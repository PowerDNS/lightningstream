# Native schema

Native mode is used when the `schema_tracks_changes` setting is set to `true`. This is the recommended mode of operation,
but it can only be used if the application natively supports LightningStream headers. This mode provides nanosecond
precision timestamps for conflict resolution and the best performance.

The PowerDNS Authoritative server supports native LightningStream headers starting from version 4.8. As of March 2023,
the latest release with this support is 4.8.0-alpha1.

Native mode provides the following benefits:

- Nanosecond precision for sync conflict resolution.
- No duplication of data in shadow tables.
- Better performance, because data does not need to be copied to and from shadow tables.
- Many operations can be performed using read locks, which do no block other readers and writers.


