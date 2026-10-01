package ledger_test

import (
	"errors"
	"math/big"
	"strings"
	"testing"

	"github.com/bikkysamuel/splitsDemo/server/internal/ledger"
)

// Invariant 1 (doc 06): for every Split, Σ Shares = total, every Share is
// within one minor unit of its exact value, and ties are decided by join_seq
// alone, never by the order Members are listed in.
func FuzzSharesInvariants(f *testing.F) {
	f.Add(int64(100000), uint8(3), uint8(0), uint64(1))         // ₹1,000 three ways, equal
	f.Add(int64(1001), uint8(2), uint8(2), uint64(7))           // odd total, percentages
	f.Add(int64(10), uint8(2), uint8(3), uint64(42))            // small total, ratio
	f.Add(int64(1), uint8(49), uint8(0), uint64(3))             // fewer units than Members
	f.Add(int64(999999999999), uint8(17), uint8(1), uint64(99)) // large total, exact
	f.Fuzz(func(t *testing.T, total int64, n uint8, methodPick uint8, seed uint64) {
		total = 1 + abs64(total)%1_000_000_000_000_000
		count := 1 + int(n)%50
		method := []ledger.SplitMethod{ledger.Equal, ledger.Exact, ledger.Percentage, ledger.Ratio}[methodPick%4]
		rng := &splitmix{seed}
		ms := randomSplit(rng, total, count, method)
		if ms == nil {
			t.Skip("total too small for a positive exact or percentage Input per Member")
		}

		shares, err := ledger.Shares(total, method, ms)
		if err != nil {
			t.Fatalf("Shares(%d, %v, %d members): %v", total, method, count, err)
		}

		var sum int64
		exact := exactShares(total, method, ms)
		for i, s := range shares {
			sum += s
			if s < 0 {
				t.Fatalf("Share %d = %d is negative", i, s)
			}
			diff := new(big.Rat).Sub(new(big.Rat).SetInt64(s), exact[i])
			if diff.Abs(diff).Cmp(big.NewRat(1, 1)) >= 0 {
				t.Fatalf("Share %d = %d is a minor unit or more from exact %s", i, s, exact[i].FloatString(4))
			}
		}
		checkLeftoverOrder(t, shares, exact, ms)
		if sum != total {
			t.Fatalf("Σ Shares = %d; want %d", sum, total)
		}

		reversed := make([]ledger.SplitMember, len(ms))
		for i, m := range ms {
			reversed[len(ms)-1-i] = m
		}
		again, err := ledger.Shares(total, method, reversed)
		if err != nil {
			t.Fatalf("Shares on reversed Members: %v", err)
		}
		for i := range ms {
			if again[len(ms)-1-i] != shares[i] {
				t.Fatalf("Member with join_seq %d gets %d in one order and %d in the other",
					ms[i].JoinSeq, shares[i], again[len(ms)-1-i])
			}
		}
	})
}

// Conversion invariant: the result is the exact product rounded half-up
// once, so result − exact lies in (−½, ½].
func FuzzConvertRoundsHalfUpOnce(f *testing.F) {
	f.Add(int64(1050), uint64(8325), uint8(2), uint8(3), uint8(0))     // USD → INR at 83.25
	f.Add(int64(101), uint64(5), uint8(1), uint8(3), uint8(4))         // exact half
	f.Add(int64(1000), uint64(55), uint8(2), uint8(1), uint8(0))       // JPY → INR
	f.Add(int64(1234), uint64(2705), uint8(1), uint8(2), uint8(0))     // KWD → INR
	f.Add(int64(77), uint64(1234567891), uint8(2), uint8(0), uint8(5)) // large rate, two decimals
	currencies := []string{"INR", "JPY", "KWD", "USD", "EUR", "CLF"}
	f.Fuzz(func(t *testing.T, original int64, rateDigits uint64, decimals uint8, fromPick, toPick uint8) {
		original = 1 + abs64(original)%1_000_000_000_000
		from, to := currencies[int(fromPick)%len(currencies)], currencies[int(toPick)%len(currencies)]
		if from == to {
			to = currencies[(int(toPick)+1)%len(currencies)]
		}
		rateText := decimalString(1+rateDigits%10_000_000_000_000, int(decimals)%3)
		rate, err := ledger.ParseExchangeRate(rateText)
		if err != nil {
			t.Fatalf("ParseExchangeRate(%q): %v", rateText, err)
		}
		if rate.String() != rateText {
			t.Fatalf("rate %q round-trips as %q", rateText, rate.String())
		}

		got, err := ledger.Convert(original, from, to, rate)

		exact := exactConversion(t, original, from, to, rateText)
		if errors.Is(err, ledger.ErrConvertedToZero) {
			if exact.Cmp(big.NewRat(1, 2)) >= 0 {
				t.Fatalf("Convert reported zero for exact %s", exact.FloatString(6))
			}
			return
		}
		if err != nil {
			t.Fatalf("Convert(%d %s → %s at %s): %v", original, from, to, rateText, err)
		}
		diff := new(big.Rat).Sub(new(big.Rat).SetInt64(got), exact)
		if diff.Cmp(big.NewRat(-1, 2)) <= 0 || diff.Cmp(big.NewRat(1, 2)) > 0 {
			t.Fatalf("Convert(%d %s → %s at %s) = %d; exact %s is not within (−½, ½]",
				original, from, to, rateText, got, exact.FloatString(6))
		}
	})
}

