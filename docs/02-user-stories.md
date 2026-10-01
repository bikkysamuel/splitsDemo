# 02 · User stories

Status: Draft · Last updated: 2026-10-01 · Depends on: doc 01, GLOSSARY

The format is "As a … I want … so that …". The detailed rules behind each story are in [03-functional-requirements.md](03-functional-requirements.md) (FR ids in brackets). **M** = milestone.

## Accounts

| ID | Story | M | FR |
|---|---|---|---|
| US-01 | As a visitor, I want to sign up with my email and a password so that I can use Splits. | M1 | FR-A1–A3 |
| US-02 | As a new User, I want to verify my email with a 6-digit code so that my account is trusted and any Placeholder carrying my email becomes mine. | M1 | FR-A2, FR-M5 |
| US-03 | As a User, I want to sign in and stay signed in across launches so that I don't type my password every time. | M1 | FR-A4 |
| US-04 | As a User, I want to sign out, which ends my session on this device. | M1 | FR-A4 |
| US-05 | As a User who forgot my password, I want to reset it with a code so that I can get back in. | M1 | FR-A5 |
| US-06 | As a signed-in User, I want to change my password. | M1 | FR-A5 |
| US-07 | As a User, I want to delete my account in the app, and Groups I'm in keep their records correct. | M1 | FR-A6 |

## Groups and Members

| ID | Story | M | FR |
|---|---|---|---|
| US-10 | As a User, I want to create a Group with a name and Group Currency so that we can share costs. | M1 | FR-G1 |
| US-11 | As a Member, I want to add someone by email so that they join if they have an account, or become a Placeholder if not. | M1 | FR-M1–M3 |
| US-12 | As a Member, I want to add a Placeholder by name only so that I can record Expenses for someone without an email. | M1 | FR-M2 |
| US-13 | As an Admin, I want to make another Member an Admin. | M1 | FR-G3 |
| US-14 | As an Admin, I want to rename the Group and remove a settled Member. | M1/M3 | FR-G3, FR-M7 |
| US-15 | As a Member with a zero Balance, I want to leave a Group. | M3 | FR-M7 |
| US-16 | As a Member, I want to create a Sub-Group of some of the Group's Members for a trip. | M3 | FR-G5 |
| US-17 | As a Member, I want to share an Invite Link so that others can join. | M3 | FR-M6 |
| US-18 | As a new Member, I want to say "I'm already in this group as Dad" so that past Expenses become mine once an Admin confirms. | M3 | FR-M5 |

## Expenses

| ID | Story | M | FR |
|---|---|---|---|
| US-20 | As a Member, I want to record an Expense (amount, payer, Category, note, date) split equally, by exact amounts, by percentage or by ratio. | M1 | FR-E1–E3 |
| US-21 | As a Member, I want to see the exact Shares, including who gets the leftover minor unit, before saving. | M1 | FR-E4 |
| US-22 | As a Member, I want to record an Expense in another currency with an Exchange Rate. | M1 | FR-E5 |
| US-23 | As an Expense's creator, I want to edit or withdraw it. | M1 | FR-E6 |
| US-24 | As a Member, I want to filter Expenses by Member, Category and date. | M1 | FR-E8 |
| US-25 | As a Member, I want to be warned when an Expense looks like a duplicate. | M3 | FR-E7 |

## Agreement

| ID | Story | M | FR |
|---|---|---|---|
| US-30 | As an Approver, I want to approve a Pending Expense so that it counts. | M2 | FR-P1–P2 |
| US-31 | As an Approver, I want to Dispute an Expense with a reason, and see the creator's reply. | M2 | FR-P3 |
| US-32 | As an Approver, I want to propose a Change Request instead of disputing. | M2 | FR-P4 |
| US-33 | As a Member, I want undisputed items to be accepted automatically after 7 days so that one silent Member can't freeze the Group. | M2 | FR-P5 |
| US-34 | As an Admin, I want to approve or confirm on behalf of a Placeholder. | M2 | FR-P6 |

## Balances and Settlements

| ID | Story | M | FR |
|---|---|---|---|
| US-40 | As a Member, I want to see every Member's Balance in the Group Currency. | M1 | FR-B1 |
| US-41 | As a Member, I want Settle-up Suggestions showing who pays whom, in at most Members − 1 payments, so every Balance reaches zero. | M1 | FR-B2 |
| US-42 | As a Member, I want to record a Settlement (full or partial). | M1 | FR-S1–S2 |
| US-43 | As the receiving Member, I want to confirm or Dispute a Settlement. | M2 | FR-S3 |
| US-44 | As a Member, I want a balance report screen with a Settle button next to each suggestion. | M3 | FR-R2 |
| US-45 | As a Member, I want a bar chart of each Member's spending by Category, switchable between Share and Paid. | M3 | FR-R1 |

## History, notifications, lifecycle

| ID | Story | M | FR |
|---|---|---|---|
| US-50 | As a Member, I want the Group's Activity History, including what changed in each edit. | M1 | FR-H1 |
| US-51 | As a User, I want "My activity" across all my Groups. | M1 | FR-H2 |
| US-52 | As a User, I want an in-app Notification list of things needing my attention. | M2 | FR-N1 |
| US-53 | As a User, I want to Approve or Confirm from a notification (with Face ID), and Dispute opens the app. | M2 | FR-N3 |
| US-54 | As a User, I want tapping a notification to open the relevant screen. | M2 | FR-N4 |
| US-55 | As a User, I want to mute informational notifications for a Group. | M2 | FR-N2 |
| US-56 | As a Member, I want a settled Group to close automatically after everyone is notified, with a chance to keep it open. | M3 | FR-G6 |
| US-57 | As a Member, I want to reopen a Closed Group. | M3 | FR-G6 |

## App shell, Home and Settings

| ID | Story | M | FR |
|---|---|---|---|
| US-60 | As a User, I want the app to check my saved session with the server at launch, so that I land on Home when signed in and on sign-in/sign-up otherwise. | M1 | FR-U1 |
| US-61 | As a User, I want a clear "session expired" alert, then sign-out, when my session can't be renewed. | M1 | FR-U2 |
| US-62 | As a User, I want a clear screen when there's no internet or the server is unavailable, which retries by itself and keeps what I was typing. | M1 | FR-U3 |
| US-63 | As a developer, I want Debug builds to show the HTTP status, problem type and detail, request ID and network error code. | M1 | FR-U3 |
| US-64 | As a User, I want Home to show at a glance whether I'm owed, owe or am all settled, per currency, with my Groups below. | M1 | FR-B3, FR-U4 |
| US-65 | As a User, I want a Report tab where I pick a Group, see a summary, and open a detail screen with more data. | M1/M3 | FR-R3 |
| US-66 | As a User, I want to choose System, Light or Dark theme. | M1 | FR-U5 |
| US-67 | As a User, I want my default currency pre-filled when I create a Group. | M1 | FR-U5 |
| US-68 | As a User, I want to choose which kinds of push notification I get, without losing anything from the in-app list. | M2 | FR-N5 |
| US-69 | As a User, I want to turn reminder pushes for pending actions on or off. | M2 | FR-N6 |
| US-70 | As a User, I want Settings to hold change password, sign out, delete account and the app version. | M1 | FR-U5 |
