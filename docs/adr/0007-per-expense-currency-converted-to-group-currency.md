# Each Expense has its own currency, converted once into the Group Currency

Supersedes ADR-0002's single-currency rule. Its rule that money is integer minor units plus an ISO 4217 code, never floating point, still stands. An Expense records its Original Amount, its currency, and the Exchange Rate used. The server converts it once into the Group Currency, which is the only currency Balances and Settlements are expressed in. Exchange Rates are entered by hand in v1. We rejected keeping Balances per currency because "Bob owes ₹500 and ฿200" is hard to settle and hard to explain.

## Consequences

- Exchange Rates are stored as exact decimals (never floats), and the converted amount is rounded once, at conversion.
- Changing an Exchange Rate is an edit and resets Approvals (ADR-0009).
