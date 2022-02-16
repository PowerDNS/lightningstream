# LightningStream

LightningStream is a tool to stream Lightning Memory-Mapped Database (LMDB) changes to an S3 bucket in
near realtime. Receiving instances can update a local LMDB from these snapshots in near realtime to
reflect remote changes, with a typical replication delay of a few seconds.

It is inspired by [Litestream](https://litestream.io/), which does something similar for sqlite3 databases,
but without the realtime receiving capabilities.


## Building

At the moment of writing, this project requires Go 1.17. Please check the `go.mod` file for the current
version.

To install the binary in a given location, simply run:

    GOBIN=$HOME/bin go install ./cmd/lightningstream/

Or run `./build.sh` to install it in a `bin/` subdirectory of this repo. 

Easy cross compiling is not supported, because the LMDB bindings require CGo.


## Example in Docker Compose

This repo includes an example of syncing the PowerDNS Authoritative Nameserver LMDB. It runs two DNS
servers with each their own syncer, syncing to a common volume.

The LightningStream config used can be found in `docker/pdns/lightningstream.yaml`.

To get it up and running:

    docker-compose up -d

You may need to rerun this command once, because of a race condition creating the LMDBs.

To see the services:

    docker-compose ps

This should show output like:

```
         Name                        Command               State                                    Ports
-------------------------------------------------------------------------------------------------------------------------------------------
lightningstream_auth1_1   /run.sh                          Up      127.0.0.1:4751->53/tcp, 127.0.0.1:4751->53/udp, 127.0.0.1:4781->8081/tcp
lightningstream_auth2_1   /run.sh                          Up      127.0.0.1:4752->53/tcp, 127.0.0.1:4752->53/udp, 127.0.0.1:4782->8081/tcp
lightningstream_sync1_1   /usr/local/bin/lightningst ...   Up
lightningstream_sync2_1   /usr/local/bin/lightningst ...   Up
```

Open one terminal with all the logs:

    docker-compose logs

Then in another terminal call these convenience scripts, with a delay between them to allow for syncing:

    docker/pdns/pdnsutil -i 1 create-zone example.org
    docker/pdns/pdnsutil -i 1 secure-zone example.org
    docker/pdns/pdnsutil -i 1 set-meta example.org foo bar
    docker/pdns/pdnsutil -i 2 generate-tsig-key example123

    sleep 2

    docker/pdns/curl-api -i 2 /api/v1/servers/localhost/zones/example.org
    docker/pdns/curl-api -i 2 /api/v1/servers/localhost/zones/example.org/metadata
    docker/pdns/curl-api -i 1 /api/v1/servers/localhost/tsigkeys

To view a dump of the LMDB contents:

    docker/pdns/dump-lmdb -i 1
    docker/pdns/dump-lmdb -i 2

