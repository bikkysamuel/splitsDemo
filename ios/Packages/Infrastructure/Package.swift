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
  dependencies: [
    .package(url: "https://github.com/apple/swift-openapi-runtime", exact: "1.12.2"),
    .package(url: "https://github.com/apple/swift-openapi-urlsession", exact: "1.3.2"),
    .package(url: "https://github.com/apple/swift-http-types", exact: "1.8.0"),
  ],
  targets: [
    .target(
      name: "APIClient",
      dependencies: [
        .product(name: "OpenAPIRuntime", package: "swift-openapi-runtime"),
        .product(name: "OpenAPIURLSession", package: "swift-openapi-urlsession"),
      ]
    ),
    .testTarget(
      name: "APIClientTests",
      dependencies: [
        "APIClient",
        .product(name: "OpenAPIRuntime", package: "swift-openapi-runtime"),
        .product(name: "HTTPTypes", package: "swift-http-types"),
      ]
    ),
  ],
  swiftLanguageModes: [.v6]
)
