# Re-running /to-spec; M0–M1 tickets (/to-tickets)

Follows [08](08-ui-ux-suggestions.md). PR #4 (UI/UX decisions Q77–Q84) was squash-merged as `36f175a`.

| UTC | Who | What happened | Decided | Details |
|---|---|---|---|---|
| 2026-10-01 11:06:00Z | User | Cleared context (`/clear`) and ran `/to-spec` | — | — |
| 2026-10-01 11:06:28Z | Claude | Checked the repo and issue tracker; found nothing new to spec and published no issue | No duplicate spec | Spec [#2](https://github.com/bikkysamuel/splitsDemo/issues/2) already covers M0–M3 and includes Q77–Q84 (stories 125–142, section "App shell, navigation and preferences"). No code exists yet. Suggested `/to-tickets` on #2 as the next step. |
| 2026-10-01 11:10:00Z | User | Ran `/to-tickets` | — | — |
| 2026-10-01 11:12:00Z | Claude | Proposed 25 vertical-slice tickets for M0 and M1, with blocking edges | Ticket M0–M1 now; M2 and M3 after their milestone `/retro` | M0: server skeleton, iOS shell, contract pipeline. M1: two `ledger` tickets, then auth (×3), connectivity, Groups, Members, automatic Claim, Expenses (equal → other Splits, foreign currency, edit/withdraw, filters), Balances, Settlements, Home summary, Activity History, delete account, Settings, performance check, nightly smoke. |
| 2026-10-01 11:14:00Z | User | Approved the breakdown | Breakdown as proposed | — |
| 2026-10-01 11:24:45Z | Claude | Published issues #5–#29, labelled `ready-for-agent`, as sub-issues of #2 with native blocked-by dependencies | — | Frontier: #5 (server skeleton) and #6 (iOS shell). |
