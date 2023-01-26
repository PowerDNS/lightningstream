package header

import (
	"encoding/binary"
	"time"

	"github.com/pkg/errors"
)

// Header describes the header of a native schema value
type Header struct {
	Timestamp time.Time // time of last change
	TxnID     uint64    // type matches lmdb.EnvInfo.LastTxnID
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

	binary.BigEndian.PutUint64(b[:8], uint64(h.Timestamp.UnixNano()))
	binary.BigEndian.PutUint64(b[8:16], h.TxnID)
	b[VersionOffset] = uint8(h.Version)
	b[FlagsOffset] = uint8(h.Flags)
	b[NumExtraOffset] = uint8(n)
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
	VersionOffset  = 16
	FlagsOffset    = 17
	NumExtraOffset = 23

	reserved1Offset = 18
	reserved2Offset = 19
	reserved3Offset = 20
	reserved4Offset = 21
	reserved5Offset = 22
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
	numExtra := int(val[NumExtraOffset])
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
		Timestamp: time.Unix(0, int64(binary.BigEndian.Uint64(val[:8]))),
		TxnID:     binary.BigEndian.Uint64(val[8:16]),
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
	numExtra := int(val[NumExtraOffset])
	if numExtra > 0 {
		numExtraBytes := BlockSize * numExtra
		if len(val) < MinHeaderSize+numExtraBytes {
			return nil, ErrTooShort
		}
		offset += numExtraBytes
	}

	return val[offset:], nil
}

// PutBasic creates a basic header in the provided slice. The slice must
// have a length of at least MinHeaderSize.
func PutBasic(b []byte, ts uint64, txnid uint64, flags Flags) {
	b = b[:MinHeaderSize] // Prevents further bounds checks
	binary.BigEndian.PutUint64(b[:8], ts)
	binary.BigEndian.PutUint64(b[8:16], txnid)
	b[VersionOffset] = 0
	b[FlagsOffset] = uint8(flags)
	b[reserved1Offset] = 0
	b[reserved2Offset] = 0
	b[reserved3Offset] = 0
	b[reserved4Offset] = 0
	b[reserved5Offset] = 0
	b[NumExtraOffset] = 0
}
