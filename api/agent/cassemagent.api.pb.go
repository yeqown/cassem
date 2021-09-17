// Code generated by protoc-gen-go. DO NOT EDIT.
// source: cassemagent.api.proto

package agent

import (
	context "context"
	fmt "fmt"
	_ "github.com/envoyproxy/protoc-gen-validate/validate"
	proto "github.com/golang/protobuf/proto"
	concept "github.com/yeqown/cassem/api/concept"
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

type GetElementReq struct {
	App                  string   `protobuf:"bytes,1,opt,name=app,proto3" json:"app,omitempty"`
	Env                  string   `protobuf:"bytes,2,opt,name=env,proto3" json:"env,omitempty"`
	Keys                 []string `protobuf:"bytes,3,rep,name=keys,proto3" json:"keys,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *GetElementReq) Reset()         { *m = GetElementReq{} }
func (m *GetElementReq) String() string { return proto.CompactTextString(m) }
func (*GetElementReq) ProtoMessage()    {}
func (*GetElementReq) Descriptor() ([]byte, []int) {
	return fileDescriptor_55c82b521937e82e, []int{0}
}

func (m *GetElementReq) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_GetElementReq.Unmarshal(m, b)
}
func (m *GetElementReq) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_GetElementReq.Marshal(b, m, deterministic)
}
func (m *GetElementReq) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GetElementReq.Merge(m, src)
}
func (m *GetElementReq) XXX_Size() int {
	return xxx_messageInfo_GetElementReq.Size(m)
}
func (m *GetElementReq) XXX_DiscardUnknown() {
	xxx_messageInfo_GetElementReq.DiscardUnknown(m)
}

var xxx_messageInfo_GetElementReq proto.InternalMessageInfo

func (m *GetElementReq) GetApp() string {
	if m != nil {
		return m.App
	}
	return ""
}

func (m *GetElementReq) GetEnv() string {
	if m != nil {
		return m.Env
	}
	return ""
}

func (m *GetElementReq) GetKeys() []string {
	if m != nil {
		return m.Keys
	}
	return nil
}

type GetElementResp struct {
	Elems                []*concept.Element `protobuf:"bytes,1,rep,name=elems,proto3" json:"elems,omitempty"`
	XXX_NoUnkeyedLiteral struct{}           `json:"-"`
	XXX_unrecognized     []byte             `json:"-"`
	XXX_sizecache        int32              `json:"-"`
}

func (m *GetElementResp) Reset()         { *m = GetElementResp{} }
func (m *GetElementResp) String() string { return proto.CompactTextString(m) }
func (*GetElementResp) ProtoMessage()    {}
func (*GetElementResp) Descriptor() ([]byte, []int) {
	return fileDescriptor_55c82b521937e82e, []int{1}
}

func (m *GetElementResp) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_GetElementResp.Unmarshal(m, b)
}
func (m *GetElementResp) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_GetElementResp.Marshal(b, m, deterministic)
}
func (m *GetElementResp) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GetElementResp.Merge(m, src)
}
func (m *GetElementResp) XXX_Size() int {
	return xxx_messageInfo_GetElementResp.Size(m)
}
func (m *GetElementResp) XXX_DiscardUnknown() {
	xxx_messageInfo_GetElementResp.DiscardUnknown(m)
}

var xxx_messageInfo_GetElementResp proto.InternalMessageInfo

func (m *GetElementResp) GetElems() []*concept.Element {
	if m != nil {
		return m.Elems
	}
	return nil
}

type UnregisterReq struct {
	ClientId             string   `protobuf:"bytes,1,opt,name=clientId,proto3" json:"clientId,omitempty"`
	ClientIp             string   `protobuf:"bytes,2,opt,name=clientIp,proto3" json:"clientIp,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *UnregisterReq) Reset()         { *m = UnregisterReq{} }
func (m *UnregisterReq) String() string { return proto.CompactTextString(m) }
func (*UnregisterReq) ProtoMessage()    {}
func (*UnregisterReq) Descriptor() ([]byte, []int) {
	return fileDescriptor_55c82b521937e82e, []int{2}
}

func (m *UnregisterReq) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_UnregisterReq.Unmarshal(m, b)
}
func (m *UnregisterReq) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_UnregisterReq.Marshal(b, m, deterministic)
}
func (m *UnregisterReq) XXX_Merge(src proto.Message) {
	xxx_messageInfo_UnregisterReq.Merge(m, src)
}
func (m *UnregisterReq) XXX_Size() int {
	return xxx_messageInfo_UnregisterReq.Size(m)
}
func (m *UnregisterReq) XXX_DiscardUnknown() {
	xxx_messageInfo_UnregisterReq.DiscardUnknown(m)
}

