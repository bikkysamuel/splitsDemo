# Implementing #8: ledger Shares and conversion (/implement)

Follows [16](16-log-prompts.md). Branch `feat/ledger-shares`.

| UTC | Who | What happened | Decided | Prompt |
|---|---|---|---|---|
| 2026-10-01 13:58:07Z | User | Ran `/implement #8` | — | `/implement #8` |
| 2026-10-01 14:01:00Z | Claude | Read #8, ADR-0002/0006/0007/0010, FR-E2/E5, doc 06 invariants; took the ticket's seams (table tests + fuzz on `ledger`'s public API) as agreed | Decided four open details myself and recorded them as Q87 for review | — |
| 2026-10-01 14:04:00Z | Claude | Built `ledger.Shares` test-first: one allocation (floor, largest remainder, ties to lowest `join_seq`) for all four methods, with typed `InvalidSplitError` reasons | — | — |
| 2026-10-01 14:08:00Z | Claude | Built `Convert`, `ParseExchangeRate` and `MinorUnits` test-first: exact `big.Rat` rate kept as entered, half-up once, zero-/three-/four-decimal currencies | — | — |
| 2026-10-01 14:12:00Z | Claude | Added fuzz targets for the Share and conversion invariants, a committed corpus (24 inputs each), a no-floating-point source check, and a fuzz step in `scripts/check.sh server` (`FUZZTIME`, default 30 s per target) | CI fuzzes every target found by `go test -list` | — |
