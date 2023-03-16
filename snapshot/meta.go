package snapshot

import "github.com/CrowdStrike/csproto"

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
