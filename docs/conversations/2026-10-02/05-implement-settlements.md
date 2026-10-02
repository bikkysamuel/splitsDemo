# Implementing #23: record and withdraw Settlements (unattended M1 run)

Follows [04](04-implement-balances.md). Branch `feat/settlements`, built on `feat/balances` (PR #45).

| UTC | Who | What happened | Decided | Prompt |
|---|---|---|---|---|
| 2026-10-02 07:00:00Z | Claude | Opened PR #45 for #22 and started #23 | — | — |
| 2026-10-02 04:34:59Z | Claude | Contract for list, record, get and withdraw; `ledger.Overpays`; migration for `settlements` (from ≠ to); the `settlements` service (overpayment warning, creator-only withdraw with version, events in one transaction); Balances include Settlements; HTTP-seam tests incl. partial, the warning flow, withdraw rules and 404s; iOS Record Settlement (pre-filled from a Suggestion, "Save anyway?"), Settlement list and detail with Withdraw; a `GroupViewModel.make` test factory, all test-first | See Q95 | — |
