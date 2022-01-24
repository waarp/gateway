#!/usr/bin/env bash

set -e

ACTION=${1:-help}
shift || true

#####################################################################
###  TASKS
#####################################################################

t_test() {
  go test "$@" ./cmd/... ./pkg/...
}

t_check() {
  if ! which golangci-lint >/dev/null 2>&1; then
    echo "golangci-lint cannot be found. Please, install it and re-run checks"
    return 1
  fi

  echo "L'intégration continue utilise toujours la dernière version de "
  echo "golangci-lint, en utilisant le fichier de configuration .golangci.yml."

  go vet ./cmd/... ./pkg/...
  golangci-lint run
}

t_test_watch() {
  goconvey -launchBrowser=false -port=8081 -excludedDirs=doc "$@"
}

ensure-venv() {
  test -d .venv && return 0

  local VENV_CMD="python -m venv"
  if ! $VENV_CMD -h >/dev/null 2>&1; then
    VENV_CMD=virtualenv
  fi

  $VENV_CMD .venv
  ./.venv/bin/pip install -r requirement.txt

}

t_doc() {
  pushd doc || return 2
  ensure-venv
  make html SPHINXBUILD=".venv/bin/sphinx-build" "$@"
  popd || return 2
}

t_doc_watch() {
  pushd doc || return 2
  ensure-venv
  PATH=.venv/bin:$PATH sphinx-autobuild --port 8082 "$@" source/ build/html/
  popd || return 2
}

t_doc_dist() {
  local name
  name="waarp-gateway-doc-$(cat VERSION)"

  pushd doc || return 2
  ensure-venv
  make clean html \
    SPHINXBUILD=".venv/bin/sphinx-build" \
    SPHINXOPTS="-D todo_include_todos=0" \
    "$@"

  pushd build || return 2
  cp -r html "$name"
  zip -rm9 "$name.zip" "$name"
  popd || return 2
  popd || return 2

  mkdir -p build
  test -f "build/$name.zip" && rm -f "build/$name.zip"
  mv "doc/build/$name.zip" build
}

t_generate() {
  go get -u golang.org/x/tools/cmd/stringer
  go generate ./cmd/... ./pkg/...
}

t_build() {
  mkdir -p build

  CGO_ENABLED=1 go build -ldflags " \
    -X code.waarp.fr/apps/gateway/gateway/pkg/version.Date=$(date -u --iso-8601=seconds) \
    -X code.waarp.fr/apps/gateway/gateway/pkg/version.Num=$(git describe --tags --dirty) \
    -X code.waarp.fr/apps/gateway/gateway/pkg/version.Commit=$(git rev-parse --short HEAD)" \
    -o "build/waarp-gateway" ./cmd/waarp-gateway
  CGO_ENABLED=1 go build -ldflags " \
    -X code.waarp.fr/apps/gateway/gateway/pkg/version.Date=$(date -u --iso-8601=seconds) \
    -X code.waarp.fr/apps/gateway/gateway/pkg/version.Num=$(git describe --tags --dirty) \
    -X code.waarp.fr/apps/gateway/gateway/pkg/version.Commit=$(git rev-parse --short HEAD)" \
    -o "build/waarp-gatewayd" ./cmd/waarp-gatewayd
}

build_static_binaries() {
  mkdir -p build

  echo "==> building for $GOOS/$GOARCH"

  # TODO: Run tests

  CGO_ENABLED=1 go build -ldflags "-s -w -extldflags '-fno-PIC -static' \
    -X code.waarp.fr/apps/gateway/gateway/pkg/version.Date=$(date -u --iso-8601=seconds) \
    -X code.waarp.fr/apps/gateway/gateway/pkg/version.Num=$(git describe --tags --dirty) \
    -X code.waarp.fr/apps/gateway/gateway/pkg/version.Commit=$(git rev-parse --short HEAD)" \
    -buildmode pie \
    -tags 'osusergo netgo static_build sqlite_omit_load_extension' \
    -o "build/waarp-gateway_${GOOS}_${GOARCH}" ./cmd/waarp-gateway
  CGO_ENABLED=1 go build -ldflags "-s -w -extldflags '-fno-PIC -static' \
    -X code.waarp.fr/apps/gateway/gateway/pkg/version.Date=$(date -u --iso-8601=seconds) \
    -X code.waarp.fr/apps/gateway/gateway/pkg/version.Num=$(git describe --tags --dirty) \
    -X code.waarp.fr/apps/gateway/gateway/pkg/version.Commit=$(git rev-parse --short HEAD)" \
    -buildmode pie \
    -tags 'osusergo netgo static_build sqlite_omit_load_extension' \
    -o "build/waarp-gatewayd_${GOOS}_${GOARCH}" ./cmd/waarp-gatewayd

  # get-remotes
  CGO_ENABLED=0 go build -ldflags "-s -w" \
    -tags 'osusergo netgo static_build sqlite_omit_load_extension' \
    -o "build/get-remote_${GOOS}_${GOARCH}" ./dist/get-remote

  # updateconf
  CGO_ENABLED=0 go build -ldflags "-s -w" \
    -tags 'osusergo netgo static_build sqlite_omit_load_extension' \
    -o "build/updateconf_${GOOS}_${GOARCH}" ./dist/updateconf

  # Checking compiled binaries
  if [ "$GOOS" = "linux" ]; then
    local out binary_file

    for name in waarp-gateway waarp-gatewayd updateconf get-remote; do
      binary_file="build/${name}_${GOOS}_${GOARCH}"
      out=$(file "$binary_file")

      echo "-> Verifying $binary_file:"
      echo "$out"
      echo

      if [[ ! "$out" = *statically* ]]; then
        echo "ERROR: $binary_file is not a static binary"
        return 2
      fi
      if [[ ! "$out" = *stripped* ]]; then
        echo "ERROR: $binary_file is not a stripped binary"
        return 2
      fi
      if [[ "$out" = *"for GNU/Linux 4.4.0"* ]]; then
        echo "ERROR: $binary_file is only compatible with Linux 4.4.0+"
        return 2
      fi
    done
  fi
}

