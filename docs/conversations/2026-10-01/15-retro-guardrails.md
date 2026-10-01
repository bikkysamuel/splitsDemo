# M0 retro follow-up: guardrails (items 2–5)

Follows [14](14-retro-m0.md). Branch `chore/m0-retro-guardrails`.

| UTC | Who | What happened | Decided | Prompt |
|---|---|---|---|---|
| 2026-10-01 13:21:10Z | Claude | Item 5: `TestEveryProblemKindIsDocumentedInTheContract` reads `api/openapi.yaml` and fails for any `problemKind` slug missing from the Problem schema's list | `problemKinds` is the single list of kinds the server sends | — |
| 2026-10-01 13:21:10Z | Claude | Item 3: `ios/scripts/check-scheme-tests.sh` fails when a package `testTarget` is not a Testable in the `Splits` scheme | — | — |
| 2026-10-01 13:21:10Z | Claude | Item 2: `scripts/check.sh [server\|api\|ios\|secrets\|all]` runs every CI check with pinned versions via `go run …@version` and a pinned gitleaks image; loads `TEST_DATABASE_URL` from `.env` when unset | All four workflows call it, so versions live in one file; the third-party gitleaks action is gone | — |
| 2026-10-01 13:21:10Z | Claude | Item 4: the log rule now says to bring the log up to date before each `/code-review` (CLAUDE.md); the hand-kept README index is gone (file names already sort chronologically); removed the duplicate auto-memory copy of the rule | Rule lives in CLAUDE.md, not in the vendored `implement` skill | — |
| 2026-10-01 13:22:00Z | Claude | Committed `aa8da03`; opened draft [PR #34](https://github.com/bikkysamuel/splitsDemo/pull/34); ran `/code-review` (Standards and Spec in parallel) | — | — |
| 2026-10-01 13:25:54Z | Claude | Fixed the review findings | Slug test parses every `problemKind` literal in `httpapi` and matches only inside the Problem schema; gitleaks image pinned by digest; scheme check counts only `skipped = "NO"` Testables and is now a `.py`; `.env` value unquoted; drift check also watches the generator's `Package.resolved`, via one `generated_state` helper; doc 09 rows and Q86 updated; CLAUDE.md line points at the script header instead of copying its usage | — |
| 2026-10-01 13:29:33Z | User | Asked to merge #34 | Squash-merged as `1a6d5ee` once all four checks passed | merge #34 |
