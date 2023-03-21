
# Non-native (shadow) mode

When the LMDB is not in a native format, Lightning Stream needs to create _shadow_ copies of the data DBIs to keep track of
change timestamps. Every second it will check if the local LMDB has changes (last transaction ID has changed), and merge
the local data into the shadow DBIs when changes have been detected. This requires a full scan over all records.

Non-native mode can be used by setting the `schema_tracks_changes` setting is set to `false`.

If the LMDB also uses `MDB_DUPSORT` functionality, Lightning Stream can support it by setting `dupsort_hack` to `true`.
This comes with additional caveats. `MDB_DUPSORT` is not supported at all in native mode.

## Older PowerDNS Authoritative versions

PowerDNS Authoritative server 4.7 did not use a native schema and required both `schema_tracks_change: false` and
`dupsort_hack: true`.

Additionally, it requires the `lmdb-random-ids=yes` setting for the LMDB backend.


## Caveats

This non-native mode does come with several limitations and downsides.

### Timestamp precision

Changed records will get the current timestamp, but these will only have a precision of around 1 second, since we only
check it once per second. This can cause updates to the same record on multiple instances to applied in the wrong order.

Consider this example, in order of execution, but within one second or so:

- On instance A, key "example" is set to "A".
- On instance B, key "example" is set to "B".
- Lightning Stream on B performs a sync iteration. 
- Lightning Stream on A performs a sync iteration.

In this example, the value "A" will end up with a later timestamp than value "B", and the instances will soon converge
on this value "A", which is an older value.

With a native schema, the application would have set a nanosecond precision timestamp on the records, and the newer
value "B" would have won, assuming that the clocks on both instances are in sync.

This issue can be mitigated by sticking to one server for updates from the same client. This guarantees that the client
will always immediately see the changes it applied, and in the right order.

### The dupsort_hack

When the schema contains DBIs with `MDB_DUPSORT` set, Lightning Stream needs to perform an additional transformation, as
its merge algorithm requires unique keys. The `dupsort_hack` rewrites keys in the shadow tables for these DBIs to
include part of the value, to make all keys unique.

For these DBIs, every time a change is detected Lightning Stream currently needs to completely rewrite the shadow DBI, and
the original DBI if it needs to sync back changes from remote instances.

### Long write locks

Every sync operation, including creating a local snapshot, requires a write lock on the LMDB, because the shadow DBIs
need to be updated. Write locks -- unlike read locks -- prevent the application from writing at the same time. This
issue is exacerbated by the extra time needed to update shadow tables, especially when the `dupsort_hack` is also
needed.

This likely constrains its use to relatively small LMDBs with thousands of records, not millions.

### Double LMDB disk and memory usage

The need to create the shadow DBIs effectively doubles the required disk space and memory usage of the LMDBs.


