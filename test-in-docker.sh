#!/bin/sh

set -ex

image=lightningstream-test

docker build --target=builder -t "$image" .
docker run \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -w /src --entrypoint '' "$image" /src/test.sh "$@"

