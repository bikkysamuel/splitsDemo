// Package pgtest gives tests an isolated schema in a real Postgres database
// (doc 09: never a mocked database).
//
// Tests read the database from TEST_DATABASE_URL (see .env.example); start it
// with `docker compose up -d postgres` from the repository root.
package pgtest

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// NewSchema creates an empty schema in the test database and returns a
// database URL whose connections use it, falling back to public for shared
// extensions. The schema is dropped when the test ends.
func NewSchema(t testing.TB) string {
	t.Helper()
	base := os.Getenv("TEST_DATABASE_URL")
	if base == "" {
		t.Fatal("TEST_DATABASE_URL is not set: start Postgres with `docker compose up -d postgres` " +
			"and export TEST_DATABASE_URL from .env (see .env.example)")
	}

	name := "test_" + randomHex(t, 8)
	exec(t, base, "CREATE SCHEMA "+pgx.Identifier{name}.Sanitize())
	t.Cleanup(func() {
		exec(t, base, "DROP SCHEMA "+pgx.Identifier{name}.Sanitize()+" CASCADE")
	})

	u, err := url.Parse(base)
	if err != nil {
		t.Fatalf("parse TEST_DATABASE_URL: %v", err)
	}
	q := u.Query()
	q.Set("search_path", name+",public")
	u.RawQuery = q.Encode()
	return u.String()
}

func exec(t testing.TB, databaseURL, sql string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}
	defer func() { _ = conn.Close(ctx) }()
	if _, err := conn.Exec(ctx, sql); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
}

func randomHex(t testing.TB, n int) string {
	t.Helper()
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("random schema name: %v", err)
	}
	return hex.EncodeToString(b)
}
