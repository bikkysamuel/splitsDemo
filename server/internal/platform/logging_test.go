package platform_test

import (
	"bytes"
	"context"
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

// Every log line written with a request's context carries its X-Request-ID,
// so one request's lines can be found together (doc 08: IDs only).
func TestLoggerAddsTheRequestIDFromTheContext(t *testing.T) {
	var buf bytes.Buffer
	logger := platform.NewLogger(&buf)
	ctx := platform.WithRequestID(context.Background(), "req-123")

	logger.InfoContext(ctx, "handled")
	logger.Info("no context")

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d log lines; want 2:\n%s", len(lines), buf.String())
	}
	var first, second map[string]any
	_ = json.Unmarshal([]byte(lines[0]), &first)
	_ = json.Unmarshal([]byte(lines[1]), &second)
	if first["request_id"] != "req-123" {
		t.Errorf("request_id = %v; want req-123", first["request_id"])
	}
	if _, ok := second["request_id"]; ok {
		t.Errorf("a line without a request context has request_id %v", second["request_id"])
	}
	if got := platform.RequestID(ctx); got != "req-123" {
		t.Errorf("RequestID(ctx) = %q; want req-123", got)
	}
}
