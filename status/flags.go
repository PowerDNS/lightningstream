package status

import (
	"fmt"
	"strings"

	"github.com/PowerDNS/lmdb-go/lmdb"
)

func displayFlags(fl uint) string {
	var names []string
	for _, fd := range flagNames {
		if fl&fd.flag > 0 {
			names = append(names, fd.name)
		}
	}
	unknown := fl &^ knownFlags
	if unknown > 0 {
		names = append(names, fmt.Sprintf("%02x", unknown))
	}
	return strings.Join(names, ",")
}

const (
	LMDBIntegerKey uint = 0x08
	LMDBIntegerDup uint = 0x20
)

var flagNames = []struct {
	name string
	flag uint
}{
	{"REVERSEKEY", lmdb.ReverseKey},
	{"DUPSORT", lmdb.DupSort},
	{"DUPFIXED", lmdb.DupFixed},
	{"REVERSEDUP", lmdb.ReverseDup},
	// Not usable in Go bindings
	{"INTEGERKEY", LMDBIntegerKey},
	{"INTEGERDUP", LMDBIntegerDup},
}

var knownFlags uint = lmdb.ReverseKey | lmdb.DupSort | lmdb.DupFixed | lmdb.ReverseDup | LMDBIntegerKey | LMDBIntegerDup
