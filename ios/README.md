# Splits iOS app

SwiftUI, iOS 26+, Swift 6 language mode with strict concurrency. Design: `docs/05-architecture.md`.

## Layout (ADR-0015)

```
SplitsApp/                composition root: the only module that imports every layer
Config/                   Info.plist per configuration (base URL, ATS)
Packages/
  Domain/                 entities, repository protocols, use cases. Pure Swift
  Data/                   repository implementations and mappers (→ Domain, Infrastructure)
  Infrastructure/         APIClient (URLSession with no cache, ADR-0005); PushNotifications later
  Presentation/           Features (SwiftUI views + @Observable ViewModels; → Domain only)
```

`Presentation → Domain ← Data → Infrastructure`. Each layer is its own Swift package, so the compiler enforces the rule.

## Run it

Start the server first (`docker compose up` from the repository root), then open `Splits.xcodeproj`, pick the `Splits` scheme and an iOS Simulator, and run. Debug talks to `http://localhost:8080` (local networking allowed). Release requires HTTPS; its base URL is a placeholder until a host exists (Q75).

## Test and lint

```sh
xcodebuild test -project Splits.xcodeproj -scheme Splits -destination 'platform=iOS Simulator,name=iPhone 17,OS=27.0'
xcrun swift-format lint --strict --recursive SplitsApp Packages/*/Package.swift Packages/*/Sources Packages/*/Tests
```

Each package also runs on the Mac with `swift test` from its directory, which is faster while iterating.

ViewModels are tested with Swift Testing and a fake repository (see `FeaturesTests`).

## Strings

Every user-facing string lives in a String Catalog (`Localizable.xcstrings`) in the package that shows it. SwiftPM does not extract strings, so add each new key and its comment to the catalog by hand. `StringCatalogTests` fails when a key is missing.
