// swift-tools-version: 6.2
// Infrastructure: the API client (and, later, push notifications). Talks to
// the outside world; knows nothing about Domain (ADR-0015).
import PackageDescription

let package = Package(
  name: "Infrastructure",
  platforms: [.iOS(.v26), .macOS(.v26)],
  products: [
    .library(name: "APIClient", targets: ["APIClient"])
  ],
  targets: [
    .target(name: "APIClient"),
    .testTarget(name: "APIClientTests", dependencies: ["APIClient"]),
  ],
  swiftLanguageModes: [.v6]
)
