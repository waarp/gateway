#!/usr/bin/env bash

GOOS=$1 GOARCH=$2

echo "==> building for $GOOS/$GOARCH"

mkdir -p build

git_tag=$(git describe --tags --dirty)
version=${git_tag#"v"}

CGO_ENABLED=0 go build -ldflags "-s -w \
  -X code.waarp.fr/apps/gateway/gateway/pkg/version.Date=$(date -u --iso-8601=seconds) \
  -X code.waarp.fr/apps/gateway/gateway/pkg/version.Num=$version \
  -X code.waarp.fr/apps/gateway/gateway/pkg/version.Commit=$(git rev-parse --short HEAD)" \
  -tags 'osusergo netgo static_build sqlite_omit_load_extension' \
  -o "build/waarp-gateway_${GOOS}_${GOARCH}" ./cmd/waarp-gateway
CGO_ENABLED=0 go build -ldflags "-s -w \
  -X code.waarp.fr/apps/gateway/gateway/pkg/version.Date=$(date -u --iso-8601=seconds) \
  -X code.waarp.fr/apps/gateway/gateway/pkg/version.Num=$version \
  -X code.waarp.fr/apps/gateway/gateway/pkg/version.Commit=$(git rev-parse --short HEAD)" \
  -tags 'osusergo netgo static_build sqlite_omit_load_extension' \
  -o "build/waarp-gatewayd_${GOOS}_${GOARCH}" ./cmd/waarp-gatewayd