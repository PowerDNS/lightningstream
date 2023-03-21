# Schema migration

As applications evolve, they may at a certain moment require a schema migration that transforms old data into a new format.

In a standalone application without Lightning Stream, it is safe to simply rewrite all data on
startup, and then continue working with the new data. With Lightning Stream and other active
instances, things can get a bit more complicated: you do not want Lightning Stream to merge
in data in a format from a previous schema version.

There are three ways to avoid such issues:

1. Never do breaking schema upgrades and always support values written by an older version.
2. For the new schema version data, create new DBIs with new names.
3. Use different storage buckets for different schema versions.

The best practical solution may be a combination of these three approaches, for different complexities
of changes.


## Never perform breaking schema upgrades

If data is stored in an extensible format, like Google Protobuf, you may be able to avoid
schema migration altogether. 

New values are written with the new extended fields, and it is fine if entries in an older format
are merged in by Lightning Stream. Over time, perhaps all entries are rewritten when updated.

This does require that the application is forever able to read any old values that it wrote.

If there are multiple active instances, you also need to consider the reverse: if the new version
writes an entry in the new format, how will any older instances that are still running and syncing
handle this data?


## Use new DBIs with different names

You could always create new DBIs with a new name when you migrate data.

For example, let's say you start with a `users_v1` DBI. When you perform a migrations, you
copy all the records in a new format to a new `users_v2` DBI.

Lightning Stream will pick it up and start syncing the new DBI to other instances, which will
ignore it until they are upgraded. The tricky part is how they will decide when they need to
migrate the data.

In this case, you cannot safely use a common DBI that is shared between version to tell the application
what the current schema version is, because that one will be increased as soon as the first instance
performs a schema migration. The other instances will think the database is already upgraded, and you
may lose your most recent changed.

If you can prevent any new changes during your upgrade, or writes only go to the instance you
upgrade first, this method can work.

The downside is that the old DBIs will never be removed, and they will be retained in the snapshots.


## Use a different storage bucket

You can include the schema version in your configured S3 storage bucket prefix. This is probably
the most robust solution, but it does complicate how you run Lightning Stream, as this makes
its configuration dependent on the application schema version.

You also must sure that Lightning Stream is not running on the instance during the migration, or
it could end up syncing data to the wrong bucket.

Lightning Stream allows you to use environment variables in its YAML configuration, like
`${APP_SCHEMA_VERSION}`. If you can reliably set this before starting Lightning Stream, you can use
this to automatically write different schema versions to different S3 bucket prefixes.



