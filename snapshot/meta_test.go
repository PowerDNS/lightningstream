package snapshot

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func makeTestMeta() Meta {
	const ts = 1678946171_001_002_003
	return Meta{
		GenerationID:  "gen",
		InstanceID:    "inst",
		Hostname:      "host",
		LmdbTxnID:     123,
		TimestampNano: ts,
		DatabaseName:  "db",
		FromLmdbTxnID: 42,
	}
}

func TestMeta(t *testing.T) {
	var empty Meta
	assert.Equal(t, 0, len(empty.Marshal()))

	orig := makeTestMeta()
	pb := orig.Marshal()

	// Now load from PB and compare
	var loaded Meta
	err := loaded.Unmarshal(pb)
	assert.NoError(t, err)
	assert.Equal(t, orig, loaded)
}
