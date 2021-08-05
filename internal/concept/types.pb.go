// Code generated by protoc-gen-go. DO NOT EDIT.
// source: types.proto

package concept

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

type ContentType int32

const (
	ContentType_UNKNOWN   ContentType = 0
	ContentType_JSON      ContentType = 1
	ContentType_TOML      ContentType = 2
	ContentType_INI       ContentType = 3
	ContentType_PLAINTEXT ContentType = 4
)

var ContentType_name = map[int32]string{
	0: "UNKNOWN",
	1: "JSON",
	2: "TOML",
	3: "INI",
	4: "PLAINTEXT",
}

var ContentType_value = map[string]int32{
	"UNKNOWN":   0,
	"JSON":      1,
	"TOML":      2,
	"INI":       3,
	"PLAINTEXT": 4,
}

func (x ContentType) String() string {
	return proto.EnumName(ContentType_name, int32(x))
}

func (ContentType) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_d938547f84707355, []int{0}
}

type Operation int32

const (
	Operation_INVALID Operation = 0
	Operation_SET     Operation = 1
	Operation_UNSET   Operation = 2
	Operation_PUBLISH Operation = 3
)

var Operation_name = map[int32]string{
	0: "INVALID",
	1: "SET",
	2: "UNSET",
	3: "PUBLISH",
}

var Operation_value = map[string]int32{
	"INVALID": 0,
	"SET":     1,
	"UNSET":   2,
	"PUBLISH": 3,
}

func (x Operation) String() string {
	return proto.EnumName(Operation_name, int32(x))
}

func (Operation) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_d938547f84707355, []int{1}
}

type Element struct {
	Metadata             *ElementMetadata `protobuf:"bytes,1,opt,name=metadata,proto3" json:"metadata,omitempty"`
	Raw                  []byte           `protobuf:"bytes,2,opt,name=raw,proto3" json:"raw,omitempty"`
	Version              int32            `protobuf:"varint,3,opt,name=version,proto3" json:"version,omitempty"`
	XXX_NoUnkeyedLiteral struct{}         `json:"-"`
	XXX_unrecognized     []byte           `json:"-"`
	XXX_sizecache        int32            `json:"-"`
}

func (m *Element) Reset()         { *m = Element{} }
func (m *Element) String() string { return proto.CompactTextString(m) }
func (*Element) ProtoMessage()    {}
func (*Element) Descriptor() ([]byte, []int) {
	return fileDescriptor_d938547f84707355, []int{0}
}

func (m *Element) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Element.Unmarshal(m, b)
}
func (m *Element) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Element.Marshal(b, m, deterministic)
}
func (m *Element) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Element.Merge(m, src)
}
func (m *Element) XXX_Size() int {
	return xxx_messageInfo_Element.Size(m)
}
func (m *Element) XXX_DiscardUnknown() {
	xxx_messageInfo_Element.DiscardUnknown(m)
}

var xxx_messageInfo_Element proto.InternalMessageInfo

func (m *Element) GetMetadata() *ElementMetadata {
	if m != nil {
		return m.Metadata
	}
	return nil
}

func (m *Element) GetRaw() []byte {
	if m != nil {
		return m.Raw
	}
	return nil
}

func (m *Element) GetVersion() int32 {
	if m != nil {
		return m.Version
	}
	return 0
}

