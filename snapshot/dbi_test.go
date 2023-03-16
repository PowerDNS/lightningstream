package snapshot

import (
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func testDBI(n int) *DBI {
	d := NewDBI()
	for i := 0; i < n; i++ {
		d.Append(KV{
			Key:           []byte{'k', byte(i)},
			Value:         []byte{'v', byte(i)},
			Flags:         uint32(i) % 2,
			TimestampNano: uint64(i),
		})
	}
	return d
}

func TestDBI(t *testing.T) {
	pbdata := testDBI(1_000).ProtobufData()

	d, err := NewDBIFromData(pbdata)
	assert.NoError(t, err)

	d.ResetCursor()
	for i := 0; i < 1_000; i++ {
		kv, err := d.Next()
		assert.NoError(t, err)
		assert.Equal(t, []byte{'k', byte(i)}, kv.Key)
		assert.Equal(t, []byte{'v', byte(i)}, kv.Value)
		assert.Equal(t, uint32(i)%2, kv.Flags)
		assert.Equal(t, uint64(i), kv.TimestampNano)
	}
	_, err = d.Next()
	assert.Equal(t, io.EOF, err)

	// FIXME: also test other fields

}

func BenchmarkDBI_Next(b *testing.B) {
	d := testDBI(1_000_000)
	b.Logf("Size for 1M entries: %.1f MB", float64(len(d.ProtobufData())/MB))
	d.ResetCursor()

	var doNotOptimise int
	b.ReportAllocs()
	b.ResetTimer()
	t := time.Now()
	for i := 0; i < b.N; i++ {
		kv, err := d.Next()
		if err == io.EOF {
			d.ResetCursor()
		}
		doNotOptimise += len(kv.Value)
	}
	b.StopTimer()
	dt := time.Since(t)
	b.Logf("doNotOptimise: %d", doNotOptimise)

	b.ReportMetric(float64(b.N)/dt.Seconds()/1_000_000, "Mentries/s")
}

func BenchmarkDBI_Next_compare_with_slice(b *testing.B) {
	slice := make([]KV, 1_000_000)

	var doNotOptimise int
	b.ReportAllocs()
	b.ResetTimer()
	t := time.Now()
	for i := 0; i < b.N; i++ {
		kv := slice[i%1_000_000]
		doNotOptimise += len(kv.Value)
	}
	b.StopTimer()
	dt := time.Since(t)
	b.Logf("doNotOptimise: %d", doNotOptimise)

	b.ReportMetric(float64(b.N)/dt.Seconds()/1_000_000, "Mentries/s")
}

func BenchmarkDBI_index_1M(b *testing.B) {
	d := testDBI(1_000_000)
	b.Logf("Size for 1M entries: %.1f MB", float64(len(d.ProtobufData())/MB))
	d.ResetCursor()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = d.indexData()
	}
}
