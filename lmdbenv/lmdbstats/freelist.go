package lmdbstats

import (
	"encoding/binary"
	"fmt"

	"github.com/PowerDNS/lmdb-go/lmdb"
	"github.com/PowerDNS/lmdb-go/lmdbscan"
	"github.com/c2h5oh/datasize"
	"github.com/pkg/errors"
)

// FreeListDBI is the internal DBI containing a list of free pages
// that can be reused during a next write-transaction if no older
// readers still access that version.
const FreeListDBI lmdb.DBI = 0

// FreeListInfo summarizes the pages in the LMDB freelist
type FreeListInfo struct {
	NumTransactions int               // Number of entries (transactions) in the freelist DBI
	NumPages        int               // Total number of pages in freelist for all transactions
	PagesBytes      datasize.ByteSize // Total number of bytes in these pages

	OldestTxnId       uint64 // Oldest TxnID in freelist
	OldestReaderTxnId uint64 // Oldest reader TxnID in use, or last if no readers

	// Stats that distinguish between old and new transactions according to the
	// current reader list.
	UsablePages int               // No longer in use
	UsableBytes datasize.ByteSize // No longer in use
	LockedPages int               // Locked by readers
	LockedBytes datasize.ByteSize // Locked by readers

	// For reference, stats on pages that have not been allocated yet at the end
	// of the file.
	UnallocatedPages int
	UnallocatedBytes datasize.ByteSize
}

func OldestReader(env *lmdb.Env) (uint64, error) {
	err := env.ReaderList(func(s string) error {
		return nil
	})
	return 0, err
}

// GetFreeListInfo returns details about the internal LMDB freelist
func GetFreeListInfo(env *lmdb.Env, txn *lmdb.Txn) (*FreeListInfo, error) {
	info, err := env.Info()
	if err != nil {
		return nil, err
	}

	stat, err := txn.Stat(FreeListDBI)
	if err != nil {
		return nil, err
	}
	pageSize := uint64(stat.PSize)
	unallocatedPages := uint64(info.MapSize)/pageSize - uint64(info.LastPNO)

	readers, err := ParsedReaderList(env)
	if err != nil {
		return nil, errors.Wrap(err, "env reader list")
	}
	oldestReader := uint64(readers.OldestReaderOrCurrent(info.LastTxnID))

	details := new(FreeListInfo)
	details.OldestReaderTxnId = oldestReader
	details.UnallocatedPages = int(unallocatedPages)
	details.UnallocatedBytes = datasize.ByteSize(unallocatedPages * pageSize)

	sc := lmdbscan.New(txn, FreeListDBI)
	defer sc.Close()

	sc.SetNext(nil, nil, lmdb.First, lmdb.Next)
	for sc.Scan() {
		// Every entry represents the free pages released in a single transaction
		details.NumTransactions++
		if len(sc.Key()) != 8 {
			return nil, fmt.Errorf("expected 8 byte (64 bit) keys, got %d bytes", len(sc.Key()))
		}
		// The transaction that released the pages
		txnID := binary.NativeEndian.Uint64(sc.Key())
		if len(sc.Val()) < 8 {
			return nil, fmt.Errorf("freelist value too short, got %d bytes", len(sc.Val()))
		}
		if details.OldestTxnId == 0 || txnID < details.OldestTxnId {
			details.OldestTxnId = txnID
		}

		// The value is a list of uint64 values.
		// The first number is the number of pages in the entry, followed by
		// a sorted list of all the freed page numbers.
		// TODO: We could collect detailed information about those pages, but for
		//       now we only care about the size per txnID.
		a := uint64Array(sc.Val())
		nPages := a.Get(0)
		details.NumPages += int(nPages)
		if txnID < oldestReader {
			details.UsablePages += int(nPages)
			details.UsableBytes += datasize.ByteSize(nPages * pageSize)
		} else {
			details.LockedPages += int(nPages)
			details.LockedBytes += datasize.ByteSize(nPages * pageSize)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}

	details.PagesBytes = datasize.ByteSize(pageSize * uint64(details.NumPages))
	return details, nil
}

type uint64Array []byte

func (a uint64Array) Entries() int {
	return len(a) / 8
}

func (a uint64Array) Get(i int) uint64 {
	offset := i * 8
	return binary.NativeEndian.Uint64(a[offset : offset+8])
}
