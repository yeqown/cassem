// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.17.3
// source: cassemdb.raft.proto

package api

import (
	context "context"
	_ "github.com/envoyproxy/protoc-gen-validate/validate"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type LogEntry_Action int32

const (
	LogEntry_UNKNOWN      LogEntry_Action = 0
	LogEntry_Set          LogEntry_Action = 1
	LogEntry_ChangeSpread LogEntry_Action = 2
)

// Enum value maps for LogEntry_Action.
var (
	LogEntry_Action_name = map[int32]string{
		0: "UNKNOWN",
		1: "Set",
		2: "ChangeSpread",
	}
	LogEntry_Action_value = map[string]int32{
		"UNKNOWN":      0,
		"Set":          1,
		"ChangeSpread": 2,
	}
)

func (x LogEntry_Action) Enum() *LogEntry_Action {
	p := new(LogEntry_Action)
	*p = x
	return p
}

func (x LogEntry_Action) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (LogEntry_Action) Descriptor() protoreflect.EnumDescriptor {
	return file_cassemdb_raft_proto_enumTypes[0].Descriptor()
}

func (LogEntry_Action) Type() protoreflect.EnumType {
	return &file_cassemdb_raft_proto_enumTypes[0]
}

func (x LogEntry_Action) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use LogEntry_Action.Descriptor instead.
func (LogEntry_Action) EnumDescriptor() ([]byte, []int) {
	return file_cassemdb_raft_proto_rawDescGZIP(), []int{0, 0}
}

type LogEntry struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// action indicates what's the format of data dose the log entry contains.
	Action    LogEntry_Action `protobuf:"varint,1,opt,name=action,proto3,enum=cassem.db.LogEntry_Action" json:"action,omitempty"`
	Command   []byte          `protobuf:"bytes,2,opt,name=command,proto3" json:"command,omitempty"`
	CreatedAt int64           `protobuf:"varint,3,opt,name=createdAt,proto3" json:"createdAt,omitempty"`
}

func (x *LogEntry) Reset() {
	*x = LogEntry{}
	if protoimpl.UnsafeEnabled {
		mi := &file_cassemdb_raft_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *LogEntry) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*LogEntry) ProtoMessage() {}

func (x *LogEntry) ProtoReflect() protoreflect.Message {
	mi := &file_cassemdb_raft_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use LogEntry.ProtoReflect.Descriptor instead.
func (*LogEntry) Descriptor() ([]byte, []int) {
	return file_cassemdb_raft_proto_rawDescGZIP(), []int{0}
}

func (x *LogEntry) GetAction() LogEntry_Action {
	if x != nil {
		return x.Action
	}
	return LogEntry_UNKNOWN
}

func (x *LogEntry) GetCommand() []byte {
	if x != nil {
		return x.Command
	}
	return nil
}

func (x *LogEntry) GetCreatedAt() int64 {
	if x != nil {
		return x.CreatedAt
	}
	return 0
}

type SetCommand struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	DeleteKey string  `protobuf:"bytes,1,opt,name=deleteKey,proto3" json:"deleteKey,omitempty"`
	IsDir     bool    `protobuf:"varint,2,opt,name=isDir,proto3" json:"isDir,omitempty"`
	SetKey    string  `protobuf:"bytes,3,opt,name=setKey,proto3" json:"setKey,omitempty"`
	Value     *Entity `protobuf:"bytes,4,opt,name=value,proto3" json:"value,omitempty"`
}

func (x *SetCommand) Reset() {
	*x = SetCommand{}
	if protoimpl.UnsafeEnabled {
		mi := &file_cassemdb_raft_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SetCommand) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SetCommand) ProtoMessage() {}

func (x *SetCommand) ProtoReflect() protoreflect.Message {
	mi := &file_cassemdb_raft_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SetCommand.ProtoReflect.Descriptor instead.
func (*SetCommand) Descriptor() ([]byte, []int) {
	return file_cassemdb_raft_proto_rawDescGZIP(), []int{1}
}

func (x *SetCommand) GetDeleteKey() string {
	if x != nil {
		return x.DeleteKey
	}
	return ""
}

func (x *SetCommand) GetIsDir() bool {
	if x != nil {
		return x.IsDir
	}
	return false
}

func (x *SetCommand) GetSetKey() string {
	if x != nil {
		return x.SetKey
	}
	return ""
}

func (x *SetCommand) GetValue() *Entity {
	if x != nil {
		return x.Value
	}
	return nil
}

type ChangeCommand struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Change *Change `protobuf:"bytes,1,opt,name=change,proto3" json:"change,omitempty"`
}

