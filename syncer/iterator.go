package syncer

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"powerdns.com/platform/lightningstream/snapshot"
)

// HeaderSize is the size of the timestamp header for each LMDB value in bytes
const HeaderSize = 8

// AddTimestampIterator iterates over a snapshot of the main database for
// insertion into a shadow database with extra timestamp.
type AddTimestampIterator struct {
	Entries []snapshot.KV // LMDB contents without timestamp to merge
	TSNano  uint64        // Timestamp to add to entries that do not have one

	current int
	started bool
	buf     []byte
}

func (it *AddTimestampIterator) Next() (key []byte, err error) {
	if it.started {
		it.current++
	} else {
		it.started = true
	}
	if len(it.Entries) <= it.current {
		return nil, io.EOF
	}
	key = it.Entries[it.current].Key
	return key, nil
}

func (it *AddTimestampIterator) Merge(oldval []byte) (val []byte, err error) {
	mainVal := it.Entries[it.current].Value
	if len(oldval) == 0 {
		// Not in destination db, add with timestamp
		return it.addTS(mainVal)
	}
	if len(oldval) < HeaderSize {
		// Should never happen
		return nil, fmt.Errorf("marge: oldval in db too short: %v", oldval)
	}
	actualOldVal := oldval[HeaderSize:]
	if bytes.Equal(mainVal, actualOldVal) {
		// No change, so no timestamp change
		return oldval, nil
	}
	// Change data, so we need a fresh timestamp
	return it.addTS(mainVal)
}

func (it *AddTimestampIterator) addTS(oldval []byte) (val []byte, err error) {
	if cap(it.buf) < HeaderSize {
		it.buf = make([]byte, HeaderSize, 1024)
	} else {
		it.buf = it.buf[:HeaderSize]
	}
	binary.BigEndian.PutUint64(it.buf, it.TSNano)
	it.buf = append(it.buf, oldval...)
	val = it.buf
	return val, nil
}

func (it *AddTimestampIterator) Clean(oldval []byte) (val []byte, err error) {
	if len(oldval) == HeaderSize {
		return oldval, nil // already deleted, only timestamp
	}
	return it.addTS(nil)
}
