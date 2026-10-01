package ledger

import (
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"
)

// maxRateDecimals is the most decimal places an Exchange Rate may have (FR-E5).
const maxRateDecimals = 10

var rateFormat = regexp.MustCompile(fmt.Sprintf(`^[0-9]+(\.[0-9]{1,%d})?$`, maxRateDecimals))

var (
	ErrInvalidRate         = errors.New("exchange rate must be a positive decimal with at most 10 decimal places")
	ErrMissingRate         = errors.New("an exchange rate is required between different currencies")
	ErrRateForSameCurrency = errors.New("no exchange rate is allowed when the currencies are the same")
	ErrAmountNotPositive   = errors.New("original amount must be positive")
	ErrConvertedToZero     = errors.New("converted amount rounds to zero")
)

// ExchangeRate is an exact decimal: units of the Group Currency per one unit
// of the Expense currency (ADR-0007). The zero value means "no rate".
type ExchangeRate struct {
	rat      *big.Rat
	decimals int
}

// ParseExchangeRate parses a decimal string such as "0.4021": digits, an
// optional point and 1–10 decimal places, greater than zero. The value is
// kept exactly; it never passes through a float.
func ParseExchangeRate(s string) (ExchangeRate, error) {
	if !rateFormat.MatchString(s) {
		return ExchangeRate{}, fmt.Errorf("%w: %q", ErrInvalidRate, s)
	}
	r, ok := new(big.Rat).SetString(s)
	if !ok || r.Sign() <= 0 {
		return ExchangeRate{}, fmt.Errorf("%w: %q", ErrInvalidRate, s)
	}
	decimals := 0
	if _, frac, found := strings.Cut(s, "."); found {
		decimals = len(frac)
	}
	return ExchangeRate{rat: r, decimals: decimals}, nil
}

// IsZero reports whether r is the zero value (no rate).
func (r ExchangeRate) IsZero() bool { return r.rat == nil }

// String returns the rate as entered, e.g. "0.4021", for storage in a
// NUMERIC column and the API.
func (r ExchangeRate) String() string {
	if r.rat == nil {
		return ""
	}
	return r.rat.FloatString(r.decimals)
}

// Convert turns an Original Amount in from's minor units into to's minor
// units: original × rate, rounded half-up once (ADR-0007, ADR-0010). With
// equal currencies the amount is returned unchanged and no rate is allowed.
func Convert(original int64, from, to string, rate ExchangeRate) (int64, error) {
	converted, err := convert(original, from, to, rate)
	if err != nil {
		return 0, fmt.Errorf("convert %d %s to %s at %q: %w", original, from, to, rate, err)
	}
	return converted, nil
}

func convert(original int64, from, to string, rate ExchangeRate) (int64, error) {
	fromUnits, err := MinorUnits(from)
	if err != nil {
		return 0, err
	}
	toUnits, err := MinorUnits(to)
	if err != nil {
		return 0, err
	}
	if original <= 0 {
		return 0, ErrAmountNotPositive
	}
	if from == to {
		if !rate.IsZero() {
			return 0, ErrRateForSameCurrency
		}
		return original, nil
	}
	if rate.IsZero() {
		return 0, ErrMissingRate
	}

	// exact = original × rate × 10^(toUnits − fromUnits), in to's minor units.
	exact := new(big.Rat).Mul(new(big.Rat).SetInt64(original), rate.rat)
	exact.Mul(exact, pow10Rat(toUnits-fromUnits))

	converted := roundHalfUp(exact)
	if converted.Sign() == 0 {
		return 0, ErrConvertedToZero
	}
	if !converted.IsInt64() {
		return 0, fmt.Errorf("converted amount %s overflows int64", converted)
	}
	return converted.Int64(), nil
}

// roundHalfUp rounds a non-negative rational to the nearest integer, with
// exact halves going up: floor(x + 1/2).
func roundHalfUp(x *big.Rat) *big.Int {
	shifted := new(big.Rat).Add(x, big.NewRat(1, 2))
	return new(big.Int).Quo(shifted.Num(), shifted.Denom()) // Denom > 0, so Quo floors for x ≥ 0
}

func pow10Rat(n int) *big.Rat {
	p := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(abs(n))), nil)
	if n < 0 {
		return new(big.Rat).SetFrac(big.NewInt(1), p)
	}
	return new(big.Rat).SetInt(p)
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
