---
status: superseded by ADR-0005
---

# Writes require a connection; SwiftData is a read cache only

The server (Go + Postgres) is the single source of truth. The iOS app sends every write straight to the API and fails visibly when offline. SwiftData only caches server state so the app can display it quickly and show it read-only while offline. We chose this over offline-first writes because offline writes bring conflicting edits, temporary IDs, and Balances that are wrong until sync, roughly doubling the hard problems in v1.

## Consequences

- Every write request carries a client-generated UUID (idempotency key), so retries are safe and offline writes can be added later without breaking the API.
- SwiftData models are disposable projections of server data. They are never the authority for a Balance.
