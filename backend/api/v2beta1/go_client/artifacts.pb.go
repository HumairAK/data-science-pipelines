//  Copyright [2024] The Kubeflow Authors
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.17.3
// source: backend/api/v2beta1/artifacts.proto

package go_client

import (
	context "context"
	_ "google.golang.org/genproto/googleapis/api/annotations"
	_ "google.golang.org/genproto/googleapis/api/httpbody"
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

type Artifact struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ArtifactId   string `protobuf:"bytes,1,opt,name=artifact_id,json=artifactId,proto3" json:"artifact_id,omitempty"`
	ArtifactType string `protobuf:"bytes,2,opt,name=artifact_type,json=artifactType,proto3" json:"artifact_type,omitempty"`
}

func (x *Artifact) Reset() {
	*x = Artifact{}
	if protoimpl.UnsafeEnabled {
		mi := &file_backend_api_v2beta1_artifacts_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Artifact) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Artifact) ProtoMessage() {}

func (x *Artifact) ProtoReflect() protoreflect.Message {
	mi := &file_backend_api_v2beta1_artifacts_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Artifact.ProtoReflect.Descriptor instead.
func (*Artifact) Descriptor() ([]byte, []int) {
	return file_backend_api_v2beta1_artifacts_proto_rawDescGZIP(), []int{0}
}

func (x *Artifact) GetArtifactId() string {
	if x != nil {
		return x.ArtifactId
	}
	return ""
}

func (x *Artifact) GetArtifactType() string {
	if x != nil {
		return x.ArtifactType
	}
	return ""
}

type GetArtifactRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ArtifactId string `protobuf:"bytes,1,opt,name=artifact_id,json=artifactId,proto3" json:"artifact_id,omitempty"`
}

func (x *GetArtifactRequest) Reset() {
	*x = GetArtifactRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_backend_api_v2beta1_artifacts_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetArtifactRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetArtifactRequest) ProtoMessage() {}

func (x *GetArtifactRequest) ProtoReflect() protoreflect.Message {
	mi := &file_backend_api_v2beta1_artifacts_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetArtifactRequest.ProtoReflect.Descriptor instead.
func (*GetArtifactRequest) Descriptor() ([]byte, []int) {
	return file_backend_api_v2beta1_artifacts_proto_rawDescGZIP(), []int{1}
}

func (x *GetArtifactRequest) GetArtifactId() string {
	if x != nil {
		return x.ArtifactId
	}
	return ""
}

type ListArtifactRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ExecutionId string `protobuf:"bytes,1,opt,name=execution_id,json=executionId,proto3" json:"execution_id,omitempty"`
}

func (x *ListArtifactRequest) Reset() {
	*x = ListArtifactRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_backend_api_v2beta1_artifacts_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ListArtifactRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListArtifactRequest) ProtoMessage() {}

func (x *ListArtifactRequest) ProtoReflect() protoreflect.Message {
	mi := &file_backend_api_v2beta1_artifacts_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListArtifactRequest.ProtoReflect.Descriptor instead.
func (*ListArtifactRequest) Descriptor() ([]byte, []int) {
	return file_backend_api_v2beta1_artifacts_proto_rawDescGZIP(), []int{2}
}

func (x *ListArtifactRequest) GetExecutionId() string {
	if x != nil {
		return x.ExecutionId
	}
	return ""
}

type DownloadArtifactRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ArtifactId string `protobuf:"bytes,1,opt,name=artifact_id,json=artifactId,proto3" json:"artifact_id,omitempty"`
}

func (x *DownloadArtifactRequest) Reset() {
	*x = DownloadArtifactRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_backend_api_v2beta1_artifacts_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DownloadArtifactRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DownloadArtifactRequest) ProtoMessage() {}

func (x *DownloadArtifactRequest) ProtoReflect() protoreflect.Message {
	mi := &file_backend_api_v2beta1_artifacts_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DownloadArtifactRequest.ProtoReflect.Descriptor instead.
func (*DownloadArtifactRequest) Descriptor() ([]byte, []int) {
	return file_backend_api_v2beta1_artifacts_proto_rawDescGZIP(), []int{3}
}

func (x *DownloadArtifactRequest) GetArtifactId() string {
	if x != nil {
		return x.ArtifactId
	}
	return ""
}

type FileResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Chunk []byte `protobuf:"bytes,1,opt,name=chunk,proto3" json:"chunk,omitempty"` // some segment of the file
}

