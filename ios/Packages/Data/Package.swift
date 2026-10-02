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
    .package(url: "https://github.com/apple/swift-openapi-runtime", exact: "1.12.2"),
    .package(url: "https://github.com/apple/swift-http-types", exact: "1.8.0"),
  ],
  targets: [
    .target(
      name: "Data",
      dependencies: [
        "Domain",
        .product(name: "APIClient", package: "Infrastructure"),
        .product(name: "OpenAPIRuntime", package: "swift-openapi-runtime"),
      ]
    ),
    .testTarget(
      name: "DataTests",
      dependencies: [
        "Data",
        "Domain",
        .product(name: "APIClient", package: "Infrastructure"),
        .product(name: "OpenAPIRuntime", package: "swift-openapi-runtime"),
        .product(name: "HTTPTypes", package: "swift-http-types"),
      ],
      resources: [.copy("Fixtures")]
    ),
  ],
  swiftLanguageModes: [.v6]
)
