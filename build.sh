#!/bin/sh

set -e
set -x

VERSION=${VERSION:-v0.0.3}

export GO111MODULE=auto

CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' ./cmd/client
CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' ./cmd/server
docker build -t docker.io/cilium/migrate-svc-test:$VERSION .