var xxx_messageInfo_UnregisterReq proto.InternalMessageInfo

func (m *UnregisterReq) GetClientId() string {
	if m != nil {
		return m.ClientId
	}
	return ""
}

func (m *UnregisterReq) GetClientIp() string {
	if m != nil {
		return m.ClientIp
	}
	return ""
}

type RegisterReq struct {
	ClientId             string                       `protobuf:"bytes,1,opt,name=clientId,proto3" json:"clientId,omitempty"`
	ClientIp             string                       `protobuf:"bytes,2,opt,name=clientIp,proto3" json:"clientIp,omitempty"`
	Watching             []*concept.Instance_Watching `protobuf:"bytes,3,rep,name=watching,proto3" json:"watching,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                     `json:"-"`
	XXX_unrecognized     []byte                       `json:"-"`
	XXX_sizecache        int32                        `json:"-"`
}

func (m *RegisterReq) Reset()         { *m = RegisterReq{} }
func (m *RegisterReq) String() string { return proto.CompactTextString(m) }
func (*RegisterReq) ProtoMessage()    {}
func (*RegisterReq) Descriptor() ([]byte, []int) {
	return fileDescriptor_55c82b521937e82e, []int{3}
}

func (m *RegisterReq) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_RegisterReq.Unmarshal(m, b)
}
func (m *RegisterReq) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_RegisterReq.Marshal(b, m, deterministic)
}
func (m *RegisterReq) XXX_Merge(src proto.Message) {
	xxx_messageInfo_RegisterReq.Merge(m, src)
}
func (m *RegisterReq) XXX_Size() int {
	return xxx_messageInfo_RegisterReq.Size(m)
}
func (m *RegisterReq) XXX_DiscardUnknown() {
	xxx_messageInfo_RegisterReq.DiscardUnknown(m)
}

var xxx_messageInfo_RegisterReq proto.InternalMessageInfo

func (m *RegisterReq) GetClientId() string {
	if m != nil {
		return m.ClientId
	}
	return ""
}

func (m *RegisterReq) GetClientIp() string {
	if m != nil {
		return m.ClientIp
	}
	return ""
}

func (m *RegisterReq) GetWatching() []*concept.Instance_Watching {
	if m != nil {
		return m.Watching
	}
	return nil
}

type EmptyResp struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *EmptyResp) Reset()         { *m = EmptyResp{} }
func (m *EmptyResp) String() string { return proto.CompactTextString(m) }
func (*EmptyResp) ProtoMessage()    {}
func (*EmptyResp) Descriptor() ([]byte, []int) {
	return fileDescriptor_55c82b521937e82e, []int{4}
}

func (m *EmptyResp) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_EmptyResp.Unmarshal(m, b)
}
func (m *EmptyResp) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_EmptyResp.Marshal(b, m, deterministic)
}
func (m *EmptyResp) XXX_Merge(src proto.Message) {
	xxx_messageInfo_EmptyResp.Merge(m, src)
}
func (m *EmptyResp) XXX_Size() int {
	return xxx_messageInfo_EmptyResp.Size(m)
}
func (m *EmptyResp) XXX_DiscardUnknown() {
	xxx_messageInfo_EmptyResp.DiscardUnknown(m)
}

var xxx_messageInfo_EmptyResp proto.InternalMessageInfo

type WatchReq struct {
	Watching []*concept.Instance_Watching `protobuf:"bytes,1,rep,name=watching,proto3" json:"watching,omitempty"`
	//  string app = 1 [(validate.rules).string = {min_len: 3, max_len: 30}];
	//  string env = 2 [(validate.rules).string = {min_len: 3, max_len: 30}];
	//  repeated string watchingKeys = 3 [(validate.rules).repeated = {unique: true, min_items: 1, max_items: 100}];
	ClientId             string   `protobuf:"bytes,4,opt,name=clientId,proto3" json:"clientId,omitempty"`
	ClientIp             string   `protobuf:"bytes,5,opt,name=clientIp,proto3" json:"clientIp,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *WatchReq) Reset()         { *m = WatchReq{} }
func (m *WatchReq) String() string { return proto.CompactTextString(m) }
func (*WatchReq) ProtoMessage()    {}
func (*WatchReq) Descriptor() ([]byte, []int) {
	return fileDescriptor_55c82b521937e82e, []int{5}
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

func (m *WatchReq) GetWatching() []*concept.Instance_Watching {
	if m != nil {
		return m.Watching
	}
	return nil
}

func (m *WatchReq) GetClientId() string {
	if m != nil {
		return m.ClientId
	}
	return ""
}

func (m *WatchReq) GetClientIp() string {
	if m != nil {
		return m.ClientIp
	}
	return ""
}

type WatchResp struct {
	Elem                 *concept.Element `protobuf:"bytes,1,opt,name=elem,proto3" json:"elem,omitempty"`
	XXX_NoUnkeyedLiteral struct{}         `json:"-"`
	XXX_unrecognized     []byte           `json:"-"`
	XXX_sizecache        int32            `json:"-"`
}

func (m *WatchResp) Reset()         { *m = WatchResp{} }
func (m *WatchResp) String() string { return proto.CompactTextString(m) }
func (*WatchResp) ProtoMessage()    {}
func (*WatchResp) Descriptor() ([]byte, []int) {
	return fileDescriptor_55c82b521937e82e, []int{6}
}

func (m *WatchResp) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_WatchResp.Unmarshal(m, b)
}
func (m *WatchResp) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_WatchResp.Marshal(b, m, deterministic)
}
func (m *WatchResp) XXX_Merge(src proto.Message) {
	xxx_messageInfo_WatchResp.Merge(m, src)
}
func (m *WatchResp) XXX_Size() int {
	return xxx_messageInfo_WatchResp.Size(m)
}
func (m *WatchResp) XXX_DiscardUnknown() {
	xxx_messageInfo_WatchResp.DiscardUnknown(m)
}