type ElementMetadata struct {
	Key                  string      `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
	App                  string      `protobuf:"bytes,2,opt,name=app,proto3" json:"app,omitempty"`
	Env                  string      `protobuf:"bytes,3,opt,name=env,proto3" json:"env,omitempty"`
	LatestVersion        int32       `protobuf:"varint,4,opt,name=latest_version,json=latestVersion,proto3" json:"latest_version,omitempty"`
	LatestFingerprint    string      `protobuf:"bytes,5,opt,name=latest_fingerprint,json=latestFingerprint,proto3" json:"latest_fingerprint,omitempty"`
	ContentType          ContentType `protobuf:"varint,6,opt,name=content_type,json=contentType,proto3,enum=cassem.concept.ContentType" json:"content_type,omitempty"`
	XXX_NoUnkeyedLiteral struct{}    `json:"-"`
	XXX_unrecognized     []byte      `json:"-"`
	XXX_sizecache        int32       `json:"-"`
}

func (m *ElementMetadata) Reset()         { *m = ElementMetadata{} }
func (m *ElementMetadata) String() string { return proto.CompactTextString(m) }
func (*ElementMetadata) ProtoMessage()    {}
func (*ElementMetadata) Descriptor() ([]byte, []int) {
	return fileDescriptor_d938547f84707355, []int{1}
}

func (m *ElementMetadata) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ElementMetadata.Unmarshal(m, b)
}
func (m *ElementMetadata) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ElementMetadata.Marshal(b, m, deterministic)
}
func (m *ElementMetadata) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ElementMetadata.Merge(m, src)
}
func (m *ElementMetadata) XXX_Size() int {
	return xxx_messageInfo_ElementMetadata.Size(m)
}
func (m *ElementMetadata) XXX_DiscardUnknown() {
	xxx_messageInfo_ElementMetadata.DiscardUnknown(m)
}

var xxx_messageInfo_ElementMetadata proto.InternalMessageInfo

func (m *ElementMetadata) GetKey() string {
	if m != nil {
		return m.Key
	}
	return ""
}

func (m *ElementMetadata) GetApp() string {
	if m != nil {
		return m.App
	}
	return ""
}

func (m *ElementMetadata) GetEnv() string {
	if m != nil {
		return m.Env
	}
	return ""
}

func (m *ElementMetadata) GetLatestVersion() int32 {
	if m != nil {
		return m.LatestVersion
	}
	return 0
}

func (m *ElementMetadata) GetLatestFingerprint() string {
	if m != nil {
		return m.LatestFingerprint
	}
	return ""
}

func (m *ElementMetadata) GetContentType() ContentType {
	if m != nil {
		return m.ContentType
	}
	return ContentType_UNKNOWN
}

type AppMetadata struct {
	Id                   string   `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Description          string   `protobuf:"bytes,2,opt,name=description,proto3" json:"description,omitempty"`
	CreatedAt            int64    `protobuf:"varint,3,opt,name=created_at,json=createdAt,proto3" json:"created_at,omitempty"`
	Creator              string   `protobuf:"bytes,4,opt,name=creator,proto3" json:"creator,omitempty"`
	Owner                string   `protobuf:"bytes,5,opt,name=owner,proto3" json:"owner,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *AppMetadata) Reset()         { *m = AppMetadata{} }
func (m *AppMetadata) String() string { return proto.CompactTextString(m) }
func (*AppMetadata) ProtoMessage()    {}
func (*AppMetadata) Descriptor() ([]byte, []int) {
	return fileDescriptor_d938547f84707355, []int{2}
}

func (m *AppMetadata) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_AppMetadata.Unmarshal(m, b)
}
func (m *AppMetadata) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_AppMetadata.Marshal(b, m, deterministic)
}
func (m *AppMetadata) XXX_Merge(src proto.Message) {
	xxx_messageInfo_AppMetadata.Merge(m, src)
}
func (m *AppMetadata) XXX_Size() int {
	return xxx_messageInfo_AppMetadata.Size(m)
}
func (m *AppMetadata) XXX_DiscardUnknown() {
	xxx_messageInfo_AppMetadata.DiscardUnknown(m)
}

var xxx_messageInfo_AppMetadata proto.InternalMessageInfo

func (m *AppMetadata) GetId() string {
	if m != nil {
		return m.Id
	}
	return ""
}

func (m *AppMetadata) GetDescription() string {
	if m != nil {
		return m.Description
	}
	return ""
}

func (m *AppMetadata) GetCreatedAt() int64 {
	if m != nil {
		return m.CreatedAt
	}
	return 0
}

func (m *AppMetadata) GetCreator() string {
	if m != nil {
		return m.Creator
	}
	return ""
}

func (m *AppMetadata) GetOwner() string {
	if m != nil {
		return m.Owner
	}
	return ""
}

type ElementOperation struct {
	Operator             string    `protobuf:"bytes,1,opt,name=operator,proto3" json:"operator,omitempty"`
	OperatedAt           int64     `protobuf:"varint,2,opt,name=operated_at,json=operatedAt,proto3" json:"operated_at,omitempty"`
	OperatedKey          string    `protobuf:"bytes,3,opt,name=operated_key,json=operatedKey,proto3" json:"operated_key,omitempty"`
	Op                   Operation `protobuf:"varint,4,opt,name=op,proto3,enum=cassem.concept.Operation" json:"op,omitempty"`
	LastVersion          int32     `protobuf:"varint,5,opt,name=last_version,json=lastVersion,proto3" json:"last_version,omitempty"`
	CurrentVersion       int32     `protobuf:"varint,6,opt,name=current_version,json=currentVersion,proto3" json:"current_version,omitempty"`
	Remark               string    `protobuf:"bytes,7,opt,name=remark,proto3" json:"remark,omitempty"`
	XXX_NoUnkeyedLiteral struct{}  `json:"-"`
	XXX_unrecognized     []byte    `json:"-"`
	XXX_sizecache        int32     `json:"-"`
}

func (m *ElementOperation) Reset()         { *m = ElementOperation{} }
func (m *ElementOperation) String() string { return proto.CompactTextString(m) }
func (*ElementOperation) ProtoMessage()    {}
func (*ElementOperation) Descriptor() ([]byte, []int) {
	return fileDescriptor_d938547f84707355, []int{3}
}

func (m *ElementOperation) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ElementOperation.Unmarshal(m, b)
}
func (m *ElementOperation) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ElementOperation.Marshal(b, m, deterministic)
}
func (m *ElementOperation) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ElementOperation.Merge(m, src)
}
func (m *ElementOperation) XXX_Size() int {
	return xxx_messageInfo_ElementOperation.Size(m)
}
func (m *ElementOperation) XXX_DiscardUnknown() {
	xxx_messageInfo_ElementOperation.DiscardUnknown(m)
}

var xxx_messageInfo_ElementOperation proto.InternalMessageInfo

func (m *ElementOperation) GetOperator() string {
	if m != nil {
		return m.Operator
	}
	return ""
}

func (m *ElementOperation) GetOperatedAt() int64 {
	if m != nil {
		return m.OperatedAt
	}
	return 0
}

func (m *ElementOperation) GetOperatedKey() string {
	if m != nil {
		return m.OperatedKey
	}
	return ""
}

func (m *ElementOperation) GetOp() Operation {
	if m != nil {
		return m.Op
	}
	return Operation_INVALID
}

func (m *ElementOperation) GetLastVersion() int32 {
	if m != nil {
		return m.LastVersion
	}
	return 0
}

func (m *ElementOperation) GetCurrentVersion() int32 {
	if m != nil {
		return m.CurrentVersion
	}
	return 0
}

func (m *ElementOperation) GetRemark() string {
	if m != nil {
		return m.Remark
	}
	return ""
}

type Instance struct {
	ClientId             string   `protobuf:"bytes,1,opt,name=client_id,json=clientId,proto3" json:"client_id,omitempty"`
	AgentId              string   `protobuf:"bytes,2,opt,name=agent_id,json=agentId,proto3" json:"agent_id,omitempty"`
	Ip                   string   `protobuf:"bytes,3,opt,name=ip,proto3" json:"ip,omitempty"`
	App                  string   `protobuf:"bytes,4,opt,name=app,proto3" json:"app,omitempty"`
	Env                  string   `protobuf:"bytes,5,opt,name=env,proto3" json:"env,omitempty"`
	WatchKeys            []string `protobuf:"bytes,6,rep,name=watch_keys,json=watchKeys,proto3" json:"watch_keys,omitempty"`
	LastJoinTimestamp    int64    `protobuf:"varint,7,opt,name=last_join_timestamp,json=lastJoinTimestamp,proto3" json:"last_join_timestamp,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Instance) Reset()         { *m = Instance{} }
func (m *Instance) String() string { return proto.CompactTextString(m) }
func (*Instance) ProtoMessage()    {}
func (*Instance) Descriptor() ([]byte, []int) {
	return fileDescriptor_d938547f84707355, []int{4}
}

func (m *Instance) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Instance.Unmarshal(m, b)
}
func (m *Instance) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Instance.Marshal(b, m, deterministic)
}
func (m *Instance) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Instance.Merge(m, src)
}
func (m *Instance) XXX_Size() int {
	return xxx_messageInfo_Instance.Size(m)
}
func (m *Instance) XXX_DiscardUnknown() {
	xxx_messageInfo_Instance.DiscardUnknown(m)
}

