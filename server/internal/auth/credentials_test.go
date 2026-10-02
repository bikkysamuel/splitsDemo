package auth_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/bikkysamuel/splitsDemo/server/internal/auth"
)

// FR-A1: emails are trimmed and compared ignoring case; the database
// compares them with citext, so normalizing only trims and checks shape.
func TestNormalizeEmailTrimsWhitespace(t *testing.T) {
	got, err := auth.NormalizeEmail("  Alice@Example.com\t")
	if err != nil {
		t.Fatalf("NormalizeEmail: %v", err)
	}
	if got != "Alice@Example.com" {
		t.Errorf("NormalizeEmail = %q; want %q", got, "Alice@Example.com")
	}
}

func TestNormalizeEmailRefusesMalformedAddresses(t *testing.T) {
	for _, raw := range []string{
		"",
		"   ",
		"alice",
		"alice@",
		"@example.com",
		"alice@@example.com",
		"Alice <alice@example.com>",
		"al ice@example.com",
		strings.Repeat("a", 243) + "@example.com", // 255 characters
	} {
		_, err := auth.NormalizeEmail(raw)
		var fe auth.FieldError
		if !errors.As(err, &fe) || fe.Field != "email" {
			t.Errorf("NormalizeEmail(%q) = %v; want an email FieldError", raw, err)
			continue
		}
		want := auth.CodeInvalid
		if strings.TrimSpace(raw) == "" {
			want = auth.CodeRequired
		}
		if fe.Code != want {
			t.Errorf("NormalizeEmail(%q) code = %q; want %q", raw, fe.Code, want)
		}
	}
}

// FR-A3: 10–128 characters, no composition rules, not on the shipped list.
func TestCheckNewPassword(t *testing.T) {
	for _, tc := range []struct {
		password string
		want     string // "" means accepted
	}{
		{"correct horse battery", ""},
		{"ten chars!", ""},
		{"nine char", auth.CodeTooShort},
		{"ßßßßßßßßß", auth.CodeTooShort}, // 9 characters, 18 bytes
		{"éééééééééé", ""},               // 10 characters
		{strings.Repeat("x", 128), ""},
		{strings.Repeat("x", 129), auth.CodeTooLong},
		{"password123", auth.CodeTooCommon},
		{"Password123", auth.CodeTooCommon}, // compared ignoring case
		{"qwertyuiop", auth.CodeTooCommon},
	} {
		err := auth.CheckNewPassword(tc.password)
		if tc.want == "" {
			if err != nil {
				t.Errorf("CheckNewPassword(%q) = %v; want accepted", tc.password, err)
			}
			continue
		}
		var fe auth.FieldError
		if !errors.As(err, &fe) || fe.Field != "password" || fe.Code != tc.want {
			t.Errorf("CheckNewPassword(%q) = %v; want password FieldError %q", tc.password, err, tc.want)
		}
	}
}

// Every entry of the shipped list is refused; the list only holds
// passwords that would otherwise pass the length rule.
func TestEveryShippedCommonPasswordIsRefused(t *testing.T) {
	list := auth.CommonPasswords()
	if len(list) < 5000 {
		t.Fatalf("shipped list has %d entries; want the full list", len(list))
	}
	for _, pw := range list {
		var fe auth.FieldError
		if err := auth.CheckNewPassword(pw); !errors.As(err, &fe) || fe.Code != auth.CodeTooCommon {
			t.Fatalf("CheckNewPassword(%q) = %v; want too_common", pw, err)
		}
	}
}

func TestHashPasswordUsesArgon2idAndVerifies(t *testing.T) {
	encoded, err := auth.HashPassword("correct horse battery", auth.TestPasswordParams)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if !strings.HasPrefix(encoded, "$argon2id$v=19$") {
		t.Errorf("encoded hash %q is not an argon2id PHC string", encoded)
	}
	if strings.Contains(encoded, "correct horse battery") {
		t.Error("encoded hash contains the password")
	}

	if ok, err := auth.VerifyPassword(encoded, "correct horse battery"); err != nil || !ok {
		t.Errorf("VerifyPassword(right password) = %v, %v; want true", ok, err)
	}
	if ok, err := auth.VerifyPassword(encoded, "correct horse batterY"); err != nil || ok {
		t.Errorf("VerifyPassword(wrong password) = %v, %v; want false", ok, err)
	}
}

func TestHashPasswordSaltsEveryHash(t *testing.T) {
	a, _ := auth.HashPassword("correct horse battery", auth.TestPasswordParams)
	b, _ := auth.HashPassword("correct horse battery", auth.TestPasswordParams)
	if a == b {
		t.Error("two hashes of the same password are equal; want a fresh salt each time")
	}
}

// The parameters travel with the hash, so raising them later still verifies
// old hashes.
func TestVerifyPasswordReadsTheParametersFromTheHash(t *testing.T) {
	encoded, err := auth.HashPassword("correct horse battery",
		auth.PasswordParams{Memory: 8 * 1024, Iterations: 2, Parallelism: 1})
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if ok, err := auth.VerifyPassword(encoded, "correct horse battery"); err != nil || !ok {
		t.Errorf("VerifyPassword = %v, %v; want true", ok, err)
	}
}

func TestVerifyPasswordRejectsMalformedHashes(t *testing.T) {
	for _, encoded := range []string{
		"",
		"plain",
		"$argon2i$v=19$m=65536,t=3,p=2$c2FsdA$aGFzaA",
		"$argon2id$v=18$m=65536,t=3,p=2$c2FsdA$aGFzaA",
		"$argon2id$v=19$m=x,t=3,p=2$c2FsdA$aGFzaA",
		"$argon2id$v=19$m=65536,t=3,p=2$!!$aGFzaA",
	} {
		if _, err := auth.VerifyPassword(encoded, "correct horse battery"); err == nil {
			t.Errorf("VerifyPassword(%q) = nil error; want malformed", encoded)
		}
	}
}
