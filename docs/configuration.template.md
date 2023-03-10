# Configuration

The individual configuration statements are documented in the example YAML configuration file, which is reproduced
at the end of this chapter.

The configuration consists of the following general sections:

- Instance name
- Storage backend for snapshots
- LMDBs to sync
- Sync parameter tweaking
- Monitoring and logging

LightningStream allows environment variables in the YAML configuration files, e.g.:

```yaml
instance: ${LS_INSTANCE}
```


## Instance name

Every instance MUST have a unique instance name. This instance name is included in the snapshot filenames
and used to ignore its own snapshots.

If you accidentally configure two instances with the same name, the following will happen:

- They will not see each other's changes, unless a third instance happens to include them in a snapshot.
- Other instances may not see all the changes from both instances, because they will only load the most
  recent snapshot for this instance name.

The instance name can either be configured in the config file:

```yaml
instance: unique-instance-name-here
```

Or it can be passed using the `--instance` commandline flag, but then you must be careful to always pass it.

As mentioned above, environment variables can be used in the YAML configuration, which can be
useful for the instance name:

```yaml
instance: ${LS_INSTANCE}
```

The instance name should be composed of safe characters, like ASCII letters, numbers, dashes and dots.
It MUST NOT contain underscores, slashes, spaces or other special characters. Basically what is allowed
in a raw domain name or hostname is safe.


## Storage

LightningStream uses our [Simpleblob](https://github.com/PowerDNS/simpleblob) library to support
different storage backends. At the moment of writing, it supports S3 and local filesystem
backends.


### S3 backend

This is currently the only backend that makes sense for a production environment. It stores
snapshots in an S3 or compatible storage. We have tested it against Amazon AWS S3 and Minio servers.

Minio example for testing without TLS:

```yaml
storage:
  type: s3
  options:
    access_key: minioadmin
    secret_key: minioadmin
    region: us-east-1
    bucket: lightningstream
    endpoint_url: http://localhost:9000
```

Currently available options:

| Option | Type | Summary |
|--------|------|---------|
| access_key | string | S3 access key |
| secret_key | string | S3 secret key |
| region | string | S3 region (default: "us-east-1") |
| bucket | string | Name of S3 bucket |
| create_bucket | bool | Create bucket if it does not exist |
| global_prefix | string | Transparently apply a global prefix to all names before storage |
| prefix_folders | bool | Show folders in list instead of recursively listing them |
| endpoint_url | string | Use a custom endpoint URL, e.g. for Minio |
| tls | [tlsconfig.Config](https://github.com/PowerDNS/go-tlsconfig) | TLS configuration |
| init_timeout | duration | Time allowed for initialisation (default: "20s") |
| use_update_marker | bool | Reduce LIST commands, see link below |
| update_marker_force_list_interval | duration | See link below for details |

The `use_update_marker` option can be useful to reduce your AWS S3 bill in small personal
deployments, as GET operations are 10 times cheaper than LIST operations, but it cannot reliably
be used when you are using a bucket mirror mechanism to keep multiple buckets in sync.

You can find all the available S3 options with full descriptions in
[Simpleblob's S3 backend Options struct](https://github.com/PowerDNS/simpleblob/blob/main/backends/s3/s3.go#:~:text=Options%20struct).


### Filesystem backend

For local testing, it can be convenient to store all snapshots in a local directory instead of
an S3 bucket:

```yaml
storage:
  type: fs
  options:
    root_path: /tmp/snapshots
```

## LMDBs

The `lmdbs` section configures which LMDB databases to sync. One LightningStream instance can sync more than
one LMDB database. Snapshots are independent per LMDB.

Every database requires a name that must not change over time,
as it is included in the snapshot filenames. The name should only contain
lowercase letters and must not contains spaces, underscores or special characters.
If you are bad at naming things and have only one database, "main" is a good safe choice.

A basic example for an LMDB with a native schema:

```yaml
lmdbs:
  main:
    path: /path/to/lmdb/dir
    options:
      create: true
      map_size: 1GB
    schema_tracks_changes: true  # native schema
```

The `path` option is the file path to the LMDB directory, or file if `options.no_subdir` is `true`.

Some commonly use LMDB options:

- `no_subdir`: the LMDB does not use a directory, but a plain file (required for PowerDNS).
- `create`: create the LMDB if it does not exist yet.
- `map_size`: the LMDB map size, which is the maximum size the LMDB can grow to.

The `schema_tracks_changes` indicates if the LMDB supported the [native LightningStream schema](schema-native.md).

If the LMDB does not support a native schema, you can use a configuration like this:

```yaml
lmdbs:
  main:
    path: /path/to/lmdb/dir
    options:
      create: true
      map_size: 1GB
    schema_tracks_changes: false  # non-native schema
    dupsort_hack: true            # set this if the non-native LMDB uses MDB_DUPSORT
```

Do read [the section on schemas](schema.md) to evaluate if the used LMDB schema is safe for syncing.


## Sync parameters

There are some top-level sync parameters that you may want to tweak for specific deployments, but
these do have sensible defaults, so you probably do not need to.

The ones you are more likely to want to change are:

- `storage_poll_interval`: how often to list the storage to check for new snapshots (default: 1s)
- `storage_force_snapshot_interval`: how often to force a snapshot when there are no changes (default: 4h)
- `lmdb_poll_interval`: how often to check the LMDB for changes (default: 1s). This check is very fast by itself, but
  you may want to adjust it if you want to limit the rate at which an instance can generate snapshots when there
  are frequent updates.

More can be found with descriptions in the example configuration.


## Monitoring and logging

Example of a few logging and monitoring options:

```yaml
# HTTP server with status page, Prometheus metrics and /healthz endpoint.
# Disabled by default.
http:
  address: ":8500"    # listen on port 8500 on all interfaces

# Logging configuration
log:
   level: info        # "debug", "info", "warning", "error", "fatal"
   format: human      # "human", "logfmt", "json"
   timestamp: short   # "short", "disable", "full"

health:
   # ... see example config
```


## Example config with comments

This example configuration assumes a PowerDNS Authoritative server setup with native schemas, but it
explains every available option.

```yaml
__CONFIG__
```

