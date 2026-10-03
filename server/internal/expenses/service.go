package expenses

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/bikkysamuel/splitsDemo/server/internal/groups"
	"github.com/bikkysamuel/splitsDemo/server/internal/ledger"
	"github.com/bikkysamuel/splitsDemo/server/internal/platform"
)

// Field error codes (doc 07; Split reasons come from ledger).
const (
	CodeRequired         = "required"
	CodeInvalid          = "invalid"
	CodeTooLong          = "too_long"
	CodeNotPositive      = "not_positive"
	CodeNotGroupCurrency = "not_group_currency"
	CodeNotAllowed       = "not_allowed"
	CodeConvertsToZero   = "converts_to_zero"
	CodeTooLarge         = "too_large"
	CodeNotAMember       = "not_a_member"
	CodeDuplicateMember  = "duplicate_member"
	CodeNoMembers        = "no_members"
)

// FieldError is one invalid input field. Field is a JSON Pointer without
// its leading slash, such as "amount/minor".
type FieldError struct {
	Field string
	Code  string
}

// ValidationError lists every invalid field of an Input.
type ValidationError struct {
	Fields []FieldError
}

func (e *ValidationError) Error() string {
	parts := make([]string, len(e.Fields))
	for i, f := range e.Fields {
		parts[i] = f.Field + ": " + f.Code
	}
	return "expenses: invalid input (" + strings.Join(parts, ", ") + ")"
}

// Deps are the Service's collaborators.
type Deps struct {
	Repository Repository
	Groups     GroupReader
	Clock      platform.Clock
	IDs        *platform.IDGenerator
}

// Service runs the Expense use cases. Every method takes the acting User.
type Service struct {
	deps Deps
}

// NewService returns a Service.
func NewService(deps Deps) *Service { return &Service{deps: deps} }

// Preview computes the Shares an Expense would get, without saving
// (FR-E4). It refuses exactly what Create refuses.
func (s *Service) Preview(ctx context.Context, userID, groupID platform.ID, in Input) (Computed, error) {
	g, err := s.writableGroup(ctx, userID, groupID)
	if err != nil {
		return Computed{}, err
	}
	c, _, err := compute(g, in)
	return c, err
}

// Create records an Expense for any payer (FR-E1, FR-E3). In M1 it is
// accepted at once (D9).
func (s *Service) Create(ctx context.Context, userID, groupID platform.ID, in Input) (Expense, error) {
	g, err := s.writableGroup(ctx, userID, groupID)
	if err != nil {
		return Expense{}, err
	}
	c, note, err := compute(g, in)
	if err != nil {
		return Expense{}, err
	}
	now := s.deps.Clock.Now()
	e := Expense{
		ID: s.deps.IDs.New(), GroupID: g.ID, PayerID: in.PayerID, CreatedBy: g.MyMemberID,
		Amount: c.Amount, Currency: c.Currency, Original: c.Original, Category: in.Category, Note: note,
		SpentOn: dateOnly(in.SpentOn), Method: in.Method, State: StateAccepted, Revision: 1, Version: 1, CreatedAt: now,
		Shares: inJoinOrder(g, c.Shares),
	}
	if err := s.deps.Repository.Create(ctx, e); err != nil {
		var invalid *ValidationError
		if errors.Is(err, ErrGroupClosed) || errors.As(err, &invalid) {
			return Expense{}, err
		}
		return Expense{}, fmt.Errorf("expenses: create in group %s: %w", g.ID, err)
	}
	return e, nil
}

// Get returns an Expense in a Group the User is an active Member of.
func (s *Service) Get(ctx context.Context, userID, expenseID platform.ID) (Expense, error) {
	e, err := s.deps.Repository.ExpenseForUser(ctx, expenseID, userID)
	if errors.Is(err, ErrNotFound) {
		return Expense{}, ErrNotFound
	}
	if err != nil {
		return Expense{}, fmt.Errorf("expenses: get %s: %w", expenseID, err)
	}
	return e, nil
}

