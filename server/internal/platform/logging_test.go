package platform_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/bikkysamuel/splitsDemo/server/internal/platform"
)

// Logs carry IDs only (doc 08). The logger is the backstop when a call site
// passes a sensitive attribute anyway.
func TestLoggerRedactsSensitiveAttributes(t *testing.T) {
	var buf bytes.Buffer
	logger := platform.NewLogger(&buf)

	logger.Info("signed in",
		"user_id", "0190b6c4-0000-7000-8000-000000000001",
		"email", "alice@example.com",
		"password", "hunter2",
		slog.Group("request", "refresh_token", "rt-secret", "code", "123456"),
		"display_name", "Alice",
		"note", "dinner with Bob",
	)

	out := buf.String()
	for _, secret := range []string{"alice@example.com", "hunter2", "rt-secret", "123456", "Alice", "dinner with Bob"} {
		if strings.Contains(out, secret) {
			t.Errorf("log output contains %q:\n%s", secret, out)
		}
	}

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("log output is not one JSON object: %v\n%s", err, out)
	}
	if entry["user_id"] != "0190b6c4-0000-7000-8000-000000000001" {
		t.Errorf("user_id = %v; want the ID kept", entry["user_id"])
	}
	if entry["msg"] != "signed in" {
		t.Errorf("msg = %v; want %q", entry["msg"], "signed in")
	}
}
