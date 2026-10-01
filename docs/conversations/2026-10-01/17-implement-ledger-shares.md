# Implementing #8: ledger Shares and conversion (/implement)

Follows [16](16-log-prompts.md). Branch `feat/ledger-shares`.

| UTC | Who | What happened | Decided | Prompt |
|---|---|---|---|---|
| 2026-10-01 13:58:07Z | User | Ran `/implement #8` | — | `/implement #8` |
| 2026-10-01 14:01:00Z | Claude | Read #8, ADR-0002/0006/0007/0010, FR-E2/E5, doc 06 invariants; took the ticket's seams (table tests + fuzz on `ledger`'s public API) as agreed | Decided four open details myself and recorded them as Q87 for review | — |
| 2026-10-01 14:04:00Z | Claude | Built `ledger.Shares` test-first: one allocation (floor, largest remainder, ties to lowest `join_seq`) for all four methods, with typed `InvalidSplitError` reasons | — | — |
| 2026-10-01 14:08:00Z | Claude | Built `Convert`, `ParseExchangeRate` and `MinorUnits` test-first: exact `big.Rat` rate kept as entered, half-up once, zero-/three-/four-decimal currencies | — | — |
| 2026-10-01 14:12:00Z | Claude | Added fuzz targets for the Share and conversion invariants, a committed corpus (24 inputs each), a no-floating-point source check, and a fuzz step in `scripts/check.sh server` (`FUZZTIME`, default 30 s per target) | CI fuzzes every target found by `go test -list` | — |
| 2026-10-01 14:16:00Z | Claude | Committed `c8700b4` (an Xcode-rewritten `InfoPlist.xcstrings` taken back out); opened draft [PR #36](https://github.com/bikkysamuel/splitsDemo/pull/36); ran `/code-review` (Standards and Spec in parallel) | — | — |
| 2026-10-01 14:08:57 | Claude | Fixed the review findings | Fuzz oracle computes exact Shares from each method's definition and checks ADR-0010's leftover rule directly (larger remainder, then lower `join_seq`); `Method` → `SplitMethod`; `Member` → `MemberIndex`; `Convert` errors carry context; Σ weights computed once; shared integer-weight path. Kept: currency codes as strings until Groups (#14) | Probes: flipping the tie direction and giving leftovers to the smallest remainder both fail on the committed corpus alone. CI `server` with 30 s fuzz per target: green in 2m11s. Q87's two readings (10 decimal places; no zero Inputs) await the user's confirmation. |
