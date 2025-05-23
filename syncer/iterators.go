package syncer

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/PowerDNS/lightningstream/lmdbenv/header"
	"github.com/PowerDNS/lightningstream/snapshot"
	"github.com/sirupsen/logrus"
)

func NewNativeIterator(
	formatVersion uint32,
	compatVersion uint32,
	dbiMsg *snapshot.DBI,
	defaultTS header.Timestamp,
	txnID header.TxnID,
	deletedCutoff header.Timestamp,
) (*NativeIterator, error) {
	if formatVersion == 0 {
		return nil, errors.New("no snapshot formatVersion provided, or 0")
	}
	// Check if the snapshot tells us this snapshot is compatible
	if compatVersion > snapshot.CurrentFormatVersion {
		return nil, fmt.Errorf("snapshot compatVersion too new for this version (%d > %d, formatVersion %d)",
			compatVersion, snapshot.CurrentFormatVersion, formatVersion)
	}
	// Check if we still support an older snapshot type. We do try to forever
	// support old version as long as there is no very strong reason not to.
	if formatVersion < snapshot.CompatFormatVersion {
		return nil, fmt.Errorf("snapshot formatVersion no longer supported (%d < %d)",
			formatVersion, snapshot.CompatFormatVersion)
	}
	if txnID == 0 {
		return nil, ErrNoTxnID
	}
	return &NativeIterator{
		DBIMsg:               dbiMsg,
		DefaultTimestampNano: defaultTS,
		TxnID:                txnID,
		FormatVersion:        formatVersion,
		HeaderPaddingBlock:   false,
		DeletedCutoff:        deletedCutoff,
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
	DBIMsg               *snapshot.DBI    // DBI contents as raw values without header
	DefaultTimestampNano header.Timestamp // Timestamp to add to entries that do not have one
	TxnID                header.TxnID     // Current write TxnID (required)
	FormatVersion        uint32           // Snapshot FormatVersion
	HeaderPaddingBlock   bool             // Extra padding block for testing
	DeletedCutoff        header.Timestamp // Older deleted entries are considered stale

	current int
	started bool
	buf     []byte
	curKV   snapshot.KV
}

func (it *NativeIterator) Next() (key []byte, err error) {
	if it.started {
		it.current++
	} else {
		it.started = true
		it.DBIMsg.ResetCursor()
	}
	kv, err := it.DBIMsg.Next()
	if err != nil {
		return nil, err // can be io.EOF
	}
	it.curKV = kv
	return kv.Key, nil
}

// Merge compares the old LMDB value currently stored and the current iterator
// value from the dump, and decides which value the LMDB should take.
// The LMDB entries are always prefixed with a header.
func (it *NativeIterator) Merge(oldval []byte) (val []byte, err error) {
	entry := it.curKV
	entryVal := entry.Value
	//logrus.Debug("key = %s | old = %s | new = %s",
	//	string(entry.Key), string(oldval), string(entryVal))
	if len(oldval) == 0 {
		// Not in destination db

		// Sweeper: check if it is a stale deletion record to not re-add a
		// record that may just have been swept.
		entryFlags := entry.MaskedFlags()
		if entryFlags.IsDeleted() && header.Timestamp(entry.TimestampNano) < it.DeletedCutoff {
			// Remove (effectively 'do not add', because it does not exist)
			return nil, nil
		}

		// Add with header
		return it.addHeader(
			entryVal,
			header.Timestamp(entry.TimestampNano),
			entry.MaskedFlags(),
			false)
	}

	h, appVal, err := header.Parse(oldval)
	if err != nil {
		// Should never happen
		it.logDebugValue(oldval)
		return nil, fmt.Errorf("merge: oldval header parse error (%v = %v): %v",
			entry.Key, oldval, err)
	}
	oldTS := h.Timestamp
	newTS := header.Timestamp(entry.TimestampNano)
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
	entry := it.curKV
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
func (it *NativeIterator) addHeader(
	entryVal []byte,
	ts header.Timestamp,
	flags header.Flags,
	fromClean bool,
) (val []byte, err error) {
	// The minimum size is sufficient as long as we do not add extensions here
	if cap(it.buf) < header.MinHeaderSize {
		it.buf = make([]byte, header.MinHeaderSize, 1024)
	} else {
		it.buf = it.buf[:header.MinHeaderSize]
	}
	if ts == 0 {
		ts = it.DefaultTimestampNano
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
	if it.HeaderPaddingBlock {
		// Add an extra all-zero padding block to test application handling
		it.buf[header.NumExtraOffsetLow] = 1
		it.buf = append(it.buf, 0, 0, 0, 0, 0, 0, 0, 0)
	}
	it.buf = append(it.buf, entryVal...)
	val = it.buf
	return val, nil
}

// PlainIterator iterates over a snapshot of a shadow database for
// insertion into the main database without the timestamp header.
type PlainIterator struct {
	DBIMsg *snapshot.DBI // LMDB contents (timestamp is ignored)

	current int
	started bool
	curKV   snapshot.KV
}

func (it *PlainIterator) Next() (key []byte, err error) {
	if it.started {
		it.current++
	} else {
		it.started = true
		it.DBIMsg.ResetCursor()
	}
	kv, err := it.DBIMsg.Next()
	if err != nil {
		return nil, err // can be io.EOF
	}
	it.curKV = kv
	return kv.Key, nil
}

func (it *PlainIterator) Merge(oldval []byte) (val []byte, err error) {
	mainVal := it.curKV.Value
	if len(mainVal) == 0 {
		// Signal that we want deletion in case the strategy distinguishes
		// between nil and an empty value
		// FIXME: do we still want to do this with the new FlagDeleted?
		mainVal = nil
	}
	return mainVal, nil
}

func (it *PlainIterator) Clean(oldval []byte) (val []byte, err error) {
	return nil, nil // Delete the key
}