var xxx_messageInfo_Instance proto.InternalMessageInfo

func (m *Instance) GetClientId() string {
	if m != nil {
		return m.ClientId
	}
	return ""
}

func (m *Instance) GetAgentId() string {
	if m != nil {
		return m.AgentId
	}
	return ""
}

func (m *Instance) GetIp() string {
	if m != nil {
		return m.Ip
	}
	return ""
}

func (m *Instance) GetApp() string {
	if m != nil {
		return m.App
	}
	return ""
}

func (m *Instance) GetEnv() string {
	if m != nil {
		return m.Env
	}
	return ""
}

func (m *Instance) GetWatchKeys() []string {
	if m != nil {
		return m.WatchKeys
	}
	return nil
}

func (m *Instance) GetLastJoinTimestamp() int64 {
	if m != nil {
		return m.LastJoinTimestamp
	}
	return 0
}

func init() {
	proto.RegisterEnum("cassem.concept.ContentType", ContentType_name, ContentType_value)
	proto.RegisterEnum("cassem.concept.Operation", Operation_name, Operation_value)
	proto.RegisterType((*Element)(nil), "cassem.concept.Element")
	proto.RegisterType((*ElementMetadata)(nil), "cassem.concept.ElementMetadata")
	proto.RegisterType((*AppMetadata)(nil), "cassem.concept.AppMetadata")
	proto.RegisterType((*ElementOperation)(nil), "cassem.concept.ElementOperation")
	proto.RegisterType((*Instance)(nil), "cassem.concept.Instance")
}

