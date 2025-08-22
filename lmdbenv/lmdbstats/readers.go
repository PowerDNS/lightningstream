package lmdbstats

import (
	"log/slog"
	"regexp"
	"strconv"
	"strings"

	"github.com/PowerDNS/lmdb-go/lmdb"
)

type ReaderInfo struct {
	PID    int64
	Thread string // currently hex formatted, but unimportant
	TxnID  int64
}

type ReaderInfoList []ReaderInfo

func (ril ReaderInfoList) OldestReader() int64 {
	var oldest int64 = -1
	for _, ri := range ril {
		if ri.TxnID <= 0 {
			continue
		}
		if oldest >= 0 && oldest < ri.TxnID {
			continue
		}
		oldest = ri.TxnID
	}
	return oldest
}

func (ril ReaderInfoList) OldestReaderOrCurrent(lastTxnID int64) int64 {
	txnID := ril.OldestReader()
	if txnID < 0 {
		return lastTxnID
	}
	return txnID
}

// MaxAge returns the age of the longest living reader compared to the latest
// transaction ID as the difference in IDs, or 0 if there is no reader.
func (ril ReaderInfoList) MaxAge(lastTxnID int64) int64 {
	if len(ril) == 0 {
		return 0
	}
	return lastTxnID - ril.OldestReader()
}

var reSplitReaderList = regexp.MustCompile(" +")

// ParsedReaderList returns a slice of structs describing the open readers.
// It parses the text table mdb_reader_list returns to get this information.
// If parsing fails in case the format ever changes in an incompatible way, it will
// either not return the entries, or return unset fields.
func ParsedReaderList(env *lmdb.Env) (readers ReaderInfoList, err error) {
	first := true
	err = env.ReaderList(func(s string) error {
		slog.Debug("Reader list:", "list", s)
		if first {
			// Skip header ("    pid     thread     txnid\n")
			first = false
			return nil
		}
		s = strings.Trim(s, " \n\t")
		parts := reSplitReaderList.Split(s, -1)
		if len(parts) < 2 {
			return nil
		}
		var ri ReaderInfo
		ri.PID, _ = strconv.ParseInt(parts[0], 10, 64)
		ri.Thread = parts[1]
		if len(parts) >= 3 {
			ri.TxnID, _ = strconv.ParseInt(parts[2], 10, 64)
		}
		readers = append(readers, ri)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return readers, nil
}