var xxx_messageInfo_WatchResp proto.InternalMessageInfo

func (m *WatchResp) GetElem() *concept.Element {
	if m != nil {
		return m.Elem
	}
	return nil
}

type DispatchReq struct {
	Elems                []*concept.Element `protobuf:"bytes,1,rep,name=elems,proto3" json:"elems,omitempty"`
	XXX_NoUnkeyedLiteral struct{}           `json:"-"`
	XXX_unrecognized     []byte             `json:"-"`
	XXX_sizecache        int32              `json:"-"`
}

func (m *DispatchReq) Reset()         { *m = DispatchReq{} }
func (m *DispatchReq) String() string { return proto.CompactTextString(m) }
func (*DispatchReq) ProtoMessage()    {}
func (*DispatchReq) Descriptor() ([]byte, []int) {
	return fileDescriptor_55c82b521937e82e, []int{7}
}

func (m *DispatchReq) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_DispatchReq.Unmarshal(m, b)
}
func (m *DispatchReq) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_DispatchReq.Marshal(b, m, deterministic)
}
func (m *DispatchReq) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DispatchReq.Merge(m, src)
}
func (m *DispatchReq) XXX_Size() int {
	return xxx_messageInfo_DispatchReq.Size(m)
}
func (m *DispatchReq) XXX_DiscardUnknown() {
	xxx_messageInfo_DispatchReq.DiscardUnknown(m)
}

var xxx_messageInfo_DispatchReq proto.InternalMessageInfo

func (m *DispatchReq) GetElems() []*concept.Element {
	if m != nil {
		return m.Elems
	}
	return nil
}

