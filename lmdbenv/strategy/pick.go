// Package strategy implements various LMDB data insert strategies.
//
// A strategy is responsible for updating the LMDB with a new snapshot or
// delta. This package contains implementations of insert strategies and
// a function to pick the best strategy for a given situation.
package strategy

import (
	"github.com/PowerDNS/lmdb-go/lmdb"
	"github.com/pkg/errors"
)

// ErrNotSorted is returned when keys are found to be not sorted and the
// strategy requires it.
var ErrNotSorted = errors.New("keys not sorted")

// Func defines the signature of functions implementing a strategy
type Func func(txn *lmdb.Txn, dbi lmdb.DBI, it Iterator) error

// Facts are used by Pick to pick a strategy.
type Facts struct {
	// Is the database currently completely empty?
	IsEmpty bool
}

// Pick picks the best insert strategy based on Facts.
// FIXME: Trivial choice, remove
func Pick(f Facts) (Func, error) {
	if f.IsEmpty {
		return Append, nil
	}
	// Snapshot load after uuid change or broken delta chain
	return IterPut, nil // Should cause minimal increases in file size
	// Unreachable
	//return nil, ErrNoStrategy
}
