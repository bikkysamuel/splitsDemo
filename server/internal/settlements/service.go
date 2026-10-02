package settlements

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/bikkysamuel/splitsDemo/server/internal/groups"
	"github.com/bikkysamuel/splitsDemo/server/internal/ledger"
	"github.com/bikkysamuel/splitsDemo/server/internal/platform"
)

// Field error codes (doc 07).
const (
	CodeRequired         = "required"
	CodeTooLong          = "too_long"
	CodeNotPositive      = "not_positive"
	CodeNotGroupCurrency = "not_group_currency"
	CodeNotAMember       = "not_a_member"
	CodeSameMember       = "same_member"
)

// FieldError is one invalid input field (a JSON Pointer without its slash).
type FieldError struct {
	Field string
	Code  string
}

// ValidationError lists every invalid field.
type ValidationError struct {
	Fields []FieldError
}

func (e *ValidationError) Error() string {
	parts := make([]string, len(e.Fields))
	for i, f := range e.Fields {
		parts[i] = f.Field + ": " + f.Code
	}
	return "settlements: invalid input (" + strings.Join(parts, ", ") + ")"
}

// Deps are the Service's collaborators.
type Deps struct {
	Repository Repository
	Groups     GroupReader
	Balances   BalanceReader
	Clock      platform.Clock
	IDs        *platform.IDGenerator
}

// Service runs the Settlement use cases. Every method takes the acting User.
type Service struct {
	deps Deps
}

// NewService returns a Service.
func NewService(deps Deps) *Service { return &Service{deps: deps} }

// Record saves a Settlement (FR-S1). An overpayment needs Acknowledge
// (FR-S2, D16). In M1 it counts at once (D9).
func (s *Service) Record(ctx context.Context, userID, groupID platform.ID, in Input) (Settlement, error) {
	g, err := s.writableGroup(ctx, userID, groupID)
	if err != nil {
		return Settlement{}, err
	}
	note, err := validate(g, in)
	if err != nil {
		return Settlement{}, err
	}
	if !in.Acknowledge {
		warnings, err := s.warnings(ctx, userID, g, in)
		if err != nil {
			return Settlement{}, err
		}
		if len(warnings) > 0 {
			return Settlement{}, &ConfirmationRequired{Warnings: warnings}
		}
	}
	now := s.deps.Clock.Now()
	st := Settlement{
		ID: s.deps.IDs.New(), GroupID: g.ID, From: in.From, To: in.To, Amount: in.Amount, Currency: g.Currency,
		SettledOn: dateOnly(in.SettledOn), Note: note, CreatedBy: g.MyMemberID, State: StateAccepted, Version: 1,
		CreatedAt: now,
	}
	if err := s.deps.Repository.Create(ctx, st); err != nil {
		var invalid *ValidationError
		if errors.Is(err, ErrGroupClosed) || errors.As(err, &invalid) {
			return Settlement{}, err
		}
		return Settlement{}, fmt.Errorf("settlements: record in group %s: %w", g.ID, err)
	}
	return st, nil
}

// Get returns a Settlement in a Group the User is an active Member of.
func (s *Service) Get(ctx context.Context, userID, settlementID platform.ID) (Settlement, error) {
	st, _, err := s.deps.Repository.ForUser(ctx, settlementID, userID)
	if errors.Is(err, ErrNotFound) {
		return Settlement{}, ErrNotFound
	}
	if err != nil {
		return Settlement{}, fmt.Errorf("settlements: get %s: %w", settlementID, err)
	}
	return st, nil
}

// List returns one page of a Group's Settlements.
func (s *Service) List(ctx context.Context, userID, groupID platform.ID, after *Cursor, limit int) (Page, error) {
	if _, err := s.deps.Groups.Get(ctx, userID, groupID); err != nil {
		return Page{}, err
	}
	items, err := s.deps.Repository.List(ctx, groupID, after, limit+1)
	if err != nil {
		return Page{}, fmt.Errorf("settlements: list group %s: %w", groupID, err)
	}
	page := Page{Items: items}
	if len(items) > limit {
		page.Items = items[:limit]
		last := page.Items[limit-1]
		page.Next = &Cursor{SettledOn: last.SettledOn, ID: last.ID}
	}
	return page, nil
}

