#!/usr/bin/env bash
# Installs what the browser smoke test needs to drive Chromium — a Node
# runtime, the Playwright driver, the browser, axe-core — under one directory,
# cached as a whole. Writes the environment the test reads to
# .playwright/env.sh: the caller sources it as a separate step, so a failed
# install fails the caller.
set -euo pipefail
# Without it, errexit is off inside every `$(…)`.
shopt -s inherit_errexit

# `promote` removes its destination: refuse to run anywhere but the repo root.
cd "$(dirname "$0")/.."
[ -f go.mod ] || { echo "not the repository root: $PWD" >&2; exit 1; }

playwright_home=$PWD/.playwright
node_major=${NODE_MAJOR:-22}
# It ends up in a path `promote` removes.
[[ $node_major =~ ^[0-9]+$ ]] || { echo "invalid NODE_MAJOR: $node_major" >&2; exit 1; }
node_dist=https://nodejs.org/dist/latest-v$node_major.x
npm_registry=https://registry.npmjs.org
# Accessibility rules run in the browser by the smoke test.
axe_version=${AXE_CORE_VERSION:-4.10.3}
[[ $axe_version =~ ^[0-9][0-9.]*$ ]] || { echo "invalid AXE_CORE_VERSION: $axe_version" >&2; exit 1; }

node_root="$playwright_home/node-v$node_major"
node_bin="$node_root/bin/node"
axe_root="$playwright_home/axe-core-$axe_version"

fetch() {
  curl -fsSL --proto-redir '=https' "$@"
}

# The cache is uploaded even when the job fails: everything is staged, then
# renamed in, so a half-written tree never satisfies its own guard.
promote() { # <staging dir> <destination>
  rm -rf "${2:?}"
  mv -T "$1" "$2"
}

# The driver version playwright-go requires, read from the module itself.
# Digits and dots only: it ends up in a path `promote` removes.
driver_version() {
  local module version
  module=$(go list -m -f '{{.Dir}}' github.com/mxschmitt/playwright-go)
  [ -n "$module" ] || { echo "playwright-go is not in the module cache" >&2; return 1; }
  version=$(sed -n 's/^\(const \|\t\)playwrightCliVersion = "\([0-9][0-9.]*\)"$/\2/p' "$module/run.go")
  [ -n "$version" ] || { echo "playwrightCliVersion not found in $module" >&2; return 1; }
  printf '%s\n' "$version"
}

# The image's node is too old for the driver bundle (node >= 18).
install_node() {
  local staging tarball
  staging=$(mktemp -d "$playwright_home/staging.XXXXXX")
  fetch "$node_dist/SHASUMS256.txt" -o "$staging/SHASUMS256.txt"
  tarball=$(sed -n 's/^[0-9a-f]\{64\}  \(node-v[0-9.]*-linux-x64\.tar\.gz\)$/\1/p' \
    "$staging/SHASUMS256.txt")
  [ -n "$tarball" ] || { echo "no linux-x64 build under $node_dist" >&2; return 1; }
  fetch "$node_dist/$tarball" -o "$staging/$tarball"
  (cd "$staging" && awk -v f="$tarball" '$2 == f' SHASUMS256.txt | sha256sum -c -)
  tar xzf "$staging/$tarball" -C "$staging" --strip-components=1
  rm -f "$staging/$tarball" "$staging/SHASUMS256.txt"
  promote "$staging" "$node_root"
}

# The npm tarball carries the driver bundle.
install_package() { # <package> <version> <destination>
  local staging
  staging=$(mktemp -d "$playwright_home/staging.XXXXXX")
  fetch "$npm_registry/$1/-/$1-$2.tgz" \
    | tar xz -C "$staging"
  promote "$staging" "$3"
}

mkdir -p "$playwright_home"
rm -rf "${playwright_home:?}"/staging.*

# On a cold module cache `go list -m` reports an empty Dir and exits 0.
go mod download github.com/mxschmitt/playwright-go
version=$(driver_version)
driver_root="$playwright_home/driver-$version"
[ -x "$node_bin" ] || install_node
[ -e "$driver_root/package/cli.js" ] || install_package playwright-core "$version" "$driver_root"
[ -e "$axe_root/package/axe.min.js" ] || install_package axe-core "$axe_version" "$axe_root"

export PLAYWRIGHT_NODEJS_PATH="$node_bin"
export PLAYWRIGHT_DRIVER_PATH="$driver_root"
export PLAYWRIGHT_BROWSERS_PATH="$playwright_home/browsers"
export AXE_CORE_JS="$axe_root/package/axe.min.js"

# playwright-go drops the driver's stderr: run its version check here first.
"$PLAYWRIGHT_NODEJS_PATH" --version
"$PLAYWRIGHT_NODEJS_PATH" "$PLAYWRIGHT_DRIVER_PATH/package/cli.js" --version

go run github.com/mxschmitt/playwright-go/cmd/playwright install chromium

{
  printf 'export PLAYWRIGHT_NODEJS_PATH=%q\n' "$PLAYWRIGHT_NODEJS_PATH"
  printf 'export PLAYWRIGHT_DRIVER_PATH=%q\n' "$PLAYWRIGHT_DRIVER_PATH"
  printf 'export PLAYWRIGHT_BROWSERS_PATH=%q\n' "$PLAYWRIGHT_BROWSERS_PATH"
  printf 'export AXE_CORE_JS=%q\n' "$AXE_CORE_JS"
} > "$playwright_home/env.sh"
