# Lightning Stream

[User documentation can be found here](https://doc.powerdns.com/lightningstream/)

![Go build](https://github.com/PowerDNS/lightningstream/actions/workflows/go.yml/badge.svg)
![Documentation build](https://github.com/PowerDNS/lightningstream/actions/workflows/documentation.yml/badge.svg)
[![Go Reference](https://pkg.go.dev/badge/github.com/PowerDNS/lightningstream.svg)](https://pkg.go.dev/github.com/PowerDNS/lightningstream)

Lightning Stream is a tool to sync changes between a local LMDB (Lightning Memory-Mapped Database) and
an S3 bucket in near real-time. If the application schema is compatible, this can be used in a multi-writer
setup where any instance can update any data, with a global eventually-consistent view of the data in seconds.

Our main target application is the sync of LMDB databases in the
[PowerDNS Authoritative Nameserver](https://doc.powerdns.com/authoritative/) (PDNS Auth). We are excited
about how Lightning Stream simplifies running multiple distributed PowerDNS Authoritative servers, with full support
for keeping DNSSEC keys in sync.
Check the [Getting Started](docs/getting-started.md) section to understand how you can use Lightning Stream together
with the PowerDNS Authoritative server.

Its use is not limited to the PowerDNS Authoritative server, however. Lightning Stream does not make any assumptions
about the contents of the LMDB, and can be used to sync LMDBs for other applications, as long as the data is stored
using a [compatible schema](docs/schema.md).


## Basic Operation

Lightning Stream is deployed next to an application that uses an LMDB for its data storage:

![Overview](docs/images/lightningstream-overview.png)

Its operation boils down to the following:

- Whenever it detects that the LMDB has changed, it writes a snapshot of the data to an S3 bucket.
- Whenever it sees a new snapshot written by a _different instance_ in the S3 bucket, it downloads the snapshot
  and merges the data into the local LMDB.

The merge of a key is performed based on a per-record last-modified timestamp:
the most recent version of the entry wins. Deleted entries are cleared and marked as deleted, together with
their deletion timestamp. This allows Lightning Stream to provide **Eventual Consistency** across all nodes.

If the application uses a [carefully designed data schema](docs/schema.md), this approach can be used to support
multiple simultaneously active writers. In other instances, it can often be used to sync data from one writer to
multiple read-only receivers. Or it can simply create a near real-time backup of a single instance.


## Building

At the moment of writing, this project requires Go 1.19. Please check the `go.mod` file for the current
version.

To install the binary in a given location, simply run:

    GOBIN=$HOME/bin go install ./cmd/lightningstream/

Or run `./build.sh` to install it in a `bin/` subdirectory of this repo.

Easy cross compiling is not supported, because the LMDB bindings require CGo.


## Example in Docker Compose

This repo includes an example of syncing the PowerDNS Authoritative Nameserver LMDB. It runs two DNS
servers, each with their own syncer, syncing to a bucket in a MinIO server.

The Lightning Stream config used can be found in `docker/pdns/lightningstream.yaml`. Note that the
config file contents can reference environment variables.

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
lightningstream_minio_1   /usr/bin/docker-entrypoint ...   Up      127.0.0.1:4730->9000/tcp, 127.0.0.1:4731->9001/tcp
lightningstream_sync1_1   /usr/local/bin/lightningst ...   Up      127.0.0.1:4791->8500/tcp
lightningstream_sync2_1   /usr/local/bin/lightningst ...   Up      127.0.0.1:4792->8500/tcp
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

You can browse the snapshots in MinIO at <http://localhost:4731/buckets/lightningstream/browse>
(login with minioadmin / minioadmin).



## Open Source

This is the documentation for the Open Source edition of Lightning Stream.
For more information on how we provide support for Open Source products, please read
[our blog post on this topic](https://blog.powerdns.com/2016/01/18/open-source-support-out-in-the-open/).

PowerDNS also offers an Enterprise edition of Lightning Stream that includes professional support, advanced features, deployment
tooling for large deployments, Kubernetes integration, and more.



