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

// activeCurrencies are the ISO 4217 codes of currencies in circulation
// (table A.1, as of 2026), without fund codes (BOV, CLF, UYI…), precious
// metals (XAU…) and testing codes (XTS, XXX). A Group or Expense currency
// must be one of these (Q91).
var activeCurrencies = []string{
	"AED", "AFN", "ALL", "AMD", "AOA", "ARS", "AUD", "AWG", "AZN", "BAM", "BBD", "BDT", "BHD", "BIF", "BMD",
	"BND", "BOB", "BRL", "BSD", "BTN", "BWP", "BYN", "BZD", "CAD", "CDF", "CHF", "CLP", "CNY", "COP", "CRC",
	"CUP", "CVE", "CZK", "DJF", "DKK", "DOP", "DZD", "EGP", "ERN", "ETB", "EUR", "FJD", "FKP", "GBP", "GEL",
	"GHS", "GIP", "GMD", "GNF", "GTQ", "GYD", "HKD", "HNL", "HTG", "HUF", "IDR", "ILS", "INR", "IQD", "IRR",
	"ISK", "JMD", "JOD", "JPY", "KES", "KGS", "KHR", "KMF", "KPW", "KRW", "KWD", "KYD", "KZT", "LAK", "LBP",
	"LKR", "LRD", "LSL", "LYD", "MAD", "MDL", "MGA", "MKD", "MMK", "MNT", "MOP", "MRU", "MUR", "MVR", "MWK",
	"MXN", "MYR", "MZN", "NAD", "NGN", "NIO", "NOK", "NPR", "NZD", "OMR", "PAB", "PEN", "PGK", "PHP", "PKR",
	"PLN", "PYG", "QAR", "RON", "RSD", "RUB", "RWF", "SAR", "SBD", "SCR", "SDG", "SEK", "SGD", "SHP", "SLE",
	"SOS", "SRD", "SSP", "STN", "SVC", "SYP", "SZL", "THB", "TJS", "TMT", "TND", "TOP", "TRY", "TTD", "TWD",
	"TZS", "UAH", "UGX", "USD", "UYU", "UZS", "VED", "VES", "VND", "VUV", "WST", "XAF", "XCD", "XCG", "XOF",
	"XPF", "YER", "ZAR", "ZMW", "ZWG",
}

var activeCurrencySet = func() map[string]bool {
	set := make(map[string]bool, len(activeCurrencies))
	for _, c := range activeCurrencies {
		set[c] = true
	}
	return set
}()

// IsActiveCurrency reports whether code is an active ISO 4217 currency.
func IsActiveCurrency(code string) bool { return activeCurrencySet[code] }

// ActiveCurrencies lists the active ISO 4217 codes, alphabetically.
func ActiveCurrencies() []string { return append([]string(nil), activeCurrencies...) }
