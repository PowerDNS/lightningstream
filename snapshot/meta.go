package snapshot

import (
	"encoding/binary"

	"github.com/CrowdStrike/csproto"
)

// Protobuf field numbers
const (
	FieldMetaGenerationID  = 1
	FieldMetaInstanceID    = 2
	FieldMetaHostname      = 3
	FieldMetaLMDBTxnID     = 4
	FieldMetaTimestampNano = 5
	FieldMetaDatabaseName  = 7
)

type Meta struct {
	GenerationID  string
	InstanceID    string
	Hostname      string
	LmdbTxnID     int64
	TimestampNano uint64
	DatabaseName  string
}

func (m *Meta) Marshal() []byte {
	stringFields := []struct {
		tag int
		val string
	}{
		{FieldMetaGenerationID, m.GenerationID},
		{FieldMetaInstanceID, m.InstanceID},
		{FieldMetaHostname, m.Hostname},
		{FieldMetaDatabaseName, m.DatabaseName},
	}

	// Make a safe estimate of the buffer size needed, not accurate.
	var bufSizeNeeded int
	for _, sf := range stringFields {
		bufSizeNeeded += len(sf.val) + 20
	}
	bufSizeNeeded += 1000 // generous enough for the numeric fields
	b := make([]byte, bufSizeNeeded)
	offset := 0

	// Marshal data
	for _, sf := range stringFields {
		if len(sf.val) > 0 {
			offset += csproto.EncodeTag(b[offset:], sf.tag, csproto.WireTypeLengthDelimited)
			offset += csproto.EncodeVarint(b[offset:], uint64(len(sf.val)))
			offset += copy(b[offset:], sf.val)
		}
	}
	if m.LmdbTxnID > 0 {
		offset += csproto.EncodeTag(b[offset:], FieldMetaLMDBTxnID, csproto.WireTypeVarint)
		offset += csproto.EncodeVarint(b[offset:], uint64(m.LmdbTxnID))
	}
	if m.TimestampNano > 0 {
		offset += csproto.EncodeTag(b[offset:], FieldMetaTimestampNano, csproto.WireTypeFixed64)
		binary.LittleEndian.PutUint64(b[offset:offset+8], m.TimestampNano)
		offset += 8
	}

	return b[:offset]
}

func (m *Meta) Unmarshal(data []byte) error {
	d := csproto.NewDecoder(data)
	d.SetMode(csproto.DecoderModeFast)
	for d.More() {
		tag, wireType, err := d.DecodeTag()
		if err != nil {
			return err
		}
		switch tag {
		case FieldMetaGenerationID:
			m.GenerationID, err = getString(d, tag, wireType)
			if err != nil {
				return err
			}
		case FieldMetaInstanceID:
			m.InstanceID, err = getString(d, tag, wireType)
			if err != nil {
				return err
			}
		case FieldMetaHostname:
			m.Hostname, err = getString(d, tag, wireType)
			if err != nil {
				return err
			}
		case FieldMetaLMDBTxnID:
			m.LmdbTxnID, err = getInt64(d, tag, wireType)
			if err != nil {
				return err
			}
		case FieldMetaTimestampNano:
			m.TimestampNano, err = getFixed64(d, tag, wireType)
			if err != nil {
				return err
			}
		case FieldMetaDatabaseName:
			m.DatabaseName, err = getString(d, tag, wireType)
			if err != nil {
				return err
			}
		default:
			if _, err := d.Skip(tag, wireType); err != nil {
				return err
			}
		}
	}
	return nil
}
