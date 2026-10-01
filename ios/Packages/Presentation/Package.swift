// swift-tools-version: 6.2
// Presentation: SwiftUI views and @Observable ViewModels. Depends on Domain
// only, never on Data, Infrastructure or generated API types (ADR-0015).
import PackageDescription

let package = Package(
  name: "Presentation",
  defaultLocalization: "en",
  platforms: [.iOS(.v26), .macOS(.v26)],
  products: [
    .library(name: "Features", targets: ["Features"])
  ],
  dependencies: [
    .package(path: "../Domain")
  ],
  targets: [
    .target(
      name: "Features",
      dependencies: ["Domain"],
      resources: [.process("Resources")]
    ),
    .testTarget(name: "FeaturesTests", dependencies: ["Features", "Domain"]),
  ],
  swiftLanguageModes: [.v6]
)