// List returns one page of a Group's Expenses, newest first.
func (s *Service) List(ctx context.Context, userID, groupID platform.ID, after *Cursor, limit int) (Page, error) {
	if _, err := s.deps.Groups.Get(ctx, userID, groupID); err != nil {
		return Page{}, err
	}
	items, err := s.deps.Repository.List(ctx, groupID, after, limit+1)
	if err != nil {
		return Page{}, fmt.Errorf("expenses: list group %s: %w", groupID, err)
	}
	page := Page{Items: items}
	if len(items) > limit {
		page.Items = items[:limit]
		last := page.Items[limit-1]
		page.Next = &Cursor{SpentOn: last.SpentOn, ID: last.ID}
	}
	return page, nil
}

// writableGroup returns the Group if the User is an active Member
// (groups.ErrNotFound otherwise) and it isn't Closed (ErrGroupClosed).
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

// compute validates an Input against its Group and asks ledger for the
// Shares: the single code path behind both Preview and Create (FR-E4).
// It also returns the normalized note.
func compute(g groups.Group, in Input) (Computed, *string, error) {
	var fields []FieldError
	active := make(map[platform.ID]groups.Member)
	for _, m := range g.Members {
		if m.Status == groups.StatusActive {
			active[m.ID] = m
		}
	}
	if _, ok := active[in.PayerID]; !ok {
		fields = append(fields, FieldError{"payer_member_id", CodeNotAMember})
	}
	amount, original, f, err := toGroupCurrency(g, in)
	if err != nil {
		return Computed{}, nil, err
	}
	fields = append(fields, f...)
	if !slices.Contains(Categories, in.Category) {
		fields = append(fields, FieldError{"category", CodeInvalid})
	}
	note, f := checkNote(in.Note)
	fields = append(fields, f...)
	if in.SpentOn.IsZero() {
		fields = append(fields, FieldError{"spent_on", CodeRequired})
	}
	method, ok := splitMethods[in.Method]
	if !ok {
		fields = append(fields, FieldError{"split/method", CodeInvalid})
	}
	members := make([]ledger.SplitMember, 0, len(in.Members))
	entries := make([]*string, 0, len(in.Members))
	seen := make(map[platform.ID]bool)
	if len(in.Members) == 0 {
		fields = append(fields, FieldError{"split/members", CodeNoMembers})
	}
	for i, e := range in.Members {
		field := "split/members/" + strconv.Itoa(i)
		m, ok := active[e.MemberID]
		switch {
		case !ok:
			fields = append(fields, FieldError{field + "/member_id", CodeNotAMember})
		case seen[e.MemberID]:
			fields = append(fields, FieldError{field + "/member_id", CodeDuplicateMember})
		default:
			input, text, valid := parseInput(e.Input)
			if !valid {
				fields = append(fields, FieldError{field + "/input", CodeInvalid})
			}
			members = append(members, ledger.SplitMember{JoinSeq: m.JoinSeq, Input: input})
			entries = append(entries, text)
		}
		seen[e.MemberID] = true
	}
	if len(fields) > 0 {
		return Computed{}, nil, &ValidationError{Fields: fields}
	}
	amounts, err := ledger.Shares(amount, method, members)
	if err != nil {
		var invalid *ledger.InvalidSplitError
		if errors.As(err, &invalid) {
			field := "split"
			if invalid.MemberIndex >= 0 {
				field = "split/members/" + strconv.Itoa(invalid.MemberIndex) + "/input"
			}
			return Computed{}, nil, &ValidationError{Fields: []FieldError{{field, string(invalid.Reason)}}}
		}
		return Computed{}, nil, fmt.Errorf("expenses: compute shares: %w", err)
	}
	c := Computed{Amount: amount, Currency: g.Currency, Original: original, Shares: make([]Share, len(amounts))}
	for i, a := range amounts {
		c.Shares[i] = Share{MemberID: in.Members[i].MemberID, Amount: a, Input: entries[i]}
	}
	return c, note, nil
}

// maxRateLength bounds an Exchange Rate's text, as the contract says.
const maxRateLength = 20

