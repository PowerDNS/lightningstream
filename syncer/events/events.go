package events

import (
	"github.com/PowerDNS/lightningstream/snapshot"
	"github.com/PowerDNS/lightningstream/utils/topics"
	"github.com/PowerDNS/simpleblob"
)

// New returns an initialized Events struct
func New() *Events {
	return &Events{
		List:                       topics.New[simpleblob.BlobList](),
		LastSeenSnapshotByInstance: topics.New[map[string]snapshot.NameInfo](),
		UpdateLoaded:               topics.New[snapshot.NameInfo](),
	}
}

// Events contains event topics that can be subscribed to.
type Events struct {
	// List is triggered when a remote listing was completed successfully
	List *topics.Topic[simpleblob.BlobList]

	// LastSeenSnapshotByInstance when we have determined the latest snapshot
	// per instance. If an instance no longer has any snapshots, it will
	// disappear from the map.
	LastSeenSnapshotByInstance *topics.Topic[map[string]snapshot.NameInfo]

	// UpdateLoaded is triggered when an update is successfully loaded into
	// the LMDB.
	// FIXME: LSE: Also add transaction info? Stats?
	UpdateLoaded *topics.Topic[snapshot.NameInfo]

	// UpdateStored is triggered when we successfully uploaded a snapshot.
	// FIXME: LSE: Also add transaction info? Stats?
	UpdateStored *topics.Topic[snapshot.NameInfo]
}
