# 07 · API contract

Status: Draft · Last updated: 2026-10-01 · Depends on: ADR-0006, 0011, 0012, 0016, 0017; docs 03, 06

`api/openapi.yaml` (OpenAPI 3.1) is the source of truth (ADR-0012). This document gives the conventions and the endpoint catalogue that the contract will implement. If the two disagree, the YAML wins and this doc gets fixed.

## Conventions

- **Base**: `/v1`. Development base URL: `http://localhost:8080/v1`. The server prints all routes at startup (NFR-O3).
- **Auth**: `Authorization: Bearer <access token>` on everything except `auth/*` sign-up, sign-in, verify, refresh and reset endpoints, and the health checks.
- **Money** in JSON: `{ "minor": 12345, "currency": "INR" }`. `minor` is an integer. Exchange Rates are **strings** holding exact decimals (`"0.4021"`), never JSON numbers.
- **IDs**: UUID strings. **Timestamps**: RFC 3339 UTC. **Dates**: `YYYY-MM-DD`.
- **Writes**: `Idempotency-Key: <uuid>` is required on every POST/PATCH/DELETE (NFR-R1). Updates send `version` and get `409` if it is stale (NFR-R4).
- **Lists**: `?cursor=&limit=` (default 50, max 200), with a `{ "items": [...], "next_cursor": "..." | null }` response.
- **Errors**: RFC 9457 `application/problem+json` with a stable `type` URI (`https://splits.dev/problems/<slug>`), `title`, `status`, `detail`, plus `errors[]` for field validation. The app maps `type` → localized message.
- **Warnings** (duplicate Expense, overpayment): `422` with `type …/confirmation-required` and a `warnings[]` list. The client resends with `"acknowledge_warnings": true`.

## Endpoint catalogue

### Health
| Method | Path | Purpose | M |
|---|---|---|---|
| GET | `/healthz`, `/readyz` | Liveness / database readiness (no `/v1`, no auth) | M0 |

### Auth (FR-A*)
| Method | Path | Purpose | M |
|---|---|---|---|
| POST | `/v1/auth/signup` | Email + password → unverified User; sends a code (dev: `123456`) | M1 |
| POST | `/v1/auth/verify-email` | Code → verified; triggers automatic Claims (ADR-0017); returns tokens | M1 |
| POST | `/v1/auth/verify-email/resend` | New code | M1 |
| POST | `/v1/auth/signin` | Email + password → tokens (or "verification required") | M1 |
| POST | `/v1/auth/refresh` | Refresh token → new token pair (rotation) | M1 |
| POST | `/v1/auth/signout` | Revoke the current session | M1 |
| POST | `/v1/auth/password-reset` | Request a code (always 202) | M1 |
| POST | `/v1/auth/password-reset/confirm` | Code + new password; revokes all sessions | M1 |
| POST | `/v1/me/password` | Change password (current password required) | M1 |
| GET | `/v1/me` | Profile | M1 |
| DELETE | `/v1/me` | Delete account (password required) (ADR-0013) | M1 |
| PUT | `/v1/me/devices/{fcmToken}` | Register a push token | M2 |

### Groups and Members (FR-G*, FR-M*)
| Method | Path | Purpose | M |
|---|---|---|---|
| GET / POST | `/v1/groups` | List my Groups / create | M1 |
| GET / PATCH | `/v1/groups/{id}` | Detail / rename | M1 |
| POST | `/v1/groups/{id}/members` | Add by email or as a name-only Placeholder | M1 |
| PATCH | `/v1/groups/{id}/members/{memberId}` | Change role (grant Admin) | M1 |
| DELETE | `/v1/groups/{id}/members/{memberId}` | Remove (→ Former Member) | M3 |
| POST | `/v1/groups/{id}/leave` | Leave (→ Former Member) | M3 |
| POST | `/v1/groups/{id}/subgroups` | Create a Sub-Group from the parent's Members | M3 |
| POST | `/v1/groups/{id}/keep-open` · `/reopen` | Lifecycle actions | M3 |
| GET / POST | `/v1/groups/{id}/invite-links` | List / create (optionally aimed at a Placeholder) | M3 |
| DELETE | `/v1/groups/{id}/invite-links/{linkId}` | Revoke | M3 |
| POST | `/v1/invites/{token}/join` | Join via link | M3 |
| POST | `/v1/groups/{id}/claims` | "I'm already in this group as…" | M3 |
| POST | `/v1/groups/{id}/claims/{claimId}/approve` · `/reject` | Admin decision | M3 |

### Expenses (FR-E*, FR-P*)
| Method | Path | Purpose | M |
|---|---|---|---|
| POST | `/v1/groups/{id}/expenses/preview` | Split preview: exact Shares, conversion, no save (ADR-0006) | M1 |
| GET / POST | `/v1/groups/{id}/expenses` | List (filters: `member`, `category`, `from`, `to`, `state`) / create | M1 |
| GET / PATCH | `/v1/expenses/{id}` | Detail (Shares, approvals, Disputes) / edit (creator) | M1 |
| POST | `/v1/expenses/{id}/withdraw` | Withdraw (needs approval if Accepted) | M1 |
| POST | `/v1/expenses/{id}/approve` | Approve (`on_behalf_of` for Admins acting for Placeholders) | M2 |
| POST | `/v1/expenses/{id}/disputes` | Raise a Dispute | M2 |
| POST | `/v1/disputes/{id}/reply` | Creator's single reply | M2 |
| POST | `/v1/expenses/{id}/change-requests` | Propose an edit | M2 |
| POST | `/v1/change-requests/{id}/accept` · `/reject` | Creator's decision | M2 |

### Settlements and Balances (FR-S*, FR-B*, FR-R*)
| Method | Path | Purpose | M |
|---|---|---|---|
| GET | `/v1/groups/{id}/balances` | Balances + Settle-up Suggestions | M1 |
| GET / POST | `/v1/groups/{id}/settlements` | List / record | M1 |
| GET | `/v1/settlements/{id}` | Detail | M1 |
| POST | `/v1/settlements/{id}/withdraw` | Withdraw | M1 |
| POST | `/v1/settlements/{id}/confirm` | Confirm (receiver or an Admin for a Placeholder) | M2 |
| POST | `/v1/settlements/{id}/disputes` | Dispute | M2 |
| GET | `/v1/groups/{id}/reports/categories?basis=share\|paid` | Chart data (FR-R1) | M3 |

### Activity and Notifications (FR-H*, FR-N*)
| Method | Path | Purpose | M |
|---|---|---|---|
| GET | `/v1/groups/{id}/activity` | Group Activity History | M1 |
| GET | `/v1/me/activity` | My activity across Groups | M1 |
| GET | `/v1/me/notifications` | List (`?unread=true`) | M2 |
| POST | `/v1/me/notifications/{id}/read` | Mark read | M2 |
| PUT / DELETE | `/v1/groups/{id}/mute` | Mute / unmute informational Notifications | M2 |

## Authorization rules (enforced in services, not handlers)

- Only Members of a Group (active, not Former) can see it. Sub-Groups are visible only to their own Members.
- Editing an Expense or replying to a Dispute: creator only. Approving: Approvers only. Confirming: the receiving Member, or an Admin for a Placeholder.
- Rename, remove Member, grant Admin, revoke link, approve a Claim: Admins only.
- Closed Groups reject every write except `reopen`.
- An unauthorized request for a resource that exists returns `404`, never `403`, so IDs of other Groups aren't revealed.