func (x *ChangeCommand) Reset() {
	*x = ChangeCommand{}
	if protoimpl.UnsafeEnabled {
		mi := &file_cassemdb_raft_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ChangeCommand) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ChangeCommand) ProtoMessage() {}

func (x *ChangeCommand) ProtoReflect() protoreflect.Message {
	mi := &file_cassemdb_raft_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ChangeCommand.ProtoReflect.Descriptor instead.
func (*ChangeCommand) Descriptor() ([]byte, []int) {
	return file_cassemdb_raft_proto_rawDescGZIP(), []int{2}
}

func (x *ChangeCommand) GetChange() *Change {
	if x != nil {
		return x.Change
	}
	return nil
}

type AddNodeRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Addr string `protobuf:"bytes,1,opt,name=addr,proto3" json:"addr,omitempty"`
}

func (x *AddNodeRequest) Reset() {
	*x = AddNodeRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_cassemdb_raft_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AddNodeRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AddNodeRequest) ProtoMessage() {}

func (x *AddNodeRequest) ProtoReflect() protoreflect.Message {
	mi := &file_cassemdb_raft_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AddNodeRequest.ProtoReflect.Descriptor instead.
func (*AddNodeRequest) Descriptor() ([]byte, []int) {
	return file_cassemdb_raft_proto_rawDescGZIP(), []int{3}
}

func (x *AddNodeRequest) GetAddr() string {
	if x != nil {
		return x.Addr
	}
	return ""
}

type AddNodeResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	NodeId uint64   `protobuf:"varint,1,opt,name=node_id,json=nodeId,proto3" json:"node_id,omitempty"`
	Peers  []string `protobuf:"bytes,2,rep,name=peers,proto3" json:"peers,omitempty"`
}

func (x *AddNodeResponse) Reset() {
	*x = AddNodeResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_cassemdb_raft_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AddNodeResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AddNodeResponse) ProtoMessage() {}

func (x *AddNodeResponse) ProtoReflect() protoreflect.Message {
	mi := &file_cassemdb_raft_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AddNodeResponse.ProtoReflect.Descriptor instead.
func (*AddNodeResponse) Descriptor() ([]byte, []int) {
	return file_cassemdb_raft_proto_rawDescGZIP(), []int{4}
}

func (x *AddNodeResponse) GetNodeId() uint64 {
	if x != nil {
		return x.NodeId
	}
	return 0
}

func (x *AddNodeResponse) GetPeers() []string {
	if x != nil {
		return x.Peers
	}
	return nil
}

type RemoveNodeRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	NodeId uint64 `protobuf:"varint,1,opt,name=node_id,json=nodeId,proto3" json:"node_id,omitempty"`
}

func (x *RemoveNodeRequest) Reset() {
	*x = RemoveNodeRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_cassemdb_raft_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RemoveNodeRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RemoveNodeRequest) ProtoMessage() {}

func (x *RemoveNodeRequest) ProtoReflect() protoreflect.Message {
	mi := &file_cassemdb_raft_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RemoveNodeRequest.ProtoReflect.Descriptor instead.
func (*RemoveNodeRequest) Descriptor() ([]byte, []int) {
	return file_cassemdb_raft_proto_rawDescGZIP(), []int{5}
}

func (x *RemoveNodeRequest) GetNodeId() uint64 {
	if x != nil {
		return x.NodeId
	}
	return 0
}

type RemoveNodeResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *RemoveNodeResponse) Reset() {
	*x = RemoveNodeResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_cassemdb_raft_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RemoveNodeResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RemoveNodeResponse) ProtoMessage() {}

