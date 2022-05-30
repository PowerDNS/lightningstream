#!/bin/sh

set -ex

image=lightningstream-test

docker build -t "$image" .
docker run -w /src --entrypoint '' "$image" /src/test.sh "$@"

