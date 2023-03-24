# Getting started

The easiest way try out Lightning Stream with the PowerDNS Authoritative server is through the Docker Compose demo
in the [Lightning Stream GitHub repository](https://github.com/PowerDNS/lightningstream/).

For manual installation instructions, please check [this section](pdns-auth-installation.md).


## Docker Compose demo

The Lightning Stream repository contains a Docker Compose demo of Lightning Stream running alongside
the PowerDNS Authoritative server, to sync the LMDB backend data.

!!! warning

    This demo does NOT handle [schema migrations](schema-migration.md) between different versions of PowerDNS Authoritative server.
    It is NOT suitable for production use!

The compose setup runs two read-write DNS servers, each with their own syncer, syncing to a bucket in a MinIO server.
Additionally, a third server has Lightning Stream running in receive-only mode.

The Lightning Stream config used can be found in `docker/pdns/lightningstream.yaml`. Note that the
config file contents can reference environment variables.

To get it up and running:

    docker-compose up -d

You may need to rerun this command once, because of a race condition when creating the LMDBs.

To see the services:

    docker-compose ps

This should show output like:

```
NAME                      IMAGE                   SERVICE   PORTS
lightningstream-auth1-1   powerdns/pdns-auth-48   auth1     127.0.0.1:4751->53/tcp, 127.0.0.1:4751->53/udp, 127.0.0.1:4781->8081/tcp
lightningstream-auth2-1   powerdns/pdns-auth-48   auth2     127.0.0.1:4752->53/tcp, 127.0.0.1:4752->53/udp, 127.0.0.1:4782->8081/tcp
lightningstream-auth3-1   powerdns/pdns-auth-48   auth3     127.0.0.1:4753->53/tcp, 127.0.0.1:4753->53/udp, 127.0.0.1:4783->8081/tcp
lightningstream-minio-1   minio/minio             minio     127.0.0.1:4730->9000/tcp, 127.0.0.1:4731->9001/tcp
lightningstream-sync1-1   lightningstream-sync1   sync1     127.0.0.1:4791->8500/tcp
lightningstream-sync2-1   lightningstream-sync2   sync2     127.0.0.1:4792->8500/tcp
lightningstream-sync3-1   lightningstream-sync3   sync3     127.0.0.1:4793->8500/tcp
```

Open a new terminal to see all the logs, to get a feeling of how and when Lightning Stream syncs data:

    docker-compose logs

Then, in another terminal, call these convenience scripts, with a delay between them to allow for syncing between instances:

    docker/pdns/pdnsutil -i 1 create-zone example.org
    docker/pdns/pdnsutil -i 1 secure-zone example.org
    docker/pdns/pdnsutil -i 1 set-meta example.org foo bar
    docker/pdns/pdnsutil -i 2 generate-tsig-key example123
    docker/pdns/pdnsutil -i 1 add-record example.org www A 60 192.0.2.3

    sleep 2

    docker/pdns/curl-api -i 2 /api/v1/servers/localhost/zones/example.org
    docker/pdns/curl-api -i 2 /api/v1/servers/localhost/zones/example.org/metadata
    docker/pdns/curl-api -i 1 /api/v1/servers/localhost/tsigkeys

These scripts execute `pdnsutil` and `curl` inside of the Docker containers. The `-i` flag specifies the instance number
to operate on.

For example, you can check if the records were correctly synced to instance 2 with this command:

    $ docker/pdns/pdnsutil -i 2 list-zone example.org
    $ORIGIN .
    example.org	3600	IN	SOA	a.misconfigured.dns.server.invalid hostmaster.example.org 0 10800 3600 604800 3600
    www.example.org	60	IN	A	192.0.2.3

The same should be true for instance 3, which is in receive-only mode:

    $ docker/pdns/pdnsutil -i 3 list-zone example.org
    $ORIGIN .
    example.org	3600	IN	SOA	a.misconfigured.dns.server.invalid hostmaster.example.org 0 10800 3600 604800 3600
    www.example.org	60	IN	A	192.0.2.3

You can use `dig` against the DNS servers to verify that the new `A` record works:

    $ dig +short -p 4753 @127.0.0.1 www.example.org
    192.0.2.3

To list all the generated snapshots, you can use this script:

    docker/pdns/lightningstream -i 1 snapshots list -l

To view a dump of the LMDB contents, you can use the `dump-lmdb` command:

    docker/pdns/dump-lmdb -i 1
    docker/pdns/dump-lmdb -i 2

You can also browse the snapshots in MinIO at <http://localhost:4731/buckets/lightningstream/browse>
(login with minioadmin / minioadmin).


