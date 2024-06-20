package syncer

import (
	"bytes"
	"testing"

	"github.com/PowerDNS/lightningstream/lmdbenv/header"
	"github.com/PowerDNS/lightningstream/snapshot"
	"github.com/stretchr/testify/assert"
)

func rep(ch byte, n int) []byte {
	return bytes.Repeat([]byte{ch}, n)
}

func concat(slices ...[]byte) []byte {
	var res []byte
	for _, s := range slices {
		res = append(res, s...)
	}
	return res
}

func Test_dupSortHackEncodeOne(t *testing.T) {
	const flagDeleted = uint32(header.FlagDeleted)
	tests := []struct {
		name       string
		orig       snapshot.KV
		wantResult snapshot.KV
		wantErr    bool
	}{
		{
			"empty-key-not-allowed",
			snapshot.KV{Key: []byte(""), Value: []byte("foo")},
			snapshot.KV{},
			true,
		},
		{
			"normal",
			snapshot.KV{Key: []byte("key"), Value: []byte("foo1")},
			snapshot.KV{Key: []byte("key\x00\x00\x00\x00foo1\x03"), Value: []byte("foo1")},
			false,
		},
		{
			"large-value",
			snapshot.KV{Key: []byte("key"), Value: rep('A', 1000)},
			snapshot.KV{
				Key:   concat([]byte{'k', 'e', 'y', 0, 0, 0, 0}, rep('A', 503), []byte{3}),
				Value: rep('A', 1000),
			},
			false,
		},
		{
			"too-large-key",
			snapshot.KV{Key: rep('K', 256), Value: []byte("value")},
			snapshot.KV{},
			true,
		},
		{
			"preserve-deleted-flag",
			snapshot.KV{Key: []byte("key"), Value: []byte(""), Flags: flagDeleted},
			snapshot.KV{Key: []byte("key\x00\x00\x00\x00\x03"), Value: []byte(""), Flags: flagDeleted},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResult, err := dupSortHackEncodeOne(tt.orig)
			if tt.wantErr != (err != nil) {
				t.Errorf("error retval: wanErr %v, got %v", tt.wantErr, err)
			}
			if err != nil {
				return
			}
			assert.Equalf(t, tt.wantResult, gotResult, "dupSortHackEncode(%v)", tt.orig)
			back, err := dupSortHackDecodeOne(gotResult)
			assert.NoError(t, err)
			assert.Equal(t, tt.orig, back, "reverse not same as original")
		})
	}
}
