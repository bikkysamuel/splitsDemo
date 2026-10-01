# Expenses and Settlements count toward Balances only once accepted

A new or edited Expense is Pending until every Approver (its payer and every Member in its Split, minus its creator and any Placeholder Members) has approved it. A Settlement is Pending until the Member receiving the money confirms it. Pending items are visible but excluded from Balances. Anything not disputed is accepted automatically after 7 days, with a reminder at day 5, so one unresponsive Member cannot freeze a Group. We chose this over counting items immediately and reversing them on dispute, because a Balance that jumps back and forth erodes trust more than one that updates late.

## Consequences

- Any edit to an accepted Expense, including its Exchange Rate, makes it Pending again.
- Withdrawing an accepted Expense or confirmed Settlement needs the same Approval or Confirmation before it takes effect; the item stays in the Activity History.
- An Admin gives Approval or Confirmation on behalf of a Placeholder Member, and the Activity History records that.
- A Disputed item stays out of Balances until its creator edits or withdraws it.
- Group closure (Closing) cannot start while anything is Pending or Disputed.
