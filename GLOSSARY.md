# Splits

An app for groups of people to record shared expenses, see who owes whom, and record when debts are paid back.

## People

**User**:
A person who has signed up and can sign in with an email and password.
_Avoid_: Account, customer, login

**Member**:
A person's place in one Group. Expenses and Balances refer to Members, never directly to Users.
_Avoid_: Participant, friend, user (when meaning someone inside a group)

**Placeholder Member**:
A Member who is not linked to any User yet, added by name or by email address so the group can record expenses before that person signs up. Admins act on its behalf until a User Claims it.
_Avoid_: Ghost, guest, offline user

**Former Member**:
A Member who has left or been removed from a Group after their Balance reached zero. They remain in the Group's records but have no access.
_Avoid_: Ex-member, removed user, inactive member

**Claim**:
Linking a User to an existing Placeholder Member, so everything recorded against that Placeholder becomes theirs. It happens automatically when a User verifies the email address the Placeholder was added with.
_Avoid_: Merge, takeover, link account

**Admin**:
A Member with extra powers in one Group: renaming it, removing Members, granting Admin, and acting for Placeholder Members. A Group's creator is its first Admin.
_Avoid_: Owner, moderator, organiser

**Invite Link**:
A shareable, expiring, revocable link that lets whoever opens it join a Group as a Member. It only opens the app once the project has its own domain.
_Avoid_: Invitation code, join link

## Groups

**Group**:
A named set of Members who share expenses. Every Expense belongs to exactly one Group.
_Avoid_: Circle, team, trip (a trip is one use of a Group)

**Parent Group**:
A Group that contains Sub-Groups.

**Sub-Group**:
A Group created inside a Parent Group, whose Members all come from the Parent Group. It is one level deep, visible only to its own Members, and its Balances are separate from the parent's.
_Avoid_: Child group, nested group, subgroup

**Active**:
The normal state of a Group, in which Expenses and Settlements can be added.

**Closing**:
The state a Group enters automatically when every Balance is zero and nothing is Pending or Disputed. It becomes Closed after a waiting period unless a Member keeps it open.
_Avoid_: Closure mode, settling

**Closed**:
The read-only state of a Group whose closing period has ended. Any Member can reopen it.
_Avoid_: Archived, deleted, finished

## Money

**Group Currency**:
The single currency in which a Group's Balances and Settlements are expressed.
_Avoid_: Default currency, base currency, settlement currency

**Original Amount**:
An Expense's amount in the currency it was actually paid in.

**Exchange Rate**:
The rate, entered by a Member, used to convert an Expense's Original Amount into the Group Currency.
_Avoid_: FX rate, conversion rate

**Category**:
A label from a fixed list (for example, food or travel) describing what an Expense was for.
_Avoid_: Type, tag

**Expense**:
An amount of money one Member paid on behalf of some Members of a Group.
_Avoid_: Bill, transaction, cost

**Split**:
How an Expense is divided among Members: equally, by exact amounts, by percentages, or by ratio (for example 2:1:1).
_Avoid_: Allocation, division

**Share**:
The portion of an Expense that one Member owes.
_Avoid_: Part, cut

**Balance**:
A Member's net position in a Group, computed only from accepted Expenses and confirmed Settlements. Positive means the Member is owed money.
_Avoid_: Debt, total, owing

**Settlement**:
A record that one Member paid another Member back, outside the app.
_Avoid_: Payment, repayment, transfer

**Settle-up Suggestion**:
A server-proposed payment from one Member to another that, together with the other suggestions, brings every Balance to zero in as few payments as possible.
_Avoid_: Simplified debt, optimisation

## Agreement

**Pending**:
The state of an Expense or Settlement that is not yet accepted, and so does not count toward Balances.
_Avoid_: Draft, unconfirmed

**Approver**:
A Member whose Approval a Pending Expense needs: its payer and every Member in its Split, except its creator and any Placeholder Members.

**Approval**:
An Approver's agreement that a Pending Expense is correct.
_Avoid_: Agreement, acceptance, sign-off

**Confirmation**:
The receiving Member's agreement that a Pending Settlement's money arrived.
_Avoid_: Approval (when meaning a Settlement)

**Dispute**:
A Member's claim, with a reason, that an Expense or Settlement is wrong. It keeps the item out of Balances until resolved.
_Avoid_: Objection, rejection, complaint

**Change Request**:
A proposed edit to an Expense that its creator can accept or reject. Accepting it edits the Expense, which makes it Pending again.
_Avoid_: Suggestion, amendment

**Withdrawal Pending**:
The state of an accepted Expense or confirmed Settlement whose withdrawal awaits agreement. It still counts toward Balances until the withdrawal is agreed.
_Avoid_: Pending deletion

**Withdrawn**:
The state of an Expense or Settlement its creator has cancelled. It stays in the Activity History but never counts toward Balances.
_Avoid_: Deleted, cancelled, voided

**Auto-acceptance**:
The rule that an undisputed Pending Expense or Settlement becomes accepted after a fixed period.
_Avoid_: Timeout, default approval

## History and notifications

**Activity History**:
The permanent, append-only record of every event in a Group, visible only to that Group's Members.
_Avoid_: Log, audit trail, timeline, feed

**Notification**:
An in-app record that something needs, or may interest, a User. It may also be delivered as a push.
_Avoid_: Alert, message, push (push is only the delivery)
