# 03 · Functional requirements

Status: Draft · Last updated: 2026-10-01 · Depends on: ADR-0002, 0003, 0006, 0007, 0009, 0010, 0013, 0016, 0017; decision log Q2–Q76

The rules the server enforces. Each FR is testable. "The server" computes and validates everything (ADR-0006); the app only displays results and checks that form input is well-formed.

## A · Accounts

- **FR-A1** Sign-up takes an email and password. Emails are unique, compared ignoring case and trimmed of whitespace.
- **FR-A2** Sign-up requires a 6-digit verification code. Codes expire after 15 minutes and allow at most 5 attempts. In development the code is always `123456`; in production it is random and emailed (ADR-0016). An unverified account can't use the app beyond the verification screen.
- **FR-A3** Passwords are 10–128 characters with no composition rules, and are rejected if they appear on the shipped common/breached list (Q34).
- **FR-A4** Sign-in issues a 15-minute access token and a 30-day refresh token. The refresh token is replaced on each use, and reusing an old one revokes the session. Sign-out revokes the session (ADR-0011). After 5 failed sign-ins, the wait between attempts grows, both per account and per IP. There is no hard lockout.
- **FR-A5** Password reset: request a code (the response never reveals whether the email exists) → submit code + new password → all sessions revoked. Change password requires the current password and revokes the other sessions.
- **FR-A6** Account deletion asks for the password, warns if any Balance is non-zero, then erases the User. Each of their Members becomes a Placeholder with the same display name (ADR-0013).

## G · Groups

- **FR-G1** Creating a Group takes a name and a Group Currency (ISO 4217), which can't change once any Expense exists. The creator becomes its first Admin.
- **FR-G2** Limits: at most 50 Members (including Placeholders and Former Members) per Group, and 200 Groups per User (Q30).
- **FR-G3** Admins can rename the Group, remove Members (FR-M7), grant Admin, and act for Placeholders. Every Member can add Members, Expenses and Settlements and create Sub-Groups (Q21, Q76).
- **FR-G4** Groups are never deleted.
- **FR-G5** A Sub-Group's Members must all be Members of its Parent Group, and it can't have Sub-Groups of its own. Only its Members can see it. Its Balances and Activity History are separate from the parent's (Q40).
- **FR-G6** Lifecycle (Q41): **Active → Closing** happens automatically when every Balance is zero and nothing is Pending or Disputed, and all Members are notified. **Closing → Closed** happens after 7 days. During Closing, "Keep open" or adding an Expense returns the Group to Active. **Closed** Groups are read-only, and any Member can reopen one (→ Active). A Parent Group can't enter Closing while any of its Sub-Groups is Active or Closing.

## M · Members

- **FR-M1** Adding by email: if a verified User has that email, they become a Member at once and get an in-app Notification. No email is sent (ADR-0017).
- **FR-M2** Otherwise a Placeholder is created, carrying the email if one was given. A Placeholder can also be added by name only.
- **FR-M3** The same email can't be added twice to one Group.
- **FR-M4** Display names are unique within a Group.
- **FR-M5** Claim (Q12, ADR-0017): (a) automatically, when a User verifies an email that Placeholders carry, in every Group; (b) by joining through an Invite Link aimed at a Placeholder; (c) a joining User picks a Placeholder and an Admin confirms.
- **FR-M6** Invite Links: created by any Member, expire after 7 days, revocable by Admins, optionally aimed at a Placeholder. Joining through a link adds the User directly. The link only opens the app once a domain exists, but the join API is built now.
- **FR-M7** Leaving or removal is allowed only when the Member's Balance is zero and they have nothing Pending or Disputed. They then become a Former Member: still shown in history, no access (Q22).

## E · Expenses

