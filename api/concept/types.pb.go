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

// ContentType enumerates all built-in content types those Element's
// raw could be.
type ContentType int32

const (
	ContentType_UNKNOWN ContentType = 0
	// application/json
	ContentType_JSON ContentType = 1
	// application/toml
	ContentType_TOML ContentType = 2
	// application/ini
	ContentType_INI ContentType = 3
	// application/plaintext
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

type ChangeOp int32

const (
	ChangeOp_UNDEFINED ChangeOp = 0
	ChangeOp_NEW       ChangeOp = 1
	ChangeOp_UPDATE    ChangeOp = 2
	ChangeOp_DELETE    ChangeOp = 3
)

var ChangeOp_name = map[int32]string{
	0: "UNDEFINED",
	1: "NEW",
	2: "UPDATE",
	3: "DELETE",
}

var ChangeOp_value = map[string]int32{
	"UNDEFINED": 0,
	"NEW":       1,
	"UPDATE":    2,
	"DELETE":    3,
}

func (x ChangeOp) String() string {
	return proto.EnumName(ChangeOp_name, int32(x))
}

func (ChangeOp) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_d938547f84707355, []int{1}
}

type AppMetadata_Status int32

const (
	AppMetadata_INUSE AppMetadata_Status = 0
	// DEPRECATED represents the app is deprecated. It can only read but update.
	AppMetadata_DEPRECATED AppMetadata_Status = 1
)

var AppMetadata_Status_name = map[int32]string{
	0: "INUSE",
	1: "DEPRECATED",
}

var AppMetadata_Status_value = map[string]int32{
	"INUSE":      0,
	"DEPRECATED": 1,
}

func (x AppMetadata_Status) String() string {
	return proto.EnumName(AppMetadata_Status_name, int32(x))
}

func (AppMetadata_Status) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_d938547f84707355, []int{2, 0}
}

type ElementOperation_Op int32

const (
	ElementOperation_INVALID ElementOperation_Op = 0
	ElementOperation_SET     ElementOperation_Op = 1
	ElementOperation_UNSET   ElementOperation_Op = 2
	ElementOperation_PUBLISH ElementOperation_Op = 3
)

var ElementOperation_Op_name = map[int32]string{
	0: "INVALID",
	1: "SET",
	2: "UNSET",
	3: "PUBLISH",
}

var ElementOperation_Op_value = map[string]int32{
	"INVALID": 0,
	"SET":     1,
	"UNSET":   2,
	"PUBLISH": 3,
}

func (x ElementOperation_Op) String() string {
	return proto.EnumName(ElementOperation_Op_name, int32(x))
}

func (ElementOperation_Op) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_d938547f84707355, []int{3, 0}
}

