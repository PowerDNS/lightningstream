#!/bin/sh
# Generates Go bindings for our modified dnsmessage.proto.
# The output files need to be checked into the repo.
# Only rerun this if proto was updated.

export GOBIN="$PWD/../bin"
export PATH="$GOBIN:$PATH"

set -ex

# No longer updated, so latest should be safe
go install github.com/gogo/protobuf/protoc-gen-gogofast@latest
# Last one to include the protoc compat command
go install github.com/bufbuild/buf/cmd/buf@v1.0.0-rc12
buf protoc --gogofast_out=. snapshot.proto

# FIXME: Currently need this to prevent copying all byte slices
patch -p1 < patch.diff
