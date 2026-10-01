// swift-tools-version: 6.2
// Domain: entities, repository protocols and use cases. Pure Swift: no
// SwiftUI, networking or Firebase (ADR-0015).
import PackageDescription

let package = Package(
  name: "Domain",
  platforms: [.iOS(.v26), .macOS(.v26)],
  products: [
    .library(name: "Domain", targets: ["Domain"])
  ],
  targets: [
    .target(name: "Domain")
  ],
  swiftLanguageModes: [.v6]
)
