# 01 · Product requirements

Status: Draft · Last updated: 2026-10-01 · Depends on: Q1, Q46, Q47, F1–F13, ADR-0005, ADR-0006

Terms in **bold capitals** (Group, Member, Expense…) are defined in [`GLOSSARY.md`](../GLOSSARY.md).

## Purpose

Splits helps groups of people who share costs (flatmates, trips, events) record Expenses, agree on them, see who owes whom, and record Settlements, without arguments about whether the numbers are right.

## Audience and ambition

- A portfolio/learning project **built to production standards** (Q1): tested, secure and maintainable.
- A public App Store launch must stay possible, so App Store rules that affect the design (in-app account deletion, privacy policy) are built in now (Q10).
- Runs **locally only** for now (Q25): Go server + Postgres in Docker Compose on the developer's Mac, iOS app on the Simulator (Q75). Hosting, domain, email provider and the paid Apple Developer Program are decided later (Q25, Q70, Q74).

## Product principles

1. **The server is the single source of truth.** All money calculation happens on the server (ADR-0006), and the app keeps no domain data on the device (ADR-0005).
2. **Nothing counts until it's agreed.** Expenses and Settlements affect Balances only once accepted (ADR-0009).
3. **History is permanent.** Groups are never deleted, only Closed. Expenses and Settlements are Withdrawn, never erased. Every event lands in the Activity History.
4. **Exact money.** Integer minor units, deterministic rounding (ADR-0002, ADR-0010).

## Platforms

- iPhone, iOS 26 minimum, built with Xcode 27 (Q26).
- One Go HTTP API (REST + JSON, OpenAPI contract-first, ADR-0012). It is client-agnostic, but iOS is the only client in scope.

## Scope by milestone (Q46)

Everything is designed now and built in milestones. Each milestone ends with working software.

| Milestone | Capabilities |
|---|---|
| **M0 Foundations** | Monorepo scaffold, Docker Compose, CI, OpenAPI pipeline, empty app shell |
| **M1 Core ledger** | Accounts (sign-up with email verification code, sign-in, sign-out, change password, reset password, delete account); Groups and Group Currency; adding Members by email and Placeholder Members; automatic Claim by verified email; Admin role; Expenses with Categories, notes, per-Expense currency and Exchange Rate; Splits equal / exact / percentage / ratio with split preview; Balances; Settlements (including partial); Settle-up Suggestions; Withdrawn items; Activity History per Group and "My activity"; basic filter by Member, Category and date |
| **M2 Agreement and notifications** | Approvals and Pending state; Disputes (one reason + one reply); Change Requests; Settlement Confirmation; Auto-acceptance after 7 days with a day-5 reminder; Admins acting for Placeholders; in-app Notification list; notification actions and deep links (tested on the Simulator); Notification preferences (mute informational per Group); push delivery via FCM (**blocked** until the Apple Developer Program is bought) |
| **M3 Groups and insight** | Sub-Groups; Group lifecycle Active → Closing → Closed (+ reopen); Former Members; Invite Links (built, but only open the app once a domain exists); duplicate-Expense warning; per-Member Category bar chart (Share/Paid); in-app balance report with Settle-up Suggestions |

## Development-only behaviour (until production setup)

| Area | Development | Production (later) |
|---|---|---|
| Verification and reset codes | Fixed code `123456`, nothing sent (ADR-0016) | Random codes sent by email; provider to be chosen |
| Invite Links | Created, shared, revocable; tapping doesn't open the app | Universal Links on our own domain |
| Push notifications | Simulated with `xcrun simctl push` | FCM → APNs (ADR-0008) |
| Hosting | Docker Compose on localhost | To be chosen; research in `docs/research/hosting-and-email.md` |

## Later (not v1)

Receipt photos, recurring Expenses, several payers per Expense, export (CSV/PDF), comment threads on Disputes, cross-Group spending chart, Exchange Rates pre-filled from a rate service, offline writes, other languages.

## Out of scope

Friends list or Expenses outside Groups (Q6); real payments, Apple Pay or payment links (Q23); more than one Group Currency per Group; iPad, Mac, Android and web clients; Sign in with Apple or other social logins (ADR-0003).

## Success criteria for v1

- Every capability in M1–M3 is implemented with tests, passes CI, and works end to end on the Simulator against the local server.
- The ledger's fuzz tests prove the money invariants (doc 09).
- Two accounts on the Simulator can run the full flow: add Expense → approve or dispute → settle → confirm → Group reaches Closing.
