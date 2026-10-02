# 06 · Data model

Status: Draft · Last updated: 2026-10-01 · Depends on: ADR-0002, 0007, 0009, 0010, 0011, 0013, 0014, 0016, 0017; doc 03

The logical Postgres 18 schema. Exact DDL lives in `server/migrations/` (goose, forward-only, ADR-0014). Money is always `bigint` minor units plus a `char(3)` ISO 4217 code, never `float` or `numeric` amounts (ADR-0002). IDs are server-generated UUIDv7. Every mutable table has `created_at`, `updated_at` and a `version int` for optimistic concurrency.

## Entity overview

```
users ─1───*─ sessions
  │  └─1───*─ one_time_codes
  │  └─1───*─ devices
  │  └─1───*─ notifications ─*───1─ groups
  │
  └─0..1───*─ members ─*───1─ groups ─0..1─ parent group (one level)
                │                 │
                │                 ├─*─ invite_links
                │                 ├─*─ expenses ─1───*─ expense_shares ─*───1─ members
                │                 │        ├─*─ expense_approvals
                │                 │        ├─*─ change_requests
                │                 │        └─*─ disputes (subject = expense | settlement)
                │                 ├─*─ settlements (from_member, to_member)
                │                 └─*─ activity_events (append-only)
```

## Tables

### Identity

| Table | Key columns | Notes |
|---|---|---|
| `users` | `id`, `email citext UNIQUE`, `password_hash`, `email_verified_at null` | Erased on account deletion (ADR-0013). |
| `sessions` | `id`, `family_id`, `user_id`, `access_hash`, `access_expires_at`, `refresh_hash`, `refresh_expires_at`, `replaced_by null`, `revoked_at null` | Only SHA-256 hashes of tokens are stored (ADR-0011). Each refresh adds a row to the Session's family (`family_id` = its first row); reuse of a replaced refresh token revokes the family. |
| `one_time_codes` | `id`, `user_id`, `purpose (verify_email \| reset_password)`, `code_hash`, `expires_at`, `attempts`, `consumed_at null` | 15-minute expiry, at most 5 attempts (ADR-0016). |
| `devices` | `id`, `user_id`, `fcm_token UNIQUE`, `last_seen_at` | Used by push later (Q74). |
| `login_throttle` | `scope (account \| ip)`, `key` (SHA-256), `failures`, `last_failure_at`, `next_allowed_at` | Growing delays (Q34, Q90). |

### Groups and Members

| Table | Key columns | Notes |
|---|---|---|
| `groups` | `id`, `parent_group_id null`, `name`, `currency char(3)`, `state (active \| closing \| closed)`, `closing_since null`, `closed_at null` | Trigger: a parent must have `parent_group_id IS NULL` (one level). `currency` is immutable once an Expense exists. |
| `members` | `id`, `group_id`, `user_id null`, `display_name`, `email citext null`, `role (admin \| member)`, `status (active \| former)`, `join_seq int`, `parent_member_id null` | Placeholder ⇔ `user_id IS NULL`. `UNIQUE(group_id, lower(display_name))` (FR-M4: unique ignoring case), `UNIQUE(group_id, email)`, `UNIQUE(group_id, user_id)`. `join_seq` breaks rounding ties (ADR-0010). In a Sub-Group, `parent_member_id` points at the same person in the Parent Group. |
| `invite_links` | `id`, `group_id`, `token_hash`, `target_member_id null`, `created_by`, `expires_at`, `revoked_at null` | Built now; opens the app once a domain exists (ADR-0017). |
| `claim_requests` | `id`, `group_id`, `user_id`, `placeholder_member_id`, `status (open \| approved \| rejected)`, `decided_by null` | For the "pick a Placeholder + Admin confirms" path (FR-M5c). |

### Expenses

