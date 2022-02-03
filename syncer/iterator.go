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

// MainToShadowIterator iterates over a snapshot of the main database for
// insertion into a shadow database with extra timestamp.
type MainToShadowIterator struct {
	DBIMsg  *snapshot.DBI
	TSNano  uint64
	current int
	started bool
	buf     []byte
}

func (it *MainToShadowIterator) Next() (key []byte, err error) {
	if it.started {
		it.current++
	} else {
		it.started = true
	}
	if len(it.DBIMsg.Entries) <= it.current {
		return nil, io.EOF
	}
	key = it.DBIMsg.Entries[it.current].Key
	return key, nil
}

func (it *MainToShadowIterator) Merge(oldval []byte) (val []byte, err error) {
	mainVal := it.DBIMsg.Entries[it.current].Value
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

func (it *MainToShadowIterator) addTS(oldval []byte) (val []byte, err error) {
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

func (it *MainToShadowIterator) Clean(oldval []byte) (val []byte, err error) {
	if len(oldval) == HeaderSize {
		return oldval, nil // already deleted, only timestamp
	}
	return it.addTS(nil)
}
