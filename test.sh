#!/bin/sh

if [ ! -z "$LIGHTNINGSTREAM_TEST_S3_CONFIG" ]; then
    echo "* Using existing LIGHTNINGSTREAM_TEST_S3_CONFIG=$LIGHTNINGSTREAM_TEST_S3_CONFIG"
elif curl -v --connect-timeout 2 http://localhost:4730/ 2>&1 | grep --silent MinIO ; then
    echo "* Using MinIO in Docker Compose for tests"
    LIGHTNINGSTREAM_TEST_S3_CONFIG="$PWD/docker/test-minio.json"
else
    echo "* MinIO not running in Docker Compose, skipping S3 tests"
fi

set -ex

go test "$@" ./...

# Configure linters in .golangci.yml
GOBIN="$PWD/bin" go install github.com/golangci/golangci-lint/cmd/golangci-lint
./bin//golangci-lint run

