// Package settlements owns Settlements (FR-S*): one Member paying another
// back, in the Group Currency.
package settlements

import (
	"context"
	"errors"
	"time"

	"github.com/bikkysamuel/splitsDemo/server/internal/balances"
	"github.com/bikkysamuel/splitsDemo/server/internal/groups"
	"github.com/bikkysamuel/splitsDemo/server/internal/platform"
)

const maxNoteLength = 500

// States, as stored in settlements.state (doc 06).
const (
	StatePending   = "pending"
	StateAccepted  = "accepted"
	StateWithdrawn = "withdrawn"
)

// WarningOverpayment: the Settlement is more than the payer owes, or more
// than the receiver is owed (FR-S2).
const WarningOverpayment = "overpayment"

var (
	// ErrNotFound: no such Settlement, or one in a Group the User isn't an
	// active Member of (404, never 403).
	ErrNotFound = errors.New("settlements: not found")
	// ErrGroupClosed: the Group is Closed and read-only (FR-G6).
	ErrGroupClosed = errors.New("settlements: group closed")
	// ErrNotCreator: only the Settlement's creator may withdraw it.
	ErrNotCreator = errors.New("settlements: not the creator")
	// ErrInvalidState: the Settlement's state doesn't allow this.
	ErrInvalidState = errors.New("settlements: invalid state")
	// ErrVersionConflict: the Settlement changed since the version sent.
	ErrVersionConflict = errors.New("settlements: version conflict")
)

// ConfirmationRequired lists the warnings the User must acknowledge
// (D16).
type ConfirmationRequired struct {
	Warnings []string
}

func (e *ConfirmationRequired) Error() string { return "settlements: confirmation required" }

// Input is a Settlement as entered (FR-S1).
type Input struct {
	From        platform.ID
	To          platform.ID
	Amount      int64
	Currency    string
	SettledOn   time.Time
	Note        *string
	Acknowledge bool
}

// Settlement is a saved Settlement.
type Settlement struct {
	ID        platform.ID
	GroupID   platform.ID
	From      platform.ID
	To        platform.ID
	Amount    int64
	Currency  string
	SettledOn time.Time
	Note      *string
	CreatedBy platform.ID
	State     string
	Version   int
	CreatedAt time.Time
}

// Cursor is where a page of Settlements ends.
type Cursor struct {
	SettledOn time.Time
	ID        platform.ID
}

// Page is one page of a Group's Settlements, newest first.
type Page struct {
	Items []Settlement
	Next  *Cursor
}

// GroupReader returns a Group the User is an active Member of, or
// groups.ErrNotFound.
type GroupReader interface {
	Get(ctx context.Context, userID, groupID platform.ID) (groups.Group, error)
}

// BalanceReader returns a Group's Balances (balances.Service).
type BalanceReader interface {
	Group(ctx context.Context, userID, groupID platform.ID) (balances.Result, error)
}

// Repository stores Settlements. Each method is one transaction.
type Repository interface {
	// Create stores the Settlement and its `settlement_recorded` Activity
	// History event together, with the Group's state held steady; it
	// returns ErrGroupClosed for a Closed Group, or a *ValidationError if
	// the Group Currency is no longer s.Currency.
	Create(ctx context.Context, s Settlement) error
	// ForUser returns the Settlement and the User's Member in its Group,
	// or ErrNotFound.
	ForUser(ctx context.Context, settlementID, userID platform.ID) (Settlement, platform.ID, error)
	// List returns up to limit Settlements after the cursor, newest first.
	List(ctx context.Context, groupID platform.ID, after *Cursor, limit int) ([]Settlement, error)
	// Withdraw moves the Settlement from fromState to Withdrawn if its
	// version is still version, writing a `settlement_withdrawn` event by
	// actor; it returns ErrVersionConflict otherwise.
	Withdraw(ctx context.Context, s Settlement, fromState string, version int, actor platform.ID, now time.Time) (Settlement, error)
}
