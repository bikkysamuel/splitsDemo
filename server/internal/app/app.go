// Package app is the server's composition root: it opens the database,
// applies migrations and wires the domain services into the HTTP layer.
// cmd/server and the HTTP-seam tests both build the server through New.
package app

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/bikkysamuel/splitsDemo/server/internal/httpapi"
	"github.com/bikkysamuel/splitsDemo/server/internal/platform"
	"github.com/bikkysamuel/splitsDemo/server/internal/store"
)

// App is a fully wired server.
type App struct {
	db  *store.DB
	api *httpapi.Server
}

// New connects to the database, applies pending migrations and wires the
// HTTP handler. Call Close when done.
func New(ctx context.Context, cfg platform.Config, logger *slog.Logger) (*App, error) {
	db, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	if err := db.Migrate(ctx); err != nil {
		db.Close()
		return nil, err
	}
	api := httpapi.New(httpapi.Deps{Logger: logger, Readiness: db})
	return &App{db: db, api: api}, nil
}

// Handler serves every route.
func (a *App) Handler() http.Handler { return a.api }

// Routes lists every registered route as "METHOD /path".
func (a *App) Routes() []string { return a.api.Routes() }

// Close releases the database connections.
func (a *App) Close() { a.db.Close() }
