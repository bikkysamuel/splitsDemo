package ledger_test

import (
	"errors"
	"math/big"
	"slices"
	"testing"

	"github.com/bikkysamuel/splitsDemo/server/internal/ledger"
)

func rat(t *testing.T, s string) *big.Rat {
	t.Helper()
	r, ok := new(big.Rat).SetString(s)
	if !ok {
		t.Fatalf("bad rational %q", s)
	}
	return r
}

// members builds Split Members with the given join_seq values and inputs
// (nil for an equal Split).
func members(t *testing.T, joinSeqs []int, inputs ...string) []ledger.SplitMember {
	t.Helper()
	ms := make([]ledger.SplitMember, len(joinSeqs))
	for i, seq := range joinSeqs {
		ms[i].JoinSeq = seq
		if len(inputs) > 0 {
			ms[i].Input = rat(t, inputs[i])
		}
	}
	return ms
}

// Expected Shares below are worked by hand from ADR-0010: floor every
// exact Share, then give leftover minor units by largest remainder, ties to
// the lowest join_seq.
func TestShares(t *testing.T) {
	tests := []struct {
		name    string
		total   int64
		method  ledger.SplitMethod
		members func(t *testing.T) []ledger.SplitMember
		want    []int64
	}{
		{
			name:  "₹1,000 three ways: the first joiner gets the extra paisa",
			total: 100000, method: ledger.Equal,
			members: func(t *testing.T) []ledger.SplitMember { return members(t, []int{1, 2, 3}) },
			want:    []int64{33334, 33333, 33333},
		},
		{
			name:  "equal tie goes by join_seq, not input order",
			total: 100, method: ledger.Equal,
			members: func(t *testing.T) []ledger.SplitMember { return members(t, []int{3, 1, 2}) },
			want:    []int64{33, 34, 33},
		},
		{
			name:  "two leftover units go to the two earliest joiners",
			total: 11, method: ledger.Equal,
			members: func(t *testing.T) []ledger.SplitMember { return members(t, []int{4, 2, 9}) },
			want:    []int64{4, 4, 3},
		},
		{
			name:  "single Member takes everything",
			total: 999, method: ledger.Equal,
			members: func(t *testing.T) []ledger.SplitMember { return members(t, []int{7}) },
			want:    []int64{999},
		},
		{
			name:  "exact amounts are kept as entered",
			total: 100000, method: ledger.Exact,
			members: func(t *testing.T) []ledger.SplitMember { return members(t, []int{1, 2}, "60000", "40000") },
			want:    []int64{60000, 40000},
		},
		{
			name:  "percentages 33.33 / 33.33 / 33.34 of ₹1,000 divide exactly",
			total: 100000, method: ledger.Percentage,
			members: func(t *testing.T) []ledger.SplitMember {
				return members(t, []int{1, 2, 3}, "33.33", "33.33", "33.34")
			},
			want: []int64{33330, 33330, 33340},
		},
		{
			name:  "50 / 50 of an odd amount: tie to the earlier joiner",
			total: 1001, method: ledger.Percentage,
			members: func(t *testing.T) []ledger.SplitMember { return members(t, []int{2, 1}, "50", "50") },
			want:    []int64{500, 501},
		},
		{
			name:  "ratio 2:1:1",
			total: 1000, method: ledger.Ratio,
			members: func(t *testing.T) []ledger.SplitMember { return members(t, []int{1, 2, 3}, "2", "1", "1") },
			want:    []int64{500, 250, 250},
		},
		{
			name:  "ratio 1:2 of 10: largest remainder beats join order",
			total: 10, method: ledger.Ratio,
			// exact 3.33… and 6.66…: floors 3 and 6, the leftover unit goes to
			// the larger remainder (0.66…) even though that Member joined later.
			members: func(t *testing.T) []ledger.SplitMember { return members(t, []int{1, 2}, "1", "2") },
			want:    []int64{3, 7},
		},
		{
			name:  "zero-decimal currency: ¥1,000 three ways",
			total: 1000, method: ledger.Equal,
			members: func(t *testing.T) []ledger.SplitMember { return members(t, []int{1, 2, 3}) },
			want:    []int64{334, 333, 333},
		},
		{
			name:  "three-decimal currency: KWD 10.000 by 1:1:1",
			total: 10000, method: ledger.Ratio,
			members: func(t *testing.T) []ledger.SplitMember { return members(t, []int{1, 2, 3}, "1", "1", "1") },
			want:    []int64{3334, 3333, 3333},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ledger.Shares(tt.total, tt.method, tt.members(t))
			if err != nil {
				t.Fatalf("Shares: %v", err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("Shares = %v; want %v", got, tt.want)
			}
		})
	}
}

