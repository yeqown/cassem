// Code generated by protoc-gen-go. DO NOT EDIT.
// source: api.proto

package cassem_cassemdb

import (
	context "context"
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
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

type ChangeOp int32

const (
	ChangeOp_Invalid ChangeOp = 0
	ChangeOp_Set     ChangeOp = 1
	ChangeOp_Unset   ChangeOp = 2
)

var ChangeOp_name = map[int32]string{
	0: "Invalid",
	1: "Set",
	2: "Unset",
}

var ChangeOp_value = map[string]int32{
	"Invalid": 0,
	"Set":     1,
	"Unset":   2,
}

func (x ChangeOp) String() string {
	return proto.EnumName(ChangeOp_name, int32(x))
}

func (ChangeOp) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_00212fb1f9d3bf1c, []int{0}
}

type Entity struct {
	Fingerprint          string   `protobuf:"bytes,1,opt,name=fingerprint,proto3" json:"fingerprint,omitempty"`
	Key                  string   `protobuf:"bytes,2,opt,name=key,proto3" json:"key,omitempty"`
	Val                  []byte   `protobuf:"bytes,3,opt,name=val,proto3" json:"val,omitempty"`
	CreatedAt            int64    `protobuf:"varint,4,opt,name=created_at,json=createdAt,proto3" json:"created_at,omitempty"`
	UpdatedAt            int64    `protobuf:"varint,5,opt,name=updated_at,json=updatedAt,proto3" json:"updated_at,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Entity) Reset()         { *m = Entity{} }
func (m *Entity) String() string { return proto.CompactTextString(m) }
func (*Entity) ProtoMessage()    {}
func (*Entity) Descriptor() ([]byte, []int) {
	return fileDescriptor_00212fb1f9d3bf1c, []int{0}
}

func (m *Entity) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Entity.Unmarshal(m, b)
}
func (m *Entity) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Entity.Marshal(b, m, deterministic)
}
func (m *Entity) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Entity.Merge(m, src)
}
func (m *Entity) XXX_Size() int {
	return xxx_messageInfo_Entity.Size(m)
}
func (m *Entity) XXX_DiscardUnknown() {
	xxx_messageInfo_Entity.DiscardUnknown(m)
}

var xxx_messageInfo_Entity proto.InternalMessageInfo

func (m *Entity) GetFingerprint() string {
	if m != nil {
		return m.Fingerprint
	}
	return ""
}

func (m *Entity) GetKey() string {
	if m != nil {
		return m.Key
	}
	return ""
}

func (m *Entity) GetVal() []byte {
	if m != nil {
		return m.Val
	}
	return nil
}

func (m *Entity) GetCreatedAt() int64 {
	if m != nil {
		return m.CreatedAt
	}
	return 0
}

func (m *Entity) GetUpdatedAt() int64 {
	if m != nil {
		return m.UpdatedAt
	}
	return 0
}

type Change struct {
	Op                   ChangeOp `protobuf:"varint,1,opt,name=op,proto3,enum=cassem.cassemdb.ChangeOp" json:"op,omitempty"`
	Key                  string   `protobuf:"bytes,2,opt,name=key,proto3" json:"key,omitempty"`
	Last                 *Entity  `protobuf:"bytes,3,opt,name=last,proto3" json:"last,omitempty"`
	Current              *Entity  `protobuf:"bytes,4,opt,name=current,proto3" json:"current,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Change) Reset()         { *m = Change{} }
func (m *Change) String() string { return proto.CompactTextString(m) }
func (*Change) ProtoMessage()    {}
func (*Change) Descriptor() ([]byte, []int) {
	return fileDescriptor_00212fb1f9d3bf1c, []int{1}
}

func (m *Change) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Change.Unmarshal(m, b)
}
func (m *Change) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Change.Marshal(b, m, deterministic)
}
func (m *Change) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Change.Merge(m, src)
}
func (m *Change) XXX_Size() int {
	return xxx_messageInfo_Change.Size(m)
}
func (m *Change) XXX_DiscardUnknown() {
	xxx_messageInfo_Change.DiscardUnknown(m)
}

