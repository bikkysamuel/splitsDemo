// Package ledger holds all money arithmetic (ADR-0006): Shares for every
// Split method, currency conversion, and (later) Balances and Settle-up
// Suggestions. It is pure: no I/O, no clock, and no floating-point types.
// Amounts are integer minor units; rates and inputs are exact big.Rat values.
package ledger

import (
	"fmt"
	"math/big"
	"sort"
)

// Method is how an Expense is divided among the Members of its Split.
type Method int

const (
	// Equal divides the total equally; Members carry no Input.
	Equal Method = iota + 1
	// Exact takes each Member's Input as their Share in minor units; the
	// Inputs must sum to the total.
	Exact
	// Percentage takes each Input as a percentage with at most two decimal
	// places; the Inputs must sum to exactly 100.
	Percentage
	// Ratio takes each Input as a positive integer weight (2:1:1).
	Ratio
)

// SplitMember is one Member of a Split.
type SplitMember struct {
	// JoinSeq is the Member's join order in the Group; it breaks rounding
	// ties (ADR-0010) and must be unique within the Split.
	JoinSeq int
	// Input is the amount, percentage or weight entered for this Member, as
	// stored in expense_shares.input. Nil for an Equal Split.
	Input *big.Rat
}

// Shares returns each Member's Share of total, in minor units and in the
// order of members. The Shares always sum exactly to total: every exact Share
// is floored and the leftover minor units go one at a time by largest
// remainder, ties to the lowest JoinSeq (ADR-0010).
func Shares(total int64, method Method, members []SplitMember) ([]int64, error) {
	weights, err := validate(total, method, members)
	if err != nil {
		return nil, err
	}
	return allocate(total, weights, members), nil
}

// validate checks the Split and returns each Member's integer weight: the
// Share is total × weight / Σ weights.
func validate(total int64, method Method, members []SplitMember) ([]*big.Int, error) {
	if method < Equal || method > Ratio {
		return nil, invalid(ReasonUnknownMethod, -1)
	}
	if len(members) == 0 {
		return nil, invalid(ReasonNoMembers, -1)
	}
	if total <= 0 {
		return nil, invalid(ReasonTotalNotPositive, -1)
	}
	seen := make(map[int]bool, len(members))
	for i, m := range members {
		if seen[m.JoinSeq] {
			return nil, invalid(ReasonDuplicateMember, i)
		}
		seen[m.JoinSeq] = true
	}

	weights := make([]*big.Int, len(members))
	sum := new(big.Int)
	for i, m := range members {
		w, err := weight(method, m.Input, i)
		if err != nil {
			return nil, err
		}
		weights[i] = w
		sum.Add(sum, w)
	}

	switch method {
	case Exact:
		if sum.Cmp(big.NewInt(total)) != 0 {
			return nil, invalid(ReasonExactSumMismatch, -1)
		}
	case Percentage:
		if sum.Cmp(hundredPercent) != 0 {
			return nil, invalid(ReasonPercentagesNot100, -1)
		}
	}
	return weights, nil
}

// hundredPercent is 100% in hundredths of a percent, the Percentage weight unit.
var hundredPercent = big.NewInt(100_00)

func weight(method Method, input *big.Rat, member int) (*big.Int, error) {
	if method == Equal {
		if input != nil {
			return nil, invalid(ReasonUnexpectedInput, member)
		}
		return big.NewInt(1), nil
	}
	if input == nil {
		return nil, invalid(ReasonMissingInput, member)
	}
	if input.Sign() <= 0 {
		return nil, invalid(ReasonInputNotPositive, member)
	}
	switch method {
	case Exact:
		if !input.IsInt() {
			return nil, invalid(ReasonNotWholeMinorUnits, member)
		}
		return new(big.Int).Set(input.Num()), nil
	case Percentage:
		hundredths := new(big.Rat).Mul(input, big.NewRat(100, 1))
		if !hundredths.IsInt() {
			return nil, invalid(ReasonTooManyDecimals, member)
		}
		return new(big.Int).Set(hundredths.Num()), nil
	default: // Ratio
		if !input.IsInt() {
			return nil, invalid(ReasonRatioNotInteger, member)
		}
		return new(big.Int).Set(input.Num()), nil
	}
}

// allocate divides total in proportion to weights by floor, then largest
// remainder, ties to the lowest JoinSeq.
func allocate(total int64, weights []*big.Int, members []SplitMember) []int64 {
	sum := new(big.Int)
	for _, w := range weights {
		sum.Add(sum, w)
	}
	shares := make([]int64, len(weights))
	remainders := make([]*big.Int, len(weights))
	leftover := total
	for i, w := range weights {
		q, r := new(big.Int).QuoRem(new(big.Int).Mul(big.NewInt(total), w), sum, new(big.Int))
		shares[i] = q.Int64() // q ≤ total, so it fits
		remainders[i] = r
		leftover -= shares[i]
	}

	order := make([]int, len(weights))
	for i := range order {
		order[i] = i
	}
	sort.Slice(order, func(a, b int) bool {
		i, j := order[a], order[b]
		if c := remainders[i].Cmp(remainders[j]); c != 0 {
			return c > 0
		}
		return members[i].JoinSeq < members[j].JoinSeq
	})
	for _, i := range order[:leftover] {
		shares[i]++
	}
	return shares
}

// Reason says why a Split is invalid. Values are stable and double as the
// `code` of a problem+json field error.
type Reason string

const (
	ReasonUnknownMethod      Reason = "unknown_method"
	ReasonNoMembers          Reason = "no_members"
	ReasonTotalNotPositive   Reason = "total_not_positive"
	ReasonDuplicateMember    Reason = "duplicate_member"
	ReasonUnexpectedInput    Reason = "unexpected_input"
	ReasonMissingInput       Reason = "missing_input"
	ReasonInputNotPositive   Reason = "input_not_positive"
	ReasonNotWholeMinorUnits Reason = "not_whole_minor_units"
	ReasonExactSumMismatch   Reason = "exact_sum_mismatch"
	ReasonTooManyDecimals    Reason = "too_many_decimals"
	ReasonPercentagesNot100  Reason = "percentages_not_100"
	ReasonRatioNotInteger    Reason = "ratio_not_integer"
)

// InvalidSplitError reports an invalid Split.
type InvalidSplitError struct {
	Reason Reason
	// Member is the index of the offending Member, or -1 when the Split as a
	// whole is at fault.
	Member int
}

func (e *InvalidSplitError) Error() string {
	if e.Member < 0 {
		return fmt.Sprintf("invalid split: %s", e.Reason)
	}
	return fmt.Sprintf("invalid split: member %d: %s", e.Member, e.Reason)
}

func invalid(reason Reason, member int) error {
	return &InvalidSplitError{Reason: reason, Member: member}
}
