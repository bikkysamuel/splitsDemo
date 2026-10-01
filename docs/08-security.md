# 08 · Security

Status: Draft · Last updated: 2026-10-01 · Depends on: ADR-0003, 0005, 0011, 0013, 0016, 0017; Q9, Q10, Q28, Q32–Q34

## What we protect

| Asset | Threat | Main control |
|---|---|---|
| Accounts | Credential stuffing, guessing, takeover | Argon2id, breached-password list, throttling, revocable sessions |
| Financial records | Tampering, unauthorized viewing | Server-side authorization, append-only history, Approvals |
| Personal data (email, names) | Leaks via logs, the repo or a lost device | No PII in logs, no domain data on device, secret scanning |
| Secrets (DB URL, later FCM and email keys) | Leaking through the public repo | `.env` git-ignored, gitleaks in CI, GitHub push protection |

## Authentication (ADR-0003, ADR-0011)

- Passwords: **Argon2id** (`golang.org/x/crypto/argon2`), parameters chosen to take ≈ 250 ms on the reference setup, stored with salt and parameters. 10–128 characters, no composition rules, rejected if on the shipped common/breached list (Q34).
- Tokens: 32 random bytes each, base64url. Only SHA-256 hashes are stored. The access token lasts 15 min; the refresh token lasts 30 days, is replaced on each use, and reuse revokes the chain.
- Revocation: sign-out (this session); password reset, password change (other sessions) and account deletion (all sessions).
- Throttling: growing delays after 5 failures, per account and per IP. No hard lockout (Q34).
- Responses don't reveal whether an email exists (sign-in errors, password reset).

## One-time codes (ADR-0016)

- 6 digits, 15-minute expiry, at most 5 attempts, stored hashed, single use.
- **Development**: the fixed code `123456` for every account.
- **Guard (hard rule)**: the server refuses to start if `OTP_MODE=fixed` and `APP_ENV` ≠ `development`. A test covers this. A fixed code outside development would let anyone take over any account, including through the automatic Claim (ADR-0017).

## Authorization

- Every service method takes the acting User and checks Group Membership and role (doc 07 rules) before touching data. Handlers never authorize on their own.
- Resources of Groups the User can't see return `404`.
- Admin actions on behalf of Placeholders are recorded with both actor and subject in the Activity History.
- Automatic Claim requires a **verified** email (ADR-0017).

## Data on the device (ADR-0005)

- Keychain only: access and refresh tokens (`kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly`), and later the FCM token.
- No domain data in SwiftData, files, `UserDefaults` or the HTTP cache (`URLSession` `urlCache = nil`). Non-financial UI preferences only in `UserDefaults`.
- Lock-screen Approve/Confirm actions require device authentication (`.authenticationRequired`).

## Transport

- Debug builds: HTTP to localhost via `NSAllowsLocalNetworking` (Q75).
- Release builds: HTTPS only, no ATS exceptions. TLS terminates at the host once one exists.

## Input and API hardening

- All input is validated against the OpenAPI schema (generated) plus domain rules in services.
- Request body limit is 64 KB, and text fields have length limits (note and reason ≤ 500).
- SQL only via sqlc-generated parameterized queries (ADR-0014).
- Idempotency keys are scoped per User.
- Rate limiting on auth endpoints (above) and a general per-User limit (proposed default: 120 requests/minute).

## Logging and privacy (Q28)

- Logs carry IDs only: no passwords, tokens, codes, emails, display names or notes.
- Crashlytics: no custom keys containing PII.
- Account deletion erases the User row, sessions, devices and codes. Display names remain on the resulting Placeholders so Group records stay correct (ADR-0013). The privacy policy page (Q10) must say this.

## Repository and supply chain (Q9, Q61)

- The repo is public: **gitleaks** runs on every PR, GitHub secret scanning and push protection are on, and `.env.example` holds only placeholder values.
- Dependencies are limited to the allowed list. `govulncheck` runs in CI, and SPM package versions are pinned in `Package.resolved`.
- GitHub Actions are pinned to commit SHAs, and workflow `permissions` are read-only by default.

## Before production (tracked in doc 10)

Email provider for real codes, HTTPS host, domain + Universal Links (with apple-app-site-association), APNs key per environment, privacy policy page, and a review of this document against the production setup.
