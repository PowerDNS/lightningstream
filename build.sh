#!/bin/sh

set -e

mkdir -p ./bin

if [ -z "$GOBIN" ]; then
    echo "+ GOBIN="./bin""
    export GOBIN="$PWD/bin"
fi

for cmd in cmd/*; do
    if [ -d "$cmd" ]; then
        echo "+ go install \"./$cmd\""
        go install "./$cmd"
    fi
done    

ls -lh "$GOBIN"
