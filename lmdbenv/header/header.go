package header

import (
	"encoding/binary"
	"time"

	"github.com/pkg/errors"
)

// Header describes the header of a native schema value
type Header struct {
	Timestamp time.Time // time of last change
	TxnID     int64     // type matches lmdb.EnvInfo.LastTxnID
	Version   int       // header version (currently always 0)
	Flags     uint8     // header flags
	NumExtra  int       // extra number of 8-byte blocks (set automatically in .Bytes())
	Extra     []byte    // bytes in extra block
}

func (h Header) MarshalBinary() (data []byte, err error) {
	return h.Bytes(), nil
}

func (h Header) Bytes() []byte {
	// fixed allocation size allows stack allocation
	b := make([]byte, MinHeaderSize, HeaderPreAllocSize)
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
	binary.BigEndian.PutUint64(b[8:16], uint64(h.TxnID))
	b[VersionOffset] = uint8(h.Version)
	b[FlagsOffset] = h.Flags
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
	// HeaderPreAllocSize is the number of bytes pre-allocated to fit a header
	// with extensions. Larger headers will trigger an additional heap alloc.
	HeaderPreAllocSize = 64
	// BlockSize is the number of bytes per extra block
	BlockSize = 8
)

const (
	VersionOffset  = 16
	FlagsOffset    = 17
	NumExtraOffset = 23
)

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
		TxnID:     int64(binary.BigEndian.Uint64(val[8:16])),
		Version:   int(val[VersionOffset]),
		Flags:     val[FlagsOffset],
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