var xxx_messageInfo_Change proto.InternalMessageInfo

func (m *Change) GetOp() ChangeOp {
	if m != nil {
		return m.Op
	}
	return ChangeOp_Invalid
}

func (m *Change) GetKey() string {
	if m != nil {
		return m.Key
	}
	return ""
}

func (m *Change) GetLast() *Entity {
	if m != nil {
		return m.Last
	}
	return nil
}

func (m *Change) GetCurrent() *Entity {
	if m != nil {
		return m.Current
	}
	return nil
}

type Empty struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Empty) Reset()         { *m = Empty{} }
func (m *Empty) String() string { return proto.CompactTextString(m) }
func (*Empty) ProtoMessage()    {}
func (*Empty) Descriptor() ([]byte, []int) {
	return fileDescriptor_00212fb1f9d3bf1c, []int{2}
}

func (m *Empty) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Empty.Unmarshal(m, b)
}
func (m *Empty) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Empty.Marshal(b, m, deterministic)
}
func (m *Empty) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Empty.Merge(m, src)
}
func (m *Empty) XXX_Size() int {
	return xxx_messageInfo_Empty.Size(m)
}
func (m *Empty) XXX_DiscardUnknown() {
	xxx_messageInfo_Empty.DiscardUnknown(m)
}

var xxx_messageInfo_Empty proto.InternalMessageInfo

type GetKVReq struct {
	Key                  string   `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *GetKVReq) Reset()         { *m = GetKVReq{} }
func (m *GetKVReq) String() string { return proto.CompactTextString(m) }
func (*GetKVReq) ProtoMessage()    {}
func (*GetKVReq) Descriptor() ([]byte, []int) {
	return fileDescriptor_00212fb1f9d3bf1c, []int{3}
}

func (m *GetKVReq) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_GetKVReq.Unmarshal(m, b)
}
func (m *GetKVReq) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_GetKVReq.Marshal(b, m, deterministic)
}
func (m *GetKVReq) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GetKVReq.Merge(m, src)
}
func (m *GetKVReq) XXX_Size() int {
	return xxx_messageInfo_GetKVReq.Size(m)
}
func (m *GetKVReq) XXX_DiscardUnknown() {
	xxx_messageInfo_GetKVReq.DiscardUnknown(m)
}

var xxx_messageInfo_GetKVReq proto.InternalMessageInfo

func (m *GetKVReq) GetKey() string {
	if m != nil {
		return m.Key
	}
	return ""
}

type GetKVResp struct {
	Entity               *Entity  `protobuf:"bytes,1,opt,name=entity,proto3" json:"entity,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *GetKVResp) Reset()         { *m = GetKVResp{} }
func (m *GetKVResp) String() string { return proto.CompactTextString(m) }
func (*GetKVResp) ProtoMessage()    {}
func (*GetKVResp) Descriptor() ([]byte, []int) {
	return fileDescriptor_00212fb1f9d3bf1c, []int{4}
}

func (m *GetKVResp) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_GetKVResp.Unmarshal(m, b)
}
func (m *GetKVResp) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_GetKVResp.Marshal(b, m, deterministic)
}
func (m *GetKVResp) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GetKVResp.Merge(m, src)
}
func (m *GetKVResp) XXX_Size() int {
	return xxx_messageInfo_GetKVResp.Size(m)
}
func (m *GetKVResp) XXX_DiscardUnknown() {
	xxx_messageInfo_GetKVResp.DiscardUnknown(m)
}

var xxx_messageInfo_GetKVResp proto.InternalMessageInfo

func (m *GetKVResp) GetEntity() *Entity {
	if m != nil {
		return m.Entity
	}
	return nil
}

