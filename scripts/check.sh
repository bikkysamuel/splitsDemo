#!/usr/bin/env bash
# Runs the same checks as CI, locally or in CI (the workflows call it).
# Tool versions are pinned here only; no tool needs installing.
#
#   scripts/check.sh [server|api|ios|secrets|all]   (default: all)
#
# server: needs Postgres (`docker compose up -d postgres`); TEST_DATABASE_URL
#         comes from the environment or, if unset, from .env. FUZZTIME (default
#         30s) is the fuzz time per target, e.g. FUZZTIME=5s for a quick run.
# ios:    needs Xcode 27 and an iOS 27.0 iPhone Simulator.
# secrets: needs Docker.
set -euo pipefail

GOLANGCI_LINT=github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0
GOVULNCHECK=golang.org/x/vuln/cmd/govulncheck@v1.8.0
VACUUM=github.com/daveshanley/vacuum@v0.30.6
# v8.30.1, pinned by digest (doc 08: pins are immutable)
GITLEAKS_IMAGE=ghcr.io/gitleaks/gitleaks@sha256:c00b6bd0aeb3071cbcb79009cb16a60dd9e0a7c60e2be9ab65d25e6bc8abbb7f

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

step() { printf '\n\033[1m==> %s\033[0m\n' "$*"; }

# Paths regenerated from api/openapi.yaml, plus the generator pins.
GENERATED=(
  server/internal/httpapi/apigen
  ios/Packages/Infrastructure/Sources/APIClient/Generated
  ios/Tools/APIGenerator/Package.resolved
)

# generated_state fingerprints GENERATED's working-tree state.
generated_state() {
  git status --porcelain -- "${GENERATED[@]}"
  git diff -- "${GENERATED[@]}" | git hash-object --stdin
}

check_server() {
  if [ -z "${TEST_DATABASE_URL:-}" ] && [ -f .env ]; then
    # Last assignment wins; surrounding quotes are stripped.
    TEST_DATABASE_URL="$(sed -n 's/^TEST_DATABASE_URL=//p' .env | tail -n 1 | sed -e 's/^["'"'"']//' -e 's/["'"'"']$//')"
    export TEST_DATABASE_URL
  fi
  (
    cd server
    step "server: gofmt"
    unformatted="$(gofmt -l .)"
    if [ -n "$unformatted" ]; then
      echo "Run gofmt -w on:"; echo "$unformatted"; exit 1
    fi
    step "server: sqlc generate, fail on drift (ADR-0014)"
    sqlc_state() { git status --porcelain -- internal/store/sqlcgen; git diff -- internal/store/sqlcgen | git hash-object --stdin; }
    before="$(sqlc_state)"
    go generate ./internal/store/
    if [ "$before" != "$(sqlc_state)" ]; then
      echo "sqlc code was out of date with the queries or migrations; regenerated it. Review and commit:"
      git status --porcelain -- internal/store/sqlcgen
      exit 1
    fi
    step "server: go vet"
    go vet ./...
    step "server: golangci-lint"
    go run "$GOLANGCI_LINT" run ./...
    step "server: go test -race (real Postgres)"
    go test -race ./...
    step "server: fuzz, ${FUZZTIME:-30s} per target (doc 09)"
    # `go test -list` prints each package's Fuzz targets, then "ok <package>".
    go test -list '^Fuzz' ./... | awk '/^Fuzz/ {names = names " " $1} /^ok/ {if (names != "") print $2 names; names = ""}' |
      while read -r pkg targets; do
        for target in $targets; do
          go test -run '^$' -fuzz "^${target}\$" -fuzztime "${FUZZTIME:-30s}" "$pkg"
        done
      done
    step "server: govulncheck"
    go run "$GOVULNCHECK" ./...
  )
}

check_api() {
  step "api: vacuum lint"
  go run "$VACUUM" lint -r api/vacuum-ruleset.yaml --fail-severity warn -d api/openapi.yaml

  step "api: regenerate Go and Swift code, fail on drift"
  local before
  before="$(generated_state)"
  scripts/generate-api.sh
  if [ "$before" != "$(generated_state)" ]; then
    echo "Generated code was out of date with api/openapi.yaml; regenerated it. Review and commit:"
    git status --porcelain -- "${GENERATED[@]}"
    exit 1
  fi
}

check_ios() {
  (
    cd ios
    step "ios: swift-format lint"
    scripts/lint.sh
    step "ios: every package test target is in the Splits scheme"
    scripts/check-scheme-tests.py
    step "ios: xcodebuild test (iOS 27.0 Simulator)"
    local udid
    udid="$(xcrun simctl list devices available 'iOS 27.0' | grep -m1 -E 'iPhone' | grep -oE '[0-9A-F-]{36}' || true)"
    if [ -z "$udid" ]; then
      echo "No iOS 27.0 iPhone Simulator available:"; xcrun simctl list runtimes; exit 1
    fi
    xcodebuild test -project Splits.xcodeproj -scheme Splits -destination "id=$udid" CODE_SIGNING_ALLOWED=NO -quiet
  )
}

check_secrets() {
  step "secrets: gitleaks (git history)"
  docker run --rm -v "$ROOT:/repo:ro" "$GITLEAKS_IMAGE" git /repo --no-banner --redact
}

case "${1:-all}" in
  server) check_server ;;
  api) check_api ;;
  ios) check_ios ;;
  secrets) check_secrets ;;
  all) check_api; check_server; check_ios; check_secrets ;;
  *) echo "usage: scripts/check.sh [server|api|ios|secrets|all]" >&2; exit 2 ;;
esac
step "OK: ${1:-all}"
