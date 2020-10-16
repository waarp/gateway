#!/usr/bin/env bash

ACTION=$1
shift

#####################################################################
###  TASKS
#####################################################################

t_test() {
  go test "$@" ./cmd/... ./pkg/...
}

t_check() {
  local CI_VERSION
  CI_VERSION=$(grep GOLANGCI_LINT_VERSION .gitlab-ci.yml | head -n 1 | cut -d' ' -f4)

  if ! which golangci-lint >/dev/null 2>&1; then
    echo "golangci-lint cannot be found. Please, install it and re-run checks"
    return 1
  fi
  if ! golangci-lint --version | grep "$CI_VERSION" >/dev/null 2>&1; then
    echo "***********************************************"
    echo "WARNING"
    echo "***********************************************"
    echo "Your version os golangci-lint is: $(golangci-lint --version | sed 's:.*\s\([0-9\.]\+\)\s.*:\1:')"
    echo "CI runs version $CI_VERSION"
    echo "***********************************************"
  fi
  go vet ./cmd/... ./pkg/...
  golangci-lint run \
    --enable bodyclose,dogsled,dupl,funlen,gocognit,gocyclo,gofmt,goimports,golint,gosec,misspell,nakedret,scopelint,stylecheck,unconvert,unparam \
    --max-issues-per-linter 0 --max-same-issues 0 \
    --exclude-use-default=false  \
    --exclude 'Potential file inclusion via variable' \
    --exclude 'Error return value of .((os\.)?std(out|err)\..*|.*Close|.*Flush|os\.Remove(All)?|.*printf?|os\.(Un)?Setenv). is not checked' \
    --exclude 'SA5008' \
    --skip-dirs .gocache \
    --exclude 'ST1016.*tec' \
    --tests=false --no-config
      # The exclude of 'ST1016.*tec' is needed to avoid false positives caused by stylecheck
      # not ignoring generated files (https://github.com/golangci/golangci-lint/issues/846)
      # In this case, it does not ignores a file generated by stringer
}

t_test_watch() {
  goconvey -launchBrowser=false -port=8081 -excludedDirs=doc "$@"
}

t_doc() {
  pushd doc || return 2
  make html SPHINXBUILD=".venv/bin/sphinx-build" "$@"
  popd || return 2
}

t_doc_watch() {
  cd doc || return 2
  PATH=.venv/bin:$PATH sphinx-autobuild -p 8082 "$@" source/ build/html/
}

t_doc_dist() {
  local name
  name="waarp-gateway-doc-$(cat VERSION)"

  pushd doc || return 2
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

t_build() {
  mkdir -p build
  go generate ./cmd/... ./pkg/...
  CGO_ENABLED=1 go build -ldflags " \
    -X code.waarp.fr/waarp-gateway/waarp-gateway/pkg/version.Date=$(date -u --iso-8601=seconds) \
    -X code.waarp.fr/waarp-gateway/waarp-gateway/pkg/version.Num=$(git describe --tags --dirty) \
    -X code.waarp.fr/waarp-gateway/waarp-gateway/pkg/version.Commit=$(git rev-parse --short HEAD)" \
    -o "build/waarp-gateway" ./cmd/waarp-gateway
  CGO_ENABLED=1 go build -ldflags " \
    -X code.waarp.fr/waarp-gateway/waarp-gateway/pkg/version.Date=$(date -u --iso-8601=seconds) \
    -X code.waarp.fr/waarp-gateway/waarp-gateway/pkg/version.Num=$(git describe --tags --dirty) \
    -X code.waarp.fr/waarp-gateway/waarp-gateway/pkg/version.Commit=$(git rev-parse --short HEAD)" \
    -o "build/waarp-gatewayd" ./cmd/waarp-gatewayd
}

build_static_binaries() {
  mkdir -p build

  echo "==> building for $GOOS/$GOARCH"

  # TODO: Run tests

  CGO_ENABLED=1 go build -ldflags "-s -w -extldflags '-fno-PIC -static' \
    -X code.waarp.fr/waarp-gateway/waarp-gateway/pkg/version.Date=$(date -u --iso-8601=seconds) \
    -X code.waarp.fr/waarp-gateway/waarp-gateway/pkg/version.Num=$(git describe --tags --dirty) \
    -X code.waarp.fr/waarp-gateway/waarp-gateway/pkg/version.Commit=$(git rev-parse --short HEAD)" \
    -buildmode pie \
    -tags 'osusergo netgo static_build sqlite_omit_load_extension' \
    -o "build/waarp-gateway_${GOOS}_${GOARCH}" ./cmd/waarp-gateway
  CGO_ENABLED=1 go build -ldflags "-s -w -extldflags '-fno-PIC -static' \
    -X code.waarp.fr/waarp-gateway/waarp-gateway/pkg/version.Date=$(date -u --iso-8601=seconds) \
    -X code.waarp.fr/waarp-gateway/waarp-gateway/pkg/version.Num=$(git describe --tags --dirty) \
    -X code.waarp.fr/waarp-gateway/waarp-gateway/pkg/version.Commit=$(git rev-parse --short HEAD)" \
    -buildmode pie \
    -tags 'osusergo netgo static_build sqlite_omit_load_extension' \
    -o "build/waarp-gatewayd_${GOOS}_${GOARCH}" ./cmd/waarp-gatewayd
}

t_build_dist() {
  cat <<EOW
This procedure will build static stripped binaries for Linux (32 and 64 bits)
and Windows (32 and 64 bits).

Warning:
  It needs a gcc compiler for linux 32bits (in archlinux, the package
  "lib32-gcc") and for Windows ()


EOW
  
  mkdir -p build
  go generate ./cmd/... ./pkg/...
  GOOS=linux GOARCH=amd64 build_static_binaries
  GOOS=linux GOARCH=386 build_static_binaries
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
  sed -i -e "s|version:.*|version: v$(cat VERSION)-1|" dist/nfpm.yaml
  nfpm pkg -p rpm -f dist/nfpm.yaml --target build/
  nfpm pkg -p deb -f dist/nfpm.yaml --target build/

  t_doc_dist
}

t_usage() {
  echo "Usage $0 [ACTION]"
  echo ""
  echo "Available actions"
  echo ""
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
  echo ""
}

#####################################################################
###  MAIN
#####################################################################

case $ACTION in
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

  *)
    t_usage
    ;;
esac
