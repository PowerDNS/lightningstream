package hooks

import "github.com/PowerDNS/lightningstream/snapshot"

func New() *Hooks {
	return &Hooks{}
}

// Hooks allow running extra code at specific points of the sync flow (LSE).
type Hooks struct {
	AfterSnapshotUnpack func(*snapshot.Snapshot) error
	UpdateSnapshotInfo  func(*snapshot.Snapshot, *snapshot.NameInfo) error
}
