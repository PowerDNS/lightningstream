package syncer

import (
	"github.com/PowerDNS/lightningstream/syncer/events"
	"github.com/PowerDNS/lightningstream/syncer/hooks"
)

type Options struct {
	// ReceiveOnly prevents writing snapshots, we will only receive them
	ReceiveOnly bool
	// Events are used to publish events to
	Events *events.Events
	// Hooks can be used to update data at certain points.
	Hooks *hooks.Hooks
}