func init() { proto.RegisterFile("types.proto", fileDescriptor_d938547f84707355) }

var fileDescriptor_d938547f84707355 = []byte{
	// 637 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x64, 0x54, 0xd1, 0x6e, 0xd3, 0x30,
	0x14, 0x5d, 0x92, 0xb6, 0x69, 0x6e, 0xb6, 0x2e, 0x18, 0x84, 0x32, 0xa6, 0x69, 0xa5, 0x12, 0xa2,
	0x4c, 0xa2, 0x0f, 0xe3, 0x09, 0x21, 0x21, 0x75, 0xb0, 0x89, 0x6c, 0x5d, 0x3a, 0x65, 0xdd, 0x40,
	0xbc, 0x54, 0x26, 0x31, 0x23, 0xac, 0xb1, 0x2d, 0xc7, 0x6c, 0xea, 0x23, 0x3f, 0xc0, 0x5f, 0xf1,
	0x19, 0xfc, 0x0b, 0xb2, 0xe3, 0xa4, 0x63, 0xbc, 0xdd, 0x73, 0x7c, 0x7c, 0x7d, 0xef, 0x3d, 0x57,
	0x06, 0x5f, 0x2e, 0x39, 0x29, 0x47, 0x5c, 0x30, 0xc9, 0x50, 0x2f, 0xc5, 0x65, 0x49, 0x8a, 0x51,
	0xca, 0x68, 0x4a, 0xb8, 0x1c, 0x08, 0x70, 0x0f, 0x17, 0xa4, 0x20, 0x54, 0xa2, 0x37, 0xd0, 0x2d,
	0x88, 0xc4, 0x19, 0x96, 0x38, 0xb4, 0xfa, 0xd6, 0xd0, 0xdf, 0xdf, 0x1d, 0xfd, 0xab, 0x1e, 0x19,
	0xe9, 0xa9, 0x91, 0x25, 0xcd, 0x05, 0x14, 0x80, 0x23, 0xf0, 0x6d, 0x68, 0xf7, 0xad, 0xe1, 0x7a,
	0xa2, 0x42, 0x14, 0x82, 0x7b, 0x43, 0x44, 0x99, 0x33, 0x1a, 0x3a, 0x7d, 0x6b, 0xd8, 0x4e, 0x6a,
	0x38, 0xf8, 0x63, 0xc1, 0xe6, 0xbd, 0x4c, 0xea, 0xfe, 0x35, 0x59, 0xea, 0x77, 0xbd, 0x44, 0x85,
	0x8a, 0xc1, 0x9c, 0xeb, 0x8c, 0x5e, 0xa2, 0x42, 0xc5, 0x10, 0x7a, 0xa3, 0xb3, 0x79, 0x89, 0x0a,
	0xd1, 0x33, 0xe8, 0x2d, 0xb0, 0x24, 0xa5, 0x9c, 0xd7, 0x4f, 0xb5, 0xf4, 0x53, 0x1b, 0x15, 0x7b,
	0x59, 0x91, 0xe8, 0x25, 0x20, 0x23, 0xfb, 0x9a, 0xd3, 0x2b, 0x22, 0xb8, 0xc8, 0xa9, 0x0c, 0xdb,
	0x3a, 0xcf, 0x83, 0xea, 0xe4, 0x68, 0x75, 0x80, 0xde, 0xc2, 0x7a, 0xca, 0xa8, 0x24, 0x54, 0xce,
	0xd5, 0xe8, 0xc2, 0x4e, 0xdf, 0x1a, 0xf6, 0xf6, 0xb7, 0xef, 0x0f, 0xe3, 0x5d, 0xa5, 0x99, 0x2d,
	0x39, 0x49, 0xfc, 0x74, 0x05, 0x06, 0xbf, 0x2c, 0xf0, 0xc7, 0x9c, 0x37, 0xbd, 0xf5, 0xc0, 0xce,
	0x33, 0xd3, 0x9a, 0x9d, 0x67, 0xa8, 0x0f, 0x7e, 0x46, 0xca, 0x54, 0xe4, 0x5c, 0xaa, 0x92, 0xab,
	0x0e, 0xef, 0x52, 0x68, 0x07, 0x20, 0x15, 0x04, 0x4b, 0x92, 0xcd, 0xb1, 0xd4, 0x0d, 0x3b, 0x89,
	0x67, 0x98, 0xb1, 0x54, 0xa3, 0xd5, 0x80, 0x09, 0xdd, 0xaf, 0x97, 0xd4, 0x10, 0x3d, 0x82, 0x36,
	0xbb, 0xa5, 0x44, 0x98, 0xe6, 0x2a, 0x30, 0xf8, 0x69, 0x43, 0x60, 0x06, 0x3e, 0xe5, 0x44, 0x60,
	0xfd, 0xc6, 0x13, 0xe8, 0x32, 0x0d, 0x98, 0x30, 0xb5, 0x35, 0x18, 0xed, 0x82, 0x5f, 0xc5, 0x55,
	0x01, 0xb6, 0x2e, 0x00, 0x6a, 0x6a, 0x2c, 0xd1, 0x53, 0x58, 0x6f, 0x04, 0xca, 0xb7, 0xca, 0x93,
	0xe6, 0xd2, 0x09, 0x59, 0xa2, 0x17, 0x60, 0x33, 0xae, 0xeb, 0xeb, 0xed, 0x6f, 0xdd, 0x9f, 0x5d,
	0x53, 0x46, 0x62, 0x33, 0xae, 0xb2, 0x2d, 0xf0, 0x1d, 0x13, 0xdb, 0xda, 0x44, 0x5f, 0x71, 0xb5,
	0x85, 0xcf, 0x61, 0x33, 0xfd, 0x21, 0x84, 0xf2, 0xa4, 0x56, 0x75, 0xb4, 0xaa, 0x67, 0xe8, 0x5a,
	0xf8, 0x18, 0x3a, 0x82, 0x14, 0x58, 0x5c, 0x87, 0xae, 0xae, 0xc9, 0xa0, 0xc1, 0x6f, 0x0b, 0xba,
	0x11, 0x2d, 0x25, 0xa6, 0x29, 0x41, 0xdb, 0xe0, 0xa5, 0x8b, 0x5c, 0x25, 0x6b, 0x8c, 0xe9, 0x56,
	0x44, 0x94, 0xa1, 0x2d, 0xe8, 0xe2, 0x2b, 0x73, 0x56, 0x79, 0xe3, 0x6a, 0x1c, 0x65, 0xda, 0x49,
	0x6e, 0x9a, 0xb5, 0x73, 0x5e, 0xef, 0x68, 0xeb, 0xbf, 0x1d, 0x6d, 0xaf, 0x76, 0x74, 0x07, 0xe0,
	0x16, 0xcb, 0xf4, 0x9b, 0x9a, 0x53, 0x19, 0x76, 0xfa, 0xce, 0xd0, 0x4b, 0x3c, 0xcd, 0x9c, 0x90,
	0x65, 0x89, 0x46, 0xf0, 0x50, 0xf7, 0xfe, 0x9d, 0xe5, 0x74, 0x2e, 0xf3, 0x82, 0x94, 0x12, 0x17,
	0x5c, 0x17, 0xef, 0xa8, 0xe5, 0x2c, 0xe5, 0x31, 0xcb, 0xe9, 0xac, 0x3e, 0xd8, 0x3b, 0x02, 0xff,
	0xce, 0xe2, 0x21, 0x1f, 0xdc, 0x8b, 0xf8, 0x24, 0x9e, 0x7e, 0x8c, 0x83, 0x35, 0xd4, 0x85, 0xd6,
	0xf1, 0xf9, 0x34, 0x0e, 0x2c, 0x15, 0xcd, 0xa6, 0xa7, 0x93, 0xc0, 0x46, 0x2e, 0x38, 0x51, 0x1c,
	0x05, 0x0e, 0xda, 0x00, 0xef, 0x6c, 0x32, 0x8e, 0xe2, 0xd9, 0xe1, 0xa7, 0x59, 0xd0, 0xda, 0x7b,
	0x0d, 0xde, 0x6a, 0x17, 0x7c, 0x70, 0xa3, 0xf8, 0x72, 0x3c, 0x89, 0xde, 0x07, 0x6b, 0xea, 0xc6,
	0xf9, 0xe1, 0x2c, 0xb0, 0x90, 0x07, 0xed, 0x8b, 0x58, 0x85, 0xb6, 0x12, 0x9c, 0x5d, 0x1c, 0x4c,
	0xa2, 0xf3, 0x0f, 0x81, 0x73, 0xe0, 0x7d, 0x76, 0x8d, 0x8f, 0x5f, 0x3a, 0xfa, 0x57, 0x79, 0xf5,
	0x37, 0x00, 0x00, 0xff, 0xff, 0xa8, 0xd3, 0xa0, 0x9c, 0x64, 0x04, 0x00, 0x00,
}
