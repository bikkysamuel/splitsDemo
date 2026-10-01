# Settle-up Suggestions: the user confirms the simple rule

Follows [18](18-implement-ledger-balances.md). Branch `docs/settle-up-simple`.

| UTC | Who | What happened | Decided | Prompt |
|---|---|---|---|---|
| 2026-10-01 15:10:00Z | User | Asked to merge #37 once CI was green | — | merge #37 once CI is green |
| 2026-10-01 15:15:00Z | Claude | Waited for `server` and `gitleaks` to pass on the head commit; squash-merged #37 (`ef65fcf`), closing #9 | — | — |
| 2026-10-01 15:18:00Z | User | Answered the glossary question with a worked example: debtors pay creditors until every Balance is zero | Settle-up Suggestions stay simple (greedy); no search for the true minimum | payment will be simple - say 4 peaople are present - members paid: 100, 70, 10, 20 - sum it up: 200 <br>share per person : 50<br>now subtract per paid - per head : 50, 20, -40, -30<br>now -ve member will pay the +ve member until all are settled to 0 |
| 2026-10-01 15:20:00Z | Claude | Traced the example through `ledger` (3 → 1: 40, 4 → 2: 20, 4 → 1: 10); added it as Balances and Settle-up table tests; reworded the Settle-up Suggestion entry in GLOSSARY and US-41 to the greedy rule; marked Q88 confirmed | Glossary no longer says "as few payments as possible" | — |
| 2026-10-01 15:25:00Z | User | Confirmed the rest of Q88 and asked to merge #38 once CI was green | One inconsistent item, even a Pending one, makes the Group's Balances fail with `ErrInconsistentLedger`; Q88 fully confirmed | use the what you suggested for Q88<br><br>and<br>merge #38 once CI is green |
| 2026-10-01 15:26:00Z | Claude | Marked Q88 confirmed on #38; waiting for CI to merge | — | — |
