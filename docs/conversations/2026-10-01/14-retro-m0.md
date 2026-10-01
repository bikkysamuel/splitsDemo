# M0 retrospective (/retro)

Follows [13](13-implement-contract-pipeline.md). #33 was squash-merged as `896f415`, completing M0.

| UTC | Who | What happened | Decided | Details |
|---|---|---|---|---|
| 2026-10-01 13:16:23Z | User | Asked to merge #33, then ran `/retro` | — | — |
| 2026-10-01 13:16:23Z | Claude | Reviewed the M0 session (#5–#7) and proposed environment improvements, ranked by severity | Awaiting the user's pick | Top items: `master` has no branch protection although doc 09 says it does; no single local check command (lint tools reinstalled per session, `.env` sourced by hand); package test targets must be added to the Xcode scheme by hand, so a new test target can silently never run; the conversation log was written after review and the hand-kept index conflicts between parallel PRs. |
| 2026-10-01 13:21:10Z | User | Chose items 2, 3, 4 and 5, in one PR | Item 1 (branch protection) and item 6 (`CODING_STANDARDS.md`) not now | Work continues in [15](15-retro-guardrails.md). |