// exactConversion computes original × rate in to's minor units from the
// decimal text directly, independently of ledger's arithmetic.
func exactConversion(t *testing.T, original int64, from, to, rateText string) *big.Rat {
	t.Helper()
	fromUnits, err := ledger.MinorUnits(from)
	if err != nil {
		t.Fatal(err)
	}
	toUnits, err := ledger.MinorUnits(to)
	if err != nil {
		t.Fatal(err)
	}
	intPart, frac, _ := strings.Cut(rateText, ".")
	num, _ := new(big.Int).SetString(intPart+frac, 10)
	// value = original × num / 10^len(frac) × 10^(toUnits − fromUnits)
	numer := new(big.Int).Mul(big.NewInt(original), num)
	exp := toUnits - fromUnits - len(frac)
	ten := func(n int) *big.Int { return new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(n)), nil) }
	if exp >= 0 {
		return new(big.Rat).SetInt(numer.Mul(numer, ten(exp)))
	}
	return new(big.Rat).SetFrac(numer, ten(-exp))
}

// randomSplit builds a valid Split of total among count Members with distinct
// shuffled join_seq values, or nil when total can't give every Member a
// positive exact amount.
func randomSplit(rng *splitmix, total int64, count int, method ledger.SplitMethod) []ledger.SplitMember {
	ms := make([]ledger.SplitMember, count)
	seqs := rng.perm(count)
	for i := range ms {
		ms[i].JoinSeq = seqs[i] * 3 // gaps, like real join_seq values after removals
	}
	switch method {
	case ledger.Exact:
		if total < int64(count) {
			return nil
		}
		for i, part := range rng.cuts(total, count) {
			ms[i].Input = new(big.Rat).SetInt64(part)
		}
	case ledger.Percentage:
		if count > 100_00 {
			return nil
		}
		for i, part := range rng.cuts(100_00, count) {
			ms[i].Input = big.NewRat(part, 100)
		}
	case ledger.Ratio:
		for i := range ms {
			ms[i].Input = new(big.Rat).SetInt64(1 + int64(rng.next()%1000))
		}
	}
	return ms
}

// exactShares is the oracle: each Member's exact Share straight from the
// Split method's definition (FR-E2), independent of ledger's integer weights.
func exactShares(total int64, method ledger.SplitMethod, ms []ledger.SplitMember) []*big.Rat {
	exact := make([]*big.Rat, len(ms))
	t := new(big.Rat).SetInt64(total)
	ratioSum := new(big.Rat)
	for _, m := range ms {
		if method == ledger.Ratio {
			ratioSum.Add(ratioSum, m.Input)
		}
	}
	for i, m := range ms {
		switch method {
		case ledger.Equal:
			exact[i] = new(big.Rat).Quo(t, big.NewRat(int64(len(ms)), 1))
		case ledger.Exact:
			exact[i] = new(big.Rat).Set(m.Input)
		case ledger.Percentage:
			exact[i] = new(big.Rat).Quo(new(big.Rat).Mul(t, m.Input), big.NewRat(100, 1))
		case ledger.Ratio:
			exact[i] = new(big.Rat).Quo(new(big.Rat).Mul(t, m.Input), ratioSum)
		}
	}
	return exact
}

// checkLeftoverOrder checks ADR-0010's rule directly: whenever one Member got
// a leftover minor unit and another didn't, the first has the larger
// fractional remainder, or an equal remainder and the lower join_seq.
func checkLeftoverOrder(t *testing.T, shares []int64, exact []*big.Rat, ms []ledger.SplitMember) {
	t.Helper()
	got := make([]bool, len(shares))
	frac := make([]*big.Rat, len(shares))
	for i := range shares {
		floor := new(big.Int).Quo(exact[i].Num(), exact[i].Denom()) // exact ≥ 0
		got[i] = shares[i] > floor.Int64()
		frac[i] = new(big.Rat).Sub(exact[i], new(big.Rat).SetInt(floor))
	}
	for i := range shares {
		for j := range shares {
			if !got[i] || got[j] {
				continue
			}
			c := frac[i].Cmp(frac[j])
			if c < 0 || (c == 0 && ms[i].JoinSeq > ms[j].JoinSeq) {
				t.Fatalf("join_seq %d (remainder %s) got a leftover unit but join_seq %d (remainder %s) didn't",
					ms[i].JoinSeq, frac[i].FloatString(6), ms[j].JoinSeq, frac[j].FloatString(6))
			}
		}
	}
}

func decimalString(digits uint64, decimals int) string {
	s := new(big.Int).SetUint64(digits).String()
	if decimals == 0 {
		return s
	}
	for len(s) <= decimals {
		s = "0" + s
	}
	return s[:len(s)-decimals] + "." + s[len(s)-decimals:]
}

func abs64(n int64) int64 {
	if n < 0 {
		if n == -n { // math.MinInt64
			return 0
		}
		return -n
	}
	return n
}

// splitmix is a tiny deterministic generator, so each fuzz input maps to one
// Split.
type splitmix struct{ state uint64 }

func (s *splitmix) next() uint64 {
	s.state += 0x9e3779b97f4a7c15
	z := s.state
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	return z ^ (z >> 31)
}

func (s *splitmix) perm(n int) []int {
	p := make([]int, n)
	for i := range p {
		p[i] = i + 1
	}
	for i := n - 1; i > 0; i-- {
		j := int(s.next() % uint64(i+1))
		p[i], p[j] = p[j], p[i]
	}
	return p
}

// cuts splits total into count positive parts.
func (s *splitmix) cuts(total int64, count int) []int64 {
	parts := make([]int64, count)
	rest := total - int64(count) // reserve 1 per part
	for i := 0; i < count-1; i++ {
		take := int64(0)
		if rest > 0 {
			take = int64(s.next() % uint64(rest+1))
		}
		parts[i] = 1 + take
		rest -= take
	}
	parts[count-1] = 1 + rest
	return parts
}
