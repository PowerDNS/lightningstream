# LightningStream

LightningStream is a tool to sync changes between a local LMDB (Lightning Memory-Mapped Database) and 
an S3 bucket in near real-time. If the application schema is compatible, this can be used in a multi-writer
setup where any instance can update any data, with a global eventually consistent view of the data in seconds.

Our main target application is the sync of LMDB databases in the 
[PowerDNS Authoritative Nameserver](https://doc.powerdns.com/authoritative/) (PDNS Auth), but LightningStream
does not make any assumptions about the contents, and can be used to sync other LMDBs, as long as the data
is stored using a [compatible schema](schema.md).

### Name origin

TODO: move to FAQ

The project was inspired by [Litestream](https://litestream.io/), which does something similar for sqlite3 databases,
but without the simultaneous multi-writer capabilities.



