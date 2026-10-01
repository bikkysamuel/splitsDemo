# Open questions and decision log

Status: Living · Last updated: 2026-10-01

Every product or technical decision goes through this list. Docs 01–10 point at a Q-id rather than leaving a silent TBD. When a question is answered, record the answer here and link the ADR or doc that holds the detail. Terms are defined in `GLOSSARY.md`.

## Decided

| Q | Question | Answer | Detail |
|---|---|---|---|
| Q1 | Audience and launch goal | Portfolio/learning project built to production standards; a public App Store launch must stay possible | 01 |
| Q2 | Can Members exist without a User? | Yes: Placeholder Members, claimable later | GLOSSARY |
| Q3 | How do Users sign in? | Email + password, handled by our own server | ADR-0003 |
| Q4 | Currency model | Per-Expense currency, converted into the Group Currency (Q36). Integer minor units, never floats | ADR-0007 |
| Q5 | Offline behaviour | No offline writes and no local data (Q49) | ADR-0005 |
| Q6 | Expenses outside a Group? | No | GLOSSARY |
| Q7 | Repo layout | Monorepo: `ios/`, `server/`, `api/`, `docs/` | ADR-0004 |
| Q8 | Hosting | Docker Compose locally; one managed staging host (provider: Q25) | 05 |
| Q9 | Public repo OK? | Yes, with secret scanning and push protection | 08 |
| Q16 | Several payers per Expense? | Later (not v1) | 01 |
| Q17 | Debt simplification | Yes: Settle-up Suggestions | GLOSSARY |
| Q18 / Q19 | What SwiftData caches; how the cache refreshes | Obsolete: no local cache (Q49) | ADR-0005 |
| Q23 | Real payments? | No. Settlements only record that a payment happened outside the app | GLOSSARY |
| Q29 | Push notifications in v1? | Yes, via Firebase Cloud Messaging | ADR-0008 |
| Q36 | Multi-currency mechanics | Original Amount + currency + manually entered Exchange Rate, converted once into the Group Currency; a rate change is an edit | ADR-0007 |
| Q37 | Who approves an Expense; does Pending count? | Every Member in the split except the creator and Placeholder Members; Pending is excluded from Balances; Auto-acceptance after 7 days, reminder at day 5 | ADR-0009 |
| Q38 | Dispute vs Change Request | Separate actions; any edit makes the Expense Pending again and is kept in history | ADR-0009 |
| Q39 | Settlement confirmation | Pending until the receiving Member confirms; Auto-acceptance after 7 days | ADR-0009 |
| Q40 | Sub-Groups | Members come from the Parent Group; separate Balances; visible to its Members only; one level deep | GLOSSARY |
| Q41 | Group lifecycle | Active → Closing (automatic when all Balances are zero and nothing is Pending or Disputed; all Members notified) → Closed after 7 days unless kept open. Closed is read-only; any Member can reopen. A Parent Group can't close while a Sub-Group is open | GLOSSARY |
| Q42 | Duplicate warning | Same Group + same Category + same converted amount within 24 h → warning only | 03 |
| Q43 | Chart | Group screen: per-Member bar chart by Category, Share/Paid switch, accepted Expenses only, Swift Charts. Cross-group chart later | 03 |
| Q44 | Balance report | In-app: net Balances + Settle-up Suggestions with a Settle button. Export later | 03 |
| Q45 | Notification behaviour | FCM; lock-screen Approve/Confirm need device authentication; Dispute opens the app; tapping opens the relevant screen; an in-app Notification list is the reliable record | ADR-0008 |
| Q46 | Build staging | Design everything now; build in milestones M1 core ledger → M2 agreement and notifications → M3 Sub-Groups, closure, chart, duplicate warning, report | 10 |
| Q47 | Remaining suggestions | In: Categories, note field, split by ratio, Settle-up Suggestions, basic filter. Later: receipts, recurring, several payers, export. Out: friends list, real payments. Archive replaced by Closed | 01 |
| Q48 | Where calculations happen | Server only; the app validates input and displays results; a split-preview endpoint gives live feedback | ADR-0006 |
| Q49 | Local caching | None. Only the Keychain auth token, the FCM device token and non-financial UI preferences are on device | ADR-0005 |
| Q10 | App Store requirements in v1 | In-app account deletion and a privacy policy page | 08 |
| Q11 | How invites work | Invite Link (share sheet; joins directly; expires in 7 days; Admin-revocable) + add by email. Refined by Q73/Q76: Invite Links built now, working once a domain exists | ADR-0017 |
| Q15 | Rounding leftovers | Largest remainder, ties by join order; conversion rounds half-up once | ADR-0010 |
| Q20 | Who may edit/delete | Anyone may record an Expense paid by someone else; Approvers = payer + Split Members − creator − Placeholders. Only the creator edits; others use a Change Request. Deleting = Withdrawn, which needs Approval if the item was accepted | ADR-0009 |
| Q21 | Roles | Admin and Member. Any Member adds Members, Expenses, Settlements, Sub-Groups. Admins rename, remove Members, grant Admin, act for Placeholders. Groups are never deleted, only Closed | GLOSSARY |
| Q24 | Partial Settlements | Any positive amount; warn (don't block) on overpayment | 03 |
| Q50 | Acting for Placeholder Members | Any Admin, recorded in Activity History as on-behalf | ADR-0009 |
| Q51 | Notification preferences | Action-required Notifications always in the in-app list; informational ones mutable per Group. Push delivery per kind refined by Q83 | 03 |
| Q53 | Activity History | Per-Group append-only history, edits shown as diffs, Group Members only (Sub-Group history not visible in parent) + a cross-Group "My activity" feed | GLOSSARY |
| Q54 | Discussing disputes | One reason + one creator reply, then edit or withdraw. Threads later | 03 |
| Q55 | Term: ratio vs shares | "Split by ratio"; Share means only the amount one Member owes | GLOSSARY |
| Q12 | Claiming a Placeholder Member | Automatically when a User verifies the email the Placeholder was added with (Q76); or via an Invite Link aimed at the Placeholder; or by picking "I'm already in this group as…" with Admin confirmation | ADR-0017, GLOSSARY |
| Q13 | Session model | Opaque tokens hashed in Postgres: 15-min access + 30-day rotating refresh with reuse detection; Keychain only; revoked on sign-out, reset, deletion | ADR-0011 |
| Q14 | API style | REST + JSON, OpenAPI 3.1 contract-first, oapi-codegen + swift-openapi-generator, problem+json, `/v1`, cursor pagination, Idempotency-Key | ADR-0012 |
| Q22 | Leaving/removal with a Balance | Blocked while Balance ≠ 0 or anything Pending; afterwards the person becomes a Former Member | GLOSSARY |
| Q26 | Minimum iOS version | iOS 26 (built with Xcode 27 / iOS 27 SDK) | 04 |
| Q28 | Observability | Go `log/slog` JSON with request IDs, `/healthz`; Firebase Crashlytics on iOS; no secrets, emails or names in logs; OpenTelemetry later | 05 |
| Q30 | Limits and targets | ≤50 Members/Group, ≤200 Groups/User, fast at 10k+ Expenses/Group; p95 < 300 ms on staging; daily backups + PITR; 99.5% if launched | 04 |
| Q31 | Localization and accessibility | English only, all strings in String Catalogs, locale-aware currency and dates; VoiceOver, Dynamic Type, 44pt targets, dark mode required on every screen | 04 |
| Q32 | Email verification | Required at sign-up via a 6-digit code (15 min) | 08 |
| Q33 | Password reset flow | 6-digit emailed code, 15 min, 5 attempts, revokes all sessions, no account-existence leak; Mailpit locally. **Provider still open** (research pending) | 08 |
| Q34 | Password rules and throttling | 10–128 chars, no composition rules, blocklist of common/breached passwords shipped with the server, Argon2id, growing delays per account and IP after 5 failures, no hard lockout | 08 |
| Q52 | Account deletion | User erased; their Members become Placeholder Members; warn but never block | ADR-0013 |
| Q27 | Go stack | stdlib `net/http` + oapi-codegen strict server, pgx v5 + sqlc, goose forward-only migrations, `NUMERIC` ↔ `math/big.Rat`, Argon2id via x/crypto, env-var config | ADR-0014 |
| Q56 | Go server structure | One deployable server, packages by domain under `server/internal/`; `ledger` is pure money logic; `store` = sqlc; `httpapi` = generated adapters | 05 |
| Q57 | iOS architecture | **Clean architecture** (user override), organised neatly; plus Observation view models, local SPM packages, protocol at the API seam, Swift 6 strict concurrency, no TCA. Layer details: Q64–Q66 | 05 |
| Q58 | Testing strategy | Ledger table + fuzz invariant tests; endpoint integration tests on real Postgres; contract/codegen drift checks; Swift Testing with fake API client; nightly XCUITest smoke; no global coverage gate; regression test with every fix | 09 |
| Q59 | CI and merge gates | Path-filtered GitHub Actions: server (gofmt, vet, golangci-lint, race tests on Postgres, govulncheck), ios (swift-format, xcodebuild test), api (lint + codegen drift), gitleaks always; protected `master`, PR + green checks | 09 |
| Q60 | Git conventions | Short-lived `feat/ fix/ docs/ chore/` branches off `master`, Conventional Commits, squash-merge, `/pr` bodies, keep `master` name | CLAUDE.md |
| Q61 | Dependency rules | Stdlib first; justification per new dependency, ADR if hard to replace; allowed list for Go and iOS; FCM via HTTP v1 + x/oauth2 (no Admin SDK); anything else needs approval | CLAUDE.md |
| Q62 | Formatting and lint | gofmt + golangci-lint; swift-format (no SwiftLint); code names follow GLOSSARY | CLAUDE.md |
| Q63 | Claude Code autonomy | May branch, commit, push, open PRs after /tdd + /code-review. Must ask before merging, touching `master`, adding dependencies, destructive migrations, changing Accepted ADRs, secrets or paid services. Must stop on ADR/GLOSSARY conflicts | CLAUDE.md |
| Q64 | iOS layers | `SplitsApp` composition root; packages Domain (pure), Data, Infrastructure (APIClient, PushNotifications), Presentation (Features, DesignSystem); `Presentation → Domain ← Data → Infrastructure`; `Money` formats but has no arithmetic | ADR-0015 |
| Q67 | Go dependency rule | Per domain package: entities + service + repository interface; `store` implements, `httpapi` calls; depguard enforces; no project-wide entity/usecase trees | ADR-0015 |
| Q65 | iOS use cases | Only where they combine more than one repository call or have several entry points (screen + notification action); otherwise ViewModels call repository protocols | ADR-0015 |
| Q66 | API type mapping | Generated types stay in `Data`; tested mappers produce Domain entities and typed errors | ADR-0015 |
| Q25 | Hosting | **Local only for now** (Docker Compose on the developer's Mac). Host, staging and production deferred, to be decided later (research kept in docs/research/hosting-and-email.md) | 10 |
| Q33b | Email provider | **None for now.** Accounts are created with a unique email + password. Knock-on effects: Q71, Q72 | 08 |
| Q68 | Shared understanding | Confirmed by the user, subject to the knock-on items Q71–Q75 from Round 9 | — |
| Q69 | CI Xcode | `xcode-27` preview runner as a required check, pinned to Xcode 27.0; switch to `macos-27` when GitHub ships it | 09 |
| Q70 | Domain | **None for now**; local server only. Purchases decided later. Knock-on effects: Q73, Q74 | 10 |
| Q71 | Email verification without a provider | Keep verification. Development: fixed code `123456` for every account; production: emailed random codes. Server refuses fixed codes outside development | ADR-0016 |
| Q72 | Password reset without a provider | Same code flow as Q33; `123456` in development, emailed in production. Signed-in users can also change their password | ADR-0016 |
| Q74 | Push without the paid Apple program | Later. M2 builds the in-app Notification list first, then actions and deep links tested with `xcrun simctl push`, then FCM delivery (waits on the purchase). ADR-0008 stands | ADR-0008 |
| Q75 | Reaching the local server | Simulator only, `http://localhost:8080`; Debug allows local HTTP (`NSAllowsLocalNetworking`), Release requires HTTPS; base URL from build settings; `docker compose up` starts everything; **server prints every API route and the base URL at startup** | 05 |
| Q73 | Invites without a domain | **Build Invite Links now** (create, share, expire, revoke, join API + screen); they only open the app once a domain exists for Universal Links | ADR-0017 |
| Q76 | How Users get into Groups | Any Member (Admin or not) adds a person by email: an existing User joins at once with an in-app Notification; otherwise a Placeholder carrying that email is created and is Claimed automatically when that email is verified at sign-up. Unregistered people remain Placeholders | ADR-0017 |
| Q77 | App launch and session expiry | Splash → saved tokens checked with `GET /v1/me` → Home; no or invalid session → sign-in/sign-up; unverified → verification. A `401` triggers one silent refresh; only when that fails does the app show a "session expired" alert and sign out | 03 FR-U1–U2, 05 |
| Q78 | No connection and server errors | Blocking overlay ("No connection" / "Server unavailable") that retries with backoff + Retry button, keeping in-memory screen state underneath. Debug builds show HTTP status, problem `type`/`detail`, `X-Request-ID` and `URLError` code; Release builds don't contain it | 03 FR-U3, 05 |
| Q79 | Navigation and visual principle | Tab bar: Home, Report, Settings. Home holds the summary, the Groups list, a Notification bell with badge and My activity. Simple screens: one primary action each, details one tap away | 03 FR-U4, 05 |
| Q80 | Home summary across currencies | Server returns the User's net Balance **per currency** across their Groups, or "all settled", plus their Balance in each Group (`GET /v1/me/summary`). No conversion between currencies | 03 FR-B3, 07 |
| Q81 | Report tab | Group picker (last used remembered). Summary: that Group's Balances (M1) + Category bar chart (M3); detail: full balance report + per-Member breakdown. Keeps Q43; cross-Group chart stays Later | 03 FR-R3 |
| Q82 | Settings contents | Theme (System/Light/Dark, extendable), default currency for new Groups, push toggles per Notification kind, reminder toggle, change password, sign out, delete account, app version. Theme and default currency in `UserDefaults` | 03 FR-U5 |
| Q83 | Push preferences per kind | Toggles control **push delivery only**; every Notification still lands in the in-app list, action-required ones can't be hidden there, per-Group mute stays. Stored on the server. Refines Q51 | 03 FR-N5, 06, 07 |
| Q84 | Reminder setting | One toggle turns the day-5 reminder push on or off; the in-app reminder and 7-day Auto-acceptance are unchanged (Q37) | 03 FR-N6 |
| Q85 | Contract test tooling (2026-10-01, #7) | **kin-openapi** (`openapi3filter`) validates every HTTP-seam response against `api/openapi.yaml`, in test code only (the generated server embeds no spec). **vacuum** lints the contract in the `api` CI job (recommended rules, warnings fail; `oas3-unused-component` off because shared schemas precede their endpoints) | 09, CLAUDE.md |
| Q86 | One check entry point (2026-10-01, M0 retro) | `scripts/check.sh [server\|api\|ios\|secrets\|all]` runs every CI check with pinned versions, and every workflow calls it. gitleaks runs from a digest-pinned image instead of the third-party action. The conversation log has no hand-kept index (file names sort chronologically). Refines Q59 | 09, CLAUDE.md |
| Q87 | `ledger` Split and conversion details (2026-10-01, #8; confirmed by the user) | Every Split Input must be > 0: exact amounts (whole Group Currency minor units, summing to the converted total), percentages (≤ 2 dp, summing to exactly 100) and ratio weights (integers). A Share can still be 0 when the total has fewer minor units than Members. **Exchange Rates have at most 2 decimal places** (the user's choice, replacing D4's 10), so a rate below 0.01 can't be entered: pick the larger-unit currency as the Group Currency when rates are small. Minor units follow ISO 4217: 0 (JPY, KRW, …), 3 (KWD, BHD, …), 4 (CLF, UYW), otherwise 2; which codes a Group may use is decided with Groups (#14). Invalid Splits carry a stable `Reason` that doubles as the problem+json field-error `code` | 03 FR-E2, FR-E5 |
| Q88 | `ledger` Balances and Settle-up Suggestions (2026-10-01, #9; greedy confirmed by the user) | `ledger.Balances` takes every Member's `join_seq` (Former Members included), the Expenses with their stored Shares and the Settlements, and returns one Balance per Member by FR-B1, counting only Accepted and WithdrawalPending items (a confirmed Settlement is Accepted). Every item is checked whatever its state; a broken invariant (unknown Member or state, Shares ≠ amount, non-positive amount, totals beyond int64) returns `ErrInconsistentLedger`, a bug or corrupt data, never user input. `ledger.SettleUpSuggestions` is greedy: the Member who owes the most pays the Member owed the most, ties to the lowest `join_seq`, repeated until every Balance is zero, so at most (Members with a non-zero Balance − 1) payments, listed in the order they are chosen | 03 FR-B1, FR-B2 |

## Open

| Q | Question | Blocked by |
|---|---|---|
| — | None. All questions decided as of 2026-10-01 | |

## Proposed defaults (introduced while writing docs 01–10; accept or change at review)

| D | Default | Where |
|---|---|---|
| D1 | Note, Dispute reason and reply: max 500 characters | 03 FR-E1, FR-P3 |
| D2 | Display names unique within a Group | 03 FR-M4 |
| D3 | Group Currency can't change once any Expense exists | 03 FR-G1 |
| D4 | Percentages up to 2 decimal places, summing to exactly 100; Exchange Rates at most 2 decimal places (changed from 10 by Q87) | 03 FR-E2, FR-E5 |
| D5 | An Expense with no Approvers (for example, only the creator and Placeholders) is accepted immediately | 03 FR-P1 |
| D6 | The 50-Member limit counts Placeholders and Former Members | 03 FR-G2 |
| D7 | Fixed Category list: food & drink, groceries, transport, accommodation, rent, utilities, entertainment, shopping, health, travel, other | 06 |
| D8 | New Expense state **WithdrawalPending**: withdrawing an accepted item still counts until approved | 06 |
| D9 | In M1 (before agreement exists) Expenses and Settlements are accepted immediately; M2 switches Pending on | 10 |
| D10 | Balances compute in < 100 ms for 50 Members / 10k Expenses; warm launch shows content in < 1 s | 04 NFR-P2, P4 |
| D11 | Lists: default page 50, max 200 | 04, 07 |
| D12 | Optimistic concurrency with `version`; stale write → `409` | 04 NFR-R4, 07 |
| D13 | `/readyz` endpoint as well as `/healthz` | 04, 07 |
| D14 | Server-generated UUIDv7 IDs; idempotency keys kept 24 h | 06 |
| D15 | Money JSON shape `{minor, currency}`; Exchange Rates as strings | 07 |
| D16 | Duplicate and overpayment warnings returned as `422 confirmation-required`, resent with `acknowledge_warnings` | 07 |
| D17 | A resource the User can't see returns `404`, not `403` | 07, 08 |
| D18 | Request body max 64 KB; general rate limit 120 requests/min per User; Argon2id tuned to ≈ 250 ms | 08 |
| D19 | Activity History made append-only by a database trigger | 06 |

## Requested features (2026-10-01)

| F | Feature (as requested) | Resolved by |
|---|---|---|
| F1 | Warn when two Members add the same type of expense in a Group | Q42 |
| F2 | Bar chart of how each User pays per Category | Q43 |
| F3 | Report of who owes / is owed | Q44 |
| F4 | Different currency per Expense, editable Exchange Rate | Q36 |
| F5 | Members must approve a new Expense; can Dispute or make a Change Request | Q37, Q38 |
| F6 | Notify on Settlement; Member can Dispute it | Q39 |
| F7 | Push notifications (Firebase) for new Expenses and anything needing attention | Q45 |
| F8 | Notification action buttons | Q45 |
| F9 | Notifications open the specific screen | Q45 |
| F10 | Create a Sub-Group; close it when settled | Q40, Q41 |
| F11 | Group enters Closing when settled, auto-closes, notifies all Members | Q41 |
| F12 | Backend is the single source of truth; all calculations via API | Q48 |
| F13 | No local data caching | Q49 |
