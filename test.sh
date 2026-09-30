#!/bin/sh

set -ex

go test -count=1 "$@" ./...

# Run again with race detector
go test -race -count=5 "$@" ./...
GOMAXPROCS=1 go test -race -count=5 "$@" ./...

# This one used to be flaky, run a few more times
go test -count 20 -run TestSyncer_Sync_startup ./syncer

# Configure linters in .golangci.yml
lint_version=2.14.0
if [ ! -x ./bin/golangci-lint ] || [ "$(./bin/golangci-lint version --short)" != "$lint_version" ]; then
    curl -sSfL https://golangci-lint.run/install.sh | sh -s v$lint_version
fi
./bin/golangci-lint run
