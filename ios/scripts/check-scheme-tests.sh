#!/usr/bin/env bash
# Fails when a package declares a test target that the shared Splits scheme
# doesn't run. `xcodebuild test -scheme Splits` only runs the targets the
# scheme lists, so a missing one would pass CI without ever running.
set -euo pipefail
cd "$(dirname "$0")/.."
python3 - <<'PY'
import glob, re, sys

declared = {}
for manifest in sorted(glob.glob("Packages/*/Package.swift")):
    for name in re.findall(r'\.testTarget\(\s*name:\s*"([^"]+)"', open(manifest).read()):
        declared[name] = manifest

scheme = open("Splits.xcodeproj/xcshareddata/xcschemes/Splits.xcscheme").read()
testables = re.search(r"<Testables>(.*?)</Testables>", scheme, re.S)
listed = set(re.findall(r'BlueprintName\s*=\s*"([^"]+)"', testables.group(1) if testables else ""))

missing = sorted(set(declared) - listed)
for name in missing:
    print(f"{name} ({declared[name]}) is not a Testable in the Splits scheme; add it to "
          "ios/Splits.xcodeproj/xcshareddata/xcschemes/Splits.xcscheme", file=sys.stderr)
if missing:
    sys.exit(1)
print(f"Splits scheme runs all {len(declared)} package test targets: {', '.join(sorted(declared))}")
PY
