# Builder
# Note: Not using Alpine, because of CGo deps
FROM golang:1.25-bookworm AS builder
ENV GOBIN=/usr/local/bin
ARG GOPROXY=

# GOFLAGS workaround for Go 1.19 when building from subrepos
# Issue: https://github.com/golang/go/issues/53640
# TODO: Check if still needed for 1.24+
ARG GOFLAGS
ENV GOFLAGS ${GOFLAGS}

RUN mkdir /src
ADD go.mod go.sum /src/
WORKDIR /src
RUN echo "GOPROXY=$GOPROXY"; go mod download
ADD . /src
RUN echo "GOFLAGS=$GOFLAGS"; go install ./cmd/...

# Dist
# Note: When using Alpine, ls prevents auth api from correctly functioning (most likely some locking issue)
# Reason: When you have multiple processing accessing the same LMDB, they MUST use the same libc, otherwise
#         way file locking is done will conflict.
FROM debian:bookworm-slim

RUN ulimit -n 2000 \
        && apt-get update \
        && apt-get install -y --no-install-recommends ca-certificates \
        && apt-get -y clean \
        && rm /etc/ld.so.cache \
        && rm -rf /var/lib/apt/lists/* \
        && rm /var/log/apt/history.log \
        && rm /var/log/dpkg.log \
        && rm /var/log/apt/term.log
RUN update-ca-certificates

COPY --from=builder /usr/local/bin/lightningstream /usr/local/bin/lightningstream
RUN mkdir /snapshots && chmod 777 /snapshots

ENTRYPOINT ["/usr/local/bin/lightningstream"]
