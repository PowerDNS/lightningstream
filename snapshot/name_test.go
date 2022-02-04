package snapshot

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseName(t *testing.T) {
	ts := time.Date(2022, 1, 2, 3, 4, 5, 12345678, time.UTC)
	tests := []struct {
		testName string
		name     string
		want     NameInfo
		wantErr  bool
	}{
		{
			"roundtrip",
			Name("db1", "inst1", "gen1", ts),
			NameInfo{
				FullName:        "db1__inst1__20220102-030405-012345678__gen1.pb.gz",
				Extension:       "pb.gz",
				SyncerName:      "db1",
				InstanceID:      "inst1",
				GenerationID:    "gen1",
				TimestampString: "20220102-030405-012345678",
				Timestamp:       ts,
			},
			false,
		},
		{
			"extra-fields",
			"db1__inst1__20220102-030405-012345678__gen1__extra__extra.pb.gz",
			NameInfo{
				FullName:        "db1__inst1__20220102-030405-012345678__gen1__extra__extra.pb.gz",
				Extension:       "pb.gz",
				SyncerName:      "db1",
				InstanceID:      "inst1",
				GenerationID:    "gen1",
				TimestampString: "20220102-030405-012345678",
				Timestamp:       ts,
			},
			false,
		},
		{
			"invalid",
			"invalid",
			NameInfo{},
			true,
		},
		{
			"invalid-ext",
			"db1__inst1__20220102-030405-012345678__gen1__extra__extra.pb.bz2",
			NameInfo{},
			true,
		},
		{
			"too-few-fields",
			"db1__inst1__20220102-030405-012345678.pb.gz",
			NameInfo{},
			true,
		},
		{
			"invalid-ts",
			"db1__inst1__20220102-030405-012__gen1.pb.gz",
			NameInfo{},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			got, err := ParseName(tt.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
