# 05 · Architecture

Status: Draft · Last updated: 2026-10-01 · Depends on: ADR-0004, 0005, 0006, 0008, 0011, 0012, 0014, 0015, 0016; Q25, Q56, Q57, Q64–Q67, Q75

## System context

```
┌──────────────────────┐   HTTPS (prod) / HTTP localhost (dev)   ┌──────────────────────┐      ┌──────────────┐
│  iOS app (SwiftUI)   │ ─────────── REST + JSON /v1 ──────────▶ │  Go API server       │ ───▶ │ Postgres 18  │
│  no domain data      │ ◀────────── problem+json errors ─────── │  single deployable   │      └──────────────┘
│  on device           │                                         │  + background jobs   │
└─────────▲────────────┘                                         └───┬──────────┬───────┘
          │ push (later)                                             │          │ email (later; dev: fixed code)
     ┌────┴─────┐            FCM HTTP v1 (later, Q74)                │          ▼
     │  APNs    │ ◀──────────── Firebase Cloud Messaging ◀───────────┘     Email provider (TBD)
     └──────────┘
```

- **Single source of truth**: the server. The app holds state in memory only (ADR-0005) and computes no money (ADR-0006).
- **Environments**: `development` only for now, run with Docker Compose on the developer's Mac (Q25). `production` is designed for but not provisioned. Hosting research is in `docs/research/hosting-and-email.md`.

## Repository layout (ADR-0004)

```
api/            openapi.yaml: the contract (ADR-0012)
server/         Go module
ios/            Xcode project + local Swift packages
docs/           requirements, ADRs, research, conversations
docker-compose.yml   postgres + server (+ mailpit later)
.github/workflows/   CI (doc 09)
```

## Server (Go)

```
server/
  cmd/server/            main: config → wiring → HTTP server + job runner; prints base URL and all routes at startup
  internal/
    ledger/              PURE money logic: Splits, rounding (ADR-0010), conversion (ADR-0007), Balances, Settle-up Suggestions
    auth/                Users, passwords (Argon2id), sessions (ADR-0011), one-time codes (ADR-0016)
    groups/              Groups, Members, Placeholders, Claims, Invite Links, Sub-Groups, lifecycle
    expenses/            Expenses, Approvals, Disputes, Change Requests
    settlements/         Settlements, Confirmations
    activity/            Activity History append + queries
    notify/              Notifications, preferences, push sender interface (FCM adapter later)
    jobs/                Auto-acceptance, reminders, Closing → Closed (idempotent)
    store/               sqlc-generated queries + transaction helper; implements the domain repository interfaces
    httpapi/             oapi-codegen strict-server adapters: HTTP ⇄ domain, auth middleware, problem+json, idempotency
      apigen/            generated from api/openapi.yaml (scripts/generate-api.sh); never edited by hand
    platform/            config (env vars), logging (slog), clock, ID generation
    app/                 composition root: opens the database, migrates, wires services into httpapi (used by cmd/server and HTTP tests)
    pgtest/, apptest/    test harness: isolated schema per test on real Postgres; fully wired server over httptest
  migrations/            goose SQL, embedded, forward-only
```

**Dependency rule (ADR-0015).** Each domain package holds its entities, a service (its use cases) and the repository interfaces it needs. `store` implements those interfaces and `httpapi` calls the services. Domain packages may import `ledger`, `platform` and each other's interfaces, never `store`, `httpapi`, pgx or `net/http`. depguard enforces this.

**Key flows.**
- *Write path*: `httpapi` decodes the request and checks the idempotency key → the service validates and authorizes → `ledger` computes → `store` writes the domain change + Activity History entry + Notifications in **one transaction** → response.
- *Balances*: computed by `ledger` from accepted Expenses' Shares and confirmed Settlements. They may be cached in a per-Group table updated in the same transaction (decided in the data-model ticket; doc 06).
- *Background jobs*: a ticker in the same process scans for due items (`pending_since + 7 days`, reminder at day 5, `closing_since + 7 days`) and processes each in its own transaction. Using `SELECT … FOR UPDATE SKIP LOCKED` makes it safe if more than one instance ever runs.
- *One-time codes*: an `OTPSender` interface. `FixedCodeSender` (`123456`) is allowed only when `APP_ENV=development`, and the server refuses to start otherwise (ADR-0016). `EmailCodeSender` comes later.
- *Push*: a `PushSender` interface, a no-op in development (Q74). The FCM HTTP v1 adapter uses `x/oauth2` (Q61).

