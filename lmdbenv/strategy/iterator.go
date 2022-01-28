package strategy

import "github.com/pkg/errors"

// ErrSkip is an error returned by an Iterator to skip a Merge or Clean for an item.
var ErrSkip = errors.New("skip")

// Iterator is the interface expected by the strategy implementations.
// The Iterator is responsible for iterating of snapshots and deltas, and for
// the proper serialization and merging of values.
// Any progress stats needed must be implemented by the Iterator.
type Iterator interface {
	// Next returns the next LMDB key to insert/update. Calling it invalidates
	// earlier byte slides received from any of these methods.
	// It returns io.EOF when no more entries are available.
	Next() (key []byte, err error)
	// Merge takes the existing LMDB value for the current key, merges it
	// with the value we want to insert, and then returns the result.
	// It must never return an empty slice, and instead return nil.
	Merge(oldval []byte) (val []byte, err error)
	// Clean removes all values that refer to the current snapshot from the LMDB
	// value and returns the result.
	// It must never return an empty slice, and instead return nil.
	Clean(oldval []byte) (val []byte, err error)
}
