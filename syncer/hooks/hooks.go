package hooks

import (
	"github.com/PowerDNS/lightningstream/lmdbenv/header"
	"github.com/PowerDNS/lightningstream/snapshot"
	"github.com/PowerDNS/lightningstream/syncer/events"
)

func New() *Hooks {
	return &Hooks{}
}

// Hooks allow running extra code at specific points of the sync flow (LSE).
// These hooks are run synchronously, the calling code waits for the response.
type Hooks struct {
	// AfterSnapshotUnpack is called after a remote snapshot is unpacked into memory
	AfterSnapshotUnpack func(*snapshot.Snapshot) error

	// UpdateSnapshotInfo is called after a snapshot is created but
	// before it is stored.
	UpdateSnapshotInfo func(SnapshotInfo) error

	// UpdateStored is called when an update is written to storage.
	// This is equal to the event with the same name, but with synchronous
	// processing.
	UpdateStored func(info events.UpdateInfo) error

	// SnapshotOverdue is called when we force a snapshot due to a
	// forced snapshot interval.
	// This is equal to the event with the same name, but with synchronous
	// processing.
	SnapshotOverdue func() error

	// FilterReadDBI is called in readDBI to perform any filtering
	FilterReadDBI func(p FilterReadDBIParams) bool // included if it returns true

	// Returns a channel that can inject other updates into the sync loop
	OtherUpdateSource func() <-chan snapshot.Update
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