func TestSharesRejectsInvalidSplits(t *testing.T) {
	tests := []struct {
		name       string
		total      int64
		method     ledger.SplitMethod
		members    func(t *testing.T) []ledger.SplitMember
		wantReason ledger.Reason
		wantMember int
	}{
		{"no Members", 100, ledger.Equal,
			func(t *testing.T) []ledger.SplitMember { return nil }, ledger.ReasonNoMembers, -1},
		{"total not positive", 0, ledger.Equal,
			func(t *testing.T) []ledger.SplitMember { return members(t, []int{1}) }, ledger.ReasonTotalNotPositive, -1},
		{"duplicate join_seq", 100, ledger.Equal,
			func(t *testing.T) []ledger.SplitMember { return members(t, []int{1, 1}) }, ledger.ReasonDuplicateMember, 1},
		{"equal Split with an input", 100, ledger.Equal,
			func(t *testing.T) []ledger.SplitMember { return members(t, []int{1, 2}, "1", "1") }, ledger.ReasonUnexpectedInput, 0},
		{"exact amounts don't sum to the total", 1000, ledger.Exact,
			func(t *testing.T) []ledger.SplitMember { return members(t, []int{1, 2}, "600", "300") },
			ledger.ReasonExactSumMismatch, -1},
		{"exact amount with a fraction of a minor unit", 1000, ledger.Exact,
			func(t *testing.T) []ledger.SplitMember { return members(t, []int{1, 2}, "600.5", "399.5") },
			ledger.ReasonNotWholeMinorUnits, 0},
		{"exact amount of zero", 1000, ledger.Exact,
			func(t *testing.T) []ledger.SplitMember { return members(t, []int{1, 2}, "1000", "0") },
			ledger.ReasonInputNotPositive, 1},
		{"percentages sum to 99.99", 1000, ledger.Percentage,
			func(t *testing.T) []ledger.SplitMember { return members(t, []int{1, 2}, "50", "49.99") },
			ledger.ReasonPercentagesNot100, -1},
		{"percentage with three decimals", 1000, ledger.Percentage,
			func(t *testing.T) []ledger.SplitMember { return members(t, []int{1, 2}, "33.335", "66.665") },
			ledger.ReasonTooManyDecimals, 0},
		{"negative percentage", 1000, ledger.Percentage,
			func(t *testing.T) []ledger.SplitMember { return members(t, []int{1, 2}, "110", "-10") },
			ledger.ReasonInputNotPositive, 1},
		{"ratio weight not an integer", 1000, ledger.Ratio,
			func(t *testing.T) []ledger.SplitMember { return members(t, []int{1, 2}, "1.5", "1") },
			ledger.ReasonRatioNotInteger, 0},
		{"ratio weight of zero", 1000, ledger.Ratio,
			func(t *testing.T) []ledger.SplitMember { return members(t, []int{1, 2}, "1", "0") },
			ledger.ReasonInputNotPositive, 1},
		{"missing input", 1000, ledger.Ratio,
			func(t *testing.T) []ledger.SplitMember {
				ms := members(t, []int{1, 2}, "1", "1")
				ms[1].Input = nil
				return ms
			}, ledger.ReasonMissingInput, 1},
		{"unknown method", 1000, ledger.SplitMethod(99),
			func(t *testing.T) []ledger.SplitMember { return members(t, []int{1}) }, ledger.ReasonUnknownMethod, -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ledger.Shares(tt.total, tt.method, tt.members(t))

			var invalid *ledger.InvalidSplitError
			if !errors.As(err, &invalid) {
				t.Fatalf("Shares error = %v; want *InvalidSplitError", err)
			}
			if invalid.Reason != tt.wantReason || invalid.MemberIndex != tt.wantMember {
				t.Errorf("got reason %q member %d; want %q member %d",
					invalid.Reason, invalid.MemberIndex, tt.wantReason, tt.wantMember)
			}
		})
	}
}
