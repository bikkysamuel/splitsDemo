package ledger_test

import (
	"testing"

	"github.com/bikkysamuel/splitsDemo/server/internal/ledger"
)

// A Group or Expense currency must be an active ISO 4217 currency (Q91):
// well-formed codes that name no currency, fund codes and precious metals
// are refused.
func TestIsActiveCurrency(t *testing.T) {
	for _, code := range []string{"INR", "USD", "EUR", "JPY", "KWD", "GBP", "XOF", "XCG", "ZWG"} {
		if !ledger.IsActiveCurrency(code) {
			t.Errorf("IsActiveCurrency(%q) = false; want true", code)
		}
	}
	for _, code := range []string{"", "inr", "ZZZ", "XAU", "XXX", "XTS", "CLF", "UYI", "BOV", "HRK", "ANG", "INRR"} {
		if ledger.IsActiveCurrency(code) {
			t.Errorf("IsActiveCurrency(%q) = true; want false", code)
		}
	}
}

// Every active currency has a known minor unit.
func TestEveryActiveCurrencyHasMinorUnits(t *testing.T) {
	for _, code := range ledger.ActiveCurrencies() {
		if _, err := ledger.MinorUnits(code); err != nil {
			t.Errorf("MinorUnits(%q): %v", code, err)
		}
	}
	if n := len(ledger.ActiveCurrencies()); n < 150 {
		t.Errorf("%d active currencies; want the full ISO 4217 list", n)
	}
}
