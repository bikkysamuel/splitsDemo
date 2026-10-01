# Implementing #5: server skeleton (/implement)

Follows [10](10-to-spec-third-run.md). Branch `feat/server-skeleton`.

| UTC | Who | What happened | Decided | Prompt |
|---|---|---|---|---|
| 2026-10-01 11:42:20Z | User | Ran `/implement #5` | — | `/implement #5` |
| 2026-10-01 11:45:00Z | Claude | Read #5, ADR-0014/0015/0016 and docs 05, 07, 08, 09; took the ticket's named seams as agreed | Test at the startup seam (`run`) and the HTTP seam (`httptest` on real Postgres) | — |
| 2026-10-01 11:50:00Z | Claude | Local Postgres already holds port 5432 on the developer's Mac | Compose publishes Postgres on 127.0.0.1:55432 by default | — |
| 2026-10-01 11:52:00Z | Claude | Built the skeleton test-first | New packages `app` (composition root), `pgtest` and `apptest` (test harness); added to doc 05 | — |
| 2026-10-01 11:57:00Z | Claude | govulncheck flagged stdlib vulnerabilities in go1.26.1 | Pin `toolchain go1.26.8` in go.mod and the Dockerfile | — |
| 2026-10-01 11:58:00Z | Claude | Verified end to end | — | — |
| 2026-10-01 12:00:00Z | Claude | Ran `/code-review` against `master` (Standards and Spec agents in parallel) | Fixed: `/readyz` logs a cause category, not driver error text (host/role/db); shared JSON writer; `CutDatabase` drops its unused parameter; `exec` → `mustExec`; comment on pre-logger stderr; CI notes the fuzz step arrives with #8. Not changed: contract-first for health endpoints (owned by #7, blocked by #5); Compose still needs `.env` (no default password committed); `AppEnv` stays a string | — |
| 2026-10-01 12:01:14Z | Claude | Committed the review fixes; pushing and opening the PR | — | — |
| 2026-10-01 12:03:37Z | Claude | Opened [PR #31](https://github.com/bikkysamuel/splitsDemo/pull/31); first CI run green (`server` 1m28s, `gitleaks`) | Merge waits for the user | — |
