# M0 retro follow-up: guardrails (items 2–5)

Follows [14](14-retro-m0.md). Branch `chore/m0-retro-guardrails`.

| UTC | Who | What happened | Decided | Details |
|---|---|---|---|---|
| 2026-10-01 13:21:10Z | Claude | Item 5: `TestEveryProblemKindIsDocumentedInTheContract` reads `api/openapi.yaml` and fails for any `problemKind` slug missing from the Problem schema's list | `problemKinds` is the single list of kinds the server sends | Probe: misspelling `internal` in the contract turned it red. |
| 2026-10-01 13:21:10Z | Claude | Item 3: `ios/scripts/check-scheme-tests.sh` fails when a package `testTarget` is not a Testable in the `Splits` scheme | — | Probe: removing `DataTests` from the scheme turned it red. |
| 2026-10-01 13:21:10Z | Claude | Item 2: `scripts/check.sh [server\|api\|ios\|secrets\|all]` runs every CI check with pinned versions via `go run …@version` and a pinned gitleaks image; loads `TEST_DATABASE_URL` from `.env` when unset | All four workflows call it, so versions live in one file; the third-party gitleaks action is gone | `all` takes about 70 s locally. Drift probe: editing the contract without regenerating failed `check.sh api`. |
| 2026-10-01 13:21:10Z | Claude | Item 4: the log rule now says to bring the log up to date before each `/code-review` (CLAUDE.md); the hand-kept README index is gone (file names already sort chronologically); removed the duplicate auto-memory copy of the rule | Rule lives in CLAUDE.md, not in the vendored `implement` skill | `implement` is pinned by hash in `skills-lock.json`, so editing it would be overwritten on update. |
