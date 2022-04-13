# Builder: Not using Alpine, because of CGo deps
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
FROM alpine:3.15

RUN apk add --no-cache libc6-compat

COPY --from=builder /usr/local/bin/lightningstream /usr/local/bin/lightningstream
RUN mkdir /snapshots && chmod 777 /snapshots

ENTRYPOINT ["/usr/local/bin/lightningstream"]