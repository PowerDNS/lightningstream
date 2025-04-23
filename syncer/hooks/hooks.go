package hooks

import (
	"github.com/PowerDNS/lightningstream/lmdbenv/header"
	"github.com/PowerDNS/lightningstream/snapshot"
)

func New() *Hooks {
	return &Hooks{}
}

// Hooks allow running extra code at specific points of the sync flow (LSE).
type Hooks struct {
	// AfterSnapshotUnpack is called after a remote snapshot is unpacked into memory
	AfterSnapshotUnpack func(*snapshot.Snapshot) error

	// UpdateSnapshotInfo is called after a snapshot is created but
	// before it is stored.
	UpdateSnapshotInfo func(SnapshotInfo) error

	// FilterReadDBI is called in readDBI to perform any filtering
	FilterReadDBI func(p FilterReadDBIParams) bool // included if it returns true
}

type SnapshotInfo struct {
	Snapshot *snapshot.Snapshot
	NameInfo *snapshot.NameInfo
}

type FilterReadDBIParams struct {
	Timestamp header.Timestamp
	TxnID     header.TxnID
	Flags     header.Flags
}
