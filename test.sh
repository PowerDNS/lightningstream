#!/bin/sh

# S3 and Azure backend tests in simpleblob use testcontainers-go to spin up
# MinIO and Azurite automatically. They require a reachable Docker socket and
# will self-skip via testcontainers.SkipIfProviderIsNotHealthy if one is not available.
if [ -S /var/run/docker.sock ] || docker info > /dev/null 2>&1; then
    echo "* Docker socket available, S3 (MinIO) and Azure (Azurite) backend tests will run via testcontainers"
else
    echo "* Docker socket not available, S3 and Azure backend tests will be skipped"
fi

set -ex

go test -count=1 "$@" ./...
go test -count=1 "$@" github.com/PowerDNS/simpleblob/...

# Run again with race detector
go test -race -count=5 "$@" ./...
GOMAXPROCS=1 go test -race -count=5 "$@" ./...

# This one used to be flaky, run a few more times
go test -count 20 -run TestSyncer_Sync_startup ./syncer

# Configure linters in .golangci.yml
GOBIN="$PWD/bin" go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.0.2
./bin/golangci-lint run

