package header

import (
	"encoding/binary"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func genTestVal(ts time.Time) []byte {
	testVal := []byte{
		0, 0, 0, 0, 0, 0, 0, 0, // timestamp filled below
		0, 0, 0, 0, 0, 0, 0, 123, // txnID
		0, 0x01, 0, 0, 0, 0, 0, 1, // version, flags, ..., extra blocks
		1, 2, 3, 4, 5, 6, 7, 8, // extra block
		't', 'e', 's', 't', // application value
	}
	binary.BigEndian.PutUint64(testVal[:8], uint64(ts.UnixNano()))
	return testVal
}

func BenchmarkParse(b *testing.B) {
	ts := time.Now().Truncate(time.Millisecond)
	testVal := genTestVal(ts)

	var (
		h   Header
		v   []byte
		err error
	)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h, v, err = Parse(testVal)
	}
	b.StopTimer()

	// To ensure it did not get optimised away
	assert.NoError(b, err)
	assert.Equal(b, 0, h.Version)
	assert.Equal(b, []byte("test"), v)

}

func TestParse(t *testing.T) {
	ts := time.Now().Truncate(time.Nanosecond)
	testVal := genTestVal(ts)

	h, v, err := Parse(testVal)
	assert.NoError(t, err)
	assert.Equal(t, ts, h.Timestamp)
	assert.Equal(t, int64(123), h.TxnID)
	assert.Equal(t, 0, h.Version)
	assert.Equal(t, uint8(0x1), h.Flags)
	assert.Equal(t, []byte{1, 2, 3, 4, 5, 6, 7, 8}, h.Extra)
	assert.Equal(t, []byte("test"), v)

	h, v, err = Parse(testVal[:20])
	assert.Equal(t, ErrTooShort, err)

	h, v, err = Parse(testVal[:25])
	assert.Equal(t, ErrTooShort, err)

	h, v, err = Parse(testVal[:31])
	assert.Equal(t, ErrTooShort, err)

	h, v, err = Parse(testVal[:32])
	assert.NoError(t, err)
	assert.Equal(t, []byte{}, v)
}

func BenchmarkSkip(b *testing.B) {
	ts := time.Now().Truncate(time.Millisecond)
	testVal := genTestVal(ts)

	var (
		v   []byte
		err error
	)

	totalSize := 0

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v, err = Skip(testVal)
		totalSize += len(v)
	}
	b.StopTimer()

	// To ensure it did not get optimised away
	assert.NoError(b, err)
	assert.Equal(b, []byte("test"), v)
}

func TestSkip(t *testing.T) {
	ts := time.Now().Truncate(time.Nanosecond)
	testVal := genTestVal(ts)

	v, err := Skip(testVal)
	assert.NoError(t, err)
	assert.Equal(t, []byte("test"), v)

	v, err = Skip(testVal[:20])
	assert.Equal(t, ErrTooShort, err)

	v, err = Skip(testVal[:25])
	assert.Equal(t, ErrTooShort, err)

	v, err = Skip(testVal[:31])
	assert.Equal(t, ErrTooShort, err)

	v, err = Skip(testVal[:32])
	assert.NoError(t, err)
	assert.Equal(t, []byte{}, v)
}

func BenchmarkPutBasic(b *testing.B) {
	buf := make([]byte, MinHeaderSize)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		PutBasic(buf, 1234567890, 121212, FlagDeleted)
	}
}

func BenchmarkHeader_Bytes_basic(b *testing.B) {
	h := Header{
		Timestamp: time.Now(),
		TxnID:     123,
		Version:   0,
		Flags:     0x01,
		NumExtra:  0,
		Extra:     nil,
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = h.Bytes()
	}
}

func BenchmarkHeader_Bytes_extra(b *testing.B) {
	h := Header{
		Timestamp: time.Now(),
		TxnID:     123,
		Version:   0,
		Flags:     0x01,
		NumExtra:  1,
		Extra:     []byte("extra"),
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = h.Bytes()
	}
}

func BenchmarkHeader_Bytes_extra_large_allocates(b *testing.B) {
	numOverflow := (PreAllocSize-MinHeaderSize)/8 + 1
	h := Header{
		Timestamp: time.Now(),
		TxnID:     123,
		Version:   0,
		Flags:     0x01,
		NumExtra:  numOverflow, // large enough to trigger an alloc
		Extra:     []byte("extra"),
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = h.Bytes()
	}
}

func TestHeader_Bytes_alloc(t *testing.T) {
	h := Header{
		Timestamp: time.Now(),
		TxnID:     123,
		Version:   0,
		Flags:     0x01,
		NumExtra:  1,
		Extra:     []byte("extra"),
	}
	allocs := testing.AllocsPerRun(1000, func() {
		h.Bytes()
	})
	assert.Equal(t, 0.0, allocs)
}

func TestHeader_Bytes_alloc_large_extra(t *testing.T) {
	numOverflow := (PreAllocSize-MinHeaderSize)/8 + 1
	h := Header{
		Timestamp: time.Now(),
		TxnID:     123,
		Version:   0,
		Flags:     0x01,
		NumExtra:  numOverflow, // larger than fixed capacity
		Extra:     []byte("extra"),
	}
	allocs := testing.AllocsPerRun(1000, func() {
		h.Bytes()
	})
	assert.Equal(t, 1.0, allocs)
}
