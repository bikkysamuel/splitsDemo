# 04 · Non-functional requirements

Status: Draft · Last updated: 2026-10-01 · Depends on: Q26, Q28, Q30, Q31, Q69, Q75; ADR-0005, 0006

## Compatibility

- **NFR-C1** iOS 26 minimum, iPhone only, built with **Xcode 27.0 / Swift 6.4**, Swift 6 language mode with strict concurrency (Q26, Q57).
- **NFR-C2** The server targets Go 1.26 and Postgres 18 (Q27).
- **NFR-C3** Development and testing use the iOS Simulator against `http://localhost:8080` (Q75).

## Performance and scale (Q30)

- **NFR-P1** API latency: p95 < 300 ms for every endpoint on the reference setup (local Docker on the developer's Mac until a host exists), with a Group of 50 Members and 10,000 Expenses.
- **NFR-P2** Balances and Settle-up Suggestions for a 50-Member, 10,000-Expense Group compute in < 100 ms on the server.
- **NFR-P3** Lists are cursor-paginated (default page 50, maximum 200). No endpoint returns an unbounded list.
- **NFR-P4** App: the first screen shows content within 1 second of a warm launch on a recent Simulator (no cache exists to show earlier, per ADR-0005).

## Reliability and data integrity

- **NFR-R1** Every write accepts an `Idempotency-Key`. Repeating a request with the same key returns the original result (ADR-0012).
- **NFR-R2** Every write that changes money state runs in one database transaction, together with its Activity History entry.
- **NFR-R3** Money invariants hold at all times. They are checked by fuzz tests (doc 09) and by database constraints where possible (doc 06).
- **NFR-R4** Concurrent edits use optimistic concurrency (a version number). A stale write gets `409 Conflict`.
- **NFR-R5** Background jobs (Auto-acceptance, reminders, Closing → Closed) are idempotent and safe to re-run after a crash.
- **NFR-R6** Availability and backups (daily + point-in-time restore, 99.5% target) apply once a production host exists. Locally, a Docker volume holds the data.

## Security

See [08-security.md](08-security.md).

## Observability (Q28)

- **NFR-O1** The server logs structured JSON via `log/slog`, with a request ID on every log line, also returned as the `X-Request-ID` header.
- **NFR-O2** `GET /healthz` (process alive) and `GET /readyz` (database reachable).
- **NFR-O3** On startup the server prints its base URL and every registered API route (method + path) (Q75).
- **NFR-O4** iOS crash reporting uses Firebase Crashlytics (no paid Apple program needed).
- **NFR-O5** Logs never contain passwords, tokens, codes, emails or display names, only IDs.

## Accessibility and localization (Q31)

- **NFR-A1** Every screen supports VoiceOver (labels, traits, logical order) and Dynamic Type up to the largest accessibility sizes without truncating key information.
- **NFR-A2** Tap targets are at least 44×44 pt. Colour is never the only signal (for example, owed vs owing also uses text or a sign).
- **NFR-A3** Light and dark mode.
- **NFR-A4** English only, but every user-facing string is in a String Catalog. Money and dates are formatted with the user's locale. The server returns no user-facing text except `problem+json` `title`/`detail`, which the app maps to localized messages by error `type`.

## Maintainability

- **NFR-M1** Clean-architecture dependency rule, enforced by SPM package boundaries (iOS) and depguard (Go) (ADR-0015).
- **NFR-M2** Code names follow `GLOSSARY.md`.
- **NFR-M3** Formatters and linters (gofmt, golangci-lint, swift-format) run in CI. A failure blocks merge (Q59, Q62).
- **NFR-M4** Dependencies follow the allowed list and approval rule in CLAUDE.md (Q61).
