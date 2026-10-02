# Implementing #14: create a Group and see it on Home (unattended M1 run)

Follows [00](00-implement-auth-refresh-signout.md). Branch `feat/groups-create`, built on `feat/auth-refresh-signout` (PR #41).

| UTC | Who | What happened | Decided | Prompt |
|---|---|---|---|---|
| 2026-10-02 03:40:00Z | Claude | Opened PR #41 for #11 and started #14 as the next ticket of the unattended run | — | — |
| 2026-10-02 03:41:15Z | Claude | Contract for `GET/POST /v1/groups` and `GET/PATCH /v1/groups/{id}`; `ledger.IsActiveCurrency`; migration for `groups` and `members` (one-level trigger, case-insensitive display names); the `groups` service with authorization (404 for invisible Groups, Admin-only rename, version check, 200 limit under a row lock); monotonic UUIDv7s so ID order is creation order; iOS Home list, Create Group, Group screen and the default-currency setting, all test-first | Each Member's display name is per Group (Users have no profile name); see Q91 for the rest | — |
