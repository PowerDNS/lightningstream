package dbiflags

import (
	"testing"

	"github.com/PowerDNS/lmdb-go/lmdb"
)

func TestFlags_String(t *testing.T) {
	tests := []struct {
		name string
		f    Flags
		want string
	}{
		{"none", 0, ""},
		{"MDB_DUPSORT", lmdb.DupSort, "MDB_DUPSORT"},
		{"MDB_DUPFIXED", lmdb.DupFixed, "MDB_DUPFIXED"},
		{"MDB_REVERSEKEY", lmdb.ReverseKey, "MDB_REVERSEKEY"},
		{"MDB_REVERSEDUP", lmdb.ReverseDup, "MDB_REVERSEDUP"},
		{"MDB_INTEGERKEY", 0x08, "MDB_INTEGERKEY"},
		{"MDB_INTEGERDUP", 0x20, "MDB_INTEGERDUP"},
		{"multi", 0x28, "MDB_INTEGERKEY|MDB_INTEGERDUP"},
		{"multi-unknown", 0xFFFF, "MDB_REVERSEKEY|MDB_DUPSORT|MDB_INTEGERKEY|" +
			"MDB_DUPFIXED|MDB_INTEGERDUP|MDB_REVERSEDUP|UNKNOWN:0xff81"},
		{"single-unknown", 0x1000, "UNKNOWN:0x1000"},
		{"MDB_CREATE-ignored-overflows", Flags(lmdb.Create & 0xFFFF), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.f.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFlags_UnmarshalText(t *testing.T) {
	tests := []struct {
		input   string
		f       Flags
		wantErr bool
	}{
		{"", 0, false},
		{"MDB_INTEGERKEY", 0x08, false},
		{"MDB_INTEGERKEY|MDB_INTEGERDUP", 0x28, false},
		{"MDB_INTEGERDUP|MDB_INTEGERKEY", 0x28, false},
		{"MDB_INTEGERKEY|32", 0x28, false},
		{"MDB_INTEGERDUP|0x20", 0x28, false},
		{"0x28", 0x28, false},
		{"2", 0x02, false},
		// Relaxed parsing
		{"INTEGERKEY", 0x08, false},
		{"IntegerKey", 0x08, false},
		{"iNtEgErkEY", 0x08, false},
		{" ", 0, false},
		{"MDB_INTEGERKEY,MDB_INTEGERDUP", 0x28, false},
		{"MDB_INTEGERKEY+MDB_INTEGERDUP", 0x28, false},
		{"MDB_INTEGERKEY MDB_INTEGERDUP", 0x28, false},
		{" MDB_INTEGERKEY\n   |  \tMDB_INTEGERDUP ", 0x28, false},
		// Errors
		{"1", 0, true},      // unknown flag
		{"0x1000", 0, true}, // unknown flag
		{"-1", 0, true},
		{"foo", 0, true},
		{"MDB_CREATE", 0, true},
		{"0x40000", 0, true}, // MDB_CREATE as number
	}
	for _, tt := range tests {
		t.Run("input-"+tt.input, func(t *testing.T) {
			if err := tt.f.UnmarshalText([]byte(tt.input)); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalText() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