func (x *RemoveNodeResponse) ProtoReflect() protoreflect.Message {
	mi := &file_cassemdb_raft_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RemoveNodeResponse.ProtoReflect.Descriptor instead.
func (*RemoveNodeResponse) Descriptor() ([]byte, []int) {
	return file_cassemdb_raft_proto_rawDescGZIP(), []int{6}
}

var File_cassemdb_raft_proto protoreflect.FileDescriptor

var file_cassemdb_raft_proto_rawDesc = []byte{
	0x0a, 0x13, 0x63, 0x61, 0x73, 0x73, 0x65, 0x6d, 0x64, 0x62, 0x2e, 0x72, 0x61, 0x66, 0x74, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x09, 0x63, 0x61, 0x73, 0x73, 0x65, 0x6d, 0x2e, 0x64, 0x62,
	0x1a, 0x12, 0x63, 0x61, 0x73, 0x73, 0x65, 0x6d, 0x64, 0x62, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x22, 0x65, 0x6e, 0x76, 0x6f, 0x79, 0x70, 0x72, 0x6f, 0x78, 0x79,
	0x2d, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x65, 0x2f, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61,
	0x74, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xa8, 0x01, 0x0a, 0x08, 0x4c, 0x6f, 0x67,
	0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x32, 0x0a, 0x06, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x1a, 0x2e, 0x63, 0x61, 0x73, 0x73, 0x65, 0x6d, 0x2e, 0x64,
	0x62, 0x2e, 0x4c, 0x6f, 0x67, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x2e, 0x41, 0x63, 0x74, 0x69, 0x6f,
	0x6e, 0x52, 0x06, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x18, 0x0a, 0x07, 0x63, 0x6f, 0x6d,
	0x6d, 0x61, 0x6e, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x07, 0x63, 0x6f, 0x6d, 0x6d,
	0x61, 0x6e, 0x64, 0x12, 0x1c, 0x0a, 0x09, 0x63, 0x72, 0x65, 0x61, 0x74, 0x65, 0x64, 0x41, 0x74,
	0x18, 0x03, 0x20, 0x01, 0x28, 0x03, 0x52, 0x09, 0x63, 0x72, 0x65, 0x61, 0x74, 0x65, 0x64, 0x41,
	0x74, 0x22, 0x30, 0x0a, 0x06, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x0b, 0x0a, 0x07, 0x55,
	0x4e, 0x4b, 0x4e, 0x4f, 0x57, 0x4e, 0x10, 0x00, 0x12, 0x07, 0x0a, 0x03, 0x53, 0x65, 0x74, 0x10,
	0x01, 0x12, 0x10, 0x0a, 0x0c, 0x43, 0x68, 0x61, 0x6e, 0x67, 0x65, 0x53, 0x70, 0x72, 0x65, 0x61,
	0x64, 0x10, 0x02, 0x22, 0x81, 0x01, 0x0a, 0x0a, 0x53, 0x65, 0x74, 0x43, 0x6f, 0x6d, 0x6d, 0x61,
	0x6e, 0x64, 0x12, 0x1c, 0x0a, 0x09, 0x64, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x4b, 0x65, 0x79, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x64, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x4b, 0x65, 0x79,
	0x12, 0x14, 0x0a, 0x05, 0x69, 0x73, 0x44, 0x69, 0x72, 0x18, 0x02, 0x20, 0x01, 0x28, 0x08, 0x52,
	0x05, 0x69, 0x73, 0x44, 0x69, 0x72, 0x12, 0x16, 0x0a, 0x06, 0x73, 0x65, 0x74, 0x4b, 0x65, 0x79,
	0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x73, 0x65, 0x74, 0x4b, 0x65, 0x79, 0x12, 0x27,
	0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x11, 0x2e,
	0x63, 0x61, 0x73, 0x73, 0x65, 0x6d, 0x2e, 0x64, 0x62, 0x2e, 0x45, 0x6e, 0x74, 0x69, 0x74, 0x79,
	0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x22, 0x3a, 0x0a, 0x0d, 0x43, 0x68, 0x61, 0x6e, 0x67,
	0x65, 0x43, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x12, 0x29, 0x0a, 0x06, 0x63, 0x68, 0x61, 0x6e,
	0x67, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x11, 0x2e, 0x63, 0x61, 0x73, 0x73, 0x65,
	0x6d, 0x2e, 0x64, 0x62, 0x2e, 0x43, 0x68, 0x61, 0x6e, 0x67, 0x65, 0x52, 0x06, 0x63, 0x68, 0x61,
	0x6e, 0x67, 0x65, 0x22, 0x31, 0x0a, 0x0e, 0x61, 0x64, 0x64, 0x4e, 0x6f, 0x64, 0x65, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x1f, 0x0a, 0x04, 0x61, 0x64, 0x64, 0x72, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x09, 0x42, 0x0b, 0xfa, 0x42, 0x08, 0x72, 0x06, 0x3a, 0x04, 0x68, 0x74, 0x74, 0x70,
	0x52, 0x04, 0x61, 0x64, 0x64, 0x72, 0x22, 0x40, 0x0a, 0x0f, 0x61, 0x64, 0x64, 0x4e, 0x6f, 0x64,
	0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x17, 0x0a, 0x07, 0x6e, 0x6f, 0x64,
	0x65, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x04, 0x52, 0x06, 0x6e, 0x6f, 0x64, 0x65,
	0x49, 0x64, 0x12, 0x14, 0x0a, 0x05, 0x70, 0x65, 0x65, 0x72, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28,
	0x09, 0x52, 0x05, 0x70, 0x65, 0x65, 0x72, 0x73, 0x22, 0x35, 0x0a, 0x11, 0x72, 0x65, 0x6d, 0x6f,
	0x76, 0x65, 0x4e, 0x6f, 0x64, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x20, 0x0a,
	0x07, 0x6e, 0x6f, 0x64, 0x65, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x04, 0x42, 0x07,
	0xfa, 0x42, 0x04, 0x32, 0x02, 0x20, 0x00, 0x52, 0x06, 0x6e, 0x6f, 0x64, 0x65, 0x49, 0x64, 0x22,
	0x14, 0x0a, 0x12, 0x72, 0x65, 0x6d, 0x6f, 0x76, 0x65, 0x4e, 0x6f, 0x64, 0x65, 0x52, 0x65, 0x73,
	0x70, 0x6f, 0x6e, 0x73, 0x65, 0x32, 0x9a, 0x01, 0x0a, 0x07, 0x43, 0x6c, 0x75, 0x73, 0x74, 0x65,
	0x72, 0x12, 0x42, 0x0a, 0x07, 0x41, 0x64, 0x64, 0x4e, 0x6f, 0x64, 0x65, 0x12, 0x19, 0x2e, 0x63,
	0x61, 0x73, 0x73, 0x65, 0x6d, 0x2e, 0x64, 0x62, 0x2e, 0x61, 0x64, 0x64, 0x4e, 0x6f, 0x64, 0x65,
	0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x1a, 0x2e, 0x63, 0x61, 0x73, 0x73, 0x65, 0x6d,
	0x2e, 0x64, 0x62, 0x2e, 0x61, 0x64, 0x64, 0x4e, 0x6f, 0x64, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f,
	0x6e, 0x73, 0x65, 0x22, 0x00, 0x12, 0x4b, 0x0a, 0x0a, 0x52, 0x65, 0x6d, 0x6f, 0x76, 0x65, 0x4e,
	0x6f, 0x64, 0x65, 0x12, 0x1c, 0x2e, 0x63, 0x61, 0x73, 0x73, 0x65, 0x6d, 0x2e, 0x64, 0x62, 0x2e,
	0x72, 0x65, 0x6d, 0x6f, 0x76, 0x65, 0x4e, 0x6f, 0x64, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x1a, 0x1d, 0x2e, 0x63, 0x61, 0x73, 0x73, 0x65, 0x6d, 0x2e, 0x64, 0x62, 0x2e, 0x72, 0x65,
	0x6d, 0x6f, 0x76, 0x65, 0x4e, 0x6f, 0x64, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65,
	0x22, 0x00, 0x42, 0x34, 0x5a, 0x32, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d,
	0x2f, 0x79, 0x65, 0x71, 0x6f, 0x77, 0x6e, 0x2f, 0x63, 0x61, 0x73, 0x73, 0x65, 0x6d, 0x2f, 0x69,
	0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x63, 0x61, 0x73, 0x73, 0x65, 0x6d, 0x64, 0x62,
	0x2f, 0x61, 0x70, 0x69, 0x3b, 0x61, 0x70, 0x69, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_cassemdb_raft_proto_rawDescOnce sync.Once
	file_cassemdb_raft_proto_rawDescData = file_cassemdb_raft_proto_rawDesc
)

func file_cassemdb_raft_proto_rawDescGZIP() []byte {
	file_cassemdb_raft_proto_rawDescOnce.Do(func() {
		file_cassemdb_raft_proto_rawDescData = protoimpl.X.CompressGZIP(file_cassemdb_raft_proto_rawDescData)
	})
	return file_cassemdb_raft_proto_rawDescData
}

var file_cassemdb_raft_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_cassemdb_raft_proto_msgTypes = make([]protoimpl.MessageInfo, 7)
var file_cassemdb_raft_proto_goTypes = []interface{}{
	(LogEntry_Action)(0),       // 0: cassem.db.LogEntry.Action
	(*LogEntry)(nil),           // 1: cassem.db.LogEntry
	(*SetCommand)(nil),         // 2: cassem.db.SetCommand
	(*ChangeCommand)(nil),      // 3: cassem.db.ChangeCommand
	(*AddNodeRequest)(nil),     // 4: cassem.db.addNodeRequest
	(*AddNodeResponse)(nil),    // 5: cassem.db.addNodeResponse
	(*RemoveNodeRequest)(nil),  // 6: cassem.db.removeNodeRequest
	(*RemoveNodeResponse)(nil), // 7: cassem.db.removeNodeResponse
	(*Entity)(nil),             // 8: cassem.db.Entity
	(*Change)(nil),             // 9: cassem.db.Change
}
var file_cassemdb_raft_proto_depIdxs = []int32{
	0, // 0: cassem.db.LogEntry.action:type_name -> cassem.db.LogEntry.Action
	8, // 1: cassem.db.SetCommand.value:type_name -> cassem.db.Entity
	9, // 2: cassem.db.ChangeCommand.change:type_name -> cassem.db.Change
	4, // 3: cassem.db.Cluster.AddNode:input_type -> cassem.db.addNodeRequest
	6, // 4: cassem.db.Cluster.RemoveNode:input_type -> cassem.db.removeNodeRequest
	5, // 5: cassem.db.Cluster.AddNode:output_type -> cassem.db.addNodeResponse
	7, // 6: cassem.db.Cluster.RemoveNode:output_type -> cassem.db.removeNodeResponse
	5, // [5:7] is the sub-list for method output_type
	3, // [3:5] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_cassemdb_raft_proto_init() }
func file_cassemdb_raft_proto_init() {
	if File_cassemdb_raft_proto != nil {
		return
	}
	file_cassemdb_api_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_cassemdb_raft_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*LogEntry); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_cassemdb_raft_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*SetCommand); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_cassemdb_raft_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ChangeCommand); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_cassemdb_raft_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AddNodeRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_cassemdb_raft_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AddNodeResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_cassemdb_raft_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RemoveNodeRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_cassemdb_raft_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RemoveNodeResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_cassemdb_raft_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   7,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_cassemdb_raft_proto_goTypes,
		DependencyIndexes: file_cassemdb_raft_proto_depIdxs,
		EnumInfos:         file_cassemdb_raft_proto_enumTypes,
		MessageInfos:      file_cassemdb_raft_proto_msgTypes,
	}.Build()
	File_cassemdb_raft_proto = out.File
	file_cassemdb_raft_proto_rawDesc = nil
	file_cassemdb_raft_proto_goTypes = nil
	file_cassemdb_raft_proto_depIdxs = nil
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConnInterface

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion6

