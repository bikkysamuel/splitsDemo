# The iOS app stores no domain data on the device

The Go server is the single source of truth, and the app holds Groups, Expenses, Balances and Settlements in memory only. It never writes them to SwiftData, files, `UserDefaults` or the HTTP cache. When offline, the app shows a no-connection state rather than possibly stale data. This supersedes ADR-0001, which allowed a SwiftData read cache. We dropped the cache because a stale Balance shown as current is worse than no Balance, and keeping no financial data at rest shrinks what a lost device can expose.

## Consequences

- SwiftData is not used in v1, despite being in the original technology list.
- `URLSession` uses a configuration with no disk cache.
- Allowed on device: the auth token in the Keychain, the Firebase device token, and non-financial UI preferences. Nothing else.
- Writes still carry a client-generated idempotency key so retries are safe.
