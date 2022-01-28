#!/bin/sh
# Generates Go bindings for our modified dnsmessage.proto.
# The output files need to be checked into the repo.
# Only rerun this if proto was updated.

export GOBIN="$PWD/../bin"
export PATH="$GOBIN:$PATH"

set -ex

# Also mentioned in tools.go
go install github.com/gogo/protobuf/protoc-gen-gogofast
protoc --gogofast_out=. snapshot.proto

