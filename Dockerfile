# Not using Alpine, because of CGo deps
FROM golang:1.17-buster
ENV GOBIN=/usr/local/bin
ARG GOPROXY=
RUN mkdir /src
ADD go.mod go.sum /src
WORKDIR /src
RUN echo "GOPROXY=$GOPROXY"; go mod download
ADD . /src
RUN go install ./cmd/...
RUN mkdir /snapshots && chmod 777 /snapshots
ENTRYPOINT ["/usr/local/bin/lightningstream"]
