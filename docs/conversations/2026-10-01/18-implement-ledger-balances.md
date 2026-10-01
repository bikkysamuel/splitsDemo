# Implementing #9: ledger Balances and Settle-up Suggestions (/implement)

Follows [17](17-implement-ledger-shares.md). Branch `feat/ledger-balances`.

| UTC | Who | What happened | Decided | Prompt |
|---|---|---|---|---|
| 2026-10-01 14:38:00Z | User | Ran `/implement #9` | — | `/implement #9` |
| 2026-10-01 14:41:00Z | Claude | Read #9 (blocker #5 closed), ADR-0006/0009/0010, FR-B1/B2, doc 06 state machine and invariants 2–4, GLOSSARY; took the ticket's seams (table tests + fuzz on `ledger`'s public API) | — | — |
| 2026-10-01 14:46:00Z | Claude | Built `ledger.Balances` and `ledger.SettleUpSuggestions` test-first: table tests for all settled, one debtor, chains, Settlements, Pending/Disputed/Withdrawn ignored, WithdrawalPending counted, ties; inconsistent items return `ErrInconsistentLedger` | Members are identified by `join_seq`, as in Splits; greedy is largest debtor pays largest creditor, ties to the lowest `join_seq`; recorded as Q88 | — |
| 2026-10-01 14:52:00Z | Claude | Added three fuzz targets (Σ Balances = 0 against an independent FR-B1 oracle; suggestions zero every Balance in ≤ non-zero Members − 1 payments; excluded states and Accepted ↔ WithdrawalPending never change Balances) with 24 committed corpus inputs each; `scripts/check.sh server` already fuzzes every target | Probes: counting Disputed items, a lossy Settlement and overpaying a creditor each fail their target within a second | `FUZZTIME=5s scripts/check.sh server` green |
| 2026-10-01 14:56:00Z | Claude | Committed `ad5d874`; ran `/code-review` (Standards and Spec in parallel). Spec: every criterion met, nothing wrong. Standards: no hard violations, a few smells | Fixed: one pass per list in `Balances`; `before` → `pickedFirst`; the excluded-states fuzz no longer uses `CountsTowardBalances` to choose its flips; `FuzzBalancesSumToZero` → `FuzzBalancesFollowFRB1`. Kept: bare `int` join_seq, as in `SplitMember`; greedy over a true minimum, which FR-B2 and #9 allow even though GLOSSARY says "as few payments as possible" | — |
| 2026-10-01 15:05:00Z | User | Asked to push and open the PR | — | push and open the PR |
| 2026-10-01 15:06:00Z | Claude | Pushed `feat/ledger-balances`; opened [PR #37](https://github.com/bikkysamuel/splitsDemo/pull/37) via `/pr` | — | — |
