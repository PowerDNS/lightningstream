// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: snapshot.proto

package snapshot

import (
	encoding_binary "encoding/binary"
	fmt "fmt"
	_ "github.com/gogo/protobuf/gogoproto"
	proto "github.com/gogo/protobuf/proto"
	io "io"
	math "math"
	math_bits "math/bits"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

type KV struct {
	Key                  []byte   `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
	Value                []byte   `protobuf:"bytes,2,opt,name=value,proto3" json:"value,omitempty"`
	TimestampNano        uint64   `protobuf:"fixed64,3,opt,name=timestampNano,proto3" json:"timestampNano,omitempty"`
	Flags                uint32   `protobuf:"varint,4,opt,name=flags,proto3" json:"flags,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
}

func (m *KV) Reset()         { *m = KV{} }
func (m *KV) String() string { return proto.CompactTextString(m) }
func (*KV) ProtoMessage()    {}
func (*KV) Descriptor() ([]byte, []int) {
	return fileDescriptor_0c8aab8e59648e0b, []int{0}
}
func (m *KV) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *KV) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_KV.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *KV) XXX_Merge(src proto.Message) {
	xxx_messageInfo_KV.Merge(m, src)
}
func (m *KV) XXX_Size() int {
	return m.Size()
}
func (m *KV) XXX_DiscardUnknown() {
	xxx_messageInfo_KV.DiscardUnknown(m)
}

var xxx_messageInfo_KV proto.InternalMessageInfo

func (m *KV) GetKey() []byte {
	if m != nil {
		return m.Key
	}
	return nil
}

func (m *KV) GetValue() []byte {
	if m != nil {
		return m.Value
	}
	return nil
}

func (m *KV) GetTimestampNano() uint64 {
	if m != nil {
		return m.TimestampNano
	}
	return 0
}

func (m *KV) GetFlags() uint32 {
	if m != nil {
		return m.Flags
	}
	return 0
}

type DBI struct {
	Name                 string   `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Entries              []KV     `protobuf:"bytes,2,rep,name=entries,proto3" json:"entries"`
	Flags                uint64   `protobuf:"varint,3,opt,name=flags,proto3" json:"flags,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
}

func (m *DBI) Reset()         { *m = DBI{} }
func (m *DBI) String() string { return proto.CompactTextString(m) }
func (*DBI) ProtoMessage()    {}
func (*DBI) Descriptor() ([]byte, []int) {
	return fileDescriptor_0c8aab8e59648e0b, []int{1}
}
func (m *DBI) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *DBI) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_DBI.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *DBI) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DBI.Merge(m, src)
}
func (m *DBI) XXX_Size() int {
	return m.Size()
}
func (m *DBI) XXX_DiscardUnknown() {
	xxx_messageInfo_DBI.DiscardUnknown(m)
}

var xxx_messageInfo_DBI proto.InternalMessageInfo

func (m *DBI) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *DBI) GetEntries() []KV {
	if m != nil {
		return m.Entries
	}
	return nil
}

func (m *DBI) GetFlags() uint64 {
	if m != nil {
		return m.Flags
	}
	return 0
}

type Snapshot struct {
	FormatVersion        uint32        `protobuf:"varint,1,opt,name=formatVersion,proto3" json:"formatVersion,omitempty"`
	Meta                 Snapshot_Meta `protobuf:"bytes,2,opt,name=meta,proto3" json:"meta"`
	Databases            []*DBI        `protobuf:"bytes,3,rep,name=databases,proto3" json:"databases,omitempty"`
	XXX_NoUnkeyedLiteral struct{}      `json:"-"`
}

func (m *Snapshot) Reset()         { *m = Snapshot{} }
func (m *Snapshot) String() string { return proto.CompactTextString(m) }
func (*Snapshot) ProtoMessage()    {}
func (*Snapshot) Descriptor() ([]byte, []int) {
	return fileDescriptor_0c8aab8e59648e0b, []int{2}
}
func (m *Snapshot) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Snapshot) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Snapshot.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Snapshot) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Snapshot.Merge(m, src)
}
func (m *Snapshot) XXX_Size() int {
	return m.Size()
}
func (m *Snapshot) XXX_DiscardUnknown() {
	xxx_messageInfo_Snapshot.DiscardUnknown(m)
}

var xxx_messageInfo_Snapshot proto.InternalMessageInfo

