---
status: superseded by ADR-0007
---

# Money is integer minor units in one currency per Group

Every amount is stored and sent over the API as an integer count of the currency's minor unit (for example, cents) together with an ISO 4217 code. Floating-point is never used for money on either client or server. Each Group has one Group Currency, and there is no conversion between currencies in v1. Keeping the currency code on each amount means per-Expense currencies can be added later without a schema rewrite.
