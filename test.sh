#!/bin/sh

if [ ! -z "$SIMPLEBLOB_TEST_S3_CONFIG" ]; then
    echo "* Using existing SIMPLEBLOB_TEST_S3_CONFIG=$SIMPLEBLOB_TEST_S3_CONFIG"
elif curl -v --connect-timeout 2 http://localhost:4730/ 2>&1 | grep --silent MinIO ; then
    echo "* Using MinIO in Docker Compose for tests"
    export SIMPLEBLOB_TEST_S3_CONFIG="$PWD/docker/test-minio.json"
else
    echo "* MinIO not running in Docker Compose, skipping S3 tests"
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

