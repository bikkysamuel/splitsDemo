#!/usr/bin/env bash
# Regenerates the Go server and Swift client code from api/openapi.yaml
# (ADR-0012). Run after every contract change and commit the result; the
# `api` CI job fails if the committed code differs from a fresh run.
set -euo pipefail
cd "$(dirname "$0")/.."

echo "Go: oapi-codegen → server/internal/httpapi/apigen"
(cd server && go generate ./internal/httpapi/apigen/)

echo "Swift: swift-openapi-generator → ios/Packages/Infrastructure/Sources/APIClient/Generated"
out=ios/Packages/Infrastructure/Sources/APIClient/Generated
mkdir -p "$out"
swift run --package-path ios/Tools/APIGenerator --quiet swift-openapi-generator generate \
  api/openapi.yaml \
  --config ios/Tools/APIGenerator/openapi-generator-config.yaml \
  --output-directory "$out"
