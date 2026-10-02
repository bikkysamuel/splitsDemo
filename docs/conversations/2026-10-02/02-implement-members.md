# Implementing #15: add Members by email or as Placeholders, grant Admin (unattended M1 run)

Follows [01](01-implement-groups-create.md). Branch `feat/members`, built on `feat/groups-create` (PR #42).

| UTC | Who | What happened | Decided | Prompt |
|---|---|---|---|---|
| 2026-10-02 04:40:00Z | Claude | Opened PR #42 for #14 and started #15 | — | — |
| 2026-10-02 03:53:56Z | Claude | Contract for `POST /v1/groups/{id}/members` and `PATCH …/members/{memberId}`; Members carry `version`; `groups.AddMember` (Group row lock for `join_seq` and the limit; verified User links, else Placeholder) and `MakeAdmin`; HTTP-seam tests incl. parallel adds, duplicates, the 50 limit, non-Admin 403; iOS Add Member sheet and a Make Admin swipe/VoiceOver action, all test-first | See Q92 | — |
| 2026-10-02 03:58:20Z | Claude | Ran `/code-review` against `feat/groups-create`. Fixed: Make Admin refuses Closed Groups; a unique violation on insert maps to the `taken` field error (the database is the backstop); neutral "Enter a name." copy and the GLOSSARY term "Placeholder Member"; `changeError` name; one catalog key for the Make Admin message; doc 05 allows domain packages to call each other's pure functions | A test with Former Members in the 50 limit waits for removal (FR-M7, M3) | — |
