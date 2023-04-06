package snapshot

import (
	"fmt"
	"io"

	"github.com/CrowdStrike/csproto"
)

const MB = 1024 * 1024

type ErrUnexpectedWireType struct {
	Tag         int
	WireType    csproto.WireType
	ExpWireType csproto.WireType
}

func (e ErrUnexpectedWireType) Error() string {
	return fmt.Sprintf("unexpected wiretype for tag %d: got %v, expected %v",
		e.Tag, e.WireType, e.ExpWireType)
}

func expectWT(tag int, got, exp csproto.WireType) error {
	if got != exp {
		return ErrUnexpectedWireType{
			Tag:         tag,
			WireType:    got,
			ExpWireType: exp,
		}
	}
	return nil
}

func getUInt32(d *csproto.Decoder, tag int, wireType csproto.WireType) (uint32, error) {
	if err := expectWT(tag, wireType, csproto.WireTypeVarint); err != nil {
		return 0, err
	}
	val, err := d.DecodeUInt32()
	if err != nil {
		return 0, err
	}
	return val, nil
}

func getFixed64(d *csproto.Decoder, tag int, wireType csproto.WireType) (uint64, error) {
	if err := expectWT(tag, wireType, csproto.WireTypeFixed64); err != nil {
		return 0, err
	}
	val, err := d.DecodeFixed64()
	if err != nil {
		return 0, err
	}
	return val, nil
}

func getInt64(d *csproto.Decoder, tag int, wireType csproto.WireType) (int64, error) {
	if err := expectWT(tag, wireType, csproto.WireTypeVarint); err != nil {
		return 0, err
	}
	val, err := d.DecodeInt64()
	if err != nil {
		return 0, err
	}
	return val, nil
}

func getBytes(d *csproto.Decoder, tag int, wireType csproto.WireType) ([]byte, error) {
	if err := expectWT(tag, wireType, csproto.WireTypeLengthDelimited); err != nil {
		return nil, err
	}
	val, err := d.DecodeBytes()
	if err != nil {
		return nil, err
	}
	n := len(val)
	return val[0:n:n], nil
}

func getString(d *csproto.Decoder, tag int, wireType csproto.WireType) (string, error) {
	if err := expectWT(tag, wireType, csproto.WireTypeLengthDelimited); err != nil {
		return "", err
	}
	val, err := d.DecodeString()
	if err != nil {
		return "", err
	}
	return val, nil
}

// skipTag skips over the next tag data
func skipTag(data []byte, wireType csproto.WireType) (skip int, err error) {
	switch wireType {
	case csproto.WireTypeVarint:
		_, n, err := csproto.DecodeVarint(data)
		if err != nil {
			return 0, err
		}
		skip = n
	case csproto.WireTypeLengthDelimited:
		size, n, err := csproto.DecodeVarint(data)
		if err != nil {
			return 0, err
		}
		skip = int(size) + n
	case csproto.WireTypeFixed32:
		skip = 4
	case csproto.WireTypeFixed64:
		skip = 8
	default:
		return 0, fmt.Errorf("unsupported wire type: %v", wireType)
	}
	if skip > len(data) {
		return 0, io.ErrUnexpectedEOF
	}
	return skip, nil
}
