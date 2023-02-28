package snapshot

const (
	// CurrentFormatVersion is the current snapshot format we write
	// Version 2 added the flags fields and Deleted flags, before this version
	// empty values indicated deleted entries.
	// Version 3 fixed the DBI flags to always represent the original DBI
	// instead of the shadow DBI, added the compatVersion field, and
	// added the per-database 'transform' field.
	CurrentFormatVersion uint32 = 3

	// CompatFormatVersion is the oldest snapshot version we can read.
	// v1 is the first version of our snapshots.
	// We will try to always support any old version, unless there is very
	// strong reason not to.
	CompatFormatVersion uint32 = 1

	// WriteCompatFormatVersion is the oldest snapshot version that snapshots
	// which were made with this program version are compatible with.
	// v1 is the first version of our snapshots, which indicates that any
	// old client is able to read these snapshots.
	// Note that there are limitations regarding support for newer features.
	// For example, during v1 an empty entry indicated deletion, while v2
	// introduced a flag for this, so an old v1 client cannot support
	// non-deleted empty entries.
	WriteCompatFormatVersion uint32 = 1
)
