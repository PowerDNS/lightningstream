package syncer

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	"github.com/sirupsen/logrus"
	"powerdns.com/platform/lightningstream/lmdbenv/header"
	"powerdns.com/platform/lightningstream/snapshot"
)

func NewNativeIterator(formatVersion uint32, entries []snapshot.KV, defaultTS, txnID uint64) (*NativeIterator, error) {
	if formatVersion == 0 {
		return nil, errors.New("no snapshot formatVersion provided, or 0")
	}
	if formatVersion < snapshot.CompatFormatVersion {
		return nil, fmt.Errorf("snapshot formatVersion no longer supported (%d < %d)",
			formatVersion, snapshot.CompatFormatVersion)
	}
	if formatVersion > snapshot.CurrentFormatVersion {
		return nil, fmt.Errorf("snapshot formatVersion too new for this version (%d > %d)",
			formatVersion, snapshot.CurrentFormatVersion)
	}
	if txnID == 0 {
		return nil, ErrNoTxnID
	}
	return &NativeIterator{
		Entries:              entries,
		DefaultTimestampNano: defaultTS,
		TxnID:                txnID,
		FormatVersion:        formatVersion,
	}, nil
}

// NativeIterator iterates over a snapshot DBI and updates the LMDB with
// values that are prefixed with a native header.
// This iterator has two uses:
// * Merge the main database into a shadow database with a default timestamp.
// * Merge a remote snapshot with the timestamp values into a DBI with headers.
// The LMDB values the iterator operates on MUST always have a header. If no
// header is present, an error is returned.
type NativeIterator struct {
	Entries              []snapshot.KV // LMDB contents as raw values without header
	DefaultTimestampNano uint64        // Timestamp to add to entries that do not have one
	TxnID                uint64        // Current write TxnID (required)
	FormatVersion        uint32        // Snapshot FormatVersion

	current int
	started bool
	buf     []byte
}

func (it *NativeIterator) Next() (key []byte, err error) {
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

// Merge compares the old LMDB value currently stored and the current iterator
// value from the dump, and decides which value the LMDB should take.
// The LMDB entries are always prefixed with a header.
func (it *NativeIterator) Merge(oldval []byte) (val []byte, err error) {
	entry := it.Entries[it.current]
	entryVal := entry.Value
	//logrus.Debug("key = %s | old = %s | new = %s",
	//	string(entry.Key), string(oldval), string(entryVal))
	if len(oldval) == 0 {
		// Not in destination db, add with header
		return it.addHeader(entryVal, entry.TimestampNano, entry.MaskedFlags(), false)
	}
	h, appVal, err := header.Parse(oldval)
	if err != nil {
		// Should never happen
		it.logDebugValue(oldval)
		return nil, fmt.Errorf("merge: oldval header parse error (%v = %v): %v",
			entry.Key, oldval, err)
	}
	oldTS := uint64(h.Timestamp.UnixNano()) // TODO: inefficient double conversion
	newTS := entry.TimestampNano
	actualOldVal := appVal
	if newTS == 0 {
		// Special handling for main to shadow copy that uses a default timestamp
		if bytes.Equal(actualOldVal, entryVal) {
			return oldval, nil // do not update timestamp
		}
		newTS = it.DefaultTimestampNano
	}
	if newTS < oldTS {
		// Current LMDB value has a higher timestamp, so keep that one
		return oldval, nil
	}
	if newTS == oldTS && bytes.Compare(actualOldVal, entryVal) <= 0 {
		// Same timestamp, lexicographic lower app value wins for deterministic values,
		// so return the old value if the plain value was lower or equal.
		return oldval, nil
	}
	// Update LMDB value
	return it.addHeader(entryVal, newTS, entry.MaskedFlags(), false)
}

func (it *NativeIterator) Clean(oldval []byte) (val []byte, err error) {
	// Clean effectively instructs us to delete the entry
	h, _, err := header.Parse(oldval)
	if err != nil {
		return nil, err
	}
	if h.Flags.IsDeleted() {
		return oldval, nil // already deleted
	}
	return it.addHeader(nil, 0, header.FlagDeleted, true)
}

func (it *NativeIterator) logDebugValue(val []byte) {
	entry := it.Entries[it.current]
	logrus.WithFields(logrus.Fields{
		"key": hex.Dump(entry.Key),
		"val": hex.Dump(val),
	}).Debug("LMDB value dump")
}

// addHeader prepends a header to a plain value. It uses the ts parameter
// passed in if non-zero, or the default one set on the iterator.
// A timestamp is mandatory. If both are 0, an error is returned.
// entryVal is the plain application value.
// The TxnID is also mandatory.
// fromClean indicates if this was called from Clean
func (it *NativeIterator) addHeader(entryVal []byte, ts uint64, flags header.Flags, fromClean bool) (val []byte, err error) {
	// The minimum size is sufficient as long as we do not add extensions here
	if cap(it.buf) < header.MinHeaderSize {
		it.buf = make([]byte, header.MinHeaderSize, 1024)
	} else {
		it.buf = it.buf[:header.MinHeaderSize]
	}
	if ts == 0 {
		ts = it.DefaultTimestampNano
		if ts == 0 {
			// When we write an entry, it MUST have a valid timestamp.
			// Only applications are allowed to use 0 when they migrate old
			// data to the native schema, but there is no reason for us
			// to ever do that.
			if fromClean {
				return nil, ErrNoTimestamp // no extra info here
			} else {
				key := it.Entries[it.current].Key
				return nil, ErrEntry{
					Key: key,
					Err: ErrNoTimestamp,
				}
			}
		}
	}
	if len(entryVal) == 0 && it.FormatVersion < 2 {
		// Earlier snapshots did not have a deleted flag and indicated deleted
		// entries with an empty application value.
		flags |= header.FlagDeleted
	}
	if flags.IsDeleted() {
		entryVal = nil
	}
	header.PutBasic(it.buf, ts, it.TxnID, flags)
	it.buf = append(it.buf, entryVal...)
	val = it.buf
	return val, nil
}

// PlainIterator iterates over a snapshot of a shadow database for
// insertion into the main database without the timestamp header.
type PlainIterator struct {
	Entries []snapshot.KV // LMDB contents (timestamp is ignored)

	current int
	started bool
}

func (it *PlainIterator) Next() (key []byte, err error) {
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

func (it *PlainIterator) Merge(oldval []byte) (val []byte, err error) {
	mainVal := it.Entries[it.current].Value
	if len(mainVal) == 0 {
		// Signal that we want deletion in case the strategy distinguishes
		// between nil and an empty value
		mainVal = nil
	}
	return mainVal, nil
}

func (it *PlainIterator) Clean(oldval []byte) (val []byte, err error) {
	return nil, nil // Delete the key
}
