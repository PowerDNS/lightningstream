# Using Debian 13 (trixie). Prefer using the same libc for everything touching the LMDB. When mixing with e.g. Alpine,
# locking issues can occur because of differences in file locking primitives implementations.

# Builder
FROM golang:1.26.4-trixie AS builder

WORKDIR /src
ADD . ./
RUN --mount=type=cache,target="/root/.cache/go-build" --mount=type=cache,target="/go/pkg/mod" \
    GOBIN=/usr/local/bin go install ./cmd/...

# Dist
FROM debian:trixie-slim

RUN <<EOF
set -euo pipefail

apt-get update -qq
apt-get install -yqq --no-install-recommends \
    ca-certificates

rm -rf /var/lib/apt/lists/*
rm /var/log/apt/history.log
rm /var/log/dpkg.log
rm /var/log/apt/term.log

mkdir /snapshots
chmod 777 /snapshots
EOF

COPY --from=builder /usr/local/bin/lightningstream /usr/local/bin/lightningstream

ENTRYPOINT ["/usr/local/bin/lightningstream"]
