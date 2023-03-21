# Native header schema

Native mode is used when the `schema_tracks_changes` setting is set to `true`. This is the recommended mode of operation,
but it can only be used if the application natively supports Lightning Stream headers. This mode provides nanosecond
precision timestamps for conflict resolution and the best performance.

The PowerDNS Authoritative server supports native Lightning Stream headers starting from version 4.8. As of March 2023,
the latest release with this support is 4.8.0-alpha1.

Native mode benefits include:

- Nanosecond precision for sync conflict resolution.
- No duplication of data in shadow tables.
- Better performance, because data does not need to be copied to and from shadow tables.
- Many operations can be performed using read locks, which do not block other readers and writers.

## Native header

The native Lightning Stream header (LS header) MUST be added to every value in the LMDB. The basic header is 24 bytes long, with the
possibility to extend this with additional 8-byte blocks in the future. 

The header consists of the following fields:

| Size      | Description                                            |
|-----------|--------------------------------------------------------|
| 8 bytes   | Timestamp (nanoseconds since UNIX epoch)               |
| 8 bytes   | LMDB transaction ID of last update                     |
| 1 byte    | Header schema version (currently always 0)             |
| 1 byte    | Flags (currently only: 0x01 = Deleted, see below)      |
| 4 bytes   | Reserved for future use, currently all 0               |
| 2 bytes   | Number of 8-byte extension blocks (N)                  |
| N*8 bytes | Header extensions, depending on the previous field     |

**All values are in network (Big Endian) byte order.**

The 8-byte size multiples were chosen to simplify alignment requirements if the value consists of a C struct, but be aware that
if you read a value from the LMDB without copying, there is no guarantee that the value pointer you receive is itself aligned.
It is recommended to use a cross-platform serialisation format for the values, instead of C structs.

### Timestamp

The **timestamp** is the number of nanoseconds since the UNIX epoch of the last update to this value. If a user explicitly writes the same old
value to an entry, the timestamp SHOULD also be updated, in case this is to undo another change to this entry that has not been
merged yet. An application MAY decide to only update the timestamp when the actual value changes, depending on its update semantics.

When migrating an old database to the new schema, care SHOULD be taken to use a timestamp of 0 so that the value is always considered older
than an entry added on another node that already uses the new schema.

### Transaction ID

The **LMDB transaction IDs** are 64 bit (on 64 bit systems) sequential IDs, starting at 1 for the very first write transaction. The ID is 0
before the first write to the LMDB. The application MUST write the current transaction ID when updating an entry. 
Transaction IDs have no meaning outside of the local instance and are never included in snapshots. They are used for local change detection.

For an initial migration, the transaction ID MAY be set to 0, but it MUST be set to the current transaction ID in future migrations. It is
recommended to always set it to the current transaction ID, even in migrations.

### Schema version

The **schema version** is currently always 0 and SHOULD be checked by the application. We do not plan to ever increase it, unless we really need
to make incompatible changes to the header structure. Since we have a mechanism for header extensions, we do not expect this to ever
be necessary.

### Flags

There is currently only one flag defined:

| Value | Name    | Description                                      |
|-------|---------|--------------------------------------------------|
| 0x01  | Deleted | Current entry is deleted                         |

The Deleted flag is set when the current entry is considered deleted from the LMDB. When keys are to be removed by the application, it
MUST NOT actually delete the key, and instead mark the entry as deleted. The application value MUST be reset to an empty value. These
records allow deletes to be propagated to other instances.

!!! note

    We will provide a mechanism to make Lightning Stream automatically clean old deleted entries in the future.
    Lightning Stream needs to be in charge of this, because it also needs to ignore old deleted entries in snapshots
    to not recreate them.

Applications MUST ignore unknown flags when reading, and they MUST NOT set or retain flags they do not understand. Any flag additions
will be made with the understanding that applications are allowed to ignore them.

### Reserved bytes

Applications MUST set the **reserved bytes** to 0 when writing a value, and ignore any value set by other applications.

### Extension blocks

Applications MUST take into account the **number of extension blocks** when determining the header size.
Applications MUST NOT hard code a header size of 24 bytes.
Applications MUST NOT add or retain any extension blocks when writing them, and they MUST ignore unknown extension
blocks when present. Currently the format of any extension blocks has not been defined.

We intend to define a format where extension blocks are optional and identified by an ID and length, but currently this idea has
not been worked out yet.

## DBI flag limitations

In native mode, Lightning Stream only supports DBIs without any special DBI flags. More specifically, the following DBI flags
are NOT supported in native mode:

- `MDB_DUPSORT`
- `MDB_DUPFIXED`
- `MDB_INTEGER`
- `MDB_INTEGERDUP`
- `MDB_REVERSEKEY`
- `MDB_REVERSEDUP`

The reverse keys are currently also not supported in non-native mode, or at least not tested.


## Old timestamp-only headers

Before version 0.3.0, Lightning Stream used a simpler header with only an 8 byte timestamp. No native application ever used this
(it was used in the old shadow DBIs), but you may see references to this in parts of the code or older documentation.
These are obsolete and no longer supported.

