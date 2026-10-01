package platform

import (
	"context"
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
// values of sensitive attribute keys and adds the request ID of the
// context, if any, as request_id.
func NewLogger(w io.Writer) *slog.Logger {
	return slog.New(requestIDHandler{slog.NewJSONHandler(w, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if sensitiveKeys[a.Key] {
				return slog.String(a.Key, redacted)
			}
			return a
		},
	})})
}

type requestIDKey struct{}

// WithRequestID returns ctx carrying the request's X-Request-ID.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, id)
}

// RequestID returns the request ID ctx carries, or "".
func RequestID(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

// requestIDHandler adds request_id from the record's context.
type requestIDHandler struct{ slog.Handler }

func (h requestIDHandler) Handle(ctx context.Context, r slog.Record) error {
	if id := RequestID(ctx); id != "" {
		r.AddAttrs(slog.String("request_id", id))
	}
	return h.Handler.Handle(ctx, r)
}

func (h requestIDHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return requestIDHandler{h.Handler.WithAttrs(attrs)}
}

func (h requestIDHandler) WithGroup(name string) slog.Handler {
	return requestIDHandler{h.Handler.WithGroup(name)}
}