// ClusterClient is the client API for Cluster service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type ClusterClient interface {
	AddNode(ctx context.Context, in *AddNodeRequest, opts ...grpc.CallOption) (*AddNodeResponse, error)
	RemoveNode(ctx context.Context, in *RemoveNodeRequest, opts ...grpc.CallOption) (*RemoveNodeResponse, error)
}

type clusterClient struct {
	cc grpc.ClientConnInterface
}

func NewClusterClient(cc grpc.ClientConnInterface) ClusterClient {
	return &clusterClient{cc}
}

func (c *clusterClient) AddNode(ctx context.Context, in *AddNodeRequest, opts ...grpc.CallOption) (*AddNodeResponse, error) {
	out := new(AddNodeResponse)
	err := c.cc.Invoke(ctx, "/cassem.db.Cluster/AddNode", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *clusterClient) RemoveNode(ctx context.Context, in *RemoveNodeRequest, opts ...grpc.CallOption) (*RemoveNodeResponse, error) {
	out := new(RemoveNodeResponse)
	err := c.cc.Invoke(ctx, "/cassem.db.Cluster/RemoveNode", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ClusterServer is the server API for Cluster service.
type ClusterServer interface {
	AddNode(context.Context, *AddNodeRequest) (*AddNodeResponse, error)
	RemoveNode(context.Context, *RemoveNodeRequest) (*RemoveNodeResponse, error)
}

// UnimplementedClusterServer can be embedded to have forward compatible implementations.
type UnimplementedClusterServer struct {
}

func (*UnimplementedClusterServer) AddNode(context.Context, *AddNodeRequest) (*AddNodeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AddNode not implemented")
}
func (*UnimplementedClusterServer) RemoveNode(context.Context, *RemoveNodeRequest) (*RemoveNodeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RemoveNode not implemented")
}

func RegisterClusterServer(s *grpc.Server, srv ClusterServer) {
	s.RegisterService(&_Cluster_serviceDesc, srv)
}

func _Cluster_AddNode_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AddNodeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ClusterServer).AddNode(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cassem.db.Cluster/AddNode",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ClusterServer).AddNode(ctx, req.(*AddNodeRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Cluster_RemoveNode_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RemoveNodeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ClusterServer).RemoveNode(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cassem.db.Cluster/RemoveNode",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ClusterServer).RemoveNode(ctx, req.(*RemoveNodeRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _Cluster_serviceDesc = grpc.ServiceDesc{
	ServiceName: "cassem.db.Cluster",
	HandlerType: (*ClusterServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "AddNode",
			Handler:    _Cluster_AddNode_Handler,
		},
		{
			MethodName: "RemoveNode",
			Handler:    _Cluster_RemoveNode_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "cassemdb.raft.proto",
}
