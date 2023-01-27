package cleaner

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/PowerDNS/simpleblob/backends/memory"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"powerdns.com/platform/lightningstream/config"
	"powerdns.com/platform/lightningstream/snapshot"
)

func mt(timeString string) time.Time {
	t, err := time.Parse("2006-01-02 15:04:05", timeString)
	if err != nil {
		panic(err)
	}
	return t
}

func snap(syncerName, instanceID, timeString string) string {
	return snapshot.Name(syncerName, instanceID, "G", mt(timeString))
}

var initialSnapshots = []string{
	// The top two are ignored, because we run a cleaner for the "test" prefix.
	snap("ignored", "old", "2020-01-01 01:00:00"),
	snap("ignored", "old", "2020-01-01 01:01:00"),
	snap("test", "a", "2020-01-30 08:00:00"),
	snap("test", "a", "2020-01-30 08:01:00"),
	snap("test", "a", "2020-01-30 08:02:00"),
	snap("test", "a", "2020-01-30 08:03:00"),
	snap("test", "old", "2020-01-01 06:00:00"),
	snap("test", "old", "2020-01-01 07:00:00"),
}

func TestWorker(t *testing.T) {
	st := memory.New()
	logger := logrus.New() // TODO: redirect to testing.T

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// We can control the clock in this test through the 'now' param
	w := New("test", st, config.Cleanup{
		Enabled:                    true,
		Interval:                   time.Minute, // not used in test
		MustKeepInterval:           10 * time.Minute,
		RemoveOldInstancesInterval: 7 * 24 * time.Hour,
	}, logger)

	addSnap := func(name string) {
		assert.NoError(t, st.Store(ctx, name, []byte{'x'}))
	}

	for _, name := range initialSnapshots {
		addSnap(name)
	}

	doRun := func(timeString string, expectedSnapshots []string) {
		// RunOnce with fake time
		now := mt(timeString)
		assert.NoError(t, w.RunOnce(ctx, now), timeString)

		// Get the new list of snapshots
		list, err := st.List(ctx, "")
		assert.NoError(t, err, timeString)

		// Check if it matches our expectations
		names := list.Names()
		sort.Strings(names)
		assert.Equal(t, expectedSnapshots, names, timeString)
	}

	// Service started later than 'a' snapshots.
	// After the initial run, nothing was changed yet, because we consider
	// the time interval since we have first seen a snapshot.
	doRun("2020-01-30 10:00:00", initialSnapshots)

	// Same one minute later
	doRun("2020-01-30 10:01:00", initialSnapshots)

	// Once 10 minutes (MustKeepInterval) pass, we see old snapshots being deleted
	doRun("2020-01-30 10:10:01", []string{
		snap("ignored", "old", "2020-01-01 01:00:00"),
		snap("ignored", "old", "2020-01-01 01:01:00"),
		//snap("test", "a", "2020-01-30 08:00:00"),
		//snap("test", "a", "2020-01-30 08:01:00"),
		//snap("test", "a", "2020-01-30 08:02:00"),
		snap("test", "a", "2020-01-30 08:03:00"),
		//snap("test", "old", "2020-01-01 06:00:00"),
		snap("test", "old", "2020-01-01 07:00:00"),
	})

	// If we add a more recent snapshot, that old one will not yet be deleted
	// on the first run in case it is being accessed.
	addSnap(snap("test", "a", "2020-01-30 10:10:50"))
	doRun("2020-01-30 10:11:00", []string{
		snap("ignored", "old", "2020-01-01 01:00:00"),
		snap("ignored", "old", "2020-01-01 01:01:00"),
		snap("test", "a", "2020-01-30 08:03:00"),
		snap("test", "a", "2020-01-30 10:10:50"),
		snap("test", "old", "2020-01-01 07:00:00"),
	})

	// But it will get deleted on the next run.
	doRun("2020-01-30 10:12:00", []string{
		snap("ignored", "old", "2020-01-01 01:00:00"),
		snap("ignored", "old", "2020-01-01 01:01:00"),
		//snap("test", "a", "2020-01-30 08:03:00"),
		snap("test", "a", "2020-01-30 10:10:50"),
		snap("test", "old", "2020-01-01 07:00:00"),
	})

	// Up until now the "old" instance snapshot was kept, even though
	// it exceeds RemoveOldInstancesInterval. This one is only deleted
	// after we explicitly mark it as loaded, and we have saved a new
	// snapshot ourselves. When this happened, SetCommitted is called by
	// the syncing code.
	w.SetCommitted(map[string]time.Time{
		"old": mt("2020-01-01 07:00:00"),
	})
	doRun("2020-01-30 10:13:00", []string{
		snap("ignored", "old", "2020-01-01 01:00:00"),
		snap("ignored", "old", "2020-01-01 01:01:00"),
		snap("test", "a", "2020-01-30 10:10:50"),
		//snap("test", "old", "2020-01-01 07:00:00"),
	})

	// Once we have indicated that a certain newer version has been committed,
	// any older versions for the stale instance will be cleaned, even if they
	// appear later.
	// But you need to wait for the MustKeepInterval before that happens, so
	// not yet here:
	addSnap(snap("test", "old", "2020-01-01 05:00:00"))
	doRun("2020-01-30 10:20:00", []string{
		snap("ignored", "old", "2020-01-01 01:00:00"),
		snap("ignored", "old", "2020-01-01 01:01:00"),
		snap("test", "a", "2020-01-30 10:10:50"),
		snap("test", "old", "2020-01-01 05:00:00"),
	})
	// But here:
	doRun("2020-01-30 10:30:01", []string{
		snap("ignored", "old", "2020-01-01 01:00:00"),
		snap("ignored", "old", "2020-01-01 01:01:00"),
		snap("test", "a", "2020-01-30 10:10:50"),
		//snap("test", "old", "2020-01-01 05:00:00"),
	})

}
