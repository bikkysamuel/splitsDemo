package groups

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/bikkysamuel/splitsDemo/server/internal/auth"
	"github.com/bikkysamuel/splitsDemo/server/internal/ledger"
	"github.com/bikkysamuel/splitsDemo/server/internal/platform"
)

// Field error codes, sent as problem+json `errors[].code` (doc 07).
const (
	CodeRequired = "required"
	CodeInvalid  = "invalid"
	CodeTooLong  = "too_long"
	CodeTaken    = "taken"
)

// FieldError is one invalid input field.
type FieldError struct {
	Field string
	Code  string
}

// ValidationError lists every invalid field of a request.
type ValidationError struct {
	Fields []FieldError
}

func (e *ValidationError) Error() string {
	parts := make([]string, len(e.Fields))
	for i, f := range e.Fields {
		parts[i] = f.Field + ": " + f.Code
	}
	return "groups: invalid input (" + strings.Join(parts, ", ") + ")"
}

// Deps are the Service's collaborators.
type Deps struct {
	Repository Repository
	Clock      platform.Clock
	IDs        *platform.IDGenerator
}

// Service runs the Group use cases. Every method takes the acting User.
type Service struct {
	deps Deps
}

// NewService returns a Service.
func NewService(deps Deps) *Service { return &Service{deps: deps} }

// Create makes a Group with the User as its first Admin (FR-G1).
func (s *Service) Create(ctx context.Context, userID platform.ID, name, currency, displayName string) (Group, error) {
	var fields []FieldError
	name, f := checkGroupName(name)
	fields = append(fields, f...)
	if !ledger.IsActiveCurrency(currency) {
		fields = append(fields, FieldError{"currency", CodeInvalid})
	}
	displayName, f = checkDisplayName(displayName)
	fields = append(fields, f...)
	if len(fields) > 0 {
		return Group{}, &ValidationError{Fields: fields}
	}
	g := NewGroup{
		ID: s.deps.IDs.New(), Name: name, Currency: currency,
		CreatorID: userID, MemberID: s.deps.IDs.New(), DisplayName: displayName,
		Now: s.deps.Clock.Now(),
	}
	if err := s.deps.Repository.CreateGroup(ctx, g, MaxGroupsPerUser); err != nil {
		if errors.Is(err, ErrGroupLimit) {
			return Group{}, ErrGroupLimit
		}
		return Group{}, fmt.Errorf("groups: create group: %w", err)
	}
	return s.Get(ctx, userID, g.ID)
}

// Get returns a Group the User is an active Member of, or ErrNotFound.
func (s *Service) Get(ctx context.Context, userID, groupID platform.ID) (Group, error) {
	g, err := s.deps.Repository.GroupForUser(ctx, groupID, userID)
	if errors.Is(err, ErrNotFound) {
		return Group{}, ErrNotFound
	}
	if err != nil {
		return Group{}, fmt.Errorf("groups: get group %s: %w", groupID, err)
	}
	return g, nil
}

// List returns one page of the User's Groups. after is the previous page's
// Next, or nil for the first page.
func (s *Service) List(ctx context.Context, userID platform.ID, after *platform.ID, limit int) (Page, error) {
	var from platform.ID
	if after != nil {
		from = *after
	}
	// One extra row tells whether another page follows.
	items, err := s.deps.Repository.ListGroups(ctx, userID, from, limit+1)
	if err != nil {
		return Page{}, fmt.Errorf("groups: list groups: %w", err)
	}
	page := Page{Items: items}
	if len(items) > limit {
		page.Items = items[:limit]
		next := page.Items[limit-1].ID
		page.Next = &next
	}
	return page, nil
}

// Rename renames a Group (Admins only, FR-G3).
func (s *Service) Rename(ctx context.Context, userID, groupID platform.ID, name string, version int) (Group, error) {
	g, err := s.Get(ctx, userID, groupID)
	if err != nil {
		return Group{}, err
	}
	if g.MyRole != RoleAdmin {
		return Group{}, ErrAdminRequired
	}
	name, fields := checkGroupName(name)
	if len(fields) > 0 {
		return Group{}, &ValidationError{Fields: fields}
	}
	err = s.deps.Repository.RenameGroup(ctx, groupID, name, version, s.deps.Clock.Now())
	if errors.Is(err, ErrVersionConflict) {
		return Group{}, ErrVersionConflict
	}
	if err != nil {
		return Group{}, fmt.Errorf("groups: rename group %s: %w", groupID, err)
	}
	return s.Get(ctx, userID, groupID)
}

// AddMember adds a person to a Group the User is an active Member of
// (FR-M1, FR-M2): by email (an existing verified User joins at once,
// anyone else becomes a Placeholder carrying it) or as a name-only
// Placeholder.
func (s *Service) AddMember(ctx context.Context, userID, groupID platform.ID, displayName string, rawEmail *string) (Member, error) {
	if _, err := s.Get(ctx, userID, groupID); err != nil {
		return Member{}, err
	}
	var fields []FieldError
	displayName, f := checkDisplayName(displayName)
	fields = append(fields, f...)
	var email *string
	if rawEmail != nil && strings.TrimSpace(*rawEmail) != "" {
		e, err := auth.NormalizeEmail(*rawEmail)
		if err != nil {
			fields = append(fields, FieldError{"email", CodeInvalid})
		} else {
			email = &e
		}
	}
	if len(fields) > 0 {
		return Member{}, &ValidationError{Fields: fields}
	}
	m, err := s.deps.Repository.AddMember(ctx, NewMember{
		ID: s.deps.IDs.New(), GroupID: groupID, DisplayName: displayName, Email: email, Now: s.deps.Clock.Now(),
	}, MaxMembersPerGroup)
	var invalid *ValidationError
	switch {
	case errors.As(err, &invalid), errors.Is(err, ErrMemberLimit), errors.Is(err, ErrGroupClosed):
		return Member{}, err
	case err != nil:
		return Member{}, fmt.Errorf("groups: add member to %s: %w", groupID, err)
	}
	return m, nil
}

// MakeAdmin grants Admin to a Member (Admins only, FR-G3).
func (s *Service) MakeAdmin(ctx context.Context, userID, groupID, memberID platform.ID, version int) (Member, error) {
	g, err := s.Get(ctx, userID, groupID)
	if err != nil {
		return Member{}, err
	}
	if g.MyRole != RoleAdmin {
		return Member{}, ErrAdminRequired
	}
	m, err := s.deps.Repository.MakeAdmin(ctx, groupID, memberID, version, s.deps.Clock.Now())
	switch {
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrMemberNotEligible), errors.Is(err, ErrVersionConflict):
		return Member{}, err
	case err != nil:
		return Member{}, fmt.Errorf("groups: make member %s admin: %w", memberID, err)
	}
	return m, nil
}

func checkGroupName(raw string) (string, []FieldError) {
	name := strings.TrimSpace(raw)
	switch {
	case name == "":
		return "", []FieldError{{"name", CodeRequired}}
	case utf8.RuneCountInString(name) > maxGroupNameLength:
		return "", []FieldError{{"name", CodeTooLong}}
	}
	return name, nil
}

// checkDisplayName trims a Member's display name and checks its length;
// uniqueness within the Group is the store's (FR-M4).
func checkDisplayName(raw string) (string, []FieldError) {
	name := strings.TrimSpace(raw)
	switch {
	case name == "":
		return "", []FieldError{{"display_name", CodeRequired}}
	case utf8.RuneCountInString(name) > maxDisplayNameLen:
		return "", []FieldError{{"display_name", CodeTooLong}}
	}
	return name, nil
}
