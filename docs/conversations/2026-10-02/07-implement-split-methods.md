# Implementing #18: exact, percentage and ratio Splits (unattended M1 run)

Follows [06](06-merge-m1-prs.md). Branch `feat/split-methods`, built on `master` after #46.

| UTC | Who | What happened | Decided | Prompt |
|---|---|---|---|---|
| 2026-10-02 14:12:00Z | Claude | Reviewed the WIP commit (contract `SplitInputValue`, regenerated code, service, store, HTTP-seam tests per method) and ran it on real Postgres: green. Added a regression test and fix so an entry reads back the same from create and from a later read (leading zeros dropped, scale kept); listed the Split field codes and their fields in the contract | See Q96 | — |
| 2026-10-02 14:22:40Z | Claude | iOS, test-first: Domain `SplitMethod` (reads what was typed into the server's decimal string, well-formedness only) and `SplitEntry`; `ExpenseInput` carries the method and entries, `Share` its entry, `Expense` its method; the Data mapper and new field codes; Add Expense gets a Split method picker and a per-Member entry field (VoiceOver labels, String Catalog), the server's sum errors under the Split and per-entry errors under the Member; the Expense detail shows the method and each percentage or ratio part | See Q96 | — |