- **FR-E1** An Expense has: payer (any Member, including a Placeholder), Original Amount (> 0, integer minor units), currency, Exchange Rate (if the currency differs from the Group Currency), Category (fixed list), optional note (≤ 500 characters), date, and Split.
- **FR-E2** Split methods: **equal** (among selected Members), **exact** (amounts must sum to the converted total), **percentage** (must sum to exactly 100, up to 2 decimal places), **ratio** (positive integers, for example 2:1:1).
- **FR-E3** Anyone may record an Expense that someone else paid (Q20).
- **FR-E4** Shares are computed by the server with largest-remainder rounding, ties broken by join order (ADR-0010). A **split-preview** endpoint returns the exact Shares without saving. The saved Shares always sum exactly to the converted amount.
- **FR-E5** Currency conversion (ADR-0007): the Original Amount × Exchange Rate gives the Group Currency amount, rounded half-up once. Exchange Rates are entered by hand, positive, with up to 10 significant decimal places, and stored exactly.
- **FR-E6** Only the creator edits directly. Others use a Change Request (FR-P4). Every edit, including an Exchange Rate change, makes the Expense Pending again. Withdrawing is done by the creator; withdrawing an accepted Expense needs its Approvers' agreement (ADR-0009). Withdrawn Expenses remain in history.
- **FR-E7** Duplicate warning (Q42): when saving, if the same Group has an Expense with the same Category and the same converted amount within 24 hours, the server returns a warning with the earlier Expense, and the client may resubmit with an acknowledgement flag.
- **FR-E8** Filter Expenses by Member, Category and date range, with cursor pagination.

## P · Agreement (Approvals, Disputes, Change Requests)

- **FR-P1** New and edited Expenses are **Pending**. Approvers are the payer plus every Member in the Split, minus the creator and minus Placeholders (ADR-0009). If there are no Approvers, the Expense is accepted immediately.
- **FR-P2** When every Approver has approved, the Expense is accepted and counts toward Balances.
- **FR-P3** Dispute: an Approver gives one reason (≤ 500 characters). The creator may reply once, then must edit (→ Pending, approvals reset) or withdraw. A Disputed Expense never counts.
- **FR-P4** Change Request: a proposed edit plus an optional reason. If the creator accepts, the Expense is edited (→ Pending); if the creator rejects, the request is closed with an optional reason.
- **FR-P5** Auto-acceptance: Pending items with no open Dispute are accepted 7 days after entering Pending, with a reminder Notification at day 5.
- **FR-P6** Admins give Approval or Confirmation on behalf of Placeholders. The Activity History records "on behalf of".

## B · Balances

- **FR-B1** Balance = Σ(amounts paid in accepted Expenses) − Σ(Shares owed in accepted Expenses) + Σ(confirmed Settlements paid) − Σ(confirmed Settlements received), in the Group Currency. The Balances in a Group always sum to zero.
- **FR-B2** Settle-up Suggestions: a list of payments that brings every Balance to zero, minimising the number of payments (a greedy approach is acceptable; at most Members − 1 payments).

## S · Settlements

- **FR-S1** A Settlement records from-Member, to-Member, amount (> 0) in the Group Currency, date and optional note. Any Member may record one.
- **FR-S2** Any positive amount is allowed. Overpayment produces a warning, not an error (Q24).
- **FR-S3** A Settlement is Pending until the receiving Member confirms it (an Admin confirms for a Placeholder). The receiver may Dispute it instead. Auto-acceptance after 7 days (FR-P5). Withdrawal follows FR-E6.

## H · Activity History

- **FR-H1** Every event in a Group is appended to its Activity History. Events include: Expense created, edited (with field-level diff), approved, disputed, replied, change requested, accepted, rejected, withdrawn, auto-accepted; Settlement recorded, confirmed, disputed, withdrawn; Member added, Claimed, left, removed, made Admin; Group renamed, Closing, kept open, Closed, reopened; Sub-Group created. History entries can never be edited or deleted.
- **FR-H2** "My activity": the events across all Groups the User belongs to, filtered by Group visibility (Sub-Group events only for its Members).

## N · Notifications

- **FR-N1** An in-app Notification list holds action-required items (approve, confirm, respond to a Dispute or Change Request) and informational items. Users can mark them read. Action-required items can't be muted (Q51).
- **FR-N2** Informational Notifications can be muted per Group.
- **FR-N3** Push actions: Approve and Confirm require device authentication; Dispute opens the app (ADR-0008). Push delivery is blocked until the paid Apple Developer Program is bought (Q74). It is tested with `xcrun simctl push`.
- **FR-N4** Every Notification carries a deep link to its screen (Expense, Settlement, Group, Balance report).

## R · Reports

- **FR-R1** Group chart: per-Member totals by Category over accepted Expenses, switchable between Share and Paid, in the Group Currency (Q43).
- **FR-R2** Balance report: each Member's Balance plus the Settle-up Suggestions, with a Settle action (Q44).
