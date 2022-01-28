#!/bin/sh

set -ex

# Configure linters in .golangci.yml
GOBIN="$PWD/bin" go install github.com/golangci/golangci-lint/cmd/golangci-lint
./bin//golangci-lint run

go test ./...
