package auth

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"crypto/subtle"
	_ "embed"
	"encoding/base64"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"sync"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
)

// Field error codes, sent as problem+json `errors[].code` (doc 07).
const (
	CodeRequired  = "required"
	CodeInvalid   = "invalid"
	CodeTooShort  = "too_short"
	CodeTooLong   = "too_long"
	CodeTooCommon = "too_common"
)

// FieldError is one invalid input field. Field is the request field name.
type FieldError struct {
	Field string
	Code  string
}

func (e FieldError) Error() string { return "auth: " + e.Field + " is " + e.Code }

// ValidationError lists every invalid field of a request.
type ValidationError struct {
	Fields []FieldError
}

func (e *ValidationError) Error() string {
	parts := make([]string, len(e.Fields))
	for i, f := range e.Fields {
		parts[i] = f.Field + ": " + f.Code
	}
	return "auth: invalid input (" + strings.Join(parts, ", ") + ")"
}

const (
	maxEmailLength    = 254
	minPasswordLength = 10
	maxPasswordLength = 128
)

// NormalizeEmail trims surrounding whitespace and checks that what remains is
// a bare address (FR-A1). Case is kept as typed: the database compares
// emails ignoring case (citext).
func NormalizeEmail(raw string) (string, error) {
	email := strings.TrimSpace(raw)
	if email == "" {
		return "", FieldError{Field: "email", Code: CodeRequired}
	}
	if utf8.RuneCountInString(email) > maxEmailLength {
		return "", FieldError{Field: "email", Code: CodeInvalid}
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email || addr.Name != "" || strings.ContainsAny(email, " \t\r\n") {
		return "", FieldError{Field: "email", Code: CodeInvalid}
	}
	return email, nil
}

// CheckNewPassword applies the password rules (FR-A3): 10–128 characters,
// no composition rules, not on the shipped common/breached list (compared
// ignoring case).
func CheckNewPassword(password string) error {
	switch n := utf8.RuneCountInString(password); {
	case n < minPasswordLength:
		return FieldError{Field: "password", Code: CodeTooShort}
	case n > maxPasswordLength:
		return FieldError{Field: "password", Code: CodeTooLong}
	}
	if commonPasswordSet()[strings.ToLower(password)] {
		return FieldError{Field: "password", Code: CodeTooCommon}
	}
	return nil
}

//go:embed passwords/common.txt
var commonPasswordsFile []byte

var commonPasswordSet = sync.OnceValue(func() map[string]bool {
	set := make(map[string]bool)
	for _, pw := range parseCommonPasswords() {
		set[pw] = true
	}
	return set
})

// CommonPasswords returns the shipped list of refused passwords.
func CommonPasswords() []string { return parseCommonPasswords() }

func parseCommonPasswords() []string {
	var list []string
	sc := bufio.NewScanner(bytes.NewReader(commonPasswordsFile))
	for sc.Scan() {
		line := sc.Text()
		if line == "" || strings.HasPrefix(line, "# ") {
			continue
		}
		list = append(list, line)
	}
	return list
}

// PasswordParams are Argon2id cost parameters. They are stored with each
// hash, so they can be raised without invalidating older hashes.
type PasswordParams struct {
	Memory      uint32 // KiB
	Iterations  uint32
	Parallelism uint8
}

// DefaultPasswordParams take about 250 ms on the reference setup (doc 08),
// measured on an Apple-silicon Mac (2026-10-01): 64 MiB, 16 passes, 2 lanes.
var DefaultPasswordParams = PasswordParams{Memory: 64 * 1024, Iterations: 16, Parallelism: 2}

// TestPasswordParams are deliberately cheap so tests stay fast. Never use
// them outside tests.
var TestPasswordParams = PasswordParams{Memory: 1024, Iterations: 1, Parallelism: 1}

const (
	saltLength = 16
	keyLength  = 32
)

var errMalformedHash = errors.New("auth: malformed password hash")

// HashPassword returns an Argon2id hash of password in PHC string format:
// $argon2id$v=19$m=<KiB>,t=<iterations>,p=<parallelism>$<salt>$<hash>.
func HashPassword(password string, p PasswordParams) (string, error) {
	salt := make([]byte, saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("auth: password salt: %w", err)
	}
	key := argon2.IDKey([]byte(password), salt, p.Iterations, p.Memory, p.Parallelism, keyLength)
	b64 := base64.RawStdEncoding
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, p.Memory, p.Iterations, p.Parallelism, b64.EncodeToString(salt), b64.EncodeToString(key)), nil
}

// VerifyPassword reports whether password matches encoded, in constant time.
// It returns an error only when encoded is not a hash HashPassword made.
func VerifyPassword(encoded, password string) (bool, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return false, errMalformedHash
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return false, errMalformedHash
	}
	var p PasswordParams
	if n, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.Memory, &p.Iterations, &p.Parallelism); err != nil || n != 3 ||
		p.Memory == 0 || p.Iterations == 0 || p.Parallelism == 0 {
		return false, errMalformedHash
	}
	b64 := base64.RawStdEncoding
	salt, err := b64.DecodeString(parts[4])
	if err != nil {
		return false, errMalformedHash
	}
	want, err := b64.DecodeString(parts[5])
	if err != nil || len(want) != keyLength {
		return false, errMalformedHash
	}
	got := argon2.IDKey([]byte(password), salt, p.Iterations, p.Memory, p.Parallelism, keyLength)
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}
