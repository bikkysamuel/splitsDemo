package platform

import (
	"io"
	"log/slog"
)

const redacted = "[REDACTED]"

// sensitiveKeys are attribute keys whose values never reach the logs. Logs
// carry IDs only (doc 08); this list is the backstop, not the rule.
var sensitiveKeys = map[string]bool{
	"password":      true,
	"password_hash": true,
	"token":         true,
	"access_token":  true,
	"refresh_token": true,
	"code":          true,
	"otp":           true,
	"email":         true,
	"name":          true,
	"display_name":  true,
	"note":          true,
	"notes":         true,
}

// NewLogger returns a JSON structured logger writing to w that redacts the
// values of sensitive attribute keys.
func NewLogger(w io.Writer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if sensitiveKeys[a.Key] {
				return slog.String(a.Key, redacted)
			}
			return a
		},
	}))
}
