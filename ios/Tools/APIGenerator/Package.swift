// swift-tools-version: 6.2
// Pins swift-openapi-generator for scripts/generate-api.sh. Not part of the
// app: the app links only the OpenAPI runtime and URLSession transport.
import PackageDescription

let package = Package(
  name: "APIGenerator",
  platforms: [.macOS(.v15)],
  dependencies: [
    .package(url: "https://github.com/apple/swift-openapi-generator", exact: "1.13.1")
  ],
  targets: []
)
