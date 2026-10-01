# Implementing #6: iOS app shell (/implement)

Follows [11](11-implement-server-skeleton.md) (on branch `feat/server-skeleton`, PR #31). Branch `feat/ios-shell`, from `master`.

| UTC | Who | What happened | Decided | Details |
|---|---|---|---|---|
| 2026-10-01 12:09:00Z | User | Ran `/implement #6` | — | — |
| 2026-10-01 12:10:00Z | Claude | Read #6, #7, ADR-0005/0012/0015, doc 05 iOS section, Q64–Q66, Q69, Q75 | Example ViewModel is a server-status check (`ServerHealthRepository` in Domain); seam is the ViewModel with a fake repository, as the ticket names | #7 later swaps the hand-written `HTTPClient` for the generated client. |
| 2026-10-01 12:11:00Z | Claude | XcodeGen is not installed and not on the allowed list | Hand-written `Splits.xcodeproj` (objectVersion 77, synchronized `SplitsApp` folder, local package references, shared `Splits` scheme running every package's tests) | No new tool or dependency. |
| 2026-10-01 12:15:00Z | Claude | Built packages test-first | Packages and products: Domain, Data, Infrastructure (`APIClient`), Presentation (`Features`) | Tests: `ServerStatusViewModelTests` (fake actor repository), `URLSessionFactoryTests` (`urlCache == nil`, ADR-0005), `APIConfigurationTests` (Release rejects http, Q75), `StringCatalogTests` (fails on a missing key; probed by misspelling one). |
| 2026-10-01 12:20:00Z | Claude | SwiftPM doesn't extract strings from package code | Catalog entries are added by hand; `StringCatalogTests` guards them | Documented in `ios/README.md`. |
| 2026-10-01 12:24:00Z | Claude | Ran the app on an iOS 27.0 Simulator | — | "Server reachable" with the Docker server up, "Server unreachable" with it stopped. Release Info.plist has the https placeholder and no ATS exception. Checked the `xcode-27` runner image: Xcode 27.0 at `/Applications/Xcode_27.0.app` (27A266a), iOS 27.0 iPhone Simulators installed. |
| 2026-10-01 12:26:00Z | Claude | Ran `/code-review` against `master` (Standards and Spec agents in parallel) | Fixed: 44 pt bordered retry button (NFR-A2); app name from `InfoPlist.xcstrings`; `ServerHealthError` moved to Domain; dropped `SWIFT_APPROACHABLE_CONCURRENCY` so app and packages match; test helper dedup; lint paths skip `.build`. Kept: the real `/healthz` call as a stand-in for #7; no Data test (code replaced in #7); `.unknown` status; string keys typed as `String` so the catalog test can read them | Spec: all criteria met; CI unproven until the first PR run. Standards: the missing log (this file) and the tap target were the firm findings. |
| 2026-10-01 12:18:41Z | Claude | Committed the review fixes; pushing and opening the PR | — | — |
| 2026-10-01 12:22:49Z | Claude | Opened [PR #32](https://github.com/bikkysamuel/splitsDemo/pull/32); first `ios` run on `xcode-27` green (3m13s) | Merge waits for the user | gitleaks doesn't run on #32 yet because `secrets.yml` arrives with #31. |
