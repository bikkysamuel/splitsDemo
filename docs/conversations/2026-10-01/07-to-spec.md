# Writing the v1 spec (/to-spec)

Follows [06](06-sdlc-docs-authoring.md). PR #1 (discovery docs) was merged at 2026-10-01 10:24:37Z with no review comments.

| UTC | Who | What happened | Decided | Prompt |
|---|---|---|---|---|
| 2026-10-01 10:25:53Z | User | Ran `/to-spec` in the same session (without clearing context) | — | `/to-spec` |
| 2026-10-01 10:26:19Z | Claude | Confirmed PR #1 merged; started branch `docs/spec`; proposed the test seams for the user to confirm | — | — |
| 2026-10-01 10:27:06Z | User | Cleared context (`/clear`) and ran `/to-spec` again | — | `/clear`<br>then `/to-spec` |
| 2026-10-01 10:27:40Z | Claude | Re-read GLOSSARY, docs 00–10 and all ADRs; asked the user to confirm the test seams for a single v1 spec covering M0–M3 | — | — |
| 2026-10-01 10:29:52Z | User | Confirmed the four seams | Test seams: server HTTP API (primary; fake clock, OTPSender and PushSender), `ledger` (table + fuzz), iOS repository protocols (fake repositories, mapper fixtures), nightly XCUITest smoke | Answered: "Which test seams should the v1 spec (M0–M3) use? There's no code yet, so all of these are new. The list is taken from doc 09 and keeps the number of seams as low as possible."="Four seams (Recommended)" |
| 2026-10-01 10:31:55Z | Claude | Published the v1 spec as issue [#2](https://github.com/bikkysamuel/splitsDemo/issues/2), labelled `ready-for-agent` | One spec covers M0–M3; `/to-tickets` splits it per milestone | — |
| 2026-10-01 10:33:19Z | User | Asked to commit and open the PR | — | yes, commit and open the PR |
| 2026-10-01 10:33:32Z | Claude | Committed the conversation log on `docs/spec`, pushed, and opened a PR with a `/pr` body | — | — |
