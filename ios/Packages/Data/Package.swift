// swift-tools-version: 6.2
// Data: repository implementations and mappers. Implements Domain's
// protocols on top of Infrastructure; API types never leave this layer
// (ADR-0015).
import PackageDescription

let package = Package(
  name: "Data",
  platforms: [.iOS(.v26), .macOS(.v26)],
  products: [
    .library(name: "Data", targets: ["Data"])
  ],
  dependencies: [
    .package(path: "../Domain"),
    .package(path: "../Infrastructure"),
  ],
  targets: [
    .target(
      name: "Data",
      dependencies: [
        "Domain",
        .product(name: "APIClient", package: "Infrastructure"),
      ]
    )
  ],
  swiftLanguageModes: [.v6]
)