type DispatchResp struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *DispatchResp) Reset()         { *m = DispatchResp{} }
func (m *DispatchResp) String() string { return proto.CompactTextString(m) }
func (*DispatchResp) ProtoMessage()    {}
func (*DispatchResp) Descriptor() ([]byte, []int) {
	return fileDescriptor_55c82b521937e82e, []int{8}
}

func (m *DispatchResp) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_DispatchResp.Unmarshal(m, b)
}
func (m *DispatchResp) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_DispatchResp.Marshal(b, m, deterministic)
}
func (m *DispatchResp) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DispatchResp.Merge(m, src)
}
func (m *DispatchResp) XXX_Size() int {
	return xxx_messageInfo_DispatchResp.Size(m)
}
func (m *DispatchResp) XXX_DiscardUnknown() {
	xxx_messageInfo_DispatchResp.DiscardUnknown(m)
}

var xxx_messageInfo_DispatchResp proto.InternalMessageInfo

func init() {
	proto.RegisterType((*GetElementReq)(nil), "cassem.agent.getElementReq")
	proto.RegisterType((*GetElementResp)(nil), "cassem.agent.getElementResp")
	proto.RegisterType((*UnregisterReq)(nil), "cassem.agent.unregisterReq")
	proto.RegisterType((*RegisterReq)(nil), "cassem.agent.registerReq")
	proto.RegisterType((*EmptyResp)(nil), "cassem.agent.emptyResp")
	proto.RegisterType((*WatchReq)(nil), "cassem.agent.watchReq")
	proto.RegisterType((*WatchResp)(nil), "cassem.agent.watchResp")
	proto.RegisterType((*DispatchReq)(nil), "cassem.agent.dispatchReq")
	proto.RegisterType((*DispatchResp)(nil), "cassem.agent.dispatchResp")
}

func init() { proto.RegisterFile("cassemagent.api.proto", fileDescriptor_55c82b521937e82e) }

