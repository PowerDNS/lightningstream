package receiver

import (
	"bytes"
	"compress/gzip"
	"context"
	"testing"
	"time"

	"github.com/PowerDNS/simpleblob/backends/memory"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/PowerDNS/lightningstream/config"
	"github.com/PowerDNS/lightningstream/snapshot"
)

func emptySnapshot() []byte {
	buf := bytes.NewBuffer(nil)
	g := gzip.NewWriter(buf)
	_ = g.Close()
	return buf.Bytes()
}

func TestReceiver(t *testing.T) {
	ts := time.Now()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	st := memory.New()
	r := New(st, config.Config{
		// Poll very often for fast tests
		StoragePollInterval: 10 * time.Millisecond,
		// If this is exceeded, the downloader will wait a second before
		// checking again, so this needs to be high enough.
		MemoryDownloadedSnapshots:   2,
		MemoryDecompressedSnapshots: 2,
	}, "test", logrus.New(), "self")
	go func() {
		err := r.Run(ctx)
		if err != nil && err != context.Canceled {
			assert.NoError(t, err)
		}
	}()

	// Initial sleep to check that we do not have one without adding one
	time.Sleep(100 * time.Millisecond)

	// No snapshots yet
	inst, _ := r.Next()
	assert.Equal(t, "", inst)

	// Add a snapshot
	err := st.Store(ctx, snapshot.Name("test", "other", "G-0", ts), emptySnapshot())
	assert.NoError(t, err)
	for i := 0; i < 50; i++ {
		time.Sleep(20 * time.Millisecond)
		inst, _ = r.Next()
		if inst != "" {
			break
		}
	}
	assert.Equal(t, "other", inst)

	// No snapshots anymore
	inst, _ = r.Next()
	assert.Equal(t, "", inst)

	// Add another snapshot for the same instance
	err = st.Store(ctx, snapshot.Name("test", "other", "G-0", ts.Add(time.Second)), emptySnapshot())
	assert.NoError(t, err)
	for i := 0; i < 50; i++ {
		time.Sleep(20 * time.Millisecond)
		inst, _ = r.Next()
		if inst != "" {
			break
		}
	}
	assert.Equal(t, "other", inst)

	// No snapshots anymore
	inst, _ = r.Next()
	assert.Equal(t, "", inst)
}
