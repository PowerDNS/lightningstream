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
		UpdateLoaded:               topics.New[UpdateInfo](),
		UpdateStored:               topics.New[UpdateInfo](),
		SnapshotOverdue:            topics.New[struct{}](),
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
	UpdateLoaded *topics.Topic[UpdateInfo]

	// UpdateStored is triggered when we successfully uploaded a snapshot.
	UpdateStored *topics.Topic[UpdateInfo]

	// SnapshotOverdue is triggered when we force a snapshot due to a
	// forced snapshot interval.
	SnapshotOverdue *topics.Topic[struct{}]
}

type UpdateInfo struct {
	NameInfo snapshot.NameInfo
	Meta     snapshot.Meta
}
