# Splits

An expense-sharing app: a SwiftUI iPhone app (iOS 26+, Xcode 27) and a Go 1.26 REST API on Postgres 18, in one monorepo (`ios/`, `server/`, `api/`, `docs/`). It is a portfolio project built to production standards, and it runs **locally only** for now (Docker Compose + iOS Simulator).

## Read before working

- `GLOSSARY.md`: domain terms. Use them exactly in code, tests, issues and UI copy (`PlaceholderMember`, `SettleUpSuggestion`, `Withdrawn`…).
- `docs/00-open-questions.md`: every product/technical decision, with links to its detail.
- `docs/adr/`: read the ADRs touching your area. If your change contradicts an ADR, stop and ask (see Operating rules).
- Requirements: `docs/01`–`04`. Design: `docs/05-architecture.md`, `06-data-model.md`, `07-api-contract.md`. Rules: `08-security.md`, `09-testing-strategy.md`. Milestones: `10-release-plan.md`.

## Architecture rules

- **The server is the single source of truth (ADR-0005, ADR-0006).** The iOS app keeps no domain data on the device: in-memory state only, `URLSession` without a disk cache. The Keychain holds tokens; `UserDefaults` holds non-financial UI preferences only.
- **Clean-architecture dependency rule (ADR-0015).**
  - iOS: `Presentation → Domain ← Data → Infrastructure`, as separate Swift packages; only `SplitsApp` imports them all.
  - iOS use cases exist only when they combine more than one repository call or have several entry points; otherwise ViewModels call repository protocols. Generated API types stay inside `Data`.
  - Go: domain packages under `server/internal/` own their entities, service and repository interfaces; `store` implements, `httpapi` adapts. Domain packages import neither `store`, `httpapi`, pgx nor `net/http`.
- **Contract first (ADR-0012).** Change `api/openapi.yaml` first, then regenerate (oapi-codegen, swift-openapi-generator) and commit the generated code.
- **Data access (ADR-0014).** SQL in sqlc query files, pgx v5, goose forward-only migrations.

## Money rules

- Amounts are integer minor units + ISO 4217 code (`bigint` in Postgres, `Int64` in Swift, `int64` in Go). Exchange Rates are exact decimals: `NUMERIC` ↔ `math/big.Rat`, strings in JSON. Floating-point types are reserved for non-money values.
- All money arithmetic lives in `server/internal/ledger` (ADR-0006). Swift `Money` formats only; live previews call `POST /v1/groups/{id}/expenses/preview`.
- Rounding: floor, then largest remainder, ties by `join_seq`; conversion rounds half-up once (ADR-0010).
- Only Accepted (and WithdrawalPending) items count toward Balances (ADR-0009).
- Every `ledger` change keeps its fuzz invariants green (doc 09).

## Security rules

- The fixed one-time code `123456` exists only under `APP_ENV=development`, and the startup guard that refuses it elsewhere must stay (ADR-0016).
- Logs carry IDs only: never passwords, tokens, codes, emails, names or notes.
- Secrets live in `.env` (git-ignored) or CI secrets. The repo is public; commit only `.env.example` with placeholder values.
- Authorization lives in services. A resource the User can't see returns `404`.
- Full rules: `docs/08-security.md`.

## Coding conventions

- Go: `gofmt`, golangci-lint (config in repo), errors wrapped with context, `log/slog` structured logging, context passed through every call.
- Swift: Swift 6 language mode with strict concurrency, `swift-format` (config in repo), `@Observable` ViewModels, every user-facing string in a String Catalog, VoiceOver labels and Dynamic Type on every screen.
- Names follow `GLOSSARY.md`.

## Testing requirements

- Work test-first with `/tdd`. Every bug fix starts with a failing regression test.
- Integration tests use real Postgres (Docker), never a mocked database. Fakes only at architectural seams.
- `scripts/check.sh` runs every CI check locally (usage in its header).
- Definition of done and CI jobs: `docs/09-testing-strategy.md`.

## Dependency rules

Standard library first. Allowed now:
- Go: pgx, goose, `golang.org/x/crypto`, `golang.org/x/oauth2`, oapi-codegen runtime; kin-openapi in tests only (Q85); sqlc, golangci-lint, oapi-codegen and vacuum as tools.
- iOS (SPM only): swift-openapi-generator/runtime/urlsession, Firebase Messaging + Crashlytics.

Any other dependency needs the user's approval, a one-line justification in the PR, and an ADR if it would be hard to replace.

## Git conventions

- Branch from `master`: `feat/…`, `fix/…`, `docs/…`, `chore/…`, ideally one per ticket.
- Conventional Commits (`feat(ledger): …`). Squash-merge PRs. PR bodies via `/pr`.

## Operating rules for Claude Code

- Workflow: `/to-spec` → `/to-tickets` → `/implement` per ticket (`/tdd` → `/code-review` → `/pr`) → `/retro` after each milestone. Bugs: `/diagnosing-bugs`.
- You may create branches, commit, push and open PRs.
- Ask the user first before: merging a PR, committing to `master`, adding a dependency, writing a destructive migration, changing an Accepted ADR, or touching secrets or paid services.
- When a ticket conflicts with an ADR, `GLOSSARY.md` or a decided question, stop and ask rather than working around it. Record each new decision in `docs/00-open-questions.md` (and an ADR when it is hard to reverse).

## Agent skills

### Issue tracker

Issues are tracked in this repo's GitHub Issues, using the `gh` CLI. See `docs/agents/issue-tracker.md`.

### Triage labels

Uses the five default triage labels: `needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`. See `docs/agents/triage-labels.md`.

### Domain docs

Single-context: one `GLOSSARY.md` and `docs/adr/` at the repo root. See `docs/agents/domain.md`.

## Conversation log

Record every conversation in `docs/conversations/` following `docs/conversations/README.md`: one directory per UTC date, one numbered file per task (new file on a date or task change; `-cont` when a task resumes), with a table of UTC time | Who | What happened | Decided | Prompt (the user's exact words on User rows). Update it as the conversation goes, and bring it up to date before each `/code-review`.