t_build_dist() {
  cat <<EOW
This procedure will build static stripped binaries for Linux (32 and 64 bits)
and Windows (32 and 64 bits).

Warning:
  It needs a gcc compiler for linux 32bits (in archlinux, the package
  "lib32-gcc-libs") and for Windows (mingw-w64-gcc).


EOW
  
  mkdir -p build

  GOOS=linux GOARCH=amd64 build_static_binaries
  GOOS=linux GOARCH=386 build_static_binaries
  GOOS=windows GOARCH=amd64 CC="x86_64-w64-mingw32-gcc" build_static_binaries
}

t_package() {
  rm -rf build
  t_build_dist
  ./build/waarp-gatewayd_linux_amd64 server -c build/waarp-gatewayd.ini -n

  # pre-configure the service
  sed -i \
    -e "s|; \(GatewayHome =\)|\1 /var/lib/waarp-gateway|" \
    -e "s|; \(Address =\) |\1 /var/lib/waarp-gateway/db/|" \
    -e "s|; \(AESPassphrase =\) |\1 /etc/waarp-gateway/|" \
    build/waarp-gatewayd.ini

  # build the packages
  nfpm pkg -p rpm -f dist/nfpm.yaml --target build/
  nfpm pkg -p deb -f dist/nfpm.yaml --target build/

  build_portable_archive

  t_doc_dist
}

build_portable_archive() {
  local dest version
  dest="build/waarp-gateway-$(cat VERSION)"
  version=$(cat VERSION)

  mkdir -p "$dest"/{etc,bin,log,share}
  cp ./dist/manage.sh "$dest/bin"
  cp ./build/waarp-gatewayd_linux_amd64 "$dest/bin/waarp-gatewayd"
  cp ./build/waarp-gateway_linux_amd64 "$dest/bin/waarp-gateway"
  cp ./build/get-remote_linux_amd64 "$dest/share/get-remote"
  cp ./build/updateconf_linux_amd64 "$dest/share/updateconf"

  ./build/waarp-gatewayd_linux_amd64 server -c "$dest/etc/gatewayd.ini" -n
  sed -i \
    -e "s|; \(GatewayHome =\)|\1 data|" \
    -e "s|; \(Address =\) |\1 data/db/|" \
    -e "s|; \(AESPassphrase =\) |\1 etc/|" \
    "$dest/etc/gatewayd.ini"

  pushd build || return 2
  tar czf "waarp-gateway-$version.linux.tar.gz" "waarp-gateway-$version"
  popd || return 2
}

t_bump() {
  if [ -z "$1" ]; then
    echo "ERROR: bump needs the version to be specified as the first argument"
  fi

  echo "$1" > VERSION
  sed -i -e "s|version:.*|version: v$(cat VERSION)|" dist/nfpm.yaml
}

t_usage() {
  echo "Usage $0 [ACTION]"
  echo ""
  echo "Available actions"
  echo ""
  echo "  generate    Runs go generate"
  echo "  build       Builds binaries"
  echo "  build dist  Builds binaries for distribution"
  echo "  package     Generates packages"
  echo "  check       Run static analysis"
  echo "  test        Run tests"
  echo "  test watch  Starts convey to watch code and run tests when"
  echo "              it has been changed"
  echo "  doc         Builds the doc"
  echo "  doc watch   Watch the source of the documentation and builds it when"
  echo "  doc dist    Builds doc for distribution"
  echo "              it has been changed"
  echo "  bump        Sets the version"
  echo ""
}

#####################################################################
###  MAIN
#####################################################################

case $ACTION in
  generate)
    t_generate
    ;;

  build)
    SUB=$1
    case $SUB in
      dist)
        shift
        t_build_dist
        ;;

      *)
        t_build
        ;;
    esac
    ;;

  package)
    t_package
    ;;

  check)
    t_check
    ;;

  test)
    SUB=$1
    case $SUB in
      watch)
        shift
        t_test_watch "$@"
        ;;
      *)
        t_test "$@"
        ;;
    esac
    ;;

  doc)
    SUB=$1
    case $SUB in
      watch)
        shift
        t_doc_watch "$@"
        ;;

      dist)
        shift
        t_doc_dist "$@"
        ;;

      *)
        t_doc "$@"
        ;;
    esac
    ;;

  bump)
    t_bump "$1"
    ;;

  *)
    t_usage
    ;;
esac
