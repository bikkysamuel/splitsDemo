#!/usr/bin/env python3
"""Fails when a package test target isn't run by the shared Splits scheme.

`xcodebuild test -scheme Splits` runs only the scheme's Testables that are
not skipped, so a missing or skipped target would pass CI without running.
"""
import glob
import os
import re
import sys

os.chdir(os.path.join(os.path.dirname(os.path.abspath(__file__)), ".."))
SCHEME = "Splits.xcodeproj/xcshareddata/xcschemes/Splits.xcscheme"

declared = {}
for manifest in sorted(glob.glob("Packages/*/Package.swift")):
    for name in re.findall(r'\.testTarget\(\s*name:\s*"([^"]+)"', open(manifest).read()):
        declared[name] = manifest

testables = re.search(r"<Testables>(.*?)</Testables>", open(SCHEME).read(), re.S)
running = set()
for ref in re.findall(r"<TestableReference(.*?)</TestableReference>", testables.group(1) if testables else "", re.S):
    if re.search(r'skipped\s*=\s*"NO"', ref):
        running.update(re.findall(r'BlueprintName\s*=\s*"([^"]+)"', ref))

missing = sorted(set(declared) - running)
for name in missing:
    print(f"{name} ({declared[name]}) is not run by the Splits scheme: add it as a Testable with "
          f'skipped = "NO" in ios/{SCHEME}', file=sys.stderr)
if missing:
    sys.exit(1)
print(f"Splits scheme runs all {len(declared)} package test targets: {', '.join(sorted(declared))}")
