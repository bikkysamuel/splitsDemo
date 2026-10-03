# Implementing #19: foreign-currency Expense with a hand-entered Exchange Rate (unattended M1 run)

Follows [2026-10-02/07](../2026-10-02/07-implement-split-methods.md). Branch `feat/foreign-currency`, built on `feat/split-methods` (#18, PR #47).

| UTC | Who | What happened | Decided | Prompt |
|---|---|---|---|---|
| 2026-10-03 10:17:57Z | Claude | Started #19 on `feat/foreign-currency` from `feat/split-methods`. Read the ticket, ADR-0006/0007/0010/0014, Q87, Q93, Q96 and the #17/#18 code. The schema already has `original_minor`, `original_currency` and `exchange_rate NUMERIC`, and `ledger.Convert` already rounds half-up once, so no migration and no new money arithmetic | Exact entries stay in Group Currency minor units (Q87); see Q97 | — |
| 2026-10-03 10:22:36Z | Claude | Server, test-first: HTTP-seam tests (`foreign_currency_test.go`) for half-up conversion and Shares on the converted amount, exact rate round-trip, rate checks and exact Splits of a foreign Expense, all red; `ledger.ErrConvertedTooLarge` (test first); contract: `exchange_rate` on `ExpenseInput`, `original_amount` and `exchange_rate` on `Expense` and `ExpensePreview`, new field codes; regenerated Go and Swift; service, store and adapter. All green on real Postgres | Q97 | — |
| 2026-10-03 10:27:53Z | Claude | iOS, test-first: Domain `ExchangeRate` (reads a typed rate into the server's form, displays a stored one), `Expense.originalAmount`/`exchangeRate`, `ExpenseInput.exchangeRate`, `ExpensePreview` fields and the new field reasons; Data mapper and fixtures; Add Expense gets a Currency picker, an Exchange Rate field ("INR per 1 USD") and the converted amount from the preview; the detail shows the Original Amount and the rate. Strings in the catalog, VoiceOver hints | Q97 | — |