func (m *Snapshot) GetFormatVersion() uint32 {
	if m != nil {
		return m.FormatVersion
	}
	return 0
}

func (m *Snapshot) GetMeta() Snapshot_Meta {
	if m != nil {
		return m.Meta
	}
	return Snapshot_Meta{}
}

func (m *Snapshot) GetDatabases() []*DBI {
	if m != nil {
		return m.Databases
	}
	return nil
}

type Snapshot_Meta struct {
	GenerationID         string   `protobuf:"bytes,1,opt,name=generationID,proto3" json:"generationID,omitempty"`
	InstanceID           string   `protobuf:"bytes,2,opt,name=instanceID,proto3" json:"instanceID,omitempty"`
	Hostname             string   `protobuf:"bytes,3,opt,name=hostname,proto3" json:"hostname,omitempty"`
	LmdbTxnID            int64    `protobuf:"varint,4,opt,name=lmdbTxnID,proto3" json:"lmdbTxnID,omitempty"`
	TimestampNano        uint64   `protobuf:"fixed64,5,opt,name=timestampNano,proto3" json:"timestampNano,omitempty"`
	DatabaseName         string   `protobuf:"bytes,7,opt,name=databaseName,proto3" json:"databaseName,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
}

func (m *Snapshot_Meta) Reset()         { *m = Snapshot_Meta{} }
func (m *Snapshot_Meta) String() string { return proto.CompactTextString(m) }
func (*Snapshot_Meta) ProtoMessage()    {}
func (*Snapshot_Meta) Descriptor() ([]byte, []int) {
	return fileDescriptor_0c8aab8e59648e0b, []int{2, 0}
}
func (m *Snapshot_Meta) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Snapshot_Meta) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Snapshot_Meta.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Snapshot_Meta) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Snapshot_Meta.Merge(m, src)
}
func (m *Snapshot_Meta) XXX_Size() int {
	return m.Size()
}
func (m *Snapshot_Meta) XXX_DiscardUnknown() {
	xxx_messageInfo_Snapshot_Meta.DiscardUnknown(m)
}

var xxx_messageInfo_Snapshot_Meta proto.InternalMessageInfo

func (m *Snapshot_Meta) GetGenerationID() string {
	if m != nil {
		return m.GenerationID
	}
	return ""
}

func (m *Snapshot_Meta) GetInstanceID() string {
	if m != nil {
		return m.InstanceID
	}
	return ""
}

func (m *Snapshot_Meta) GetHostname() string {
	if m != nil {
		return m.Hostname
	}
	return ""
}

func (m *Snapshot_Meta) GetLmdbTxnID() int64 {
	if m != nil {
		return m.LmdbTxnID
	}
	return 0
}

func (m *Snapshot_Meta) GetTimestampNano() uint64 {
	if m != nil {
		return m.TimestampNano
	}
	return 0
}

func (m *Snapshot_Meta) GetDatabaseName() string {
	if m != nil {
		return m.DatabaseName
	}
	return ""
}

func init() {
	proto.RegisterType((*KV)(nil), "KV")
	proto.RegisterType((*DBI)(nil), "DBI")
	proto.RegisterType((*Snapshot)(nil), "Snapshot")
	proto.RegisterType((*Snapshot_Meta)(nil), "Snapshot.Meta")
}

func init() { proto.RegisterFile("snapshot.proto", fileDescriptor_0c8aab8e59648e0b) }

var fileDescriptor_0c8aab8e59648e0b = []byte{
	// 376 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x6c, 0x92, 0xb1, 0xce, 0xda, 0x30,
	0x10, 0xc7, 0x71, 0xe2, 0x0f, 0xc8, 0x7d, 0xf0, 0x09, 0x59, 0x0c, 0x11, 0xaa, 0xd2, 0x28, 0xed,
	0x90, 0x29, 0x03, 0x7d, 0x83, 0x28, 0x4b, 0x8a, 0x60, 0x70, 0x11, 0xbb, 0x29, 0x26, 0x44, 0x25,
	0x36, 0x8a, 0xdd, 0xaa, 0x7d, 0x8b, 0x3e, 0x16, 0x23, 0x43, 0xbb, 0x56, 0x15, 0xbc, 0x48, 0x65,
	0x87, 0x14, 0x50, 0xbb, 0xdd, 0xfd, 0x6c, 0xdf, 0xff, 0x7f, 0x77, 0x86, 0x17, 0x25, 0xd8, 0x41,
	0xed, 0xa4, 0x4e, 0x0e, 0xb5, 0xd4, 0x72, 0x02, 0x85, 0x2c, 0x64, 0x13, 0x47, 0x3b, 0x70, 0x66,
	0x2b, 0x32, 0x02, 0xf7, 0x13, 0xff, 0xe6, 0xa3, 0x10, 0xc5, 0x03, 0x6a, 0x42, 0x32, 0x86, 0xa7,
	0x2f, 0x6c, 0xff, 0x99, 0xfb, 0x8e, 0x65, 0x4d, 0x42, 0xde, 0xc2, 0x50, 0x97, 0x15, 0x57, 0x9a,
	0x55, 0x87, 0x05, 0x13, 0xd2, 0x77, 0x43, 0x14, 0x77, 0xe9, 0x23, 0x34, 0x6f, 0xb7, 0x7b, 0x56,
	0x28, 0x1f, 0x87, 0x28, 0x1e, 0xd2, 0x26, 0x89, 0x96, 0xe0, 0x66, 0x69, 0x4e, 0x08, 0x60, 0xc1,
	0x2a, 0x6e, 0xb5, 0x3c, 0x6a, 0x63, 0xf2, 0x06, 0x7a, 0x5c, 0xe8, 0xba, 0xe4, 0xca, 0x77, 0x42,
	0x37, 0x7e, 0x9e, 0xba, 0xc9, 0x6c, 0x95, 0xe2, 0xe3, 0xaf, 0xd7, 0x1d, 0xda, 0x9e, 0xdc, 0xaa,
	0x1a, 0x4d, 0xdc, 0x56, 0xfd, 0xe9, 0x40, 0xff, 0xc3, 0xb5, 0x3d, 0x63, 0x6f, 0x2b, 0xeb, 0x8a,
	0xe9, 0x15, 0xaf, 0x55, 0x29, 0x85, 0x15, 0x19, 0xd2, 0x47, 0x48, 0x62, 0xc0, 0x15, 0xd7, 0xcc,
	0x76, 0xf6, 0x3c, 0x7d, 0x49, 0xda, 0xe7, 0xc9, 0x9c, 0x6b, 0x76, 0x55, 0xb5, 0x37, 0x48, 0x04,
	0xde, 0x86, 0x69, 0xb6, 0x66, 0x8a, 0x1b, 0x59, 0xe3, 0x0c, 0x27, 0x59, 0x9a, 0xd3, 0x1b, 0x9e,
	0xfc, 0x40, 0x80, 0xe7, 0xcd, 0xe5, 0x41, 0xc1, 0x05, 0xaf, 0x99, 0x2e, 0xa5, 0xc8, 0xb3, 0x6b,
	0x83, 0x0f, 0x8c, 0x04, 0x00, 0xa5, 0x50, 0x9a, 0x89, 0x8f, 0x3c, 0xcf, 0xac, 0x01, 0x8f, 0xde,
	0x11, 0x32, 0x81, 0xfe, 0x4e, 0x2a, 0x6d, 0x07, 0xe4, 0xda, 0xd3, 0xbf, 0x39, 0x79, 0x05, 0xde,
	0xbe, 0xda, 0xac, 0x97, 0x5f, 0x4d, 0x71, 0x33, 0x59, 0x97, 0xde, 0xc0, 0xbf, 0x9b, 0x79, 0xfa,
	0xdf, 0x66, 0x22, 0x18, 0xb4, 0xce, 0x17, 0x46, 0xa3, 0xd7, 0x78, 0xbc, 0x67, 0xef, 0x71, 0xbf,
	0x3b, 0xea, 0xa5, 0xe3, 0xe3, 0x39, 0x40, 0xa7, 0x73, 0x80, 0x7e, 0x9f, 0x03, 0xf4, 0xfd, 0x12,
	0x74, 0x4e, 0x97, 0xa0, 0xb3, 0xee, 0xda, 0x4f, 0xf3, 0xee, 0x4f, 0x00, 0x00, 0x00, 0xff, 0xff,
	0x85, 0x9a, 0xad, 0x58, 0x52, 0x02, 0x00, 0x00,
}

func (m *KV) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *KV) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *KV) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.Flags != 0 {
		i = encodeVarintSnapshot(dAtA, i, uint64(m.Flags))
		i--
		dAtA[i] = 0x20
	}
	if m.TimestampNano != 0 {
		i -= 8
		encoding_binary.LittleEndian.PutUint64(dAtA[i:], uint64(m.TimestampNano))
		i--
		dAtA[i] = 0x19
	}
	if len(m.Value) > 0 {
		i -= len(m.Value)
		copy(dAtA[i:], m.Value)
		i = encodeVarintSnapshot(dAtA, i, uint64(len(m.Value)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.Key) > 0 {
		i -= len(m.Key)
		copy(dAtA[i:], m.Key)
		i = encodeVarintSnapshot(dAtA, i, uint64(len(m.Key)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *DBI) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *DBI) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *DBI) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.Flags != 0 {
		i = encodeVarintSnapshot(dAtA, i, uint64(m.Flags))
		i--
		dAtA[i] = 0x18
	}
	if len(m.Entries) > 0 {
		for iNdEx := len(m.Entries) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.Entries[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintSnapshot(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x12
		}
	}
	if len(m.Name) > 0 {
		i -= len(m.Name)
		copy(dAtA[i:], m.Name)
		i = encodeVarintSnapshot(dAtA, i, uint64(len(m.Name)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *Snapshot) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Snapshot) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Snapshot) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Databases) > 0 {
		for iNdEx := len(m.Databases) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.Databases[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintSnapshot(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x1a
		}
	}
	{
		size, err := m.Meta.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintSnapshot(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x12
	if m.FormatVersion != 0 {
		i = encodeVarintSnapshot(dAtA, i, uint64(m.FormatVersion))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func (m *Snapshot_Meta) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Snapshot_Meta) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Snapshot_Meta) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.DatabaseName) > 0 {
		i -= len(m.DatabaseName)
		copy(dAtA[i:], m.DatabaseName)
		i = encodeVarintSnapshot(dAtA, i, uint64(len(m.DatabaseName)))
		i--
		dAtA[i] = 0x3a
	}
	if m.TimestampNano != 0 {
		i -= 8
		encoding_binary.LittleEndian.PutUint64(dAtA[i:], uint64(m.TimestampNano))
		i--
		dAtA[i] = 0x29
	}
	if m.LmdbTxnID != 0 {
		i = encodeVarintSnapshot(dAtA, i, uint64(m.LmdbTxnID))
		i--
		dAtA[i] = 0x20
	}
	if len(m.Hostname) > 0 {
		i -= len(m.Hostname)
		copy(dAtA[i:], m.Hostname)
		i = encodeVarintSnapshot(dAtA, i, uint64(len(m.Hostname)))
		i--
		dAtA[i] = 0x1a
	}
	if len(m.InstanceID) > 0 {
		i -= len(m.InstanceID)
		copy(dAtA[i:], m.InstanceID)
		i = encodeVarintSnapshot(dAtA, i, uint64(len(m.InstanceID)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.GenerationID) > 0 {
		i -= len(m.GenerationID)
		copy(dAtA[i:], m.GenerationID)
		i = encodeVarintSnapshot(dAtA, i, uint64(len(m.GenerationID)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintSnapshot(dAtA []byte, offset int, v uint64) int {
	offset -= sovSnapshot(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *KV) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Key)
	if l > 0 {
		n += 1 + l + sovSnapshot(uint64(l))
	}
	l = len(m.Value)
	if l > 0 {
		n += 1 + l + sovSnapshot(uint64(l))
	}
	if m.TimestampNano != 0 {
		n += 9
	}
	if m.Flags != 0 {
		n += 1 + sovSnapshot(uint64(m.Flags))
	}
	return n
}

func (m *DBI) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Name)
	if l > 0 {
		n += 1 + l + sovSnapshot(uint64(l))
	}
	if len(m.Entries) > 0 {
		for _, e := range m.Entries {
			l = e.Size()
			n += 1 + l + sovSnapshot(uint64(l))
		}
	}
	if m.Flags != 0 {
		n += 1 + sovSnapshot(uint64(m.Flags))
	}
	return n
}

func (m *Snapshot) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.FormatVersion != 0 {
		n += 1 + sovSnapshot(uint64(m.FormatVersion))
	}
	l = m.Meta.Size()
	n += 1 + l + sovSnapshot(uint64(l))
	if len(m.Databases) > 0 {
		for _, e := range m.Databases {
			l = e.Size()
			n += 1 + l + sovSnapshot(uint64(l))
		}
	}
	return n
}

func (m *Snapshot_Meta) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.GenerationID)
	if l > 0 {
		n += 1 + l + sovSnapshot(uint64(l))
	}
	l = len(m.InstanceID)
	if l > 0 {
		n += 1 + l + sovSnapshot(uint64(l))
	}
	l = len(m.Hostname)
	if l > 0 {
		n += 1 + l + sovSnapshot(uint64(l))
	}
	if m.LmdbTxnID != 0 {
		n += 1 + sovSnapshot(uint64(m.LmdbTxnID))
	}
	if m.TimestampNano != 0 {
		n += 9
	}
	l = len(m.DatabaseName)
	if l > 0 {
		n += 1 + l + sovSnapshot(uint64(l))
	}
	return n
}

func sovSnapshot(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozSnapshot(x uint64) (n int) {
	return sovSnapshot(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *KV) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowSnapshot
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: KV: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: KV: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Key", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSnapshot
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthSnapshot
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthSnapshot
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Key = append(m.Key[:0], dAtA[iNdEx:postIndex]...)
			if m.Key == nil {
				m.Key = []byte{}
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Value", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSnapshot
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthSnapshot
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthSnapshot
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Value = append(m.Value[:0], dAtA[iNdEx:postIndex]...)
			if m.Value == nil {
				m.Value = []byte{}
			}
			iNdEx = postIndex
		case 3:
			if wireType != 1 {
				return fmt.Errorf("proto: wrong wireType = %d for field TimestampNano", wireType)
			}
			m.TimestampNano = 0
			if (iNdEx + 8) > l {
				return io.ErrUnexpectedEOF
			}
			m.TimestampNano = uint64(encoding_binary.LittleEndian.Uint64(dAtA[iNdEx:]))
			iNdEx += 8
		case 4:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Flags", wireType)
			}
			m.Flags = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSnapshot
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Flags |= uint32(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipSnapshot(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthSnapshot
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *DBI) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowSnapshot
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: DBI: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: DBI: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Name", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSnapshot
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthSnapshot
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthSnapshot
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Name = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Entries", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSnapshot
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthSnapshot
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthSnapshot
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Entries = append(m.Entries, KV{})
			if err := m.Entries[len(m.Entries)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Flags", wireType)
			}
			m.Flags = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSnapshot
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Flags |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipSnapshot(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthSnapshot
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *Snapshot) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowSnapshot
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: Snapshot: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Snapshot: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field FormatVersion", wireType)
			}
			m.FormatVersion = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSnapshot
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.FormatVersion |= uint32(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Meta", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSnapshot
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthSnapshot
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthSnapshot
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Meta.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Databases", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSnapshot
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthSnapshot
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthSnapshot
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Databases = append(m.Databases, &DBI{})
			if err := m.Databases[len(m.Databases)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipSnapshot(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthSnapshot
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *Snapshot_Meta) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowSnapshot
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: Meta: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Meta: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field GenerationID", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSnapshot
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthSnapshot
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthSnapshot
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.GenerationID = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field InstanceID", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSnapshot
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthSnapshot
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthSnapshot
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.InstanceID = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Hostname", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSnapshot
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthSnapshot
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthSnapshot
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Hostname = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 4:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field LmdbTxnID", wireType)
			}
			m.LmdbTxnID = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSnapshot
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.LmdbTxnID |= int64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 5:
			if wireType != 1 {
				return fmt.Errorf("proto: wrong wireType = %d for field TimestampNano", wireType)
			}
			m.TimestampNano = 0
			if (iNdEx + 8) > l {
				return io.ErrUnexpectedEOF
			}
			m.TimestampNano = uint64(encoding_binary.LittleEndian.Uint64(dAtA[iNdEx:]))
			iNdEx += 8
		case 7:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field DatabaseName", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSnapshot
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthSnapshot
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthSnapshot
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.DatabaseName = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipSnapshot(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthSnapshot
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipSnapshot(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowSnapshot
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowSnapshot
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
		case 1:
			iNdEx += 8
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowSnapshot
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthSnapshot
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupSnapshot
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthSnapshot
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthSnapshot        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowSnapshot          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupSnapshot = fmt.Errorf("proto: unexpected end of group")
)
