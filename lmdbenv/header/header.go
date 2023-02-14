package header

import (
	"encoding/binary"
	"time"

	"github.com/pkg/errors"
)

// Timestamp is the number of nanoseconds since the UNIX epoch, which
// is how we write it to the header.
type Timestamp uint64

// Time converts a Timestamp into Time
func (ts Timestamp) Time() time.Time {
	return time.Unix(0, int64(ts))
}

// TimestampFromTime creates a Timestamp from a Time
func TimestampFromTime(t time.Time) Timestamp {
	return Timestamp(t.UnixNano())
}

// TxnID is the LMDB transaction ID.
// The Go lmdb library inconsistently uses int64 and uintptr for this in
// different places.
type TxnID uint64

// Header describes the header of a native schema value
type Header struct {
	Timestamp Timestamp // time of last change
	TxnID     TxnID     // lmdb Go lib uses uint64 and uintptr
	Version   int       // header version (currently always 0)
	Flags     Flags     // header flags
	NumExtra  int       // extra number of 8-byte blocks (set automatically in .Bytes())
	Extra     []byte    // bytes in extra block
}

func (h Header) MarshalBinary() (data []byte, err error) {
	return h.Bytes(), nil
}

func (h Header) Bytes() []byte {
	// fixed allocation size allows stack allocation
	b := make([]byte, MinHeaderSize, PreAllocSize)
	// moved to function to allow inlining this function with allocation
	return h.doBytes(b)
}

func (h Header) doBytes(b []byte) []byte {
	// Allow NumExtra to be too low, increase dynamically
	extra := h.Extra
	n := h.NumExtra
	extraLen := len(extra)
	if n > 0 || extraLen > 0 {
		if extraLen > n*BlockSize {
			n = extraLen / BlockSize
			if extraLen%BlockSize > 0 {
				n++
			}
		}
		if n > 0 {
			newLen := MinHeaderSize + n*BlockSize
			if cap(b) < newLen {
				// Allocate a new slice, old one does not fit
				b = make([]byte, newLen)
			} else {
				// reslice
				b = b[:newLen]
			}
			copy(b[MinHeaderSize:MinHeaderSize+extraLen], extra)
		}
	}

	binary.BigEndian.PutUint64(b[:8], uint64(h.Timestamp))
	binary.BigEndian.PutUint64(b[8:16], uint64(h.TxnID))
	b[VersionOffset] = uint8(h.Version)
	b[FlagsOffset] = uint8(h.Flags)
	binary.BigEndian.PutUint16(b[NumExtraOffsetHigh:NumExtraOffsetHigh+2], uint16(n))
	return b
}

var (
	ErrTooShort = errors.New("value too short to contain a header")
	ErrVersion  = errors.New("unsupported header version or not a header")
)

const (
	// MinHeaderSize is the minimum header size when no extra blocks are present
	MinHeaderSize = 24
	// PreAllocSize is the number of bytes pre-allocated to fit a header
	// with extensions. Larger headers will trigger an additional heap alloc.
	PreAllocSize = 64
	// BlockSize is the number of bytes per extra block
	BlockSize = 8
)

const (
	VersionOffset      = 16
	FlagsOffset        = 17
	NumExtraOffsetHigh = 22 // uint16 high byte
	NumExtraOffsetLow  = 23 // uint16 low byte

	reserved1Offset = 18
	reserved2Offset = 19
	reserved3Offset = 20
	reserved4Offset = 21
)

type Flags uint8

const (
	// FlagDeleted indicates that this entry has been deleted
	FlagDeleted Flags = 1

	// NoFlags can be used when no flags are needed, for readability
	NoFlags Flags = 0

	// FlagSyncMask is the mask of flags allowed to sync to/from a snapshot,
	// others will be cleared.
	FlagSyncMask = FlagDeleted
)

func (f Flags) IsDeleted() bool {
	return f&FlagDeleted > 0
}

func (f Flags) Masked() Flags {
	return f & FlagSyncMask
}

// Parse parses a value with header and returns the remaining application value.
func Parse(val []byte) (header Header, value []byte, err error) {
	if len(val) < MinHeaderSize {
		return header, nil, ErrTooShort
	}

	version := val[VersionOffset]
	if version != 0 {
		return header, nil, ErrVersion
	}

	offset := MinHeaderSize
	numExtra := getNumExtra(val)
	var numExtraBytes int
	var extra []byte
	if numExtra > 0 {
		numExtraBytes = BlockSize * numExtra
		if len(val) < MinHeaderSize+numExtraBytes {
			return header, nil, ErrTooShort
		}
		offset += numExtraBytes
		extra = val[MinHeaderSize:offset]
	}

	header = Header{
		Timestamp: Timestamp(binary.BigEndian.Uint64(val[:8])),
		TxnID:     TxnID(binary.BigEndian.Uint64(val[8:16])),
		Version:   int(val[VersionOffset]),
		Flags:     Flags(val[FlagsOffset]),
		NumExtra:  numExtra,
		Extra:     extra,
	}
	return header, val[offset:], nil
}

// Skip skips over the header and returns the remaining application value.
func Skip(val []byte) (value []byte, err error) {
	if len(val) < MinHeaderSize {
		return nil, ErrTooShort
	}

	version := val[VersionOffset]
	if version != 0 {
		return nil, ErrVersion
	}

	offset := MinHeaderSize
	numExtra := getNumExtra(val)
	if numExtra > 0 {
		numExtraBytes := BlockSize * numExtra
		if len(val) < MinHeaderSize+numExtraBytes {
			return nil, ErrTooShort
		}
		offset += numExtraBytes
	}

	return val[offset:], nil
}

func getNumExtra(val []byte) int {
	return int(binary.BigEndian.Uint16(val[NumExtraOffsetHigh : NumExtraOffsetHigh+2]))
}

// PutBasic creates a basic header in the provided slice. The slice must
// have a length of at least MinHeaderSize.
func PutBasic(b []byte, ts Timestamp, txnid TxnID, flags Flags) {
	b = b[:MinHeaderSize] // Prevents further bounds checks
	binary.BigEndian.PutUint64(b[:8], uint64(ts))
	binary.BigEndian.PutUint64(b[8:16], uint64(txnid))
	b[VersionOffset] = 0
	b[FlagsOffset] = uint8(flags)
	b[reserved1Offset] = 0
	b[reserved2Offset] = 0
	b[reserved3Offset] = 0
	b[reserved4Offset] = 0
	b[NumExtraOffsetHigh] = 0
	b[NumExtraOffsetLow] = 0
}