// Withdraw withdraws a Settlement (its creator only; FR-S3, FR-E6). In M1
// there is no agreement yet, so it is Withdrawn at once.
func (s *Service) Withdraw(ctx context.Context, userID, settlementID platform.ID, version int) (Settlement, error) {
	st, me, err := s.deps.Repository.ForUser(ctx, settlementID, userID)
	if errors.Is(err, ErrNotFound) {
		return Settlement{}, ErrNotFound
	}
	if err != nil {
		return Settlement{}, fmt.Errorf("settlements: get %s: %w", settlementID, err)
	}
	if _, err := s.writableGroup(ctx, userID, st.GroupID); err != nil {
		return Settlement{}, err
	}
	if st.CreatedBy != me {
		return Settlement{}, ErrNotCreator
	}
	if st.State == StateWithdrawn {
		return Settlement{}, ErrInvalidState
	}
	w, err := s.deps.Repository.Withdraw(ctx, st, st.State, version, me, s.deps.Clock.Now())
	if errors.Is(err, ErrVersionConflict) {
		return Settlement{}, ErrVersionConflict
	}
	if err != nil {
		return Settlement{}, fmt.Errorf("settlements: withdraw %s: %w", settlementID, err)
	}
	return w, nil
}

// warnings asks ledger whether the Settlement overpays (FR-S2).
func (s *Service) warnings(ctx context.Context, userID platform.ID, g groups.Group, in Input) ([]string, error) {
	result, err := s.deps.Balances.Group(ctx, userID, g.ID)
	if err != nil {
		return nil, fmt.Errorf("settlements: balances of group %s: %w", g.ID, err)
	}
	seq := make(map[platform.ID]int, len(g.Members))
	for _, m := range g.Members {
		seq[m.ID] = m.JoinSeq
	}
	bs := make([]ledger.Balance, len(result.Balances))
	for i, b := range result.Balances {
		bs[i] = ledger.Balance{JoinSeq: seq[b.MemberID], Amount: b.Amount}
	}
	over, err := ledger.Overpays(bs, seq[in.From], seq[in.To], in.Amount)
	if err != nil {
		return nil, fmt.Errorf("settlements: check overpayment: %w", err)
	}
	if over {
		return []string{WarningOverpayment}, nil
	}
	return nil, nil
}

func (s *Service) writableGroup(ctx context.Context, userID, groupID platform.ID) (groups.Group, error) {
	g, err := s.deps.Groups.Get(ctx, userID, groupID)
	if err != nil {
		return groups.Group{}, err
	}
	if g.State == groups.StateClosed {
		return groups.Group{}, ErrGroupClosed
	}
	return g, nil
}

func validate(g groups.Group, in Input) (*string, error) {
	var fields []FieldError
	active := make(map[platform.ID]bool)
	for _, m := range g.Members {
		if m.Status == groups.StatusActive {
			active[m.ID] = true
		}
	}
	if !active[in.From] {
		fields = append(fields, FieldError{"from_member_id", CodeNotAMember})
	}
	if !active[in.To] {
		fields = append(fields, FieldError{"to_member_id", CodeNotAMember})
	} else if in.To == in.From {
		fields = append(fields, FieldError{"to_member_id", CodeSameMember})
	}
	if in.Amount <= 0 {
		fields = append(fields, FieldError{"amount/minor", CodeNotPositive})
	}
	if in.Currency != g.Currency {
		fields = append(fields, FieldError{"amount/currency", CodeNotGroupCurrency})
	}
	if in.SettledOn.IsZero() {
		fields = append(fields, FieldError{"settled_on", CodeRequired})
	}
	var note *string
	if in.Note != nil {
		n := strings.TrimSpace(*in.Note)
		switch {
		case n == "":
		case utf8.RuneCountInString(n) > maxNoteLength:
			fields = append(fields, FieldError{"note", CodeTooLong})
		default:
			note = &n
		}
	}
	if len(fields) > 0 {
		return nil, &ValidationError{Fields: fields}
	}
	return note, nil
}

func dateOnly(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
