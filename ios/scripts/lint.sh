#!/usr/bin/env bash
# swift-format lint for every hand-written Swift file. Generated code
# (Generated/ folders, from api/openapi.yaml) and build output are skipped.
set -euo pipefail
cd "$(dirname "$0")/.."
find SplitsApp Packages -name '*.swift' \
  -not -path '*/Generated/*' -not -path '*/.build/*' -print0 |
  xargs -0 xcrun swift-format lint --strict --parallel
