# Builder
# Note: Not using Alpine, because of CGo deps
FROM golang:1.17-buster as builder
ENV GOBIN=/usr/local/bin
ARG GOPROXY=
RUN mkdir /src
ADD go.mod go.sum /src/
WORKDIR /src
RUN echo "GOPROXY=$GOPROXY"; go mod download
ADD . /src
RUN go install ./cmd/...

# Dist
# Note: When using Alpine, ls prevents auth api from correctly functioning (most likely some locking issue)
FROM debian:buster-slim

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