// Element represent a config element with specific version.
type Element struct {
	// refer ElementMetadata
	Metadata *ElementMetadata `protobuf:"bytes,1,opt,name=metadata,proto3" json:"metadata,omitempty"`
	// raw data in the version.
	Raw []byte `protobuf:"bytes,2,opt,name=raw,proto3" json:"raw,omitempty"`
	// version number start since 1.
	Version int32 `protobuf:"varint,3,opt,name=version,proto3" json:"version,omitempty"`
	// indicates published or not.
	Published            bool     `protobuf:"varint,4,opt,name=published,proto3" json:"published,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
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

func (m *Element) GetPublished() bool {
	if m != nil {
		return m.Published
	}
	return false
}

// ElementMetadata contains metadata of one element, includes
// specific key, app, env attributes, and other fields to display
// the element's version status.
type ElementMetadata struct {
	Key string `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
	App string `protobuf:"bytes,2,opt,name=app,proto3" json:"app,omitempty"`
	Env string `protobuf:"bytes,3,opt,name=env,proto3" json:"env,omitempty"`
	// the latest version in all versions of the element.
	LatestVersion int32 `protobuf:"varint,4,opt,name=latestVersion,proto3" json:"latestVersion,omitempty"`
	// if there is any unpublished version, if there's any unpublished
	// version, the element can not create a new version util all versions have been
	// published,
	UnpublishedVersion int32 `protobuf:"varint,5,opt,name=unpublishedVersion,proto3" json:"unpublishedVersion,omitempty"`
	// the in-use version.
	UsingVersion int32 `protobuf:"varint,6,opt,name=usingVersion,proto3" json:"usingVersion,omitempty"`
	// the using version's fingerprint.
	UsingFingerprint string `protobuf:"bytes,7,opt,name=usingFingerprint,proto3" json:"usingFingerprint,omitempty"`
	// indicates the content type of Element's raw data
	ContentType          ContentType `protobuf:"varint,8,opt,name=contentType,proto3,enum=cassem.concept.ContentType" json:"contentType,omitempty"`
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

func (m *ElementMetadata) GetUnpublishedVersion() int32 {
	if m != nil {
		return m.UnpublishedVersion
	}
	return 0
}

func (m *ElementMetadata) GetUsingVersion() int32 {
	if m != nil {
		return m.UsingVersion
	}
	return 0
}

func (m *ElementMetadata) GetUsingFingerprint() string {
	if m != nil {
		return m.UsingFingerprint
	}
	return ""
}

func (m *ElementMetadata) GetContentType() ContentType {
	if m != nil {
		return m.ContentType
	}
	return ContentType_UNKNOWN
}

// AppMetadata contains metadata of one app, includes specific identity,
// description, and other fields to display the app's status.
type AppMetadata struct {
	Id          string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Description string `protobuf:"bytes,2,opt,name=description,proto3" json:"description,omitempty"`
	CreatedAt   int64  `protobuf:"varint,3,opt,name=createdAt,proto3" json:"createdAt,omitempty"`
	// creator of the app indicates the user who created the app.
	Creator string `protobuf:"bytes,4,opt,name=creator,proto3" json:"creator,omitempty"`
	// owner of the app indicates the user who actually own this app.
	// of course, admin account owns all apps.
	Owner  string             `protobuf:"bytes,5,opt,name=owner,proto3" json:"owner,omitempty"`
	Status AppMetadata_Status `protobuf:"varint,6,opt,name=status,proto3,enum=cassem.concept.AppMetadata_Status" json:"status,omitempty"`
	// secrets is the key to acccecss the app's elements by different envs. If it's empty,
	// that means the app is public.
	Secrets              map[string]string `protobuf:"bytes,7,rep,name=secrets,proto3" json:"secrets,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	XXX_NoUnkeyedLiteral struct{}          `json:"-"`
	XXX_unrecognized     []byte            `json:"-"`
	XXX_sizecache        int32             `json:"-"`
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

func (m *AppMetadata) GetStatus() AppMetadata_Status {
	if m != nil {
		return m.Status
	}
	return AppMetadata_INUSE
}

func (m *AppMetadata) GetSecrets() map[string]string {
	if m != nil {
		return m.Secrets
	}
	return nil
}

// ElementOperation is used to indicate the operation of one element.
type ElementOperation struct {
	// operator indicates the user who execute the operation.
	Operator             string              `protobuf:"bytes,1,opt,name=operator,proto3" json:"operator,omitempty"`
	OperatedAt           int64               `protobuf:"varint,2,opt,name=operatedAt,proto3" json:"operatedAt,omitempty"`
	OperatedKey          string              `protobuf:"bytes,3,opt,name=operatedKey,proto3" json:"operatedKey,omitempty"`
	Op                   ElementOperation_Op `protobuf:"varint,4,opt,name=op,proto3,enum=cassem.concept.ElementOperation_Op" json:"op,omitempty"`
	LastVersion          int32               `protobuf:"varint,5,opt,name=lastVersion,proto3" json:"lastVersion,omitempty"`
	CurrentVersion       int32               `protobuf:"varint,6,opt,name=currentVersion,proto3" json:"currentVersion,omitempty"`
	Remark               string              `protobuf:"bytes,7,opt,name=remark,proto3" json:"remark,omitempty"`
	XXX_NoUnkeyedLiteral struct{}            `json:"-"`
	XXX_unrecognized     []byte              `json:"-"`
	XXX_sizecache        int32               `json:"-"`
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

func (m *ElementOperation) GetOp() ElementOperation_Op {
	if m != nil {
		return m.Op
	}
	return ElementOperation_INVALID
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

// Instance describes client instance.
type Instance struct {
	ClientId             string               `protobuf:"bytes,1,opt,name=clientId,proto3" json:"clientId,omitempty"`
	AgentId              string               `protobuf:"bytes,2,opt,name=agentId,proto3" json:"agentId,omitempty"`
	ClientIp             string               `protobuf:"bytes,3,opt,name=clientIp,proto3" json:"clientIp,omitempty"`
	Watching             []*Instance_Watching `protobuf:"bytes,4,rep,name=watching,proto3" json:"watching,omitempty"`
	LastRenewTimestamp   int64                `protobuf:"varint,5,opt,name=lastRenewTimestamp,proto3" json:"lastRenewTimestamp,omitempty"`
	XXX_NoUnkeyedLiteral struct{}             `json:"-"`
	XXX_unrecognized     []byte               `json:"-"`
	XXX_sizecache        int32                `json:"-"`
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

func (m *Instance) GetClientIp() string {
	if m != nil {
		return m.ClientIp
	}
	return ""
}

func (m *Instance) GetWatching() []*Instance_Watching {
	if m != nil {
		return m.Watching
	}
	return nil
}

func (m *Instance) GetLastRenewTimestamp() int64 {
	if m != nil {
		return m.LastRenewTimestamp
	}
	return 0
}

type Instance_Watching struct {
	App                  string   `protobuf:"bytes,1,opt,name=app,proto3" json:"app,omitempty"`
	Env                  string   `protobuf:"bytes,2,opt,name=env,proto3" json:"env,omitempty"`
	WatchKeys            []string `protobuf:"bytes,3,rep,name=watchKeys,proto3" json:"watchKeys,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Instance_Watching) Reset()         { *m = Instance_Watching{} }
func (m *Instance_Watching) String() string { return proto.CompactTextString(m) }
func (*Instance_Watching) ProtoMessage()    {}
func (*Instance_Watching) Descriptor() ([]byte, []int) {
	return fileDescriptor_d938547f84707355, []int{4, 0}
}

func (m *Instance_Watching) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Instance_Watching.Unmarshal(m, b)
}
func (m *Instance_Watching) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Instance_Watching.Marshal(b, m, deterministic)
}
func (m *Instance_Watching) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Instance_Watching.Merge(m, src)
}
func (m *Instance_Watching) XXX_Size() int {
	return xxx_messageInfo_Instance_Watching.Size(m)
}
func (m *Instance_Watching) XXX_DiscardUnknown() {
	xxx_messageInfo_Instance_Watching.DiscardUnknown(m)
}

