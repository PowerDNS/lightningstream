#!/bin/sh

set -e

mkdir -p ./bin

if [ -z "$GOBIN" ]; then
    echo "+ GOBIN="./bin""
    export GOBIN="$PWD/bin"
fi

for cmd in cmd/*; do
    if [ -d "$cmd" ]; then
        name=$(basename "$cmd")
        # go install refuses to install cross-compiled binaries
        # https://github.com/golang/go/issues/57485
        echo "go build -o $GOBIN/$name  ./$cmd"
        go build -o "$GOBIN/$name"  "./$cmd"
    fi
done    

ls -lh "$GOBIN"
