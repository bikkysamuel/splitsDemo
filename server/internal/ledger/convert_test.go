package ledger_test

import (
	"errors"
	"testing"

	"github.com/bikkysamuel/splitsDemo/server/internal/ledger"
)

// Expected values are worked by hand: Original Amount in major units ×
// Exchange Rate (Group Currency per unit of the Expense currency), in Group
// Currency minor units, rounded half-up once (ADR-0007, ADR-0010).
func TestConvert(t *testing.T) {
	tests := []struct {
		name     string
		original int64
		from, to string
		rate     string
		want     int64
	}{
		{"same currency needs no rate", 12345, "INR", "INR", "", 12345},
		{"USD → INR, two decimals each side", 1050, "USD", "INR", "83.25", 87413},   // 10.50 × 83.25 = 874.125 → 874.13
		{"exact half rounds up", 101, "USD", "EUR", "0.5", 51},                      // 1.01 × 0.5 = 0.505 → 0.51
		{"zero-decimal source: ¥1,000 → ₹", 1000, "JPY", "INR", "0.55", 55000},      // 1000 × 0.55 = 550.00
		{"zero-decimal target: ₹100.00 → ¥", 10000, "INR", "JPY", "1.8", 180},       // 100 × 1.8 = 180
		{"three-decimal source: KWD 1.234 → ₹", 1234, "KWD", "INR", "270.5", 33380}, // 1.234 × 270.5 = 333.797 → 333.80
		{"three-decimal target: ₹10.00 → KWD", 1000, "INR", "KWD", "0.0037", 37},    // 10 × 0.0037 = 0.037
		{"ten decimal places", 100, "USD", "INR", "83.1234567891", 8312},            // 1 × 83.1234567891 → 83.12
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rate ledger.ExchangeRate
			if tt.rate != "" {
				var err error
				if rate, err = ledger.ParseExchangeRate(tt.rate); err != nil {
					t.Fatalf("ParseExchangeRate(%q): %v", tt.rate, err)
				}
			}
			got, err := ledger.Convert(tt.original, tt.from, tt.to, rate)
			if err != nil {
				t.Fatalf("Convert: %v", err)
			}
			if got != tt.want {
				t.Errorf("Convert(%d %s → %s at %s) = %d; want %d", tt.original, tt.from, tt.to, tt.rate, got, tt.want)
			}
		})
	}
}

func TestConvertRejects(t *testing.T) {
	rate := func(s string) ledger.ExchangeRate {
		r, err := ledger.ParseExchangeRate(s)
		if err != nil {
			t.Fatalf("ParseExchangeRate(%q): %v", s, err)
		}
		return r
	}
	tests := []struct {
		name     string
		original int64
		from, to string
		rate     ledger.ExchangeRate
		want     error
	}{
		{"a result that rounds to zero", 1, "USD", "EUR", rate("0.4999999999"), ledger.ErrConvertedToZero},
		{"a missing rate between currencies", 100, "USD", "INR", ledger.ExchangeRate{}, ledger.ErrMissingRate},
		{"a rate between equal currencies", 100, "INR", "INR", rate("1"), ledger.ErrRateForSameCurrency},
		{"a non-positive Original Amount", 0, "USD", "INR", rate("83"), ledger.ErrAmountNotPositive},
		{"a malformed currency code", 100, "usd", "INR", rate("83"), ledger.ErrInvalidCurrency},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ledger.Convert(tt.original, tt.from, tt.to, tt.rate)
			if !errors.Is(err, tt.want) {
				t.Errorf("Convert error = %v; want %v", err, tt.want)
			}
		})
	}
}

func TestParseExchangeRate(t *testing.T) {
	for _, ok := range []string{"1", "0.4021", "83.25", "0.0000000001", "1234567.1234567891"} {
		if _, err := ledger.ParseExchangeRate(ok); err != nil {
			t.Errorf("ParseExchangeRate(%q) = %v; want ok", ok, err)
		}
	}
	for _, bad := range []string{"", "0", "0.0", "-1", "1.12345678901", "1e3", "1/3", ".5", "5.", " 1", "1,5", "abc"} {
		if _, err := ledger.ParseExchangeRate(bad); !errors.Is(err, ledger.ErrInvalidRate) {
			t.Errorf("ParseExchangeRate(%q) error = %v; want ErrInvalidRate", bad, err)
		}
	}
}

func TestParseExchangeRateKeepsTheExactDecimal(t *testing.T) {
	r, err := ledger.ParseExchangeRate("0.4021")
	if err != nil {
		t.Fatal(err)
	}
	if got := r.String(); got != "0.4021" {
		t.Errorf("String() = %q; want %q", got, "0.4021")
	}
}

func TestMinorUnits(t *testing.T) {
	for code, want := range map[string]int{"INR": 2, "USD": 2, "JPY": 0, "KRW": 0, "KWD": 3, "BHD": 3, "CLF": 4} {
		got, err := ledger.MinorUnits(code)
		if err != nil || got != want {
			t.Errorf("MinorUnits(%q) = %d, %v; want %d", code, got, err, want)
		}
	}
	for _, bad := range []string{"", "IN", "INRR", "inr", "1NR"} {
		if _, err := ledger.MinorUnits(bad); !errors.Is(err, ledger.ErrInvalidCurrency) {
			t.Errorf("MinorUnits(%q) error = %v; want ErrInvalidCurrency", bad, err)
		}
	}
}