var xxx_messageInfo_Instance_Watching proto.InternalMessageInfo

func (m *Instance_Watching) GetApp() string {
	if m != nil {
		return m.App
	}
	return ""
}

func (m *Instance_Watching) GetEnv() string {
	if m != nil {
		return m.Env
	}
	return ""
}

func (m *Instance_Watching) GetWatchKeys() []string {
	if m != nil {
		return m.WatchKeys
	}
	return nil
}

// AgentInstance describes agent node instance attributes.
type AgentInstance struct {
	// agentId is the unique identifier for agent.
	AgentId string `protobuf:"bytes,1,opt,name=agentId,proto3" json:"agentId,omitempty"`
	Addr    string `protobuf:"bytes,2,opt,name=addr,proto3" json:"addr,omitempty"`
	// annotations contains the some custom label and value of AgentInstance.
	Annotations          map[string]string `protobuf:"bytes,3,rep,name=annotations,proto3" json:"annotations,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	XXX_NoUnkeyedLiteral struct{}          `json:"-"`
	XXX_unrecognized     []byte            `json:"-"`
	XXX_sizecache        int32             `json:"-"`
}

func (m *AgentInstance) Reset()         { *m = AgentInstance{} }
func (m *AgentInstance) String() string { return proto.CompactTextString(m) }
func (*AgentInstance) ProtoMessage()    {}
func (*AgentInstance) Descriptor() ([]byte, []int) {
	return fileDescriptor_d938547f84707355, []int{5}
}

func (m *AgentInstance) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_AgentInstance.Unmarshal(m, b)
}
func (m *AgentInstance) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_AgentInstance.Marshal(b, m, deterministic)
}
func (m *AgentInstance) XXX_Merge(src proto.Message) {
	xxx_messageInfo_AgentInstance.Merge(m, src)
}
func (m *AgentInstance) XXX_Size() int {
	return xxx_messageInfo_AgentInstance.Size(m)
}
func (m *AgentInstance) XXX_DiscardUnknown() {
	xxx_messageInfo_AgentInstance.DiscardUnknown(m)
}

var xxx_messageInfo_AgentInstance proto.InternalMessageInfo

func (m *AgentInstance) GetAgentId() string {
	if m != nil {
		return m.AgentId
	}
	return ""
}

func (m *AgentInstance) GetAddr() string {
	if m != nil {
		return m.Addr
	}
	return ""
}

func (m *AgentInstance) GetAnnotations() map[string]string {
	if m != nil {
		return m.Annotations
	}
	return nil
}

type AgentInstanceChange struct {
	Ins                  *AgentInstance `protobuf:"bytes,1,opt,name=ins,proto3" json:"ins,omitempty"`
	Op                   ChangeOp       `protobuf:"varint,2,opt,name=op,proto3,enum=cassem.concept.ChangeOp" json:"op,omitempty"`
	XXX_NoUnkeyedLiteral struct{}       `json:"-"`
	XXX_unrecognized     []byte         `json:"-"`
	XXX_sizecache        int32          `json:"-"`
}

func (m *AgentInstanceChange) Reset()         { *m = AgentInstanceChange{} }
func (m *AgentInstanceChange) String() string { return proto.CompactTextString(m) }
func (*AgentInstanceChange) ProtoMessage()    {}
func (*AgentInstanceChange) Descriptor() ([]byte, []int) {
	return fileDescriptor_d938547f84707355, []int{6}
}

func (m *AgentInstanceChange) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_AgentInstanceChange.Unmarshal(m, b)
}
func (m *AgentInstanceChange) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_AgentInstanceChange.Marshal(b, m, deterministic)
}
func (m *AgentInstanceChange) XXX_Merge(src proto.Message) {
	xxx_messageInfo_AgentInstanceChange.Merge(m, src)
}
func (m *AgentInstanceChange) XXX_Size() int {
	return xxx_messageInfo_AgentInstanceChange.Size(m)
}
func (m *AgentInstanceChange) XXX_DiscardUnknown() {
	xxx_messageInfo_AgentInstanceChange.DiscardUnknown(m)
}

var xxx_messageInfo_AgentInstanceChange proto.InternalMessageInfo

func (m *AgentInstanceChange) GetIns() *AgentInstance {
	if m != nil {
		return m.Ins
	}
	return nil
}

func (m *AgentInstanceChange) GetOp() ChangeOp {
	if m != nil {
		return m.Op
	}
	return ChangeOp_UNDEFINED
}

func init() {
	proto.RegisterEnum("cassem.concept.ContentType", ContentType_name, ContentType_value)
	proto.RegisterEnum("cassem.concept.ChangeOp", ChangeOp_name, ChangeOp_value)
	proto.RegisterEnum("cassem.concept.AppMetadata_Status", AppMetadata_Status_name, AppMetadata_Status_value)
	proto.RegisterEnum("cassem.concept.ElementOperation_Op", ElementOperation_Op_name, ElementOperation_Op_value)
	proto.RegisterType((*Element)(nil), "cassem.concept.Element")
	proto.RegisterType((*ElementMetadata)(nil), "cassem.concept.ElementMetadata")
	proto.RegisterType((*AppMetadata)(nil), "cassem.concept.AppMetadata")
	proto.RegisterMapType((map[string]string)(nil), "cassem.concept.AppMetadata.SecretsEntry")
	proto.RegisterType((*ElementOperation)(nil), "cassem.concept.ElementOperation")
	proto.RegisterType((*Instance)(nil), "cassem.concept.Instance")
	proto.RegisterType((*Instance_Watching)(nil), "cassem.concept.Instance.Watching")
	proto.RegisterType((*AgentInstance)(nil), "cassem.concept.AgentInstance")
	proto.RegisterMapType((map[string]string)(nil), "cassem.concept.AgentInstance.AnnotationsEntry")
	proto.RegisterType((*AgentInstanceChange)(nil), "cassem.concept.AgentInstanceChange")
}

func init() { proto.RegisterFile("types.proto", fileDescriptor_d938547f84707355) }

var fileDescriptor_d938547f84707355 = []byte{
	// 906 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x94, 0x55, 0xdd, 0x6e, 0xe3, 0x44,
	0x14, 0xae, 0xed, 0xfc, 0xf9, 0xb8, 0x0d, 0xd6, 0xb0, 0x42, 0x56, 0x59, 0x20, 0x78, 0x11, 0x8a,
	0x7a, 0x61, 0xa4, 0xec, 0x0d, 0x2a, 0x5a, 0xa4, 0xb4, 0x71, 0x85, 0x69, 0xd6, 0x89, 0x26, 0xce,
	0x16, 0x71, 0x37, 0xeb, 0x8c, 0x52, 0x6b, 0x93, 0xf1, 0xc8, 0x9e, 0xb4, 0xca, 0x5b, 0x70, 0xcb,
	0x3b, 0xf0, 0x06, 0x3c, 0x04, 0x6f, 0xc2, 0x33, 0xa0, 0x19, 0xdb, 0x89, 0x93, 0x96, 0x95, 0xf6,
	0xee, 0x9c, 0x6f, 0xbe, 0x39, 0xe7, 0xf8, 0x9b, 0x6f, 0xc6, 0x60, 0x89, 0x2d, 0xa7, 0xb9, 0xc7,
	0xb3, 0x54, 0xa4, 0xa8, 0x1b, 0x93, 0x3c, 0xa7, 0x6b, 0x2f, 0x4e, 0x59, 0x4c, 0xb9, 0x70, 0xff,
	0xd0, 0xa0, 0xed, 0xaf, 0xe8, 0x9a, 0x32, 0x81, 0x7e, 0x82, 0xce, 0x9a, 0x0a, 0xb2, 0x20, 0x82,
	0x38, 0x5a, 0x4f, 0xeb, 0x5b, 0x83, 0x6f, 0xbc, 0x43, 0xba, 0x57, 0x52, 0xdf, 0x96, 0x34, 0xbc,
	0xdb, 0x80, 0x6c, 0x30, 0x32, 0xf2, 0xe8, 0xe8, 0x3d, 0xad, 0x7f, 0x8a, 0x65, 0x88, 0x1c, 0x68,
	0x3f, 0xd0, 0x2c, 0x4f, 0x52, 0xe6, 0x18, 0x3d, 0xad, 0xdf, 0xc4, 0x55, 0x8a, 0x5e, 0x82, 0xc9,
	0x37, 0xef, 0x57, 0x49, 0x7e, 0x4f, 0x17, 0x4e, 0xa3, 0xa7, 0xf5, 0x3b, 0x78, 0x0f, 0xb8, 0x7f,
	0xe9, 0xf0, 0xd9, 0x51, 0x1f, 0x59, 0xfd, 0x03, 0xdd, 0xaa, 0xa9, 0x4c, 0x2c, 0x43, 0x89, 0x10,
	0xce, 0x55, 0x3f, 0x13, 0xcb, 0x50, 0x22, 0x94, 0x3d, 0xa8, 0x5e, 0x26, 0x96, 0x21, 0xfa, 0x0e,
	0xce, 0x56, 0x44, 0xd0, 0x5c, 0xbc, 0x2b, 0xe7, 0x68, 0xa8, 0x39, 0x0e, 0x41, 0xe4, 0x01, 0xda,
	0xb0, 0x5d, 0xfb, 0x8a, 0xda, 0x54, 0xd4, 0x67, 0x56, 0x90, 0x0b, 0xa7, 0x9b, 0x3c, 0x61, 0xcb,
	0x8a, 0xd9, 0x52, 0xcc, 0x03, 0x0c, 0x5d, 0x80, 0xad, 0xf2, 0x9b, 0x84, 0x2d, 0x69, 0xc6, 0xb3,
	0x84, 0x09, 0xa7, 0xad, 0x06, 0x7b, 0x82, 0xa3, 0x37, 0x60, 0xc5, 0x29, 0x13, 0x94, 0x89, 0x68,
	0xcb, 0xa9, 0xd3, 0xe9, 0x69, 0xfd, 0xee, 0xe0, 0xcb, 0x63, 0xe5, 0xaf, 0xf7, 0x14, 0x5c, 0xe7,
	0xbb, 0xff, 0xea, 0x60, 0x0d, 0x39, 0xdf, 0x49, 0xd5, 0x05, 0x3d, 0x59, 0x94, 0x4a, 0xe9, 0xc9,
	0x02, 0xf5, 0xc0, 0x5a, 0xd0, 0x3c, 0xce, 0x12, 0x2e, 0xe4, 0xb4, 0x85, 0x60, 0x75, 0x48, 0x1e,
	0x47, 0x9c, 0x51, 0x22, 0xe8, 0x62, 0x28, 0x94, 0x7c, 0x06, 0xde, 0x03, 0xf2, 0x18, 0x55, 0x92,
	0x66, 0x4a, 0x3e, 0x13, 0x57, 0x29, 0x7a, 0x01, 0xcd, 0xf4, 0x91, 0xd1, 0x4c, 0x69, 0x65, 0xe2,
	0x22, 0x41, 0x97, 0xd0, 0xca, 0x05, 0x11, 0x9b, 0x5c, 0x09, 0xd3, 0x1d, 0xb8, 0xc7, 0x5f, 0x52,
	0x1b, 0xd6, 0x9b, 0x29, 0x26, 0x2e, 0x77, 0xa0, 0x2b, 0x68, 0xe7, 0x34, 0xce, 0xa8, 0xc8, 0x9d,
	0x76, 0xcf, 0xe8, 0x5b, 0x83, 0xfe, 0x47, 0x37, 0x17, 0x54, 0x9f, 0x89, 0x6c, 0x8b, 0xab, 0x8d,
	0xe7, 0x97, 0x70, 0x5a, 0x5f, 0x78, 0xc6, 0x3a, 0x2f, 0xa0, 0xf9, 0x40, 0x56, 0x1b, 0x5a, 0x6a,
	0x51, 0x24, 0x97, 0xfa, 0x8f, 0x9a, 0xfb, 0x0a, 0x5a, 0xc5, 0x44, 0xc8, 0x84, 0x66, 0x10, 0xce,
	0x67, 0xbe, 0x7d, 0x82, 0xba, 0x00, 0x23, 0x7f, 0x8a, 0xfd, 0xeb, 0x61, 0xe4, 0x8f, 0x6c, 0xcd,
	0xfd, 0x5b, 0x07, 0xbb, 0xf4, 0xe7, 0x84, 0xd3, 0x8c, 0x28, 0x0d, 0xcf, 0xa1, 0x93, 0xaa, 0x24,
	0xcd, 0xca, 0x56, 0xbb, 0x1c, 0x7d, 0x0d, 0x50, 0xc4, 0x4a, 0x60, 0x5d, 0x09, 0x5c, 0x43, 0xe4,
	0x09, 0x55, 0xd9, 0x2d, 0xdd, 0x96, 0x06, 0xae, 0x43, 0xe8, 0x35, 0xe8, 0x29, 0x57, 0xf2, 0x77,
	0x07, 0xaf, 0xfe, 0xe7, 0x4e, 0xee, 0x66, 0xf1, 0x26, 0x1c, 0xeb, 0x29, 0x97, 0x65, 0x57, 0x64,
	0xef, 0xfd, 0xc2, 0xd0, 0x75, 0x08, 0x7d, 0x0f, 0xdd, 0x78, 0x93, 0x65, 0x94, 0x89, 0x43, 0x2f,
	0x1f, 0xa1, 0xe8, 0x0b, 0x68, 0x65, 0x74, 0x4d, 0xb2, 0x0f, 0xa5, 0x87, 0xcb, 0xcc, 0x1d, 0x80,
	0x3e, 0xe1, 0xc8, 0x82, 0x76, 0x10, 0xbe, 0x1b, 0x8e, 0x83, 0x91, 0x7d, 0x82, 0xda, 0x60, 0xcc,
	0xfc, 0xc8, 0xd6, 0xa4, 0x80, 0xf3, 0x50, 0x86, 0xba, 0x24, 0x4c, 0xe7, 0x57, 0xe3, 0x60, 0xf6,
	0x8b, 0x6d, 0xb8, 0x7f, 0xea, 0xd0, 0x09, 0x58, 0x2e, 0x08, 0x8b, 0xa9, 0x54, 0x2d, 0x5e, 0x25,
	0x94, 0x89, 0xa0, 0x72, 0xec, 0x2e, 0x97, 0xbe, 0x23, 0xcb, 0x62, 0xa9, 0x38, 0xa7, 0x2a, 0xad,
	0xed, 0xe2, 0xa5, 0x58, 0xbb, 0x1c, 0xbd, 0x81, 0xce, 0x23, 0x11, 0xf1, 0x7d, 0xc2, 0x96, 0x4e,
	0x43, 0x59, 0xe8, 0xdb, 0x63, 0xbd, 0xaa, 0xee, 0xde, 0x5d, 0x49, 0xc4, 0xbb, 0x2d, 0xf2, 0x2d,
	0x90, 0x02, 0x61, 0xca, 0xe8, 0x63, 0x94, 0xac, 0x69, 0x2e, 0xc8, 0x9a, 0x2b, 0xe9, 0x0c, 0xfc,
	0xcc, 0xca, 0xf9, 0x18, 0x3a, 0x55, 0x95, 0xea, 0x45, 0xd2, 0x9e, 0xbc, 0x48, 0xfa, 0xfe, 0x45,
	0x7a, 0x09, 0xa6, 0xea, 0x75, 0x4b, 0xb7, 0xb9, 0x63, 0xf4, 0x8c, 0xbe, 0x89, 0xf7, 0x80, 0xfb,
	0x8f, 0x06, 0x67, 0x43, 0xf5, 0x91, 0x95, 0x40, 0x35, 0x11, 0xb4, 0x43, 0x11, 0x10, 0x34, 0xc8,
	0x62, 0x91, 0x95, 0xc5, 0x55, 0x8c, 0xa6, 0x60, 0x11, 0xc6, 0x52, 0xa1, 0x6c, 0x50, 0xd4, 0xb7,
	0x06, 0xde, 0x93, 0x2b, 0x54, 0xef, 0xe0, 0x0d, 0xf7, 0x1b, 0x8a, 0x8b, 0x54, 0x2f, 0x71, 0xfe,
	0x33, 0xd8, 0xc7, 0x84, 0x4f, 0xba, 0x50, 0x1c, 0x3e, 0x3f, 0x68, 0x77, 0x7d, 0x4f, 0xd8, 0x92,
	0xa2, 0x1f, 0xc0, 0x48, 0x58, 0x5e, 0xfe, 0x64, 0xbe, 0xfa, 0xe8, 0x80, 0x58, 0x32, 0x51, 0x5f,
	0x5d, 0x00, 0x5d, 0x5d, 0x00, 0xe7, 0xc9, 0xd3, 0xa8, 0x8a, 0x16, 0xae, 0xbf, 0xb8, 0x01, 0xab,
	0xf6, 0x54, 0x4a, 0xef, 0xcd, 0xc3, 0xdb, 0x70, 0x72, 0x17, 0xda, 0x27, 0xa8, 0x03, 0x8d, 0x5f,
	0x67, 0x93, 0xd0, 0xd6, 0x64, 0x14, 0x4d, 0xde, 0x8e, 0x6d, 0x5d, 0x1a, 0x36, 0x08, 0x03, 0xdb,
	0x40, 0x67, 0x60, 0x4e, 0xc7, 0xc3, 0x20, 0x8c, 0xfc, 0xdf, 0x22, 0xbb, 0x71, 0x71, 0x09, 0x9d,
	0xaa, 0xae, 0x5c, 0x9a, 0x87, 0x23, 0xff, 0x26, 0x08, 0xfd, 0xd2, 0xe3, 0xa1, 0x7f, 0x67, 0x6b,
	0x08, 0xa0, 0x35, 0x9f, 0x8e, 0x86, 0x91, 0x6f, 0xeb, 0x32, 0x1e, 0xf9, 0x63, 0x3f, 0xf2, 0x6d,
	0xe3, 0xca, 0xfc, 0xbd, 0x5d, 0xce, 0xf6, 0xbe, 0xa5, 0x7e, 0xbb, 0xaf, 0xff, 0x0b, 0x00, 0x00,
	0xff, 0xff, 0x73, 0x48, 0x7f, 0xa0, 0x85, 0x07, 0x00, 0x00,
}
