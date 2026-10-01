# Implementing #5: server skeleton (/implement)

Follows [10](10-to-spec-third-run.md). Branch `feat/server-skeleton`.

| UTC | Who | What happened | Decided | Details |
|---|---|---|---|---|
| 2026-10-01 11:43:00Z | User | Ran `/implement #5` | — | — |
| 2026-10-01 11:45:00Z | Claude | Read #5, ADR-0014/0015/0016 and docs 05, 07, 08, 09; took the ticket's named seams as agreed | Test at the startup seam (`run`) and the HTTP seam (`httptest` on real Postgres) | Plus small unit seams in `platform`: logger redaction, UUIDv7 (RFC 9562 vector), fake clock. |
| 2026-10-01 11:50:00Z | Claude | Local Postgres already holds port 5432 on the developer's Mac | Compose publishes Postgres on 127.0.0.1:55432 by default | Set in `.env.example` (`POSTGRES_PORT`, `TEST_DATABASE_URL`). Created a local `.env` from it. |
| 2026-10-01 11:52:00Z | Claude | Built the skeleton test-first | New packages `app` (composition root), `pgtest` and `apptest` (test harness); added to doc 05 | Each test gets its own schema (`search_path=<schema>,public`); a TCP proxy lets a test cut the database for the `/readyz` 503 case. Baseline migration creates `citext` in `public` so test schemas share it. |
| 2026-10-01 11:57:00Z | Claude | govulncheck flagged stdlib vulnerabilities in go1.26.1 | Pin `toolchain go1.26.8` in go.mod and the Dockerfile | `setup-go` reads the toolchain from go.mod, so CI gets the fixed release too. |
| 2026-10-01 11:58:00Z | Claude | Verified end to end | — | `docker compose up --build` logs base URL and routes; `/readyz` 503 with Postgres stopped and 200 after restart; `APP_ENV=production` + `OTP_MODE=fixed` refuses to start; depguard rejects a probe importing `net/http`/pgx from `ledger`; gitleaks clean. Committed `b623a55`. |
| 2026-10-01 12:00:00Z | Claude | Ran `/code-review` against `master` (Standards and Spec agents in parallel) | Fixed: `/readyz` logs a cause category, not driver error text (host/role/db); shared JSON writer; `CutDatabase` drops its unused parameter; `exec` → `mustExec`; comment on pre-logger stderr; CI notes the fuzz step arrives with #8. Not changed: contract-first for health endpoints (owned by #7, blocked by #5); Compose still needs `.env` (no default password committed); `AppEnv` stays a string | Spec axis: all acceptance criteria met. Standards axis: contract-first and the missing log were the hard findings; the log is this file. |
| 2026-10-01 12:01:14Z | Claude | Committed the review fixes; pushing and opening the PR | — | — |
| 2026-10-01 12:03:37Z | Claude | Opened [PR #31](https://github.com/bikkysamuel/splitsDemo/pull/31); first CI run green (`server` 1m28s, `gitleaks`) | Merge waits for the user | — |
