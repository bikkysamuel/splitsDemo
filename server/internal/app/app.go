// Package app is the server's composition root: it opens the database,
// applies migrations and wires the domain services into the HTTP layer.
// cmd/server and the HTTP-seam tests both build the server through New.
package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/bikkysamuel/splitsDemo/server/internal/auth"
	"github.com/bikkysamuel/splitsDemo/server/internal/groups"
	"github.com/bikkysamuel/splitsDemo/server/internal/httpapi"
	"github.com/bikkysamuel/splitsDemo/server/internal/idempotency"
	"github.com/bikkysamuel/splitsDemo/server/internal/platform"
	"github.com/bikkysamuel/splitsDemo/server/internal/store"
)

// App is a fully wired server.
type App struct {
	db  *store.DB
	api *httpapi.Server
}

type options struct {
	clock          platform.Clock
	passwordParams auth.PasswordParams
}

// Option changes how New wires the server. Tests use them; cmd/server takes
// the defaults.
type Option func(*options)

// WithClock replaces the system clock, so tests can expire codes and
// tokens.
func WithClock(c platform.Clock) Option { return func(o *options) { o.clock = c } }

// WithPasswordParams replaces the Argon2id cost, so tests stay fast.
func WithPasswordParams(p auth.PasswordParams) Option {
	return func(o *options) { o.passwordParams = p }
}

// New connects to the database, applies pending migrations and wires the
// HTTP handler. Call Close when done.
func New(ctx context.Context, cfg platform.Config, logger *slog.Logger, opts ...Option) (*App, error) {
	o := options{clock: platform.SystemClock{}, passwordParams: auth.DefaultPasswordParams}
	for _, opt := range opts {
		opt(&o)
	}
	sender, err := otpSender(cfg)
	if err != nil {
		return nil, err
	}

	db, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	if err := db.Migrate(ctx); err != nil {
		db.Close()
		return nil, err
	}
	ids := platform.NewIDGenerator(o.clock)
	api := httpapi.New(httpapi.Deps{
		Logger:    logger,
		Readiness: db,
		Auth: auth.NewService(auth.Deps{
			Repository:     db.Auth(),
			OTPSender:      sender,
			Clock:          o.clock,
			IDs:            ids,
			PasswordParams: o.passwordParams,
		}),
		Idempotency: idempotency.NewService(db.Idempotency(), o.clock),
		Groups:      groups.NewService(groups.Deps{Repository: db.Groups(), Clock: o.clock, IDs: ids}),
		IDs:         ids,
	})
	return &App{db: db, api: api}, nil
}

// otpSender picks the one-time-code sender for OTP_MODE. LoadConfig has
// already refused the fixed code outside development (ADR-0016); this
// checks again, so no other path into New can bypass the guard.
func otpSender(cfg platform.Config) (auth.OTPSender, error) {
	switch cfg.OTPMode {
	case platform.OTPModeFixed:
		if cfg.AppEnv != platform.EnvDevelopment {
			return nil, fmt.Errorf("OTP_MODE=%s is allowed only with APP_ENV=%s (ADR-0016)", platform.OTPModeFixed, platform.EnvDevelopment)
		}
		return auth.FixedCodeSender{}, nil
	case platform.OTPModeEmail:
		// EmailCodeSender comes with the email provider (doc 05, doc 10).
		return nil, fmt.Errorf("OTP_MODE=%s: no email sender exists yet; use OTP_MODE=%s under APP_ENV=%s",
			platform.OTPModeEmail, platform.OTPModeFixed, platform.EnvDevelopment)
	default:
		return nil, fmt.Errorf("OTP_MODE must be %q or %q", platform.OTPModeFixed, platform.OTPModeEmail)
	}
}

// Handler serves every route.
func (a *App) Handler() http.Handler { return a.api }

// Routes lists every registered route as "METHOD /path".
func (a *App) Routes() []string { return a.api.Routes() }

// Close releases the database connections.
func (a *App) Close() { a.db.Close() }