**Configuration**: environment variables only: `APP_ENV`, `DATABASE_URL`, `HTTP_ADDR` (default `:8080`), `OTP_MODE`, and later `FCM_*` and `EMAIL_*`. `.env.example` is committed; `.env` never is.

## iOS app (Swift)

```
ios/
  SplitsApp/                     composition root: @main, dependency wiring, deep-link router, push/notification delegate
  Packages/
    Domain/                      entities (Group, Member, Expense, Share, Settlement, Balance, Money…), repository protocols, use cases
    Data/                        repository implementations; mappers generated-type ⇄ entity; Keychain token store
    Infrastructure/
      APIClient/                 swift-openapi-generator client (Generated/, from api/openapi.yaml) + auth middleware (token attach/refresh)
      PushNotifications/         Firebase Messaging, notification categories and actions (FCM wiring later)
    Presentation/
      DesignSystem/              shared components, colours, typography
      Features/
        Auth/  Groups/  Expenses/  Settlements/  Balances/  Activity/  Notifications/  Settings/
          Views/  ViewModels/  Components/
```

**Dependency rule (ADR-0015)**: `Presentation → Domain ← Data → Infrastructure`. Only `SplitsApp` imports every layer. Domain is pure Swift.

- **State**: `@Observable` ViewModels hold in-memory state, loaded from the API when a screen opens and on pull-to-refresh. Nothing is persisted except the Keychain tokens, the FCM token and non-financial UI preferences (ADR-0005). `URLSession` is configured with `urlCache = nil`.
- **Use cases** exist only where they combine more than one repository call or have several entry points (for example, Approve from both a screen and a notification action). Otherwise ViewModels call repository protocols directly (ADR-0015).
- **Money**: `Money { minorUnits: Int64, currency: CurrencyCode }` can format itself for display. It has no arithmetic (ADR-0006). Live split previews call the server.
- **Errors**: `problem+json` `type` URIs map to typed Domain errors, which map to localized messages.
- **Launch and session**: an app-level session state (`launching`, `signedOut`, `needsVerification`, `signedIn`) drives the root view. The APIClient auth middleware refreshes once on `401` (single-flight) and, if the refresh fails, reports session expiry to the composition root, which shows the alert and clears the Keychain (FR-U1, FR-U2).
- **Connectivity**: `NWPathMonitor` plus classification of API failures feed one app-level connectivity state that shows the blocking overlay (FR-U3). The Debug-only error details panel is compiled under `#if DEBUG`.
- **Navigation**: a `TabView` with Home, Report and Settings, each with its own `NavigationStack`. The deep-link router picks the tab and pushes the screen (FR-U4).
- **Networking**: the base URL comes from the build configuration (`Debug` → `http://localhost:8080`). Debug's Info.plist sets `NSAllowsLocalNetworking`; Release requires HTTPS (Q75).
- **Deep links**: one `DeepLink` enum (`expense(id)`, `settlement(id)`, `group(id)`, `balances(groupID)`, `join(inviteToken)`) handled by the router in `SplitsApp`. Notification taps and (later) Universal Links feed it.

## Cross-cutting decisions index

| Concern | Decision |
|---|---|
| Source of truth / no device data | ADR-0005 |
| Money computation | ADR-0006, ADR-0002, ADR-0007, ADR-0010 |
| Agreement model | ADR-0009 |
| Auth | ADR-0003, ADR-0011, ADR-0016 |
| Members and Claims | ADR-0013, ADR-0017 |
| API style | ADR-0012 |
| Data access | ADR-0014 |
| Layering | ADR-0015 |
| Push | ADR-0008 |