// toGroupCurrency checks the Original Amount, its currency and Exchange
// Rate, returning field errors for what's wrong, and asks ledger for the
// amount in the Group Currency (FR-E5, ADR-0007).
func toGroupCurrency(g groups.Group, in Input) (int64, Original, []FieldError, error) {
	var fields []FieldError
	original := Original{Amount: in.Amount, Currency: in.Currency}
	if in.Amount <= 0 {
		fields = append(fields, FieldError{"amount/minor", CodeNotPositive})
	}
	if !ledger.IsActiveCurrency(in.Currency) {
		fields = append(fields, FieldError{"amount/currency", CodeInvalid})
	}
	var rate ledger.ExchangeRate
	switch {
	case in.ExchangeRate == nil && in.Currency != g.Currency:
		fields = append(fields, FieldError{"exchange_rate", CodeRequired})
	case in.ExchangeRate != nil && in.Currency == g.Currency:
		fields = append(fields, FieldError{"exchange_rate", CodeNotAllowed})
	case in.ExchangeRate != nil:
		var err error
		if len(*in.ExchangeRate) > maxRateLength {
			err = ledger.ErrInvalidRate
		} else {
			rate, err = ledger.ParseExchangeRate(*in.ExchangeRate)
		}
		if err != nil {
			fields = append(fields, FieldError{"exchange_rate", CodeInvalid})
		} else {
			stored := rate.String()
			original.ExchangeRate = &stored
		}
	}
	if len(fields) > 0 {
		return 0, original, fields, nil
	}
	amount, err := ledger.Convert(in.Amount, in.Currency, g.Currency, rate)
	switch {
	case errors.Is(err, ledger.ErrConvertedToZero):
		return 0, original, []FieldError{{"amount/minor", CodeConvertsToZero}}, nil
	case errors.Is(err, ledger.ErrConvertedTooLarge):
		return 0, original, []FieldError{{"amount/minor", CodeTooLarge}}, nil
	case err != nil:
		return 0, original, nil, fmt.Errorf("expenses: convert: %w", err)
	}
	return amount, original, nil, nil
}

var splitMethods = map[string]ledger.SplitMethod{
	MethodEqual: ledger.Equal, MethodExact: ledger.Exact, MethodPercentage: ledger.Percentage, MethodRatio: ledger.Ratio,
}

// decimalInput is the only shape a Split input may take: digits with an
// optional fraction, never a sign, exponent or "1/3" fraction.
var decimalInput = regexp.MustCompile(`^[0-9]{1,30}(\.[0-9]{1,8})?$`)

// parseInput reads a Split input exactly, and returns it as it will be
// stored: leading zeros dropped, scale kept ("066.70" → "66.70", as
// NUMERIC reads it back). nil stays nil (ledger decides whether the method
// needs one). It reports false for a malformed one.
func parseInput(raw *string) (*big.Rat, *string, bool) {
	if raw == nil {
		return nil, nil, true
	}
	if !decimalInput.MatchString(*raw) {
		return nil, nil, false
	}
	r, ok := new(big.Rat).SetString(*raw)
	if !ok {
		return nil, nil, false
	}
	whole, fraction, hasFraction := strings.Cut(*raw, ".")
	text := strings.TrimLeft(whole, "0")
	if text == "" {
		text = "0"
	}
	if hasFraction {
		text += "." + fraction
	}
	return r, &text, true
}

func checkNote(raw *string) (*string, []FieldError) {
	if raw == nil {
		return nil, nil
	}
	note := strings.TrimSpace(*raw)
	switch {
	case note == "":
		return nil, nil
	case utf8.RuneCountInString(note) > maxNoteLength:
		return nil, []FieldError{{"note", CodeTooLong}}
	}
	return &note, nil
}

// inJoinOrder sorts Shares by the Members' join order, as saved Expenses
// list them.
func inJoinOrder(g groups.Group, shares []Share) []Share {
	seq := make(map[platform.ID]int, len(g.Members))
	for _, m := range g.Members {
		seq[m.ID] = m.JoinSeq
	}
	sorted := slices.Clone(shares)
	slices.SortFunc(sorted, func(a, b Share) int { return seq[a.MemberID] - seq[b.MemberID] })
	return sorted
}

func dateOnly(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
