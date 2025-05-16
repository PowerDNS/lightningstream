package syncer

import (
	"testing"

	"github.com/PowerDNS/lightningstream/lmdbenv/header"
	"github.com/PowerDNS/lightningstream/snapshot"
	"github.com/stretchr/testify/assert"
)

func makeVal(ts uint64, flags header.Flags, appVal string) []byte {
	buf := make([]byte, header.MinHeaderSize+len(appVal))
	header.PutBasic(buf, header.Timestamp(ts), 123, flags)
	copy(buf[header.MinHeaderSize:], appVal)
	return buf
}

func TestNativeIterator_Merge(t *testing.T) {
	tt := []struct {
		Name          string
		KV            snapshot.KV
		OldVal        []byte
		Expected      []byte
		ExpectedError bool
	}{
		{
			Name: "add-new-entry",
			KV: snapshot.KV{
				Value:         []byte("val"),
				TimestampNano: 30,
				Flags:         0,
			},
			OldVal:   nil,
			Expected: makeVal(30, 0, "val"),
		},
		{
			Name: "add-new-entry-that-is-old",
			KV: snapshot.KV{
				Value:         []byte("val"),
				TimestampNano: 5, // before stale cutoff, but not marked deleted
				Flags:         0,
			},
			OldVal:   nil,
			Expected: makeVal(5, 0, "val"),
		},
		{
			Name: "add-new-entry-default-ts",
			KV: snapshot.KV{
				Value:         []byte("val"),
				TimestampNano: 0,
				Flags:         0,
			},
			OldVal:   nil,
			Expected: makeVal(42, 0, "val"),
		},
		{
			Name: "db-retains-newer-entry",
			KV: snapshot.KV{
				Value:         []byte("val"),
				TimestampNano: 30,
				Flags:         0,
			},
			OldVal:   makeVal(40, 0, "newer"),
			Expected: makeVal(40, 0, "newer"),
		},
		{
			Name: "add-deleted-entry",
			KV: snapshot.KV{
				Value:         nil,
				TimestampNano: 30,
				Flags:         uint32(header.FlagDeleted),
			},
			OldVal:   nil,
			Expected: makeVal(30, header.FlagDeleted, ""),
		},
		{
			Name: "skip-stale-deleted-entry",
			KV: snapshot.KV{
				Value:         nil,
				TimestampNano: 5, // stale
				Flags:         uint32(header.FlagDeleted),
			},
			OldVal:   nil,
			Expected: nil,
		},
		{
			Name: "conflict-lexicographic-lower",
			KV: snapshot.KV{
				Value:         []byte("aaa"),
				TimestampNano: 30,
				Flags:         0,
			},
			OldVal:   makeVal(30, 0, "bbb"),
			Expected: makeVal(30, 0, "aaa"),
		},
		{
			Name: "conflict-lexicographic-higher",
			KV: snapshot.KV{
				Value:         []byte("ccc"),
				TimestampNano: 30,
				Flags:         0,
			},
			OldVal:   makeVal(30, 0, "bbb"),
			Expected: makeVal(30, 0, "bbb"),
		},
		{
			Name: "same",
			KV: snapshot.KV{
				Value:         []byte("val"),
				TimestampNano: 30,
				Flags:         0,
			},
			OldVal:   makeVal(30, 0, "val"),
			Expected: makeVal(30, 0, "val"),
		},
		{
			Name: "corrupt-old-value",
			KV: snapshot.KV{
				Value:         []byte("val"),
				TimestampNano: 30,
				Flags:         0,
			},
			OldVal:        []byte("corrupt"),
			ExpectedError: true,
		},
	}

	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			it := &NativeIterator{
				DefaultTimestampNano: 42,  // used then the timestamp is zero
				TxnID:                123, // current txnid
				FormatVersion:        snapshot.CurrentFormatVersion,
				DeletedCutoff:        10, // delete markers older than this are ignored
			}
			tc.KV.Key = []byte("key")
			it.curKV = tc.KV
			res, err := it.Merge(tc.OldVal)
			if tc.ExpectedError {
				if err == nil {
					t.Errorf("expected error, got none")
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
			}
			assert.Equal(t, tc.Expected, res)
		})
	}

}
