package snapshot

const (
	// CurrentFormatVersion is the current snapshot format we write
	// Version 2 added the flags fields and Deleted flags, before this version
	// empty values indicated deleted entries.
	CurrentFormatVersion uint32 = 2

	// CompatFormatVersion is the oldest snapshot version we can read
	// v1 is the first version of our snapshots.
	CompatFormatVersion uint32 = 1
)
