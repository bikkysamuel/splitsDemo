# 09 · Testing strategy

Status: Draft · Last updated: 2026-10-01 · Depends on: Q58, Q59, Q62, Q69; ADR-0006, 0010, 0012, 0015, 0016

## Principles

- **Test-first** via `/tdd`: one failing test, then the code, then refactor. Every bug fix starts with a test that fails on that bug.
- Test **behaviour through public interfaces**: services and HTTP on the server, ViewModels and use cases on iOS. Not private functions.
- **Real Postgres, never a mocked database.** Fakes only at the seams the architecture defines (iOS `APIClient`/repository protocols; server `OTPSender`, `PushSender`, clock).
- No global coverage percentage. The bar is the invariants and flows below.

## Server (Go)

| Layer | What | How |
|---|---|---|
| `ledger` | Splits (all four methods), rounding, conversion, Balances, Settle-up Suggestions | Table tests for named scenarios + **`go test -fuzz`** for the money invariants (doc 06): Σ shares = amount; each share within one minor unit of exact; ties by `join_seq`; Σ Balances = 0; suggestions zero everything in ≤ Members − 1 payments. Fuzz corpora committed. |
| Services | Domain rules, authorization, state machines (Expense, Settlement, Group) | Go tests against real Postgres (Docker locally, a service container in CI); each test runs in its own schema or rolled-back transaction. Fake clock for the 5- and 7-day rules. |
| HTTP | Each endpoint: happy path, validation, auth, `404`-not-`403`, idempotency replay, `409` on stale version | `httptest` against the full wired server + real Postgres (`apptest.Start`). `apptest` validates every response against `api/openapi.yaml` with kin-openapi (Q85). |
| Jobs | Auto-acceptance, reminders, Closing → Closed | Fake clock; run twice to prove idempotency. |
| Config | Fixed OTP refused outside development (ADR-0016) | Startup test. |
| Migrations | Apply cleanly from empty; constraints and triggers reject invalid rows (Σ shares, append-only history, one-level Sub-Groups) | Run in CI on a fresh database. |

## Contract (ADR-0012)

- OpenAPI lint on `api/openapi.yaml` with vacuum (Q85).
- **Drift check**: regenerating server and client code must produce no diff.
- Server responses are validated against the schema in the HTTP tests above.

## iOS (Swift)

| Layer | What | How |
|---|---|---|
| Domain | Use cases, `Money` formatting, deep-link parsing | **Swift Testing** (`@Test`), pure. |
| Data | Mappers (generated type ⇄ entity), invalid server data → typed errors, token refresh | Swift Testing with recorded JSON fixtures. |
| Presentation | ViewModels: loading, empty, error and success states; actions call the right repository/use case | Swift Testing with fake repositories (ADR-0015 seam). |
| UI smoke | Sign up (code `123456`) → create Group → add Expense → second account approves → settle → confirm | **XCUITest** against the Docker server, **nightly**, not on every PR. |
| Accessibility | VoiceOver labels and Dynamic Type on key screens | XCUITest accessibility audit (`performAccessibilityAudit`) in the nightly run. |
| Notifications | Action handling and deep-link routing | Unit tests on the handler + manual `xcrun simctl push` checklist until push is live (Q74). |

## CI (GitHub Actions, Q59)

| Job | Triggers on changes to | Steps | Runner |
|---|---|---|---|
| `server` | `server/**`, `api/**` | gofmt check, `go vet`, golangci-lint (incl. depguard), `go test -race ./...` with a Postgres 18 service, a short fuzz run (≈30 s per target), govulncheck | `ubuntu-latest` |
| `ios` | `ios/**`, `api/**` | swift-format lint, build, `xcodebuild test` (unit tests) on an iOS 27.0 Simulator | **`xcode-27`** pinned to Xcode 27.0 (Q69) |
| `api` | `api/**`, generator config and generated code | vacuum lint, `scripts/generate-api.sh`, fail on any diff | `ubuntu-latest` |
| `secrets` | every PR | gitleaks | `ubuntu-latest` |
| `nightly` | schedule | longer fuzz runs, XCUITest smoke + accessibility audit against Docker Compose | `xcode-27` |

**Merge gate**: `master` is protected; a PR needs every triggered job to pass. If the `xcode-27` preview runner proves unreliable, the `ios` check can be made non-required in repo settings without code changes. Switch to `macos-27` when GitHub ships it.

## Definition of done (per ticket)

1. Tests written first and passing locally and in CI.
2. `/code-review` run: both Standards and Spec axes clean, or findings addressed.
3. API changes made in `openapi.yaml` first, with generated code committed.
4. New screens meet NFR-A1–A3.
5. Docs/ADRs updated if a decision changed. Conversation log updated.