type SetKVReq struct {
	Key                  string   `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
	Entity               *Entity  `protobuf:"bytes,2,opt,name=entity,proto3" json:"entity,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *SetKVReq) Reset()         { *m = SetKVReq{} }
func (m *SetKVReq) String() string { return proto.CompactTextString(m) }
func (*SetKVReq) ProtoMessage()    {}
func (*SetKVReq) Descriptor() ([]byte, []int) {
	return fileDescriptor_00212fb1f9d3bf1c, []int{5}
}

func (m *SetKVReq) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SetKVReq.Unmarshal(m, b)
}
func (m *SetKVReq) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SetKVReq.Marshal(b, m, deterministic)
}
func (m *SetKVReq) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SetKVReq.Merge(m, src)
}
func (m *SetKVReq) XXX_Size() int {
	return xxx_messageInfo_SetKVReq.Size(m)
}
func (m *SetKVReq) XXX_DiscardUnknown() {
	xxx_messageInfo_SetKVReq.DiscardUnknown(m)
}

var xxx_messageInfo_SetKVReq proto.InternalMessageInfo

func (m *SetKVReq) GetKey() string {
	if m != nil {
		return m.Key
	}
	return ""
}

func (m *SetKVReq) GetEntity() *Entity {
	if m != nil {
		return m.Entity
	}
	return nil
}

type UnsetKVReq struct {
	Key                  string   `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
	IsDir                bool     `protobuf:"varint,2,opt,name=is_dir,json=isDir,proto3" json:"is_dir,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *UnsetKVReq) Reset()         { *m = UnsetKVReq{} }
func (m *UnsetKVReq) String() string { return proto.CompactTextString(m) }
func (*UnsetKVReq) ProtoMessage()    {}
func (*UnsetKVReq) Descriptor() ([]byte, []int) {
	return fileDescriptor_00212fb1f9d3bf1c, []int{6}
}

func (m *UnsetKVReq) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_UnsetKVReq.Unmarshal(m, b)
}
func (m *UnsetKVReq) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_UnsetKVReq.Marshal(b, m, deterministic)
}
func (m *UnsetKVReq) XXX_Merge(src proto.Message) {
	xxx_messageInfo_UnsetKVReq.Merge(m, src)
}
func (m *UnsetKVReq) XXX_Size() int {
	return xxx_messageInfo_UnsetKVReq.Size(m)
}
func (m *UnsetKVReq) XXX_DiscardUnknown() {
	xxx_messageInfo_UnsetKVReq.DiscardUnknown(m)
}

var xxx_messageInfo_UnsetKVReq proto.InternalMessageInfo

func (m *UnsetKVReq) GetKey() string {
	if m != nil {
		return m.Key
	}
	return ""
}

func (m *UnsetKVReq) GetIsDir() bool {
	if m != nil {
		return m.IsDir
	}
	return false
}