func (x *FileResponse) Reset() {
	*x = FileResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_backend_api_v2beta1_artifacts_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *FileResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*FileResponse) ProtoMessage() {}

func (x *FileResponse) ProtoReflect() protoreflect.Message {
	mi := &file_backend_api_v2beta1_artifacts_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use FileResponse.ProtoReflect.Descriptor instead.
func (*FileResponse) Descriptor() ([]byte, []int) {
	return file_backend_api_v2beta1_artifacts_proto_rawDescGZIP(), []int{4}
}

func (x *FileResponse) GetChunk() []byte {
	if x != nil {
		return x.Chunk
	}
	return nil
}

var File_backend_api_v2beta1_artifacts_proto protoreflect.FileDescriptor

var file_backend_api_v2beta1_artifacts_proto_rawDesc = []byte{
	0x0a, 0x23, 0x62, 0x61, 0x63, 0x6b, 0x65, 0x6e, 0x64, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x76, 0x32,
	0x62, 0x65, 0x74, 0x61, 0x31, 0x2f, 0x61, 0x72, 0x74, 0x69, 0x66, 0x61, 0x63, 0x74, 0x73, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x26, 0x6b, 0x75, 0x62, 0x65, 0x66, 0x6c, 0x6f, 0x77, 0x2e,
	0x70, 0x69, 0x70, 0x65, 0x6c, 0x69, 0x6e, 0x65, 0x73, 0x2e, 0x62, 0x61, 0x63, 0x6b, 0x65, 0x6e,
	0x64, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x32, 0x62, 0x65, 0x74, 0x61, 0x31, 0x1a, 0x1c, 0x67,
	0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61,
	0x74, 0x69, 0x6f, 0x6e, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x19, 0x67, 0x6f, 0x6f,
	0x67, 0x6c, 0x65, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x68, 0x74, 0x74, 0x70, 0x62, 0x6f, 0x64, 0x79,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x50, 0x0a, 0x08, 0x41, 0x72, 0x74, 0x69, 0x66, 0x61,
	0x63, 0x74, 0x12, 0x1f, 0x0a, 0x0b, 0x61, 0x72, 0x74, 0x69, 0x66, 0x61, 0x63, 0x74, 0x5f, 0x69,
	0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x61, 0x72, 0x74, 0x69, 0x66, 0x61, 0x63,
	0x74, 0x49, 0x64, 0x12, 0x23, 0x0a, 0x0d, 0x61, 0x72, 0x74, 0x69, 0x66, 0x61, 0x63, 0x74, 0x5f,
	0x74, 0x79, 0x70, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0c, 0x61, 0x72, 0x74, 0x69,
	0x66, 0x61, 0x63, 0x74, 0x54, 0x79, 0x70, 0x65, 0x22, 0x35, 0x0a, 0x12, 0x47, 0x65, 0x74, 0x41,
	0x72, 0x74, 0x69, 0x66, 0x61, 0x63, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x1f,
	0x0a, 0x0b, 0x61, 0x72, 0x74, 0x69, 0x66, 0x61, 0x63, 0x74, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x0a, 0x61, 0x72, 0x74, 0x69, 0x66, 0x61, 0x63, 0x74, 0x49, 0x64, 0x22,
	0x38, 0x0a, 0x13, 0x4c, 0x69, 0x73, 0x74, 0x41, 0x72, 0x74, 0x69, 0x66, 0x61, 0x63, 0x74, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x21, 0x0a, 0x0c, 0x65, 0x78, 0x65, 0x63, 0x75, 0x74,
	0x69, 0x6f, 0x6e, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x65, 0x78,
	0x65, 0x63, 0x75, 0x74, 0x69, 0x6f, 0x6e, 0x49, 0x64, 0x22, 0x3a, 0x0a, 0x17, 0x44, 0x6f, 0x77,
	0x6e, 0x6c, 0x6f, 0x61, 0x64, 0x41, 0x72, 0x74, 0x69, 0x66, 0x61, 0x63, 0x74, 0x52, 0x65, 0x71,
	0x75, 0x65, 0x73, 0x74, 0x12, 0x1f, 0x0a, 0x0b, 0x61, 0x72, 0x74, 0x69, 0x66, 0x61, 0x63, 0x74,
	0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x61, 0x72, 0x74, 0x69, 0x66,
	0x61, 0x63, 0x74, 0x49, 0x64, 0x22, 0x24, 0x0a, 0x0c, 0x46, 0x69, 0x6c, 0x65, 0x52, 0x65, 0x73,
	0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x14, 0x0a, 0x05, 0x63, 0x68, 0x75, 0x6e, 0x6b, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x0c, 0x52, 0x05, 0x63, 0x68, 0x75, 0x6e, 0x6b, 0x32, 0xa7, 0x04, 0x0a, 0x0f,
	0x41, 0x72, 0x74, 0x69, 0x66, 0x61, 0x63, 0x74, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12,
	0x9f, 0x01, 0x0a, 0x0d, 0x4c, 0x69, 0x73, 0x74, 0x41, 0x72, 0x74, 0x69, 0x66, 0x61, 0x63, 0x74,
	0x73, 0x12, 0x3b, 0x2e, 0x6b, 0x75, 0x62, 0x65, 0x66, 0x6c, 0x6f, 0x77, 0x2e, 0x70, 0x69, 0x70,
	0x65, 0x6c, 0x69, 0x6e, 0x65, 0x73, 0x2e, 0x62, 0x61, 0x63, 0x6b, 0x65, 0x6e, 0x64, 0x2e, 0x61,
	0x70, 0x69, 0x2e, 0x76, 0x32, 0x62, 0x65, 0x74, 0x61, 0x31, 0x2e, 0x4c, 0x69, 0x73, 0x74, 0x41,
	0x72, 0x74, 0x69, 0x66, 0x61, 0x63, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x30,
	0x2e, 0x6b, 0x75, 0x62, 0x65, 0x66, 0x6c, 0x6f, 0x77, 0x2e, 0x70, 0x69, 0x70, 0x65, 0x6c, 0x69,
	0x6e, 0x65, 0x73, 0x2e, 0x62, 0x61, 0x63, 0x6b, 0x65, 0x6e, 0x64, 0x2e, 0x61, 0x70, 0x69, 0x2e,
	0x76, 0x32, 0x62, 0x65, 0x74, 0x61, 0x31, 0x2e, 0x41, 0x72, 0x74, 0x69, 0x66, 0x61, 0x63, 0x74,
	0x22, 0x1f, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x19, 0x12, 0x17, 0x2f, 0x61, 0x70, 0x69, 0x73, 0x2f,
	0x76, 0x32, 0x62, 0x65, 0x74, 0x61, 0x31, 0x2f, 0x61, 0x72, 0x74, 0x69, 0x66, 0x61, 0x63, 0x74,
	0x73, 0x12, 0xab, 0x01, 0x0a, 0x0c, 0x47, 0x65, 0x74, 0x41, 0x72, 0x74, 0x69, 0x66, 0x61, 0x63,
	0x74, 0x73, 0x12, 0x3a, 0x2e, 0x6b, 0x75, 0x62, 0x65, 0x66, 0x6c, 0x6f, 0x77, 0x2e, 0x70, 0x69,
	0x70, 0x65, 0x6c, 0x69, 0x6e, 0x65, 0x73, 0x2e, 0x62, 0x61, 0x63, 0x6b, 0x65, 0x6e, 0x64, 0x2e,
	0x61, 0x70, 0x69, 0x2e, 0x76, 0x32, 0x62, 0x65, 0x74, 0x61, 0x31, 0x2e, 0x47, 0x65, 0x74, 0x41,
	0x72, 0x74, 0x69, 0x66, 0x61, 0x63, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x30,
	0x2e, 0x6b, 0x75, 0x62, 0x65, 0x66, 0x6c, 0x6f, 0x77, 0x2e, 0x70, 0x69, 0x70, 0x65, 0x6c, 0x69,
	0x6e, 0x65, 0x73, 0x2e, 0x62, 0x61, 0x63, 0x6b, 0x65, 0x6e, 0x64, 0x2e, 0x61, 0x70, 0x69, 0x2e,
	0x76, 0x32, 0x62, 0x65, 0x74, 0x61, 0x31, 0x2e, 0x41, 0x72, 0x74, 0x69, 0x66, 0x61, 0x63, 0x74,
	0x22, 0x2d, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x27, 0x12, 0x25, 0x2f, 0x61, 0x70, 0x69, 0x73, 0x2f,
	0x76, 0x32, 0x62, 0x65, 0x74, 0x61, 0x31, 0x2f, 0x61, 0x72, 0x74, 0x69, 0x66, 0x61, 0x63, 0x74,
	0x73, 0x2f, 0x7b, 0x61, 0x72, 0x74, 0x69, 0x66, 0x61, 0x63, 0x74, 0x5f, 0x69, 0x64, 0x7d, 0x12,
	0xc3, 0x01, 0x0a, 0x10, 0x44, 0x6f, 0x77, 0x6e, 0x6c, 0x6f, 0x61, 0x64, 0x41, 0x72, 0x74, 0x69,
	0x66, 0x61, 0x63, 0x74, 0x12, 0x3f, 0x2e, 0x6b, 0x75, 0x62, 0x65, 0x66, 0x6c, 0x6f, 0x77, 0x2e,
	0x70, 0x69, 0x70, 0x65, 0x6c, 0x69, 0x6e, 0x65, 0x73, 0x2e, 0x62, 0x61, 0x63, 0x6b, 0x65, 0x6e,
	0x64, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x32, 0x62, 0x65, 0x74, 0x61, 0x31, 0x2e, 0x44, 0x6f,
	0x77, 0x6e, 0x6c, 0x6f, 0x61, 0x64, 0x41, 0x72, 0x74, 0x69, 0x66, 0x61, 0x63, 0x74, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x34, 0x2e, 0x6b, 0x75, 0x62, 0x65, 0x66, 0x6c, 0x6f, 0x77,
	0x2e, 0x70, 0x69, 0x70, 0x65, 0x6c, 0x69, 0x6e, 0x65, 0x73, 0x2e, 0x62, 0x61, 0x63, 0x6b, 0x65,
	0x6e, 0x64, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x32, 0x62, 0x65, 0x74, 0x61, 0x31, 0x2e, 0x46,
	0x69, 0x6c, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x36, 0x82, 0xd3, 0xe4,
	0x93, 0x02, 0x30, 0x12, 0x2e, 0x2f, 0x61, 0x70, 0x69, 0x73, 0x2f, 0x76, 0x32, 0x62, 0x65, 0x74,
	0x61, 0x31, 0x2f, 0x61, 0x72, 0x74, 0x69, 0x66, 0x61, 0x63, 0x74, 0x73, 0x2f, 0x7b, 0x61, 0x72,
	0x74, 0x69, 0x66, 0x61, 0x63, 0x74, 0x5f, 0x69, 0x64, 0x7d, 0x3a, 0x64, 0x6f, 0x77, 0x6e, 0x6c,
	0x6f, 0x61, 0x64, 0x30, 0x01, 0x42, 0x3d, 0x5a, 0x3b, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e,
	0x63, 0x6f, 0x6d, 0x2f, 0x6b, 0x75, 0x62, 0x65, 0x66, 0x6c, 0x6f, 0x77, 0x2f, 0x70, 0x69, 0x70,
	0x65, 0x6c, 0x69, 0x6e, 0x65, 0x73, 0x2f, 0x62, 0x61, 0x63, 0x6b, 0x65, 0x6e, 0x64, 0x2f, 0x61,
	0x70, 0x69, 0x2f, 0x76, 0x32, 0x62, 0x65, 0x74, 0x61, 0x31, 0x2f, 0x67, 0x6f, 0x5f, 0x63, 0x6c,
	0x69, 0x65, 0x6e, 0x74, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_backend_api_v2beta1_artifacts_proto_rawDescOnce sync.Once
	file_backend_api_v2beta1_artifacts_proto_rawDescData = file_backend_api_v2beta1_artifacts_proto_rawDesc
)

func file_backend_api_v2beta1_artifacts_proto_rawDescGZIP() []byte {
	file_backend_api_v2beta1_artifacts_proto_rawDescOnce.Do(func() {
		file_backend_api_v2beta1_artifacts_proto_rawDescData = protoimpl.X.CompressGZIP(file_backend_api_v2beta1_artifacts_proto_rawDescData)
	})
	return file_backend_api_v2beta1_artifacts_proto_rawDescData
}

var file_backend_api_v2beta1_artifacts_proto_msgTypes = make([]protoimpl.MessageInfo, 5)
var file_backend_api_v2beta1_artifacts_proto_goTypes = []interface{}{
	(*Artifact)(nil),                // 0: kubeflow.pipelines.backend.api.v2beta1.Artifact
	(*GetArtifactRequest)(nil),      // 1: kubeflow.pipelines.backend.api.v2beta1.GetArtifactRequest
	(*ListArtifactRequest)(nil),     // 2: kubeflow.pipelines.backend.api.v2beta1.ListArtifactRequest
	(*DownloadArtifactRequest)(nil), // 3: kubeflow.pipelines.backend.api.v2beta1.DownloadArtifactRequest
	(*FileResponse)(nil),            // 4: kubeflow.pipelines.backend.api.v2beta1.FileResponse
}
var file_backend_api_v2beta1_artifacts_proto_depIdxs = []int32{
	2, // 0: kubeflow.pipelines.backend.api.v2beta1.ArtifactService.ListArtifacts:input_type -> kubeflow.pipelines.backend.api.v2beta1.ListArtifactRequest
	1, // 1: kubeflow.pipelines.backend.api.v2beta1.ArtifactService.GetArtifacts:input_type -> kubeflow.pipelines.backend.api.v2beta1.GetArtifactRequest
	3, // 2: kubeflow.pipelines.backend.api.v2beta1.ArtifactService.DownloadArtifact:input_type -> kubeflow.pipelines.backend.api.v2beta1.DownloadArtifactRequest
	0, // 3: kubeflow.pipelines.backend.api.v2beta1.ArtifactService.ListArtifacts:output_type -> kubeflow.pipelines.backend.api.v2beta1.Artifact
	0, // 4: kubeflow.pipelines.backend.api.v2beta1.ArtifactService.GetArtifacts:output_type -> kubeflow.pipelines.backend.api.v2beta1.Artifact
	4, // 5: kubeflow.pipelines.backend.api.v2beta1.ArtifactService.DownloadArtifact:output_type -> kubeflow.pipelines.backend.api.v2beta1.FileResponse
	3, // [3:6] is the sub-list for method output_type
	0, // [0:3] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_backend_api_v2beta1_artifacts_proto_init() }
func file_backend_api_v2beta1_artifacts_proto_init() {
	if File_backend_api_v2beta1_artifacts_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_backend_api_v2beta1_artifacts_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Artifact); i {
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
		file_backend_api_v2beta1_artifacts_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetArtifactRequest); i {
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
		file_backend_api_v2beta1_artifacts_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ListArtifactRequest); i {
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
		file_backend_api_v2beta1_artifacts_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DownloadArtifactRequest); i {
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
		file_backend_api_v2beta1_artifacts_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*FileResponse); i {
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
			RawDescriptor: file_backend_api_v2beta1_artifacts_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   5,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_backend_api_v2beta1_artifacts_proto_goTypes,
		DependencyIndexes: file_backend_api_v2beta1_artifacts_proto_depIdxs,
		MessageInfos:      file_backend_api_v2beta1_artifacts_proto_msgTypes,
	}.Build()
	File_backend_api_v2beta1_artifacts_proto = out.File
	file_backend_api_v2beta1_artifacts_proto_rawDesc = nil
	file_backend_api_v2beta1_artifacts_proto_goTypes = nil
	file_backend_api_v2beta1_artifacts_proto_depIdxs = nil
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConnInterface

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion6

// ArtifactServiceClient is the client API for ArtifactService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type ArtifactServiceClient interface {
	ListArtifacts(ctx context.Context, in *ListArtifactRequest, opts ...grpc.CallOption) (*Artifact, error)
	GetArtifacts(ctx context.Context, in *GetArtifactRequest, opts ...grpc.CallOption) (*Artifact, error)
	DownloadArtifact(ctx context.Context, in *DownloadArtifactRequest, opts ...grpc.CallOption) (ArtifactService_DownloadArtifactClient, error)
}

type artifactServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewArtifactServiceClient(cc grpc.ClientConnInterface) ArtifactServiceClient {
	return &artifactServiceClient{cc}
}

func (c *artifactServiceClient) ListArtifacts(ctx context.Context, in *ListArtifactRequest, opts ...grpc.CallOption) (*Artifact, error) {
	out := new(Artifact)
	err := c.cc.Invoke(ctx, "/kubeflow.pipelines.backend.api.v2beta1.ArtifactService/ListArtifacts", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *artifactServiceClient) GetArtifacts(ctx context.Context, in *GetArtifactRequest, opts ...grpc.CallOption) (*Artifact, error) {
	out := new(Artifact)
	err := c.cc.Invoke(ctx, "/kubeflow.pipelines.backend.api.v2beta1.ArtifactService/GetArtifacts", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *artifactServiceClient) DownloadArtifact(ctx context.Context, in *DownloadArtifactRequest, opts ...grpc.CallOption) (ArtifactService_DownloadArtifactClient, error) {
	stream, err := c.cc.NewStream(ctx, &_ArtifactService_serviceDesc.Streams[0], "/kubeflow.pipelines.backend.api.v2beta1.ArtifactService/DownloadArtifact", opts...)
	if err != nil {
		return nil, err
	}
	x := &artifactServiceDownloadArtifactClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type ArtifactService_DownloadArtifactClient interface {
	Recv() (*FileResponse, error)
	grpc.ClientStream
}

type artifactServiceDownloadArtifactClient struct {
	grpc.ClientStream
}

func (x *artifactServiceDownloadArtifactClient) Recv() (*FileResponse, error) {
	m := new(FileResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// ArtifactServiceServer is the server API for ArtifactService service.
type ArtifactServiceServer interface {
	ListArtifacts(context.Context, *ListArtifactRequest) (*Artifact, error)
	GetArtifacts(context.Context, *GetArtifactRequest) (*Artifact, error)
	DownloadArtifact(*DownloadArtifactRequest, ArtifactService_DownloadArtifactServer) error
}

// UnimplementedArtifactServiceServer can be embedded to have forward compatible implementations.
type UnimplementedArtifactServiceServer struct {
}

func (*UnimplementedArtifactServiceServer) ListArtifacts(context.Context, *ListArtifactRequest) (*Artifact, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListArtifacts not implemented")
}
func (*UnimplementedArtifactServiceServer) GetArtifacts(context.Context, *GetArtifactRequest) (*Artifact, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetArtifacts not implemented")
}
func (*UnimplementedArtifactServiceServer) DownloadArtifact(*DownloadArtifactRequest, ArtifactService_DownloadArtifactServer) error {
	return status.Errorf(codes.Unimplemented, "method DownloadArtifact not implemented")
}

func RegisterArtifactServiceServer(s *grpc.Server, srv ArtifactServiceServer) {
	s.RegisterService(&_ArtifactService_serviceDesc, srv)
}

func _ArtifactService_ListArtifacts_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListArtifactRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ArtifactServiceServer).ListArtifacts(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/kubeflow.pipelines.backend.api.v2beta1.ArtifactService/ListArtifacts",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ArtifactServiceServer).ListArtifacts(ctx, req.(*ListArtifactRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ArtifactService_GetArtifacts_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetArtifactRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ArtifactServiceServer).GetArtifacts(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/kubeflow.pipelines.backend.api.v2beta1.ArtifactService/GetArtifacts",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ArtifactServiceServer).GetArtifacts(ctx, req.(*GetArtifactRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ArtifactService_DownloadArtifact_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(DownloadArtifactRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(ArtifactServiceServer).DownloadArtifact(m, &artifactServiceDownloadArtifactServer{stream})
}

type ArtifactService_DownloadArtifactServer interface {
	Send(*FileResponse) error
	grpc.ServerStream
}

type artifactServiceDownloadArtifactServer struct {
	grpc.ServerStream
}

func (x *artifactServiceDownloadArtifactServer) Send(m *FileResponse) error {
	return x.ServerStream.SendMsg(m)
}

var _ArtifactService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "kubeflow.pipelines.backend.api.v2beta1.ArtifactService",
	HandlerType: (*ArtifactServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "ListArtifacts",
			Handler:    _ArtifactService_ListArtifacts_Handler,
		},
		{
			MethodName: "GetArtifacts",
			Handler:    _ArtifactService_GetArtifacts_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "DownloadArtifact",
			Handler:       _ArtifactService_DownloadArtifact_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "backend/api/v2beta1/artifacts.proto",
}