var fileDescriptor_55c82b521937e82e = []byte{
	// 506 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xb4, 0x54, 0x5d, 0x6b, 0xd4, 0x40,
	0x14, 0xdd, 0xd9, 0xdd, 0x6c, 0x93, 0x9b, 0x6d, 0x29, 0x23, 0xda, 0x98, 0x8a, 0xac, 0x23, 0xc2,
	0x82, 0x34, 0x95, 0xf5, 0xc5, 0x87, 0x8a, 0x25, 0x56, 0xca, 0x3e, 0x09, 0x01, 0x11, 0xf4, 0x29,
	0x26, 0x97, 0x35, 0x34, 0x3b, 0x19, 0x33, 0x63, 0x6a, 0xfe, 0x46, 0xff, 0x80, 0xe0, 0x9f, 0xf2,
	0xbf, 0xec, 0x93, 0xe4, 0xb3, 0x9b, 0x95, 0x95, 0xaa, 0xf4, 0x2d, 0xcc, 0x39, 0xf7, 0xce, 0x3d,
	0xf7, 0x9c, 0x09, 0xdc, 0x0d, 0x7c, 0x29, 0x71, 0xe9, 0x2f, 0x90, 0x2b, 0xc7, 0x17, 0x91, 0x23,
	0xd2, 0x44, 0x25, 0x74, 0x5c, 0x1d, 0x3b, 0xe5, 0xb9, 0x7d, 0x27, 0x48, 0x78, 0x80, 0x42, 0x1d,
	0xab, 0x5c, 0xa0, 0xac, 0x28, 0x36, 0x43, 0x9e, 0x25, 0xb9, 0x48, 0x93, 0x6f, 0xf9, 0x51, 0xe6,
	0xc7, 0x51, 0xe8, 0x2b, 0x3c, 0x6e, 0x3e, 0x2a, 0x0e, 0xbb, 0x80, 0xdd, 0x05, 0xaa, 0x37, 0x31,
	0x2e, 0x91, 0x2b, 0x0f, 0xbf, 0xd0, 0x43, 0x18, 0xf8, 0x42, 0x58, 0x64, 0x42, 0xa6, 0x86, 0x6b,
	0xac, 0xdc, 0x51, 0x3a, 0xdc, 0x1f, 0x58, 0x0f, 0xbd, 0xe2, 0xb4, 0x00, 0x91, 0x67, 0x56, 0xff,
	0x37, 0x10, 0x79, 0x46, 0x27, 0x30, 0xbc, 0xc0, 0x5c, 0x5a, 0x83, 0xc9, 0x60, 0x6a, 0xb8, 0xe3,
	0x95, 0x6b, 0x5c, 0x91, 0x91, 0x4e, 0xf6, 0x43, 0x8b, 0x78, 0x25, 0xc2, 0x5e, 0xc1, 0xde, 0xfa,
	0x65, 0x52, 0xd0, 0x23, 0xd0, 0x30, 0xc6, 0xa5, 0xb4, 0xc8, 0x64, 0x30, 0x35, 0x67, 0x07, 0x4e,
	0xad, 0xaa, 0x96, 0xe3, 0x34, 0xdc, 0x8a, 0xc5, 0x3e, 0xc2, 0xee, 0x57, 0x9e, 0xe2, 0x22, 0x92,
	0x0a, 0xd3, 0x62, 0xda, 0x27, 0xa0, 0x07, 0x71, 0x84, 0x5c, 0xcd, 0xc3, 0xee, 0xc8, 0x9a, 0x75,
	0xea, 0xb5, 0x10, 0x7d, 0xdc, 0xd2, 0x44, 0x3d, 0xfc, 0xce, 0xca, 0x1d, 0xa6, 0x7d, 0x41, 0x5a,
	0x92, 0x60, 0x3f, 0x08, 0x98, 0xb7, 0xd4, 0x9b, 0x9e, 0x83, 0x7e, 0xe9, 0xab, 0xe0, 0x73, 0xc4,
	0x17, 0xe5, 0x7e, 0xcc, 0xd9, 0xa3, 0x4d, 0xa9, 0x73, 0x2e, 0x95, 0xcf, 0x03, 0x74, 0xde, 0xd7,
	0x44, 0x57, 0x5f, 0xb9, 0xda, 0x15, 0xe9, 0x4f, 0x89, 0xd7, 0x16, 0x33, 0x13, 0x0c, 0x5c, 0x0a,
	0x95, 0x17, 0xdb, 0x63, 0xdf, 0x49, 0xdd, 0xb6, 0x18, 0x77, 0xfd, 0x0a, 0xf2, 0x1f, 0x57, 0x74,
	0x74, 0x0f, 0x6f, 0xa6, 0x5b, 0xdb, 0xb6, 0xd3, 0x17, 0x60, 0xd4, 0x03, 0x4a, 0x41, 0x9f, 0xc2,
	0xb0, 0xb0, 0xb1, 0x5c, 0xe6, 0x1f, 0xbc, 0x2e, 0x49, 0xec, 0x04, 0xcc, 0x30, 0x92, 0xa2, 0x51,
	0xf7, 0x97, 0x41, 0xd9, 0x83, 0xf1, 0x75, 0xb5, 0x14, 0xb3, 0x9f, 0x7d, 0xd0, 0xca, 0x97, 0x42,
	0xe7, 0x00, 0xe7, 0x6d, 0x06, 0xe9, 0xa1, 0xb3, 0xfe, 0x8c, 0x9c, 0xce, 0x53, 0xb0, 0x1f, 0x6c,
	0x07, 0xa5, 0x60, 0x3d, 0x7a, 0x06, 0xf0, 0xae, 0x4d, 0xe3, 0x66, 0xab, 0x4e, 0x4e, 0xed, 0x83,
	0x2e, 0x78, 0x6d, 0x61, 0x8f, 0x9e, 0x82, 0xee, 0x35, 0x3d, 0xee, 0x77, 0x69, 0x37, 0xec, 0xf0,
	0x12, 0x34, 0x0f, 0x39, 0x5e, 0xfe, 0x63, 0xf9, 0x09, 0x68, 0x65, 0x1e, 0xe8, 0xbd, 0x2e, 0xa7,
	0x49, 0xd6, 0x66, 0x6d, 0x6b, 0x28, 0xeb, 0x3d, 0x23, 0xb3, 0xb7, 0xa0, 0x87, 0x18, 0x47, 0x19,
	0xa6, 0x39, 0x7d, 0x0d, 0xfa, 0x59, 0xbd, 0xf5, 0xcd, 0x59, 0xd6, 0xbc, 0xb4, 0xed, 0x6d, 0x50,
	0xd1, 0xd2, 0xdd, 0xf9, 0x50, 0x39, 0xf5, 0x69, 0x54, 0xfe, 0xa1, 0x9e, 0xff, 0x0a, 0x00, 0x00,
	0xff, 0xff, 0xde, 0xa1, 0x8c, 0xdf, 0x01, 0x05, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// AgentClient is the client API for Agent service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type AgentClient interface {
	GetElement(ctx context.Context, in *GetElementReq, opts ...grpc.CallOption) (*GetElementResp, error)
	Unregister(ctx context.Context, in *UnregisterReq, opts ...grpc.CallOption) (*EmptyResp, error)
	Register(ctx context.Context, in *RegisterReq, opts ...grpc.CallOption) (*EmptyResp, error)
	Renew(ctx context.Context, in *RegisterReq, opts ...grpc.CallOption) (*EmptyResp, error)
	Watch(ctx context.Context, in *WatchReq, opts ...grpc.CallOption) (Agent_WatchClient, error)
}

