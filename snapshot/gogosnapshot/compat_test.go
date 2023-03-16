package gogosnapshot

import (
	"bytes"
	"encoding/binary"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"powerdns.com/platform/lightningstream/snapshot"
)

func Test_roundtrip_compat_generated(t *testing.T) {
	// This will catch any new fields that we forgot to explicitly test for
	popr := rand.New(rand.NewSource(123))
	orig := NewPopulatedSnapshot(popr, false)
	_ = doRoundtripCompatTest(t, *orig)
}

func Test_roundtrip_compat_manual(t *testing.T) {
	orig := Snapshot{
		FormatVersion: 42,
		CompatVersion: 123,
		Meta: Snapshot_Meta{
			GenerationID:  "gen",
			InstanceID:    "inst",
			Hostname:      "host",
			LmdbTxnID:     12345,
			TimestampNano: 21576572351622,
			DatabaseName:  "test-db",
		},
		Databases: []*DBI{
			{
				Name: "test-dbi-1",
				Entries: []KV{
					// Filled below
				},
				Flags:     0,
				Transform: "",
			},
			{
				Name: "test-dbi-2",
				Entries: []KV{
					// Filled below
				},
				Flags:     42,
				Transform: "foo-transform",
			},
		},
	}
	for i := 0; i < 10_000; i++ {
		key := []byte("key-****-test")
		binary.BigEndian.PutUint32(key[4:8], uint32(i))
		val := []byte("val-****-testing-testing")
		binary.BigEndian.PutUint32(val[4:8], uint32(i))
		orig.Databases[0].Entries = append(orig.Databases[0].Entries,
			KV{
				Key:           key,
				Value:         val,
				TimestampNano: 0,
				Flags:         0,
			},
		)
		orig.Databases[1].Entries = append(orig.Databases[1].Entries,
			KV{
				Key:           key,
				Value:         val,
				TimestampNano: 4512761786718612,
				Flags:         11,
			},
		)
	}
	_ = doRoundtripCompatTest(t, orig)
}

func doRoundtripCompatTest(t *testing.T, orig Snapshot) snapshot.Snapshot {
	// Test if our own protobuf code actually produces and reads data
	// produced by a well-tested protobuf implementation.

	// Dump as protobuf
	origProto, err := orig.Marshal()
	assert.NoError(t, err)

	// Load in our implementation
	var ours snapshot.Snapshot
	err = ours.Unmarshal(origProto)
	assert.NoError(t, err)

	// Some checks on our version, just in case the roundtrip is
	// correct, but we do not interpret it correctly.
	assert.Equal(t, orig.CompatVersion, ours.CompatVersion)
	assert.Equal(t, orig.Meta.Hostname, ours.Meta.Hostname)
	for k := 0; k < len(orig.Databases); k++ {
		assert.Equal(t, orig.Databases[k].Name, ours.Databases[k].Name())
		assert.Equal(t, orig.Databases[k].Flags, ours.Databases[k].Flags())
		assert.Equal(t, orig.Databases[k].Transform, ours.Databases[k].Transform())
		for i := 0; i < len(orig.Databases[k].Entries); i++ {
			kv, err := ours.Databases[k].Next()
			assert.NoError(t, err)
			assert.Equal(t, orig.Databases[k].Entries[i].Key, kv.Key)
			assert.Equal(t, orig.Databases[k].Entries[i].Value, kv.Value)
			assert.Equal(t, orig.Databases[k].Entries[i].Flags, kv.Flags)
			assert.Equal(t, orig.Databases[k].Entries[i].TimestampNano, kv.TimestampNano)
		}
	}

	// Dump as protobuf from ours
	buf := bytes.NewBuffer(nil)
	_, err = ours.WriteTo(buf)
	assert.NoError(t, err)
	ourProto := buf.Bytes()

	// Load in this implementation again
	var result Snapshot
	err = result.Unmarshal(ourProto)
	assert.NoError(t, err)

	// To trigger a test failure:
	//result.CompatVersion = 999
	//result.Databases[0].Entries[1].Value = []byte("intentional-failure")

	// Check if they are the same
	assert.Equal(t, orig, result)
	if err := result.VerboseEqual(orig); err != nil {
		t.Errorf("snapshots are not equal: %v", err)
	}

	return ours
}
