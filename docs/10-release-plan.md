# 10 · Release plan

Status: Draft · Last updated: 2026-10-01 · Depends on: Q25, Q46, Q69, Q70, Q74; doc 01

## Where things run now

Local only (Q25): `docker compose up` starts Postgres 18 and the Go server on `http://localhost:8080`, and the app runs on the iOS Simulator (Q75). There is no host, domain, email provider or paid Apple Developer Program yet (Q70, Q74).

## Milestones

Each milestone is a set of tickets produced by `/to-tickets` from the `/to-spec` spec, each built with `/implement` (`/tdd` → `/code-review` → `/pr`). A milestone is done when every ticket is merged, CI is green, and its demo runs on the Simulator.

| Milestone | Exit demo | Notes |
|---|---|---|
| **M0 Foundations** | `docker compose up` → server prints routes; app shell builds on `xcode-27` CI; `/healthz` green | Monorepo layout, OpenAPI pipeline both ways, CI jobs, migrations runner, `.env.example`. First ticket. |
| **M1 Core ledger** | Two accounts (code `123456`) share a Group; record equal / exact / percentage / ratio Expenses in two currencies; Balances and Settle-up Suggestions correct; record a Settlement; history shows everything | Agreement is not active yet: Expenses are accepted immediately. The data model already has the Pending states (doc 06), so M2 only switches them on. |
| **M2 Agreement and notifications** | Second account approves, disputes and change-requests; the receiver confirms a Settlement; Auto-acceptance via fake clock; in-app Notification list; `simctl push` notification actions open the right screen | FCM delivery ticket stays **blocked** until the Apple Developer Program is bought. |
| **M3 Groups and insight** | Sub-Group trip; Group goes Active → Closing → Closed → reopened; Former Member; Invite Link created and joined via API; duplicate warning; category chart; balance report | Invite Links don't open the app until a domain exists. |

## Production-readiness track (deferred; each item is a later decision)

| Item | Unblocks | Reference |
|---|---|---|
| Choose and set up a host + managed Postgres 18 | Staging/production, TestFlight against a real server | `docs/research/hosting-and-email.md` §1, §6 |
| Buy a domain | Universal Links for Invite Links, email sending domain, HTTPS API | Q70, research §3.1 |
| Choose an email provider; implement `EmailCodeSender` | Real verification and reset codes; `APP_ENV=production` allowed | ADR-0016, research §2 |
| Apple Developer Program ($99/yr) + APNs key per environment | Push via FCM, TestFlight, App Store | ADR-0008, research §3.2 |
| Privacy policy page | App Store submission | Q10 |
| Production security review | Launch | doc 08 |

Walking through the human-only steps (accounts, DNS, secrets) will use `/wizard` when the time comes.

## Versioning and branching

- App: semantic versioning (`MAJOR.MINOR.PATCH`), with the build number from CI. Server: tags `server-vX.Y.Z`. API: `/v1` until a breaking change is unavoidable.
- Short-lived branches → PR → squash-merge to `master` (Q60). `master` is always releasable.
- Database migrations are forward-only. A destructive migration needs explicit approval (CLAUDE.md).

## After each milestone

Run `/retro` to improve CLAUDE.md, CI checks and coding standards before starting the next milestone.
