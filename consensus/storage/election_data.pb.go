// Code generated by protoc-gen-go. DO NOT EDIT.
// source: consensus/storage/election_data.proto

package storage

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

type PillarDelegationProto struct {
	ProducingAddress     []byte   `protobuf:"bytes,1,opt,name=producingAddress,proto3" json:"producingAddress,omitempty"`
	Name                 string   `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	Weight               []byte   `protobuf:"bytes,3,opt,name=weight,proto3" json:"weight,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *PillarDelegationProto) Reset()         { *m = PillarDelegationProto{} }
func (m *PillarDelegationProto) String() string { return proto.CompactTextString(m) }
func (*PillarDelegationProto) ProtoMessage()    {}
func (*PillarDelegationProto) Descriptor() ([]byte, []int) {
	return fileDescriptor_163f9f85a46af540, []int{0}
}

func (m *PillarDelegationProto) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_PillarDelegationProto.Unmarshal(m, b)
}
func (m *PillarDelegationProto) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_PillarDelegationProto.Marshal(b, m, deterministic)
}
func (m *PillarDelegationProto) XXX_Merge(src proto.Message) {
	xxx_messageInfo_PillarDelegationProto.Merge(m, src)
}
func (m *PillarDelegationProto) XXX_Size() int {
	return xxx_messageInfo_PillarDelegationProto.Size(m)
}
func (m *PillarDelegationProto) XXX_DiscardUnknown() {
	xxx_messageInfo_PillarDelegationProto.DiscardUnknown(m)
}

var xxx_messageInfo_PillarDelegationProto proto.InternalMessageInfo

func (m *PillarDelegationProto) GetProducingAddress() []byte {
	if m != nil {
		return m.ProducingAddress
	}
	return nil
}

func (m *PillarDelegationProto) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *PillarDelegationProto) GetWeight() []byte {
	if m != nil {
		return m.Weight
	}
	return nil
}

type ElectionDataProto struct {
	Producers            [][]byte                 `protobuf:"bytes,1,rep,name=producers,proto3" json:"producers,omitempty"`
	Delegations          []*PillarDelegationProto `protobuf:"bytes,2,rep,name=delegations,proto3" json:"delegations,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                 `json:"-"`
	XXX_unrecognized     []byte                   `json:"-"`
	XXX_sizecache        int32                    `json:"-"`
}

func (m *ElectionDataProto) Reset()         { *m = ElectionDataProto{} }
func (m *ElectionDataProto) String() string { return proto.CompactTextString(m) }
func (*ElectionDataProto) ProtoMessage()    {}
func (*ElectionDataProto) Descriptor() ([]byte, []int) {
	return fileDescriptor_163f9f85a46af540, []int{1}
}

func (m *ElectionDataProto) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ElectionDataProto.Unmarshal(m, b)
}
func (m *ElectionDataProto) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ElectionDataProto.Marshal(b, m, deterministic)
}
func (m *ElectionDataProto) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ElectionDataProto.Merge(m, src)
}
func (m *ElectionDataProto) XXX_Size() int {
	return xxx_messageInfo_ElectionDataProto.Size(m)
}
func (m *ElectionDataProto) XXX_DiscardUnknown() {
	xxx_messageInfo_ElectionDataProto.DiscardUnknown(m)
}

var xxx_messageInfo_ElectionDataProto proto.InternalMessageInfo

func (m *ElectionDataProto) GetProducers() [][]byte {
	if m != nil {
		return m.Producers
	}
	return nil
}

func (m *ElectionDataProto) GetDelegations() []*PillarDelegationProto {
	if m != nil {
		return m.Delegations
	}
	return nil
}

func init() {
	proto.RegisterType((*PillarDelegationProto)(nil), "storage.PillarDelegationProto")
	proto.RegisterType((*ElectionDataProto)(nil), "storage.ElectionDataProto")
}

func init() {
	proto.RegisterFile("consensus/storage/election_data.proto", fileDescriptor_163f9f85a46af540)
}

var fileDescriptor_163f9f85a46af540 = []byte{
	// 207 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x6c, 0x4f, 0xd1, 0x4a, 0x03, 0x31,
	0x10, 0xe4, 0x7a, 0x52, 0xe9, 0xb6, 0x0f, 0x1a, 0x50, 0xf2, 0x20, 0x72, 0x14, 0x84, 0xe0, 0xc3,
	0x15, 0xf4, 0x07, 0x14, 0xea, 0x7b, 0xc9, 0x0f, 0xc8, 0x7a, 0x59, 0x62, 0x20, 0x26, 0x47, 0x36,
	0x87, 0xbf, 0x2f, 0x97, 0x0b, 0x2a, 0xe8, 0xdb, 0xee, 0xec, 0xcc, 0xce, 0x0c, 0xdc, 0x0d, 0x31,
	0x30, 0x05, 0x9e, 0xf8, 0xc0, 0x39, 0x26, 0xb4, 0x74, 0x20, 0x4f, 0x43, 0x76, 0x31, 0xbc, 0x1a,
	0xcc, 0xd8, 0x8f, 0x29, 0xe6, 0x28, 0xce, 0xeb, 0x71, 0x1f, 0xe1, 0xea, 0xe4, 0xbc, 0xc7, 0x74,
	0x24, 0x4f, 0x16, 0x67, 0xde, 0xa9, 0x30, 0xee, 0xe1, 0x62, 0x4c, 0xd1, 0x4c, 0x83, 0x0b, 0xf6,
	0xd9, 0x98, 0x44, 0xcc, 0xb2, 0xe9, 0x1a, 0xb5, 0xd3, 0x7f, 0x70, 0x21, 0xe0, 0x2c, 0xe0, 0x07,
	0xc9, 0x55, 0xd7, 0xa8, 0x8d, 0x2e, 0xb3, 0xb8, 0x86, 0xf5, 0x27, 0x39, 0xfb, 0x9e, 0x65, 0x5b,
	0x54, 0x75, 0xdb, 0x33, 0x5c, 0xbe, 0xd4, 0x40, 0x47, 0xcc, 0xb8, 0x98, 0xdd, 0xc0, 0x66, 0x79,
	0x4a, 0x69, 0x76, 0x69, 0xd5, 0x4e, 0xff, 0x00, 0xe2, 0x09, 0xb6, 0xe6, 0x3b, 0x1d, 0xcb, 0x55,
	0xd7, 0xaa, 0xed, 0xc3, 0x6d, 0x5f, 0x2b, 0xf4, 0xff, 0xe6, 0xd7, 0xbf, 0x25, 0x6f, 0xeb, 0xd2,
	0xfa, 0xf1, 0x2b, 0x00, 0x00, 0xff, 0xff, 0xd9, 0x1c, 0x6f, 0xd4, 0x1e, 0x01, 0x00, 0x00,
}