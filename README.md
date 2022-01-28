# LightningStream

LightningStream is a tool to stream Lightning Memory-Mapped Database (LMDB) changes to an S3 bucket in
near realtime. Receiving instances can update a local LMDB from these snapshots in near realtime to
reflect remote changes, with a typical replication delay of a few seconds.

It is inspired by [Litestream](https://litestream.io/), which does something similar for sqlite3 databases,
but without the realtime receiving capabilities.

