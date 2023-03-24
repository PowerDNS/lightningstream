# Schema

Lightning Stream can sync LMDBs in two different modes:

1. If the application is Lightning Stream aware and uses native Lightning Stream headers for all its values, it can directly
   use these headers to sync and merge data. This is described in the [native schema section](schema-native.md).
2. If the application does not use the special Lightning Stream headers, data can still be synced, but Lightning Stream needs
   to create _shadow_ DBIs to keep track of all the sync metadata.
   This is described in the [non-native schema (shadow) section](schema-shadow.md).

## General schema considerations

No matter if you are using the [native mode](schema-native.md) or [non-native mode](schema-shadow.md), there are some
important considerations regarding how the application manages its data, that affect if it is safe to use
Lightning Stream with multiple active writers.

Conceptually, consider an LMDB with Lightning Stream like a global key-value storage that is updated by multiple
instances at the same time. Does the way these values are stored by the application allow that to be done safely?

### Application caching of state

If the application assumes that it is the only one writing to the LMDB, it may cache certain state and not be aware
of changes made by Lightning Stream when syncing data from other instances.

For example, it could have an in-memory cache of the state and only invalidate this cache when itself writes
a change to the LMDB.

Another example would be where the LMDB would map IDs to Names, and the application maintains an in-memory
reverse index from Name to ID.

#### Solution

Since LMDB is so fast, it may be feasible to store all state in the LMDB and read it on demand.

Alternatively, a cache could be short-lived (e.g. 1 second), or the application could check if the LMDB's LastTxnID
has changed since the last cache update.

!!! TODO

    Link to a section describing LMDB concepts like LastTxnID.


### Natural keys vs. Sequential IDs

If an application uses **sequential IDs** as keys, using multiple writers will quickly result in a conflict, because it is
very likely that two instances will try to create a new entry using the same ID.

**Natural keys**, on the other hand, do not have this problem. For example, if an entry describes a domain name, use
the domain name itself as the key, instead of a number, if possible. Even if two instances try to create the same
entry, it will not result in an inconsistent database, as the natural key automatically prevents the addition of
duplicate domain entries. A **hash** of the fields that make up the uniqueness constraint provides similar guarantees.

If natural keys cannot be used, use **random or globally unique IDs like UUIDs** to reduce the chance of an ID clash.
The larger the ID, the smaller the chance of a clash. In this case you do need to be aware that duplicate entries can
occur, for example if two users try to add an `example.com` entry on different instances at the same time: you will end
up with two `example.com` entries with different IDs.


### Multi-value entries and indices

Suppose you have a DBI that keeps track of tags or categories for keys. One way to organise this would be:

- "key 1" => "RED,BLUE" 
- "key 2" => "RED,GREEN,YELLOW" 

Now consider what happens if two instances want to update the tags for "key 1" at the same time with an additional
tag:

- Instance A: "key 1" => "RED,BLUE,YELLOW" 
- Instance B: "key 1" => "RED,BLUE,CYAN" 

One of these updates will win the race, and the other one will get lost.

For these tags this may be acceptable, but consider a scenario where these are not tags, but indices:

- "accounts-owned-by-jane" => "14,522,1314"

If there is no process to automatically fix these indices, this can cause serious issues.

#### Solution

Use multiple entries, with the individual IDs encoded in the key.

Example:

- "accounts-owned-by-jane:14" => ""
- "accounts-owned-by-jane:522" => ""
- "accounts-owned-by-jane:1314" => ""

The value part can be kept empty, or contain some data, as needed.

When you need a list of values, perform an `MDB_SET_RANGE` query to fetch all keys that start with
"accounts-owned-by-jane:".

If a safe separator cannot be found, the key can be prefixed with one or two bytes indicating the length
of the key. Also keep in mind that LMDB keys are typically limited to 511 bytes.

These records can safely be added and deleted by different instances of the application.

!!! warning

    You may be tempted to solve this with `MDB_DUPSORT`, but Lightning Stream only supports dupsort
    DBIs in non-native mode, and then only with [severe caveats](schema-shadow.md#the-dupsort_hack).




