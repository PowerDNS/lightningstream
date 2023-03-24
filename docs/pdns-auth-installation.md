# Lightning Stream with PowerDNS Authoritative server

This section explains how to install and run Lightning Stream with the PowerDNS Authoritative
server the old fashioned way.

!!! TODO

    Explain how to handle [schema migrations](schema-migration.md) in this setup.

## Building Lightning Stream 

At the moment of writing, this project requires Go 1.19. Please check the `go.mod` file for the current
Go version expected.

To install the binary in a given location, simply run:

    GOBIN=$HOME/bin go install ./cmd/lightningstream/

Or run `./build.sh` to install it in a `bin/` subdirectory of this repo. 

Easy cross compiling is unfortunately not supported, because the LMDB bindings require CGo.


## Configuring PowerDNS Authoritative server 4.8+

To install PowerDNS Authoritative server, please read [its installation instructions](https://doc.powerdns.com/authoritative/installation.html).
Make sure to install version 4.8 or higher for Lightning Stream. Also install the `lmdb` backend for PowerDNS Authoritative server, if packaged
separately.

!!! warning

    This section assumes the use of PowerDNS Authoritative version 4.8 and higher, which uses [native Lightning Stream value headers](schema.md).
    As of March 2023, an alpha release of 4.8 is available for testing. 

    These instructions will NOT work with earlier versions, but scroll down if you insist on trying it out with 4.7.

Lightning Stream requires the following PowerDNS Authoritative server settings:

```
# Lightning Stream uses the LMDB backend 
launch=lmdb

# Path to the directory where the LMDB databases for this instance will be stored.
# This MUST be unique per instance, if you are running more than one on the same server.
lmdb-filename=/path/to/lmdb

# Run it with a single shard, to simplify management and configuration.
# Note that this cannot safely be changed later!
lmdb-shards=1

# This MUST be enabled to safely handle multiple writers
lmdb-random-ids=yes

# This MUST be enabled to track and propagate deletes
lmdb-flag-deleted=yes

# You may want a lower number than 16000 MB, which is the default on 64 bit systems.
lmdb-map-size=1000

# You may want to reduce the cache interval to 1 second, or disable it
# altogether with 0, to quickly see your changes. The default is 300 seconds.
# An interval of 1 second will likely provide you with most of the benefits of caching,
# with a barely noticeable delay.
zone-cache-refresh-interval=0
zone-metadata-cache-ttl=0
```


## Configuring Lightning Stream

A basic Lightning Stream configuration for PowerDNS Authoritative looks like this:

```yaml
instance: unique-instance-name  # IMPORTANT: change this
lmdbs:
  main:
    # Auth 'lmdb-filename'
    path: /path/to/lmdb
    schema_tracks_changes: true
    options:
      no_subdir: true
      create: true      # optional for 'main', as auth will create it on startup, if needed
      map_size: 1000MB  # for create=true, make sure to match auth's lmdb-map-size
  shard:
    # Auth 'lmdb-filename' plus '-0' for the first shard
    path: /path/to/lmdb-0
    schema_tracks_changes: true
    options:
      no_subdir: true
      create: true      # strongly recommended for shards
      map_size: 1000MB  # for create=true, make sure to match auth's lmdb-map-size

storage:
  #type: fs
  type: s3
  options:
    #root_path: /tmp/snapshots
    access_key: minioadmin
    secret_key: minioadmin
    region: us-east-1
    bucket: lightningstream-auth48 # use a different bucket or prefix for each auth version
    create_bucket: true
    endpoint_url: http://localhost:9000

http:
  address: ":8500"  # for status and metrics
```

Please check [the configuration section](configuration.md) for details and other options.

Lightning Stream can be run in the foreground as follows:

    $ lightningstream --config=/path/to/config.yaml sync 

Ensure that both PowerDNS Authoritative and Lightning Stream have write access to the LMDBs,
for example by running them under the same system user.



## Running it with an older Authoritative server

While it is possible to run Lightning Stream with PowerDNS Authoritative 4.7 in [non-native mode](schema-shadow.md),
this is NOT recommended due to the [many downsides](schema-shadow.md#caveats) of running it in non-native mode.

!!! warning

    This is NOT recommended, other than for testing and experimenting with non-native shadow mode.
    Please use PowerDNS Authoritative 4.8+ instead.

To run it with PowerDNS Authoritative 4.7 in non-native mode, change the above Lightning Stream configuration as follows:

```yaml
# WARNING: NOT RECOMMENDED, non-native mode for PowerDNS Authoritative server 4.7
lmdbs:
  main:
    ...
    schema_tracks_changes: false
    dupsort_hack: true
  shard:
    ...
    schema_tracks_changes: false

storage:
  type: s3
  options:
    ...
    bucket: lightningstream-auth47 # use a different bucket or prefix for each auth version
```

And remove the `lmdb-flag-deletes` option from the Auth config. The Lightning Stream shadow DBIs will be
tracking these deletes instead.


