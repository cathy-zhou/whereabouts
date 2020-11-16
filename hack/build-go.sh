#!/usr/bin/env bash
set -eu
cmd=whereabouts
cmdMigrate=migrate
eval $(go env | grep -e "GOHOSTOS" -e "GOHOSTARCH")
GOOS=${GOOS:-${GOHOSTOS}}
GOARCH=${GOACH:-${GOHOSTARCH}}
GOFLAGS=${GOFLAGS:-}
GLDFLAGS=${GLDFLAGS:-}
CGO_ENABLED=0 GOOS=${GOOS} GOARCH=${GOARCH} go build ${GOFLAGS} -ldflags "${GLDFLAGS}" -o bin/${cmd} cmd/${cmd}.go
CGO_ENABLED=0 GOOS=${GOOS} GOARCH=${GOARCH} go build ${GOFLAGS} -ldflags "${GLDFLAGS}" -o bin/${cmdMigrate} cmd/${cmdMigrate}.go

