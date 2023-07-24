#!/bin/bash

set -ex

port=${port:-53}
webserver_port=${webserver_port:-8081}

mkdir -p "/lmdb/instance-$instance"
chown -R pdns:pdns "/lmdb/instance-$instance"

echo "
setuid=pdns
setgid=pdns
guardian=no
daemon=no
disable-syslog=yes
write-pid=no
socket-dir=/tmp
log-dns-queries=yes
loglevel=99

local-address=0.0.0.0,::
local-port=$port
primary=yes
secondary=yes

webserver-port=$webserver_port
webserver-password=changeme
webserver=yes
webserver-address=0.0.0.0
webserver-allow-from=0.0.0.0/0
api=yes
api-key=changeme

zone-cache-refresh-interval=0
zone-metadata-cache-ttl=0

load-modules=liblmdbbackend.so
launch=lmdb
lmdb-filename=/lmdb/instance-$instance/db
lmdb-shards=1
lmdb-lightning-stream=yes
" > /etc/powerdns/pdns.conf

pdns_server || (
    # WORKAROUND: The LMDB backend creates the database before the
    #             setuid, resulting in a permission denied error on the first run
    echo "WORKAROUND: restart after assumed permission error"
    chown -R pdns:pdns "/lmdb/instance-$instance"
    pdns_server
)
