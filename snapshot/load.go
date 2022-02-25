package snapshot

import (
	"bytes"
	"compress/gzip"
	"io"
	"time"

	"github.com/c2h5oh/datasize"
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

func DumpData(msg *Snapshot) ([]byte, DumpDataStats, error) {
	var stat DumpDataStats
	t0 := time.Now()

	// Snapshot complete, serialize it
	pb, err := msg.Marshal()
	if err != nil {
		return nil, stat, err
	}
	tMarshaled := time.Now()
	stat.TMarshaled = tMarshaled.Sub(t0)

	// Compress it
	out := bytes.NewBuffer(make([]byte, 0, datasize.MB))
	gw, err := gzip.NewWriterLevel(out, gzip.BestSpeed)
	if err != nil {
		return nil, stat, err
	}
	if _, err = gw.Write(pb); err != nil {
		return nil, stat, err
	}
	if err = gw.Close(); err != nil {
		return nil, stat, err
	}
	tCompressed := time.Now()
	stat.TCompressed = tCompressed.Sub(tMarshaled)

	return out.Bytes(), stat, nil
}

type DumpDataStats struct {
	TMarshaled  time.Duration
	TCompressed time.Duration
}
