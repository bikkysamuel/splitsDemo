// Package groups owns Groups and their Members (FR-G*, FR-M*): who may see
// and change a Group is decided here, never in handlers (doc 07).
package groups

import (
	"context"
	"errors"
	"time"

	"github.com/bikkysamuel/splitsDemo/server/internal/platform"
)

// Limits (FR-G2).
const (
	MaxGroupsPerUser   = 200
	MaxMembersPerGroup = 50
	maxGroupNameLength = 100
	maxDisplayNameLen  = 50
)

var (
	// ErrNotFound: no such Group, or one the User isn't an active Member of.
	// The two look the same, so Group IDs aren't revealed (doc 07).
	ErrNotFound = errors.New("groups: not found")
	// ErrAdminRequired: only an Admin may do this (FR-G3).
	ErrAdminRequired = errors.New("groups: admin required")
	// ErrVersionConflict: the Group changed since the version sent (NFR-R4).
	ErrVersionConflict = errors.New("groups: version conflict")
	// ErrGroupLimit: the User is already in MaxGroupsPerUser Groups.
	ErrGroupLimit = errors.New("groups: group limit reached")
	// ErrMemberLimit: the Group already has MaxMembersPerGroup Members.
	ErrMemberLimit = errors.New("groups: member limit reached")
	// ErrMemberNotEligible: a Placeholder or Former Member can't be an Admin.
	ErrMemberNotEligible = errors.New("groups: member not eligible")
	// ErrGroupClosed: the Group is Closed and read-only (FR-G6).
	ErrGroupClosed = errors.New("groups: group closed")
)

// Role is an Admin or an ordinary Member.
type Role string

// Roles, as stored in members.role.
const (
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
)

// MemberStatus is active or Former (FR-M7).
type MemberStatus string

// Member statuses, as stored in members.status.
const (
	StatusActive MemberStatus = "active"
	StatusFormer MemberStatus = "former"
)

// State is a Group's lifecycle state (FR-G6).
type State string

// Group states, as stored in groups.state.
const (
	StateActive  State = "active"
	StateClosing State = "closing"
	StateClosed  State = "closed"
)

// Summary is a Group in a User's list.
type Summary struct {
	ID       platform.ID
	Name     string
	Currency string
	State    State
}

// Member is a person's place in one Group.
type Member struct {
	ID          platform.ID
	UserID      *platform.ID // nil for a Placeholder Member
	DisplayName string
	Role        Role
	Status      MemberStatus
	JoinSeq     int
	Version     int
}

// Placeholder reports whether no User is linked to the Member yet.
func (m Member) Placeholder() bool { return m.UserID == nil }

// Group is a Group as one of its Members sees it.
type Group struct {
	Summary
	Version    int
	MyMemberID platform.ID
	MyRole     Role
	Members    []Member
}

// Page is one page of a User's Groups.
type Page struct {
	Items []Summary
	// Next is the last Group's ID when more may follow; nil on the last page.
	Next *platform.ID
}

// NewGroup is a Group and its first Admin to store.
type NewGroup struct {
	ID          platform.ID
	Name        string
	Currency    string
	CreatorID   platform.ID // the User
	MemberID    platform.ID
	DisplayName string
	Now         time.Time
}

// NewMember is a Member to add. The store links it to the verified User
// with Email, if there is one; otherwise it is a Placeholder carrying Email.
type NewMember struct {
	ID          platform.ID
	GroupID     platform.ID
	DisplayName string
	Email       *string
	Now         time.Time
}

// Repository stores Groups and Members. Each method is one transaction.
type Repository interface {
	// CreateGroup stores the Group with its creator as first Admin, unless
	// the creator is already an active Member of maxGroups Groups
	// (ErrGroupLimit). Concurrent creations by one User are serialized.
	CreateGroup(ctx context.Context, g NewGroup, maxGroups int) error
	// GroupForUser returns the Group with its Members if the User is an
	// active Member of it, or ErrNotFound.
	GroupForUser(ctx context.Context, groupID, userID platform.ID) (Group, error)
	// ListGroups returns up to limit of the User's Groups with IDs after
	// `after` (the zero ID for the first page), in ID order.
	ListGroups(ctx context.Context, userID, after platform.ID, limit int) ([]Summary, error)
	// RenameGroup renames the Group if its version is still version, or
	// returns ErrVersionConflict.
	RenameGroup(ctx context.Context, groupID platform.ID, name string, version int, now time.Time) error
	// AddMember adds m with the Group row locked, so join_seq is the next
	// one and the limit holds. It returns ErrGroupClosed, ErrMemberLimit
	// (maxMembers counts every Member), or a *ValidationError with code
	// CodeTaken for an email or display name already in the Group.
	AddMember(ctx context.Context, m NewMember, maxMembers int) (Member, error)
	// MakeAdmin makes the Member an Admin if its version is still version.
	// It returns ErrNotFound (no such Member in the Group),
	// ErrMemberNotEligible (a Placeholder or Former Member) or
	// ErrVersionConflict.
	MakeAdmin(ctx context.Context, groupID, memberID platform.ID, version int, now time.Time) (Member, error)
}
