package snapshot

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func BenchmarkDumpData_1M_entries(b *testing.B) {
	// Keep in mind that is basically just testing the compression speed,
	// as that is by far the bottleneck here now.
	const entries = 1_000_000
	snap := makeTestSnapshot(entries)
	var uncompressed uint64
	var compressed uint64
	b.ReportAllocs()
	b.ResetTimer()
	t := time.Now()
	for i := 0; i < b.N; i++ {
		data, st, err := DumpData(snap)
		if len(data) < 1*MB {
			b.Fatal("snapshot too small", len(data))
		}
		assert.NoError(b, err)
		uncompressed += uint64(st.ProtobufSize)
		compressed += uint64(st.CompressedSize)
	}
	dt := time.Since(t)
	b.StopTimer()
	b.ReportMetric(float64(uncompressed)/MB/dt.Seconds(), "uncompressed_MB/s")
	b.ReportMetric(float64(compressed)/MB/dt.Seconds(), "compressed_MB/s")
	b.ReportMetric(float64(b.N)/dt.Seconds(), "Mentries/s")
}
