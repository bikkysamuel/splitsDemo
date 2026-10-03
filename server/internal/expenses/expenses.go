// Package expenses owns Expenses and their Shares (FR-E*). Shares always
// come from ledger (ADR-0006), through one code path for previews and
// saves, so a preview always matches what is saved.
package expenses

import (
	"context"
	"errors"
	"time"

	"github.com/bikkysamuel/splitsDemo/server/internal/groups"
	"github.com/bikkysamuel/splitsDemo/server/internal/platform"
)

// Limits (D1).
const maxNoteLength = 500

var (
	// ErrNotFound: no such Expense, or one in a Group the User isn't an
	// active Member of (doc 07: 404, never 403).
	ErrNotFound = errors.New("expenses: not found")
	// ErrGroupClosed: the Group is Closed and read-only (FR-G6).
	ErrGroupClosed = errors.New("expenses: group closed")
)

// Categories is the fixed Category list (D7), as stored in
// expenses.category.
var Categories = []string{
	"food_drink", "groceries", "transport", "accommodation", "rent", "utilities", "entertainment", "shopping",
	"health", "travel", "other",
}

// Split methods, as stored in expenses.split_method (FR-E2).
const (
	MethodEqual      = "equal"
	MethodExact      = "exact"
	MethodPercentage = "percentage"
	MethodRatio      = "ratio"
)

// States, as stored in expenses.state (doc 06).
const (
	StatePending           = "pending"
	StateAccepted          = "accepted"
	StateDisputed          = "disputed"
	StateWithdrawalPending = "withdrawal_pending"
	StateWithdrawn         = "withdrawn"
)

// Input is an Expense as entered (FR-E1). Amount and Currency are the
// Original Amount; ExchangeRate is the decimal string entered when Currency
// isn't the Group Currency (FR-E5).
type Input struct {
	PayerID      platform.ID
	Amount       int64
	Currency     string
	ExchangeRate *string
	Category     string
	Note         *string
	SpentOn      time.Time // a date; the time is ignored
	Method       string
	Members      []SplitEntry
}

// SplitEntry is one Member of a Split and what was entered for them: minor
// units (exact), a percentage or a ratio weight, as a decimal string; nil
// for an equal Split.
type SplitEntry struct {
	MemberID platform.ID
	Input    *string
}

// Share is one Member's Share in minor units of the Group Currency, and
// what was entered for them (nil for an equal Split).
type Share struct {
	MemberID platform.ID
	Amount   int64
	Input    *string
}

// Computed is what ledger makes of an Input: the amount in the Group
// Currency, converted from the Original Amount if need be, and each
// Member's Share, in the Split's order.
type Computed struct {
	Amount   int64
	Currency string
	Original Original
	Shares   []Share
}

// Original is an Expense's Original Amount and, when its currency isn't
// the Group Currency, the Exchange Rate as stored ("83.20"; ADR-0007).
type Original struct {
	Amount       int64
	Currency     string
	ExchangeRate *string
}

// Expense is a saved Expense with its Shares in joining order.
type Expense struct {
	ID        platform.ID
	GroupID   platform.ID
	PayerID   platform.ID
	CreatedBy platform.ID
	Amount    int64 // in the Group Currency
	Currency  string
	Original  Original
	Category  string
	Note      *string
	SpentOn   time.Time
	Method    string
	State     string
	Version   int
	CreatedAt time.Time
	Shares    []Share
}

// Summary is an Expense in a list.
type Summary struct {
	ID       platform.ID
	PayerID  platform.ID
	Amount   int64
	Currency string
	Category string
	Note     *string
	SpentOn  time.Time
	State    string
}

// Cursor is where a page of Expenses ends: its last Expense.
type Cursor struct {
	SpentOn time.Time
	ID      platform.ID
}

// Page is one page of a Group's Expenses, newest first.
type Page struct {
	Items []Summary
	Next  *Cursor // nil on the last page
}

// GroupReader returns a Group the User is an active Member of, or
// groups.ErrNotFound. groups.Service implements it.
type GroupReader interface {
	Get(ctx context.Context, userID, groupID platform.ID) (groups.Group, error)
}

// Repository stores Expenses. Each method is one transaction.
type Repository interface {
	// Create stores the Expense, its Shares and an `expense_created`
	// Activity History event together (NFR-R2), with the Group's state
	// held steady; it returns ErrGroupClosed for a Closed Group, or a
	// *ValidationError if the Group Currency is no longer e.Currency.
	Create(ctx context.Context, e Expense) error
	// ExpenseForUser returns the Expense if the User is an active Member of
	// its Group, or ErrNotFound.
	ExpenseForUser(ctx context.Context, expenseID, userID platform.ID) (Expense, error)
	// List returns up to limit Expenses of the Group after the cursor (nil
	// for the first page), newest first.
	List(ctx context.Context, groupID platform.ID, after *Cursor, limit int) ([]Summary, error)
}
