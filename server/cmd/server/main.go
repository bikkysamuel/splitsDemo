// Command server runs the Splits API. It reads its configuration from the
// environment (doc 05), applies migrations, logs its base URL and routes, and
// serves until SIGINT or SIGTERM.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/bikkysamuel/splitsDemo/server/internal/app"
	"github.com/bikkysamuel/splitsDemo/server/internal/platform"
)

const (
	readHeaderTimeout = 5 * time.Second
	shutdownTimeout   = 10 * time.Second
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Getenv, os.Stdout); err != nil {
		// Plain stderr: the slog logger may not exist yet (bad configuration).
		fmt.Fprintln(os.Stderr, "server:", err)
		os.Exit(1)
	}
}

// run starts the server and blocks until ctx is cancelled. It refuses to
// start on invalid configuration, including a fixed one-time code outside
// development (ADR-0016).
func run(ctx context.Context, getenv func(string) string, stdout io.Writer) error {
	cfg, err := platform.LoadConfig(getenv)
	if err != nil {
		return err
	}
	logger := platform.NewLogger(stdout)

	a, err := app.New(ctx, cfg, logger)
	if err != nil {
		return err
	}
	defer a.Close()

	ln, err := new(net.ListenConfig).Listen(ctx, "tcp", cfg.HTTPAddr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", cfg.HTTPAddr, err)
	}
	srv := &http.Server{
		Handler:           a.Handler(),
		ReadHeaderTimeout: readHeaderTimeout,
		BaseContext:       func(net.Listener) context.Context { return ctx },
	}
	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(ln) }()

	logger.InfoContext(ctx, "server started",
		"app_env", cfg.AppEnv,
		"base_url", baseURL(ln.Addr()),
		"routes", a.Routes())

	select {
	case err := <-serveErr:
		return fmt.Errorf("serve: %w", err)
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shut down: %w", err)
	}
	if err := <-serveErr; !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve: %w", err)
	}
	logger.InfoContext(shutdownCtx, "server stopped")
	return nil
}

// baseURL is the URL clients use to reach addr; a wildcard listen address
// is reported as localhost.
func baseURL(addr net.Addr) string {
	tcp, ok := addr.(*net.TCPAddr)
	if !ok {
		return "http://" + addr.String()
	}
	host := tcp.IP.String()
	if tcp.IP.IsUnspecified() {
		host = "localhost"
	}
	return "http://" + net.JoinHostPort(host, strconv.Itoa(tcp.Port))
}
