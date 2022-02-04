package receiver

import (
	"bytes"
	"compress/gzip"
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"powerdns.com/platform/lightningstream/config"
	"powerdns.com/platform/lightningstream/snapshot"
	"powerdns.com/platform/lightningstream/storage/memory"
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
	r := New(st, config.Config{}, "test", logrus.New(), "self")
	go func() {
		err := r.Run(ctx)
		if err != nil && err != context.Canceled {
			assert.NoError(t, err)
		}
	}()

	// FIXME: can we get rid of the sleeps?
	time.Sleep(50 * time.Millisecond)

	// No snapshots yet
	inst, s := r.Next()
	assert.Equal(t, "", inst)

	// Add a snapshot
	err := st.Store(ctx, snapshot.Name("test", "other", "G-0", ts), emptySnapshot())
	assert.NoError(t, err)
	time.Sleep(1100 * time.Millisecond)
	inst, s = r.Next()
	assert.Equal(t, "other", inst)

	// Add another snapshot for the same instance
	err = st.Store(ctx, snapshot.Name("test", "other", "G-0", ts.Add(time.Second)), emptySnapshot())
	assert.NoError(t, err)
	time.Sleep(1100 * time.Millisecond)
	inst, s = r.Next()
	assert.Equal(t, "other", inst)

	// No snapshots anymore
	inst, s = r.Next()
	assert.Equal(t, "", inst)

	_ = s

}