type agentClient struct {
	cc *grpc.ClientConn
}

func NewAgentClient(cc *grpc.ClientConn) AgentClient {
	return &agentClient{cc}
}

func (c *agentClient) GetElement(ctx context.Context, in *GetElementReq, opts ...grpc.CallOption) (*GetElementResp, error) {
	out := new(GetElementResp)
	err := c.cc.Invoke(ctx, "/cassem.agent.agent/GetElement", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *agentClient) Unregister(ctx context.Context, in *UnregisterReq, opts ...grpc.CallOption) (*EmptyResp, error) {
	out := new(EmptyResp)
	err := c.cc.Invoke(ctx, "/cassem.agent.agent/Unregister", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *agentClient) Register(ctx context.Context, in *RegisterReq, opts ...grpc.CallOption) (*EmptyResp, error) {
	out := new(EmptyResp)
	err := c.cc.Invoke(ctx, "/cassem.agent.agent/Register", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *agentClient) Renew(ctx context.Context, in *RegisterReq, opts ...grpc.CallOption) (*EmptyResp, error) {
	out := new(EmptyResp)
	err := c.cc.Invoke(ctx, "/cassem.agent.agent/Renew", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *agentClient) Watch(ctx context.Context, in *WatchReq, opts ...grpc.CallOption) (Agent_WatchClient, error) {
	stream, err := c.cc.NewStream(ctx, &_Agent_serviceDesc.Streams[0], "/cassem.agent.agent/Watch", opts...)
	if err != nil {
		return nil, err
	}
	x := &agentWatchClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type Agent_WatchClient interface {
	Recv() (*WatchResp, error)
	grpc.ClientStream
}

type agentWatchClient struct {
	grpc.ClientStream
}

func (x *agentWatchClient) Recv() (*WatchResp, error) {
	m := new(WatchResp)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// AgentServer is the server API for Agent service.
type AgentServer interface {
	GetElement(context.Context, *GetElementReq) (*GetElementResp, error)
	Unregister(context.Context, *UnregisterReq) (*EmptyResp, error)
	Register(context.Context, *RegisterReq) (*EmptyResp, error)
	Renew(context.Context, *RegisterReq) (*EmptyResp, error)
	Watch(*WatchReq, Agent_WatchServer) error
}

// UnimplementedAgentServer can be embedded to have forward compatible implementations.
type UnimplementedAgentServer struct {
}

func (*UnimplementedAgentServer) GetElement(ctx context.Context, req *GetElementReq) (*GetElementResp, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetElement not implemented")
}
func (*UnimplementedAgentServer) Unregister(ctx context.Context, req *UnregisterReq) (*EmptyResp, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Unregister not implemented")
}
func (*UnimplementedAgentServer) Register(ctx context.Context, req *RegisterReq) (*EmptyResp, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Register not implemented")
}
func (*UnimplementedAgentServer) Renew(ctx context.Context, req *RegisterReq) (*EmptyResp, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Renew not implemented")
}
func (*UnimplementedAgentServer) Watch(req *WatchReq, srv Agent_WatchServer) error {
	return status.Errorf(codes.Unimplemented, "method Watch not implemented")
}

func RegisterAgentServer(s *grpc.Server, srv AgentServer) {
	s.RegisterService(&_Agent_serviceDesc, srv)
}

func _Agent_GetElement_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetElementReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AgentServer).GetElement(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cassem.agent.agent/GetElement",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AgentServer).GetElement(ctx, req.(*GetElementReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _Agent_Unregister_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UnregisterReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AgentServer).Unregister(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cassem.agent.agent/Unregister",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AgentServer).Unregister(ctx, req.(*UnregisterReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _Agent_Register_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RegisterReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AgentServer).Register(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cassem.agent.agent/Register",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AgentServer).Register(ctx, req.(*RegisterReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _Agent_Renew_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RegisterReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AgentServer).Renew(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cassem.agent.agent/Renew",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AgentServer).Renew(ctx, req.(*RegisterReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _Agent_Watch_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(WatchReq)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(AgentServer).Watch(m, &agentWatchServer{stream})
}

type Agent_WatchServer interface {
	Send(*WatchResp) error
	grpc.ServerStream
}

type agentWatchServer struct {
	grpc.ServerStream
}

func (x *agentWatchServer) Send(m *WatchResp) error {
	return x.ServerStream.SendMsg(m)
}

var _Agent_serviceDesc = grpc.ServiceDesc{
	ServiceName: "cassem.agent.agent",
	HandlerType: (*AgentServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetElement",
			Handler:    _Agent_GetElement_Handler,
		},
		{
			MethodName: "Unregister",
			Handler:    _Agent_Unregister_Handler,
		},
		{
			MethodName: "Register",
			Handler:    _Agent_Register_Handler,
		},
		{
			MethodName: "Renew",
			Handler:    _Agent_Renew_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Watch",
			Handler:       _Agent_Watch_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "cassemagent.api.proto",
}

// DeliveryClient is the client API for Delivery service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type DeliveryClient interface {
	Dispatch(ctx context.Context, in *DispatchReq, opts ...grpc.CallOption) (*DispatchResp, error)
}

type deliveryClient struct {
	cc *grpc.ClientConn
}

func NewDeliveryClient(cc *grpc.ClientConn) DeliveryClient {
	return &deliveryClient{cc}
}

func (c *deliveryClient) Dispatch(ctx context.Context, in *DispatchReq, opts ...grpc.CallOption) (*DispatchResp, error) {
	out := new(DispatchResp)
	err := c.cc.Invoke(ctx, "/cassem.agent.delivery/Dispatch", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// DeliveryServer is the server API for Delivery service.
type DeliveryServer interface {
	Dispatch(context.Context, *DispatchReq) (*DispatchResp, error)
}

// UnimplementedDeliveryServer can be embedded to have forward compatible implementations.
type UnimplementedDeliveryServer struct {
}

func (*UnimplementedDeliveryServer) Dispatch(ctx context.Context, req *DispatchReq) (*DispatchResp, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Dispatch not implemented")
}

func RegisterDeliveryServer(s *grpc.Server, srv DeliveryServer) {
	s.RegisterService(&_Delivery_serviceDesc, srv)
}

func _Delivery_Dispatch_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DispatchReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DeliveryServer).Dispatch(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cassem.agent.delivery/Dispatch",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DeliveryServer).Dispatch(ctx, req.(*DispatchReq))
	}
	return interceptor(ctx, in, info, handler)
}

var _Delivery_serviceDesc = grpc.ServiceDesc{
	ServiceName: "cassem.agent.delivery",
	HandlerType: (*DeliveryServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Dispatch",
			Handler:    _Delivery_Dispatch_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "cassemagent.api.proto",
}