type WatchReq struct {
	Keys                 []string `protobuf:"bytes,2,rep,name=keys,proto3" json:"keys,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *WatchReq) Reset()         { *m = WatchReq{} }
func (m *WatchReq) String() string { return proto.CompactTextString(m) }
func (*WatchReq) ProtoMessage()    {}
func (*WatchReq) Descriptor() ([]byte, []int) {
	return fileDescriptor_00212fb1f9d3bf1c, []int{7}
}

func (m *WatchReq) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_WatchReq.Unmarshal(m, b)
}
func (m *WatchReq) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_WatchReq.Marshal(b, m, deterministic)
}
func (m *WatchReq) XXX_Merge(src proto.Message) {
	xxx_messageInfo_WatchReq.Merge(m, src)
}
func (m *WatchReq) XXX_Size() int {
	return xxx_messageInfo_WatchReq.Size(m)
}
func (m *WatchReq) XXX_DiscardUnknown() {
	xxx_messageInfo_WatchReq.DiscardUnknown(m)
}

var xxx_messageInfo_WatchReq proto.InternalMessageInfo

func (m *WatchReq) GetKeys() []string {
	if m != nil {
		return m.Keys
	}
	return nil
}

func init() {
	proto.RegisterEnum("cassem.cassemdb.ChangeOp", ChangeOp_name, ChangeOp_value)
	proto.RegisterType((*Entity)(nil), "cassem.cassemdb.Entity")
	proto.RegisterType((*Change)(nil), "cassem.cassemdb.change")
	proto.RegisterType((*Empty)(nil), "cassem.cassemdb.empty")
	proto.RegisterType((*GetKVReq)(nil), "cassem.cassemdb.getKVReq")
	proto.RegisterType((*GetKVResp)(nil), "cassem.cassemdb.getKVResp")
	proto.RegisterType((*SetKVReq)(nil), "cassem.cassemdb.setKVReq")
	proto.RegisterType((*UnsetKVReq)(nil), "cassem.cassemdb.unsetKVReq")
	proto.RegisterType((*WatchReq)(nil), "cassem.cassemdb.watchReq")
}

func init() { proto.RegisterFile("api.proto", fileDescriptor_00212fb1f9d3bf1c) }

var fileDescriptor_00212fb1f9d3bf1c = []byte{
	// 436 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x8c, 0x53, 0x5d, 0x6f, 0xd3, 0x30,
	0x14, 0xad, 0x93, 0xe6, 0xeb, 0x16, 0x41, 0x75, 0x25, 0x20, 0x94, 0x0f, 0x45, 0x7e, 0x2a, 0x4c,
	0x2a, 0x50, 0xc4, 0xdb, 0x24, 0x36, 0x01, 0x42, 0x08, 0x21, 0x24, 0x4f, 0x83, 0xc7, 0xc9, 0x4b,
	0x4c, 0x67, 0xad, 0x4b, 0x8d, 0xed, 0x0e, 0xf5, 0x1f, 0xf0, 0xc0, 0xdf, 0xe0, 0x7f, 0x22, 0x3b,
	0x0d, 0x4c, 0x64, 0x8d, 0x78, 0xca, 0xd5, 0x3d, 0xc7, 0xe7, 0x1e, 0x9f, 0x1b, 0x43, 0xc6, 0x95,
	0x9c, 0x29, 0xbd, 0xb2, 0x2b, 0xbc, 0x55, 0x72, 0x63, 0xc4, 0xc5, 0xac, 0xf9, 0x54, 0xa7, 0xf4,
	0x27, 0x81, 0xf8, 0x6d, 0x6d, 0xa5, 0xdd, 0x60, 0x01, 0xa3, 0xaf, 0xb2, 0x5e, 0x08, 0xad, 0xb4,
	0xac, 0x6d, 0x4e, 0x0a, 0x32, 0xcd, 0xd8, 0xd5, 0x16, 0x8e, 0x21, 0x3c, 0x17, 0x9b, 0x3c, 0xf0,
	0x88, 0x2b, 0x5d, 0xe7, 0x92, 0x2f, 0xf3, 0xb0, 0x20, 0xd3, 0x1b, 0xcc, 0x95, 0xf8, 0x10, 0xa0,
	0xd4, 0x82, 0x5b, 0x51, 0x9d, 0x70, 0x9b, 0x0f, 0x0b, 0x32, 0x0d, 0x59, 0xb6, 0xed, 0x1c, 0x5a,
	0x07, 0xaf, 0x55, 0xd5, 0xc2, 0x51, 0x03, 0x6f, 0x3b, 0x87, 0x96, 0xfe, 0x22, 0x10, 0x97, 0x67,
	0xbc, 0x5e, 0x08, 0x7c, 0x0c, 0xc1, 0x4a, 0x79, 0x17, 0x37, 0xe7, 0xf7, 0x66, 0xff, 0xf8, 0x9e,
	0xbd, 0xf6, 0xa4, 0x4f, 0x8a, 0x05, 0x2b, 0x75, 0x8d, 0xaf, 0x3d, 0x18, 0x2e, 0xb9, 0xb1, 0xde,
	0xd8, 0x68, 0x7e, 0xb7, 0x73, 0xbc, 0xb9, 0x32, 0xf3, 0x24, 0x7c, 0x0e, 0x49, 0xb9, 0xd6, 0x5a,
	0xd4, 0x8d, 0xdf, 0x1e, 0x7e, 0xcb, 0xa3, 0x09, 0x44, 0xe2, 0x42, 0xd9, 0x0d, 0x7d, 0x00, 0xe9,
	0x42, 0xd8, 0x0f, 0x9f, 0x99, 0xf8, 0xd6, 0xda, 0x20, 0x7f, 0x6c, 0xd0, 0x7d, 0xc8, 0xb6, 0xa8,
	0x51, 0xf8, 0x14, 0x62, 0xe1, 0x65, 0x3c, 0xa3, 0x67, 0xca, 0x96, 0x46, 0x3f, 0x42, 0x6a, 0x76,
	0x6a, 0x5f, 0x91, 0x0b, 0xfe, 0x4f, 0xee, 0x25, 0xc0, 0xba, 0xee, 0x11, 0xbc, 0x0d, 0xb1, 0x34,
	0x27, 0x95, 0xd4, 0x5e, 0x30, 0x65, 0x91, 0x34, 0x6f, 0xa4, 0xa6, 0x8f, 0x20, 0xfd, 0xce, 0x6d,
	0x79, 0xe6, 0x0e, 0x21, 0x0c, 0xcf, 0xc5, 0xc6, 0xe4, 0x41, 0x11, 0x4e, 0x33, 0xe6, 0xeb, 0x27,
	0x7b, 0x90, 0xb6, 0xcb, 0xc0, 0x11, 0x24, 0xef, 0xeb, 0x4b, 0xbe, 0x94, 0xd5, 0x78, 0x80, 0x09,
	0x84, 0x47, 0xc2, 0x8e, 0x09, 0x66, 0x10, 0x1d, 0xbb, 0xc1, 0xe3, 0x60, 0xfe, 0x23, 0x80, 0x90,
	0x2b, 0x89, 0x07, 0x10, 0xbd, 0x73, 0x4e, 0xb0, 0xbb, 0xd9, 0x36, 0xce, 0xc9, 0x64, 0x17, 0x64,
	0x14, 0x1d, 0xe0, 0x3e, 0x44, 0x47, 0x3b, 0x14, 0xda, 0x3b, 0x4e, 0xee, 0x74, 0xa0, 0x66, 0x69,
	0x03, 0x3c, 0x80, 0xe4, 0xb8, 0xc9, 0x02, 0xef, 0x77, 0x48, 0x7f, 0x53, 0xea, 0x51, 0x78, 0x05,
	0xd1, 0x17, 0x17, 0xcb, 0x35, 0xf3, 0xdb, 0xb8, 0x26, 0xdd, 0x95, 0x34, 0xff, 0x36, 0x1d, 0x3c,
	0x23, 0xa7, 0xb1, 0x7f, 0x91, 0x2f, 0x7e, 0x07, 0x00, 0x00, 0xff, 0xff, 0xb3, 0x6e, 0x62, 0xa3,
	0x9e, 0x03, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// ApiClient is the client API for Api service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type ApiClient interface {
	GetKV(ctx context.Context, in *GetKVReq, opts ...grpc.CallOption) (*GetKVResp, error)
	SetKV(ctx context.Context, in *SetKVReq, opts ...grpc.CallOption) (*Empty, error)
	UnsetKV(ctx context.Context, in *UnsetKVReq, opts ...grpc.CallOption) (*Empty, error)
	// Watch will rev a stream response in client.
	Watch(ctx context.Context, in *WatchReq, opts ...grpc.CallOption) (Api_WatchClient, error)
}

type apiClient struct {
	cc *grpc.ClientConn
}

func NewApiClient(cc *grpc.ClientConn) ApiClient {
	return &apiClient{cc}
}

func (c *apiClient) GetKV(ctx context.Context, in *GetKVReq, opts ...grpc.CallOption) (*GetKVResp, error) {
	out := new(GetKVResp)
	err := c.cc.Invoke(ctx, "/cassem.cassemdb.api/GetKV", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *apiClient) SetKV(ctx context.Context, in *SetKVReq, opts ...grpc.CallOption) (*Empty, error) {
	out := new(Empty)
	err := c.cc.Invoke(ctx, "/cassem.cassemdb.api/SetKV", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *apiClient) UnsetKV(ctx context.Context, in *UnsetKVReq, opts ...grpc.CallOption) (*Empty, error) {
	out := new(Empty)
	err := c.cc.Invoke(ctx, "/cassem.cassemdb.api/UnsetKV", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *apiClient) Watch(ctx context.Context, in *WatchReq, opts ...grpc.CallOption) (Api_WatchClient, error) {
	stream, err := c.cc.NewStream(ctx, &_Api_serviceDesc.Streams[0], "/cassem.cassemdb.api/Watch", opts...)
	if err != nil {
		return nil, err
	}
	x := &apiWatchClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type Api_WatchClient interface {
	Recv() (*Change, error)
	grpc.ClientStream
}

type apiWatchClient struct {
	grpc.ClientStream
}

func (x *apiWatchClient) Recv() (*Change, error) {
	m := new(Change)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// ApiServer is the server API for Api service.
type ApiServer interface {
	GetKV(context.Context, *GetKVReq) (*GetKVResp, error)
	SetKV(context.Context, *SetKVReq) (*Empty, error)
	UnsetKV(context.Context, *UnsetKVReq) (*Empty, error)
	// Watch will rev a stream response in client.
	Watch(*WatchReq, Api_WatchServer) error
}

// UnimplementedApiServer can be embedded to have forward compatible implementations.
type UnimplementedApiServer struct {
}

func (*UnimplementedApiServer) GetKV(ctx context.Context, req *GetKVReq) (*GetKVResp, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetKV not implemented")
}
func (*UnimplementedApiServer) SetKV(ctx context.Context, req *SetKVReq) (*Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SetKV not implemented")
}
func (*UnimplementedApiServer) UnsetKV(ctx context.Context, req *UnsetKVReq) (*Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UnsetKV not implemented")
}
func (*UnimplementedApiServer) Watch(req *WatchReq, srv Api_WatchServer) error {
	return status.Errorf(codes.Unimplemented, "method Watch not implemented")
}

func RegisterApiServer(s *grpc.Server, srv ApiServer) {
	s.RegisterService(&_Api_serviceDesc, srv)
}

func _Api_GetKV_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetKVReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ApiServer).GetKV(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cassem.cassemdb.api/GetKV",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ApiServer).GetKV(ctx, req.(*GetKVReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _Api_SetKV_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SetKVReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ApiServer).SetKV(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cassem.cassemdb.api/SetKV",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ApiServer).SetKV(ctx, req.(*SetKVReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _Api_UnsetKV_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UnsetKVReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ApiServer).UnsetKV(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cassem.cassemdb.api/UnsetKV",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ApiServer).UnsetKV(ctx, req.(*UnsetKVReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _Api_Watch_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(WatchReq)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(ApiServer).Watch(m, &apiWatchServer{stream})
}

type Api_WatchServer interface {
	Send(*Change) error
	grpc.ServerStream
}

type apiWatchServer struct {
	grpc.ServerStream
}

func (x *apiWatchServer) Send(m *Change) error {
	return x.ServerStream.SendMsg(m)
}

var _Api_serviceDesc = grpc.ServiceDesc{
	ServiceName: "cassem.cassemdb.api",
	HandlerType: (*ApiServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetKV",
			Handler:    _Api_GetKV_Handler,
		},
		{
			MethodName: "SetKV",
			Handler:    _Api_SetKV_Handler,
		},
		{
			MethodName: "UnsetKV",
			Handler:    _Api_UnsetKV_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Watch",
			Handler:       _Api_Watch_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "api.proto",
}
