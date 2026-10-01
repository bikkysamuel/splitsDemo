# Implementing #6: iOS app shell (/implement)

Follows [11](11-implement-server-skeleton.md) (on branch `feat/server-skeleton`, PR #31). Branch `feat/ios-shell`, from `master`.

| UTC | Who | What happened | Decided | Prompt |
|---|---|---|---|---|
| 2026-10-01 12:04:45Z | User | Ran `/implement #6` | — | `/implement #6` |
| 2026-10-01 12:10:00Z | Claude | Read #6, #7, ADR-0005/0012/0015, doc 05 iOS section, Q64–Q66, Q69, Q75 | Example ViewModel is a server-status check (`ServerHealthRepository` in Domain); seam is the ViewModel with a fake repository, as the ticket names | — |
| 2026-10-01 12:11:00Z | Claude | XcodeGen is not installed and not on the allowed list | Hand-written `Splits.xcodeproj` (objectVersion 77, synchronized `SplitsApp` folder, local package references, shared `Splits` scheme running every package's tests) | — |
| 2026-10-01 12:15:00Z | Claude | Built packages test-first | Packages and products: Domain, Data, Infrastructure (`APIClient`), Presentation (`Features`) | — |
| 2026-10-01 12:20:00Z | Claude | SwiftPM doesn't extract strings from package code | Catalog entries are added by hand; `StringCatalogTests` guards them | — |
| 2026-10-01 12:24:00Z | Claude | Ran the app on an iOS 27.0 Simulator | — | — |
| 2026-10-01 12:26:00Z | Claude | Ran `/code-review` against `master` (Standards and Spec agents in parallel) | Fixed: 44 pt bordered retry button (NFR-A2); app name from `InfoPlist.xcstrings`; `ServerHealthError` moved to Domain; dropped `SWIFT_APPROACHABLE_CONCURRENCY` so app and packages match; test helper dedup; lint paths skip `.build`. Kept: the real `/healthz` call as a stand-in for #7; no Data test (code replaced in #7); `.unknown` status; string keys typed as `String` so the catalog test can read them | — |
| 2026-10-01 12:18:41Z | Claude | Committed the review fixes; pushing and opening the PR | — | — |
| 2026-10-01 12:22:49Z | Claude | Opened [PR #32](https://github.com/bikkysamuel/splitsDemo/pull/32); first `ios` run on `xcode-27` green (3m13s) | Merge waits for the user | — |
| 2026-10-01 12:23:38Z | User | Asked to merge #31 and #32 | Squash-merge both | merge #31 and #32 |
