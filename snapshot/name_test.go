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
			Name("db1", "inst1", "G1", ts),
			NameInfo{
				FullName:        "db1__inst1__20220102-030405-012345678__G1.pb.gz",
				BaseName:        "db1__inst1__20220102-030405-012345678__G1",
				Extension:       "pb.gz",
				Kind:            "snapshot",
				SyncerName:      "db1",
				InstanceID:      "inst1",
				GenerationID:    "G1",
				TimestampString: "20220102-030405-012345678",
				Timestamp:       ts,
			},
			false,
		},
		{
			"extra-fields",
			"db1__inst1__20220102-030405-012345678__G1__X123__Y456.pb.gz",
			NameInfo{
				FullName:        "db1__inst1__20220102-030405-012345678__G1__X123__Y456.pb.gz",
				BaseName:        "db1__inst1__20220102-030405-012345678__G1__X123__Y456",
				Extension:       "pb.gz",
				Kind:            "snapshot",
				SyncerName:      "db1",
				InstanceID:      "inst1",
				GenerationID:    "G1",
				TimestampString: "20220102-030405-012345678",
				Timestamp:       ts,
				Extra: []NameExtraItem{
					"X123",
					"Y456",
				},
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
			"db1__inst1__20220102-030405-012345678__G1__X123__Y456.pb.invalid",
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
			"db1__inst1__20220102-030405-012__G1.pb.gz",
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
