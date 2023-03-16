package snapshot

import (
	"bytes"
	"io"
	"time"

	"github.com/c2h5oh/datasize"
	"github.com/klauspost/compress/gzip"
)

// LoadData loads snapshot file contents that are gzipped protobufs
func LoadData(data []byte) (*Snapshot, error) {
	// Uncompress
	dataBuffer := bytes.NewBuffer(data)
	g, err := gzip.NewReader(dataBuffer)
	if err != nil {
		return nil, err
	}
	pbData, err := io.ReadAll(g)
	if err != nil {
		return nil, err
	}
	if err := g.Close(); err != nil {
		return nil, err
	}

	// Load protobuf
	msg := new(Snapshot)
	if err := msg.Unmarshal(pbData); err != nil {
		return nil, err
	}

	return msg, nil
}

// DumpData returns a compressed Snapshot.
func DumpData(msg *Snapshot) ([]byte, DumpDataStats, error) {
	var stat DumpDataStats
	t0 := time.Now()

	// Streaming compression
	out := bytes.NewBuffer(make([]byte, 0, datasize.MB)) // TODO: better start size
	gw, err := gzip.NewWriterLevel(out, gzip.BestSpeed)
	if err != nil {
		return nil, stat, err
	}

	// Marshal and write to gzip writer
	// The marshalling itself takes almost no time, since all the DBI data is
	// already marshaled.
	pbSize, err := msg.WriteTo(gw)
	if err != nil {
		return nil, stat, err
	}
	stat.ProtobufSize = datasize.ByteSize(pbSize)

	if err = gw.Close(); err != nil {
		return nil, stat, err
	}
	tCompressed := time.Now()
	stat.TCompressed = tCompressed.Sub(t0)

	compressedData := out.Bytes()
	stat.CompressedSize = datasize.ByteSize(len(compressedData))
	return compressedData, stat, nil
}

type DumpDataStats struct {
	TCompressed    time.Duration     // time it took to marshal (near 0) and compress
	ProtobufSize   datasize.ByteSize // uncompressed protobuf size
	CompressedSize datasize.ByteSize // uncompressed protobuf size
}
