package ledger

import (
	"errors"
	"fmt"
)

// ErrInvalidCurrency means a currency code is not three uppercase letters.
var ErrInvalidCurrency = errors.New("invalid currency code")

// minorUnitExceptions lists the ISO 4217 currencies whose minor unit is not
// two decimal places (ISO 4217 table A.1, "Minor unit" column). Every other
// well-formed code has two. Whether a code is an accepted Group or Expense
// currency is decided by the caller.
var minorUnitExceptions = map[string]int{
	// No minor unit.
	"BIF": 0, "CLP": 0, "DJF": 0, "GNF": 0, "ISK": 0, "JPY": 0, "KMF": 0, "KRW": 0, "PYG": 0,
	"RWF": 0, "UGX": 0, "UYI": 0, "VND": 0, "VUV": 0, "XAF": 0, "XOF": 0, "XPF": 0,
	// Three decimal places.
	"BHD": 3, "IQD": 3, "JOD": 3, "KWD": 3, "LYD": 3, "OMR": 3, "TND": 3,
	// Four decimal places.
	"CLF": 4, "UYW": 4,
}

// MinorUnits returns how many decimal places the currency's minor unit has:
// 2 for INR (paise), 0 for JPY, 3 for KWD (fils).
func MinorUnits(code string) (int, error) {
	if !isCurrencyCode(code) {
		return 0, fmt.Errorf("%w: %q", ErrInvalidCurrency, code)
	}
	if n, ok := minorUnitExceptions[code]; ok {
		return n, nil
	}
	return 2, nil
}

func isCurrencyCode(code string) bool {
	if len(code) != 3 {
		return false
	}
	for _, c := range code {
		if c < 'A' || c > 'Z' {
			return false
		}
	}
	return true
}
