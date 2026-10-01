# Writing the v1 spec (/to-spec)

Follows [06](06-sdlc-docs-authoring.md). PR #1 (discovery docs) was merged at 2026-10-01 10:24:37Z with no review comments.

| UTC | Who | What happened | Decided | Details |
|---|---|---|---|---|
| 2026-10-01 10:25:53Z | User | Ran `/to-spec` in the same session (without clearing context) | — | — |
| 2026-10-01 10:26:19Z | Claude | Confirmed PR #1 merged; started branch `docs/spec`; proposed the test seams for the user to confirm | — | Seams: the server's HTTP API (primary), the `ledger` module (fuzz invariants), the iOS repository-protocol boundary, and a nightly end-to-end smoke test. |
| 2026-10-01 10:27:00Z | User | Cleared context (`/clear`) and ran `/to-spec` again | — | The seams proposed above had not been confirmed. |
| 2026-10-01 10:27:40Z | Claude | Re-read GLOSSARY, docs 00–10 and all ADRs; asked the user to confirm the test seams for a single v1 spec covering M0–M3 | — | No code exists yet, so every seam is new. Same four seams proposed as before. |
| 2026-10-01 10:30:00Z | User | Confirmed the four seams | Test seams: server HTTP API (primary; fake clock, OTPSender and PushSender), `ledger` (table + fuzz), iOS repository protocols (fake repositories, mapper fixtures), nightly XCUITest smoke | Rejected: extra Go service-level seam; server-only seams. |
| 2026-10-01 10:31:55Z | Claude | Published the v1 spec as issue [#2](https://github.com/bikkysamuel/splitsDemo/issues/2), labelled `ready-for-agent` | One spec covers M0–M3; `/to-tickets` splits it per milestone | 131 user stories; implementation decisions per server module and iOS layer; D1–D19 adopted as proposed; suggested M1 ticket order starting with `ledger`. |
| 2026-10-01 10:33:32Z | User | Asked to commit and open the PR | — | — |
| 2026-10-01 10:33:32Z | Claude | Committed the conversation log on `docs/spec`, pushed, and opened a PR with a `/pr` body | — | The spec itself lives in issue #2, not in the repo. |
