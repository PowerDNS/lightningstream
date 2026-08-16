# Using Debian 13 (trixie). Prefer using the same libc for everything touching the LMDB. When mixing with e.g. Alpine,
# locking issues can occur because of differences in file locking primitives implementations.

# Builder
FROM golang:1.27rc2-trixie AS builder


WORKDIR /src
COPY . ./
RUN --mount=type=cache,id=gomod,target=/go/pkg/mod \
    --mount=type=cache,id=gobuild,target=/root/.cache/go-build \
    GOBIN=/usr/local/bin go install -trimpath ./cmd/...


# Dist
FROM debian:trixie-slim AS runtime

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

COPY --from=builder /usr/local/bin/* /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/lightningstream"]
