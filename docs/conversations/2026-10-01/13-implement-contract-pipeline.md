# Implementing #7: contract pipeline (/implement)

Follows [12](12-implement-ios-shell.md). #31 and #32 were squash-merged first (`b0add89`, `b9d6865`). Branch `feat/contract-pipeline`.

| UTC | Who | What happened | Decided | Details |
|---|---|---|---|---|
| 2026-10-01 12:32:00Z | User | Ran `/implement #7` | — | — |
| 2026-10-01 12:34:00Z | Claude | Checked tool support: oapi-codegen v2.8.0 and swift-openapi-generator 1.13.1 both support OpenAPI 3.1. Response validation and OpenAPI lint each need a tool outside the allowed list; asked the user | — | — |
| 2026-10-01 12:36:00Z | User | Chose kin-openapi for response validation and vacuum for lint | kin-openapi in tests only; vacuum in the `api` CI job (Q85) | Recorded as Q85 and in CLAUDE.md's dependency list. depguard confines kin-openapi to `apptest`. |
| 2026-10-01 12:45:00Z | Claude | Wrote `api/openapi.yaml` and generated both sides | Health ops + Problem/FieldError, Money, CurrencyCode, ExchangeRate. Go: models + strict std-http server, no embedded spec, prefixed enum values. Swift: generator pinned in `ios/Tools/APIGenerator`, output in `Infrastructure/APIClient/Generated`. One entry point: `scripts/generate-api.sh` | vacuum's `oas3-unused-component` is off because shared schemas precede their endpoints; warnings fail. |
| 2026-10-01 12:50:00Z | Claude | Ported `httpapi` onto the generated strict interface (tests unchanged and green); added contract validation to `apptest.Get` | — | Probe: `{"status":"okay"}` failed the test with "value must be 'ok'". |
| 2026-10-01 12:55:00Z | Claude | Data's repository now maps the generated client; first Data tests with a fake `ClientTransport` | swift-format skips `Generated/` via `ios/scripts/lint.sh` | Per-package and project `Package.resolved` files are committed. |
| 2026-10-01 13:00:00Z | Claude | Verified end to end and committed `98b3994`; opened draft [PR #33](https://github.com/bikkysamuel/splitsDemo/pull/33) to run the Linux drift check early | — | Rebuilt Docker server logs both contract routes; the app shows "Server reachable" through the generated client. |
| 2026-10-01 13:05:00Z | Claude | Ran `/code-review` (Standards and Spec in parallel) | Fixed: the contract now lists every problem slug the server sends and a 500 response on both ops; `responseError` logs the error type only; problem kinds in one list; Data builds the generated `Client` so `SplitsApp` never sees it; clearer names. Kept: indicator in all builds (as in #6); `main_test`'s plain `http.Get` (startup test, not an HTTP-seam test); duplicated `paths:` lists, matching the other workflows | Both axes flagged the undocumented slugs. |
| 2026-10-01 13:10:00Z | Claude | First CI run on #33 green: `api` (6m24s, Linux output matches macOS), `ios`, `server`, `gitleaks` | — | — |
| 2026-10-01 13:05:33Z | Claude | Pushed the review fixes; PR body written with `/pr`; marked ready | Merge waits for the user | — |