| Table | Key columns | Notes |
|---|---|---|
| `expenses` | `id`, `group_id`, `created_by`, `payer_id`, `category`, `note`, `spent_on date`, `original_minor bigint`, `original_currency char(3)`, `exchange_rate numeric null`, `amount_minor bigint`, `split_method (equal \| exact \| percentage \| ratio)`, `state`, `revision int`, `pending_since null` | `amount_minor` is in the Group Currency, computed by `ledger`. `exchange_rate` is NULL when the currencies match. `CHECK (original_minor > 0 AND amount_minor > 0)`. |
| `expense_shares` | `expense_id`, `member_id`, `input numeric null`, `share_minor bigint` | `input` is the exact amount / percentage / ratio weight entered. A deferred constraint trigger checks `Σ share_minor = expenses.amount_minor`. |
| `expense_approvals` | `expense_id`, `revision`, `member_id`, `acted_by_member_id`, `approved_at` | Keyed to a revision, so an edit resets approvals naturally. `acted_by ≠ member` means on behalf of a Placeholder (FR-P6). |
| `change_requests` | `id`, `expense_id`, `base_revision`, `requested_by`, `proposed jsonb`, `reason`, `status (open \| accepted \| rejected)`, `decision_reason` | |
| `disputes` | `id`, `subject_type (expense \| settlement)`, `subject_id`, `subject_revision`, `raised_by`, `reason`, `reply null`, `status (open \| resolved)` | |

### Settlements

| Table | Key columns | Notes |
|---|---|---|
| `settlements` | `id`, `group_id`, `from_member_id`, `to_member_id`, `amount_minor bigint`, `settled_on`, `note`, `created_by`, `state`, `pending_since null`, `confirmed_by null` | `CHECK (from_member_id <> to_member_id AND amount_minor > 0)`. Always in the Group Currency. |

### History, notifications, infrastructure

| Table | Key columns | Notes |
|---|---|---|
| `activity_events` | `id bigint identity`, `group_id`, `actor_member_id`, `on_behalf_of_member_id null`, `type`, `subject_type`, `subject_id`, `payload jsonb`, `occurred_at` | **Append-only**: a trigger rejects UPDATE and DELETE. `payload` holds field diffs for edits. |
| `notifications` | `id`, `user_id`, `group_id`, `kind`, `action_required bool`, `subject_type`, `subject_id`, `read_at null`, `resolved_at null` | Action-required items resolve when the underlying action happens. |
| `notification_mutes` | `user_id`, `group_id` | Mutes informational kinds only (FR-N2). |
| `notification_preferences` | `user_id`, `kind`, `push_enabled bool` | Push delivery only (FR-N5); a missing row means enabled. The reminder toggle is kind `reminder` (FR-N6). |
| `idempotency_keys` | `user_id`, `key`, `request_hash`, `status (in_progress \| completed)`, `response_status`, `response_content_type`, `response_body bytea`, `created_at` | PK `(user_id, key)`, kept 24 h (NFR-R1). The response is stored as sent, so a replay is byte-identical; 5xx responses are not kept. |

Balances are derived from accepted `expense_shares`, `expenses.payer_id` and confirmed `settlements`. Whether to also keep a `group_balances` table, updated in the same transaction, is decided when performance tests run (NFR-P2), and an ADR is written if it is added.

## State machines

**Expense** (ADR-0009, FR-E6, FR-P*)
```
            create/edit
   ┌──────────────────────────┐
   ▼                          │
Pending ──all approvals / 7d──▶ Accepted ──creator withdraws──▶ WithdrawalPending ──approved / 7d──▶ Withdrawn
   │  ▲                         ▲                                     │ (still counts until approved)
   │  └──── creator edits ──────┼──── Disputed ◀── approver disputes ─┘
   └──── approver disputes ───▶ Disputed ──creator edits──▶ Pending
                                 └──creator withdraws──▶ Withdrawn
Pending ──creator withdraws──▶ Withdrawn
```
Counts toward Balances only in **Accepted** and **WithdrawalPending**.

**Settlement**: the same shape, with Confirmation by the receiver (or an Admin for a Placeholder) in place of Approvals.

**Group** (FR-G6)
```
Active ──all Balances 0, nothing Pending/Disputed──▶ Closing ──7 days──▶ Closed ──any Member reopens──▶ Active
   ▲                                                  │
   └──────── "Keep open" or new Expense ──────────────┘
```

## Fixed Category list (proposed default)

`food_drink`, `groceries`, `transport`, `accommodation`, `rent`, `utilities`, `entertainment`, `shopping`, `health`, `travel`, `other`. Display names are localized in the app.

## Invariants (checked by fuzz tests and constraints)

1. For every Expense: `Σ shares = amount_minor`, every share ≥ 0, and shares differ by at most one minor unit from their exact value (ADR-0010).
2. For every Group: `Σ Balances = 0`.
3. Applying all Settle-up Suggestions brings every Balance to 0, using at most (Members − 1) payments.
4. Only Accepted and WithdrawalPending items affect Balances.
5. `activity_events` rows are never changed or removed.
