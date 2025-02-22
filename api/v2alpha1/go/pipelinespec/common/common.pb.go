// Copyright 2025 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.33.0
// 	protoc        v3.17.3
// source: common/common.proto

package commonspec

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	structpb "google.golang.org/protobuf/types/known/structpb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// Represents an input parameter. The value can be taken from an upstream
// task's output parameter (if specifying `producer_task` and
// `output_parameter_key`, or it can be a runtime value, which can either be
// determined at compile-time, or from a pipeline parameter.
type InputParameterSpec struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Types that are assignable to Kind:
	//
	//	*InputParameterSpec_TaskOutputParameter
	//	*InputParameterSpec_RuntimeValue
	//	*InputParameterSpec_ComponentInputParameter
	//	*InputParameterSpec_TaskFinalStatus_
	Kind isInputParameterSpec_Kind `protobuf_oneof:"kind"`
	// Selector expression of Common Expression Language (CEL)
	// that applies to the parameter found from above kind.
	//
	// The expression is applied to the Value type
	// [Value][].  For example,
	// 'size(string_value)' will return the size of the Value.string_value.
	//
	// After applying the selection, the parameter will be returned as a
	// [Value][].  The type of the Value is either deferred from the input
	// definition in the corresponding
	// [ComponentSpec.input_definitions.parameters][], or if not found,
	// automatically deferred as either string value or double value.
	//
	// In addition to the builtin functions in CEL, The value.string_value can
	// be treated as a json string and parsed to the [google.protobuf.Value][]
	// proto message. Then, the CEL expression provided in this field will be
	// used to get the requested field. For examples,
	//   - if Value.string_value is a json array of "[1.1, 2.2, 3.3]",
	//     'parseJson(string_value)[i]' will pass the ith parameter from the list
	//     to the current task, or
	//   - if the Value.string_value is a json map of "{"a": 1.1, "b": 2.2,
	//     "c": 3.3}, 'parseJson(string_value)[key]' will pass the map value from
	//     the struct map to the current task.
	//
	// If unset, the value will be passed directly to the current task.
	ParameterExpressionSelector string `protobuf:"bytes,4,opt,name=parameter_expression_selector,json=parameterExpressionSelector,proto3" json:"parameter_expression_selector,omitempty"`
}

func (x *InputParameterSpec) Reset() {
	*x = InputParameterSpec{}
	if protoimpl.UnsafeEnabled {
		mi := &file_common_common_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *InputParameterSpec) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*InputParameterSpec) ProtoMessage() {}

func (x *InputParameterSpec) ProtoReflect() protoreflect.Message {
	mi := &file_common_common_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use InputParameterSpec.ProtoReflect.Descriptor instead.
func (*InputParameterSpec) Descriptor() ([]byte, []int) {
	return file_common_common_proto_rawDescGZIP(), []int{0}
}

func (m *InputParameterSpec) GetKind() isInputParameterSpec_Kind {
	if m != nil {
		return m.Kind
	}
	return nil
}

func (x *InputParameterSpec) GetTaskOutputParameter() *InputParameterSpec_TaskOutputParameterSpec {
	if x, ok := x.GetKind().(*InputParameterSpec_TaskOutputParameter); ok {
		return x.TaskOutputParameter
	}
	return nil
}

func (x *InputParameterSpec) GetRuntimeValue() *ValueOrRuntimeParameter {
	if x, ok := x.GetKind().(*InputParameterSpec_RuntimeValue); ok {
		return x.RuntimeValue
	}
	return nil
}

func (x *InputParameterSpec) GetComponentInputParameter() string {
	if x, ok := x.GetKind().(*InputParameterSpec_ComponentInputParameter); ok {
		return x.ComponentInputParameter
	}
	return ""
}

func (x *InputParameterSpec) GetTaskFinalStatus() *InputParameterSpec_TaskFinalStatus {
	if x, ok := x.GetKind().(*InputParameterSpec_TaskFinalStatus_); ok {
		return x.TaskFinalStatus
	}
	return nil
}

func (x *InputParameterSpec) GetParameterExpressionSelector() string {
	if x != nil {
		return x.ParameterExpressionSelector
	}
	return ""
}

type isInputParameterSpec_Kind interface {
	isInputParameterSpec_Kind()
}

type InputParameterSpec_TaskOutputParameter struct {
	// Output parameter from an upstream task.
	TaskOutputParameter *InputParameterSpec_TaskOutputParameterSpec `protobuf:"bytes,1,opt,name=task_output_parameter,json=taskOutputParameter,proto3,oneof"`
}

type InputParameterSpec_RuntimeValue struct {
	// A constant value or runtime parameter.
	RuntimeValue *ValueOrRuntimeParameter `protobuf:"bytes,2,opt,name=runtime_value,json=runtimeValue,proto3,oneof"`
}

type InputParameterSpec_ComponentInputParameter struct {
	// Pass the input parameter from parent component input parameter.
	ComponentInputParameter string `protobuf:"bytes,3,opt,name=component_input_parameter,json=componentInputParameter,proto3,oneof"`
}

type InputParameterSpec_TaskFinalStatus_ struct {
	// The final status of an upstream task.
	TaskFinalStatus *InputParameterSpec_TaskFinalStatus `protobuf:"bytes,5,opt,name=task_final_status,json=taskFinalStatus,proto3,oneof"`
}

func (*InputParameterSpec_TaskOutputParameter) isInputParameterSpec_Kind() {}

func (*InputParameterSpec_RuntimeValue) isInputParameterSpec_Kind() {}

func (*InputParameterSpec_ComponentInputParameter) isInputParameterSpec_Kind() {}

func (*InputParameterSpec_TaskFinalStatus_) isInputParameterSpec_Kind() {}

// Definition for a value or reference to a runtime parameter. A
// ValueOrRuntimeParameter instance can be either a field value that is
// determined during compilation time, or a runtime parameter which will be
// determined during runtime.
type ValueOrRuntimeParameter struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Types that are assignable to Value:
	//
	//	*ValueOrRuntimeParameter_ConstantValue
	//	*ValueOrRuntimeParameter_RuntimeParameter
	//	*ValueOrRuntimeParameter_Constant
	Value isValueOrRuntimeParameter_Value `protobuf_oneof:"value"`
}

func (x *ValueOrRuntimeParameter) Reset() {
	*x = ValueOrRuntimeParameter{}
	if protoimpl.UnsafeEnabled {
		mi := &file_common_common_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ValueOrRuntimeParameter) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ValueOrRuntimeParameter) ProtoMessage() {}

func (x *ValueOrRuntimeParameter) ProtoReflect() protoreflect.Message {
	mi := &file_common_common_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ValueOrRuntimeParameter.ProtoReflect.Descriptor instead.
func (*ValueOrRuntimeParameter) Descriptor() ([]byte, []int) {
	return file_common_common_proto_rawDescGZIP(), []int{1}
}

func (m *ValueOrRuntimeParameter) GetValue() isValueOrRuntimeParameter_Value {
	if m != nil {
		return m.Value
	}
	return nil
}

// Deprecated: Marked as deprecated in common/common.proto.
func (x *ValueOrRuntimeParameter) GetConstantValue() *Value {
	if x, ok := x.GetValue().(*ValueOrRuntimeParameter_ConstantValue); ok {
		return x.ConstantValue
	}
	return nil
}

func (x *ValueOrRuntimeParameter) GetRuntimeParameter() string {
	if x, ok := x.GetValue().(*ValueOrRuntimeParameter_RuntimeParameter); ok {
		return x.RuntimeParameter
	}
	return ""
}

func (x *ValueOrRuntimeParameter) GetConstant() *structpb.Value {
	if x, ok := x.GetValue().(*ValueOrRuntimeParameter_Constant); ok {
		return x.Constant
	}
	return nil
}

type isValueOrRuntimeParameter_Value interface {
	isValueOrRuntimeParameter_Value()
}

type ValueOrRuntimeParameter_ConstantValue struct {
	// Constant value which is determined in compile time.
	// Deprecated. Use [ValueOrRuntimeParameter.constant][] instead.
	//
	// Deprecated: Marked as deprecated in common/common.proto.
	ConstantValue *Value `protobuf:"bytes,1,opt,name=constant_value,json=constantValue,proto3,oneof"`
}

type ValueOrRuntimeParameter_RuntimeParameter struct {
	// The runtime parameter refers to the parent component input parameter.
	RuntimeParameter string `protobuf:"bytes,2,opt,name=runtime_parameter,json=runtimeParameter,proto3,oneof"`
}

type ValueOrRuntimeParameter_Constant struct {
	// Constant value which is determined in compile time.
	Constant *structpb.Value `protobuf:"bytes,3,opt,name=constant,proto3,oneof"`
}

func (*ValueOrRuntimeParameter_ConstantValue) isValueOrRuntimeParameter_Value() {}

func (*ValueOrRuntimeParameter_RuntimeParameter) isValueOrRuntimeParameter_Value() {}

func (*ValueOrRuntimeParameter_Constant) isValueOrRuntimeParameter_Value() {}

// Value is the value of the field.
type Value struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Types that are assignable to Value:
	//
	//	*Value_IntValue
	//	*Value_DoubleValue
	//	*Value_StringValue
	Value isValue_Value `protobuf_oneof:"value"`
}

func (x *Value) Reset() {
	*x = Value{}
	if protoimpl.UnsafeEnabled {
		mi := &file_common_common_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Value) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Value) ProtoMessage() {}

func (x *Value) ProtoReflect() protoreflect.Message {
	mi := &file_common_common_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Value.ProtoReflect.Descriptor instead.
func (*Value) Descriptor() ([]byte, []int) {
	return file_common_common_proto_rawDescGZIP(), []int{2}
}

func (m *Value) GetValue() isValue_Value {
	if m != nil {
		return m.Value
	}
	return nil
}

func (x *Value) GetIntValue() int64 {
	if x, ok := x.GetValue().(*Value_IntValue); ok {
		return x.IntValue
	}
	return 0
}

func (x *Value) GetDoubleValue() float64 {
	if x, ok := x.GetValue().(*Value_DoubleValue); ok {
		return x.DoubleValue
	}
	return 0
}

func (x *Value) GetStringValue() string {
	if x, ok := x.GetValue().(*Value_StringValue); ok {
		return x.StringValue
	}
	return ""
}

type isValue_Value interface {
	isValue_Value()
}

type Value_IntValue struct {
	// An integer value
	IntValue int64 `protobuf:"varint,1,opt,name=int_value,json=intValue,proto3,oneof"`
}

type Value_DoubleValue struct {
	// A double value
	DoubleValue float64 `protobuf:"fixed64,2,opt,name=double_value,json=doubleValue,proto3,oneof"`
}

type Value_StringValue struct {
	// A string value
	StringValue string `protobuf:"bytes,3,opt,name=string_value,json=stringValue,proto3,oneof"`
}

func (*Value_IntValue) isValue_Value() {}

func (*Value_DoubleValue) isValue_Value() {}

func (*Value_StringValue) isValue_Value() {}

// Represents an upstream task's output parameter.
type TaskOutputParameterSpec struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The name of the upstream task which produces the output parameter that
	// matches with the `output_parameter_key`.
	ProducerTask string `protobuf:"bytes,1,opt,name=producer_task,json=producerTask,proto3" json:"producer_task,omitempty"`
	// The key of [TaskOutputsSpec.parameters][] map of the producer task.
	OutputParameterKey string `protobuf:"bytes,2,opt,name=output_parameter_key,json=outputParameterKey,proto3" json:"output_parameter_key,omitempty"`
}

func (x *TaskOutputParameterSpec) Reset() {
	*x = TaskOutputParameterSpec{}
	if protoimpl.UnsafeEnabled {
		mi := &file_common_common_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TaskOutputParameterSpec) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TaskOutputParameterSpec) ProtoMessage() {}

func (x *TaskOutputParameterSpec) ProtoReflect() protoreflect.Message {
	mi := &file_common_common_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TaskOutputParameterSpec.ProtoReflect.Descriptor instead.
func (*TaskOutputParameterSpec) Descriptor() ([]byte, []int) {
	return file_common_common_proto_rawDescGZIP(), []int{3}
}

func (x *TaskOutputParameterSpec) GetProducerTask() string {
	if x != nil {
		return x.ProducerTask
	}
	return ""
}

func (x *TaskOutputParameterSpec) GetOutputParameterKey() string {
	if x != nil {
		return x.OutputParameterKey
	}
	return ""
}

// Represents an upstream task's output parameter.
type InputParameterSpec_TaskOutputParameterSpec struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The name of the upstream task which produces the output parameter that
	// matches with the `output_parameter_key`.
	ProducerTask string `protobuf:"bytes,1,opt,name=producer_task,json=producerTask,proto3" json:"producer_task,omitempty"`
	// The key of [TaskOutputsSpec.parameters][] map of the producer task.
	OutputParameterKey string `protobuf:"bytes,2,opt,name=output_parameter_key,json=outputParameterKey,proto3" json:"output_parameter_key,omitempty"`
}

func (x *InputParameterSpec_TaskOutputParameterSpec) Reset() {
	*x = InputParameterSpec_TaskOutputParameterSpec{}
	if protoimpl.UnsafeEnabled {
		mi := &file_common_common_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *InputParameterSpec_TaskOutputParameterSpec) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*InputParameterSpec_TaskOutputParameterSpec) ProtoMessage() {}

func (x *InputParameterSpec_TaskOutputParameterSpec) ProtoReflect() protoreflect.Message {
	mi := &file_common_common_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use InputParameterSpec_TaskOutputParameterSpec.ProtoReflect.Descriptor instead.
func (*InputParameterSpec_TaskOutputParameterSpec) Descriptor() ([]byte, []int) {
	return file_common_common_proto_rawDescGZIP(), []int{0, 0}
}

func (x *InputParameterSpec_TaskOutputParameterSpec) GetProducerTask() string {
	if x != nil {
		return x.ProducerTask
	}
	return ""
}

func (x *InputParameterSpec_TaskOutputParameterSpec) GetOutputParameterKey() string {
	if x != nil {
		return x.OutputParameterKey
	}
	return ""
}

// Represents an upstream task's final status. The field can only be set if
// the schema version is `2.0.0`. The resolved input parameter will be a
// json payload in string type.
type InputParameterSpec_TaskFinalStatus struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The name of the upsteram task where the final status is coming from.
	ProducerTask string `protobuf:"bytes,1,opt,name=producer_task,json=producerTask,proto3" json:"producer_task,omitempty"`
}

func (x *InputParameterSpec_TaskFinalStatus) Reset() {
	*x = InputParameterSpec_TaskFinalStatus{}
	if protoimpl.UnsafeEnabled {
		mi := &file_common_common_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *InputParameterSpec_TaskFinalStatus) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*InputParameterSpec_TaskFinalStatus) ProtoMessage() {}

func (x *InputParameterSpec_TaskFinalStatus) ProtoReflect() protoreflect.Message {
	mi := &file_common_common_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use InputParameterSpec_TaskFinalStatus.ProtoReflect.Descriptor instead.
func (*InputParameterSpec_TaskFinalStatus) Descriptor() ([]byte, []int) {
	return file_common_common_proto_rawDescGZIP(), []int{0, 1}
}

func (x *InputParameterSpec_TaskFinalStatus) GetProducerTask() string {
	if x != nil {
		return x.ProducerTask
	}
	return ""
}

var File_common_common_proto protoreflect.FileDescriptor

var file_common_common_proto_rawDesc = []byte{
	0x0a, 0x13, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x2f, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0a, 0x6b, 0x66, 0x70, 0x5f, 0x63, 0x6f, 0x6d, 0x6d, 0x6f,
	0x6e, 0x1a, 0x1c, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62,
	0x75, 0x66, 0x2f, 0x73, 0x74, 0x72, 0x75, 0x63, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22,
	0xe0, 0x04, 0x0a, 0x12, 0x49, 0x6e, 0x70, 0x75, 0x74, 0x50, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74,
	0x65, 0x72, 0x53, 0x70, 0x65, 0x63, 0x12, 0x6c, 0x0a, 0x15, 0x74, 0x61, 0x73, 0x6b, 0x5f, 0x6f,
	0x75, 0x74, 0x70, 0x75, 0x74, 0x5f, 0x70, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74, 0x65, 0x72, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x36, 0x2e, 0x6b, 0x66, 0x70, 0x5f, 0x63, 0x6f, 0x6d, 0x6d,
	0x6f, 0x6e, 0x2e, 0x49, 0x6e, 0x70, 0x75, 0x74, 0x50, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74, 0x65,
	0x72, 0x53, 0x70, 0x65, 0x63, 0x2e, 0x54, 0x61, 0x73, 0x6b, 0x4f, 0x75, 0x74, 0x70, 0x75, 0x74,
	0x50, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74, 0x65, 0x72, 0x53, 0x70, 0x65, 0x63, 0x48, 0x00, 0x52,
	0x13, 0x74, 0x61, 0x73, 0x6b, 0x4f, 0x75, 0x74, 0x70, 0x75, 0x74, 0x50, 0x61, 0x72, 0x61, 0x6d,
	0x65, 0x74, 0x65, 0x72, 0x12, 0x4a, 0x0a, 0x0d, 0x72, 0x75, 0x6e, 0x74, 0x69, 0x6d, 0x65, 0x5f,
	0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x23, 0x2e, 0x6b, 0x66,
	0x70, 0x5f, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x2e, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x4f, 0x72,
	0x52, 0x75, 0x6e, 0x74, 0x69, 0x6d, 0x65, 0x50, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74, 0x65, 0x72,
	0x48, 0x00, 0x52, 0x0c, 0x72, 0x75, 0x6e, 0x74, 0x69, 0x6d, 0x65, 0x56, 0x61, 0x6c, 0x75, 0x65,
	0x12, 0x3c, 0x0a, 0x19, 0x63, 0x6f, 0x6d, 0x70, 0x6f, 0x6e, 0x65, 0x6e, 0x74, 0x5f, 0x69, 0x6e,
	0x70, 0x75, 0x74, 0x5f, 0x70, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74, 0x65, 0x72, 0x18, 0x03, 0x20,
	0x01, 0x28, 0x09, 0x48, 0x00, 0x52, 0x17, 0x63, 0x6f, 0x6d, 0x70, 0x6f, 0x6e, 0x65, 0x6e, 0x74,
	0x49, 0x6e, 0x70, 0x75, 0x74, 0x50, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74, 0x65, 0x72, 0x12, 0x5c,
	0x0a, 0x11, 0x74, 0x61, 0x73, 0x6b, 0x5f, 0x66, 0x69, 0x6e, 0x61, 0x6c, 0x5f, 0x73, 0x74, 0x61,
	0x74, 0x75, 0x73, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x2e, 0x2e, 0x6b, 0x66, 0x70, 0x5f,
	0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x2e, 0x49, 0x6e, 0x70, 0x75, 0x74, 0x50, 0x61, 0x72, 0x61,
	0x6d, 0x65, 0x74, 0x65, 0x72, 0x53, 0x70, 0x65, 0x63, 0x2e, 0x54, 0x61, 0x73, 0x6b, 0x46, 0x69,
	0x6e, 0x61, 0x6c, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x48, 0x00, 0x52, 0x0f, 0x74, 0x61, 0x73,
	0x6b, 0x46, 0x69, 0x6e, 0x61, 0x6c, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x42, 0x0a, 0x1d,
	0x70, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74, 0x65, 0x72, 0x5f, 0x65, 0x78, 0x70, 0x72, 0x65, 0x73,
	0x73, 0x69, 0x6f, 0x6e, 0x5f, 0x73, 0x65, 0x6c, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x18, 0x04, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x1b, 0x70, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74, 0x65, 0x72, 0x45, 0x78,
	0x70, 0x72, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x53, 0x65, 0x6c, 0x65, 0x63, 0x74, 0x6f, 0x72,
	0x1a, 0x70, 0x0a, 0x17, 0x54, 0x61, 0x73, 0x6b, 0x4f, 0x75, 0x74, 0x70, 0x75, 0x74, 0x50, 0x61,
	0x72, 0x61, 0x6d, 0x65, 0x74, 0x65, 0x72, 0x53, 0x70, 0x65, 0x63, 0x12, 0x23, 0x0a, 0x0d, 0x70,
	0x72, 0x6f, 0x64, 0x75, 0x63, 0x65, 0x72, 0x5f, 0x74, 0x61, 0x73, 0x6b, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x0c, 0x70, 0x72, 0x6f, 0x64, 0x75, 0x63, 0x65, 0x72, 0x54, 0x61, 0x73, 0x6b,
	0x12, 0x30, 0x0a, 0x14, 0x6f, 0x75, 0x74, 0x70, 0x75, 0x74, 0x5f, 0x70, 0x61, 0x72, 0x61, 0x6d,
	0x65, 0x74, 0x65, 0x72, 0x5f, 0x6b, 0x65, 0x79, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x12,
	0x6f, 0x75, 0x74, 0x70, 0x75, 0x74, 0x50, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74, 0x65, 0x72, 0x4b,
	0x65, 0x79, 0x1a, 0x36, 0x0a, 0x0f, 0x54, 0x61, 0x73, 0x6b, 0x46, 0x69, 0x6e, 0x61, 0x6c, 0x53,
	0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x23, 0x0a, 0x0d, 0x70, 0x72, 0x6f, 0x64, 0x75, 0x63, 0x65,
	0x72, 0x5f, 0x74, 0x61, 0x73, 0x6b, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0c, 0x70, 0x72,
	0x6f, 0x64, 0x75, 0x63, 0x65, 0x72, 0x54, 0x61, 0x73, 0x6b, 0x42, 0x06, 0x0a, 0x04, 0x6b, 0x69,
	0x6e, 0x64, 0x22, 0xc7, 0x01, 0x0a, 0x17, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x4f, 0x72, 0x52, 0x75,
	0x6e, 0x74, 0x69, 0x6d, 0x65, 0x50, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74, 0x65, 0x72, 0x12, 0x3e,
	0x0a, 0x0e, 0x63, 0x6f, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x74, 0x5f, 0x76, 0x61, 0x6c, 0x75, 0x65,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x11, 0x2e, 0x6b, 0x66, 0x70, 0x5f, 0x63, 0x6f, 0x6d,
	0x6d, 0x6f, 0x6e, 0x2e, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x42, 0x02, 0x18, 0x01, 0x48, 0x00, 0x52,
	0x0d, 0x63, 0x6f, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x74, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x12, 0x2d,
	0x0a, 0x11, 0x72, 0x75, 0x6e, 0x74, 0x69, 0x6d, 0x65, 0x5f, 0x70, 0x61, 0x72, 0x61, 0x6d, 0x65,
	0x74, 0x65, 0x72, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x48, 0x00, 0x52, 0x10, 0x72, 0x75, 0x6e,
	0x74, 0x69, 0x6d, 0x65, 0x50, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74, 0x65, 0x72, 0x12, 0x34, 0x0a,
	0x08, 0x63, 0x6f, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x74, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x2e, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x48, 0x00, 0x52, 0x08, 0x63, 0x6f, 0x6e, 0x73, 0x74,
	0x61, 0x6e, 0x74, 0x42, 0x07, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x22, 0x79, 0x0a, 0x05,
	0x56, 0x61, 0x6c, 0x75, 0x65, 0x12, 0x1d, 0x0a, 0x09, 0x69, 0x6e, 0x74, 0x5f, 0x76, 0x61, 0x6c,
	0x75, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x03, 0x48, 0x00, 0x52, 0x08, 0x69, 0x6e, 0x74, 0x56,
	0x61, 0x6c, 0x75, 0x65, 0x12, 0x23, 0x0a, 0x0c, 0x64, 0x6f, 0x75, 0x62, 0x6c, 0x65, 0x5f, 0x76,
	0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x01, 0x48, 0x00, 0x52, 0x0b, 0x64, 0x6f,
	0x75, 0x62, 0x6c, 0x65, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x12, 0x23, 0x0a, 0x0c, 0x73, 0x74, 0x72,
	0x69, 0x6e, 0x67, 0x5f, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x48,
	0x00, 0x52, 0x0b, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x42, 0x07,
	0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x22, 0x70, 0x0a, 0x17, 0x54, 0x61, 0x73, 0x6b, 0x4f,
	0x75, 0x74, 0x70, 0x75, 0x74, 0x50, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74, 0x65, 0x72, 0x53, 0x70,
	0x65, 0x63, 0x12, 0x23, 0x0a, 0x0d, 0x70, 0x72, 0x6f, 0x64, 0x75, 0x63, 0x65, 0x72, 0x5f, 0x74,
	0x61, 0x73, 0x6b, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0c, 0x70, 0x72, 0x6f, 0x64, 0x75,
	0x63, 0x65, 0x72, 0x54, 0x61, 0x73, 0x6b, 0x12, 0x30, 0x0a, 0x14, 0x6f, 0x75, 0x74, 0x70, 0x75,
	0x74, 0x5f, 0x70, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74, 0x65, 0x72, 0x5f, 0x6b, 0x65, 0x79, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x12, 0x6f, 0x75, 0x74, 0x70, 0x75, 0x74, 0x50, 0x61, 0x72,
	0x61, 0x6d, 0x65, 0x74, 0x65, 0x72, 0x4b, 0x65, 0x79, 0x42, 0x34, 0x5a, 0x32, 0x67, 0x69, 0x74,
	0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x6b, 0x75, 0x62, 0x65, 0x66, 0x6c, 0x6f, 0x77,
	0x2f, 0x70, 0x69, 0x70, 0x65, 0x6c, 0x69, 0x6e, 0x65, 0x73, 0x2f, 0x63, 0x6f, 0x6d, 0x6d, 0x6f,
	0x6e, 0x2f, 0x67, 0x6f, 0x2f, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x73, 0x70, 0x65, 0x63, 0x62,
	0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_common_common_proto_rawDescOnce sync.Once
	file_common_common_proto_rawDescData = file_common_common_proto_rawDesc
)

func file_common_common_proto_rawDescGZIP() []byte {
	file_common_common_proto_rawDescOnce.Do(func() {
		file_common_common_proto_rawDescData = protoimpl.X.CompressGZIP(file_common_common_proto_rawDescData)
	})
	return file_common_common_proto_rawDescData
}

var file_common_common_proto_msgTypes = make([]protoimpl.MessageInfo, 6)
var file_common_common_proto_goTypes = []interface{}{
	(*InputParameterSpec)(nil),                         // 0: kfp_common.InputParameterSpec
	(*ValueOrRuntimeParameter)(nil),                    // 1: kfp_common.ValueOrRuntimeParameter
	(*Value)(nil),                                      // 2: kfp_common.Value
	(*TaskOutputParameterSpec)(nil),                    // 3: kfp_common.TaskOutputParameterSpec
	(*InputParameterSpec_TaskOutputParameterSpec)(nil), // 4: kfp_common.InputParameterSpec.TaskOutputParameterSpec
	(*InputParameterSpec_TaskFinalStatus)(nil),         // 5: kfp_common.InputParameterSpec.TaskFinalStatus
	(*structpb.Value)(nil),                             // 6: google.protobuf.Value
}
var file_common_common_proto_depIdxs = []int32{
	4, // 0: kfp_common.InputParameterSpec.task_output_parameter:type_name -> kfp_common.InputParameterSpec.TaskOutputParameterSpec
	1, // 1: kfp_common.InputParameterSpec.runtime_value:type_name -> kfp_common.ValueOrRuntimeParameter
	5, // 2: kfp_common.InputParameterSpec.task_final_status:type_name -> kfp_common.InputParameterSpec.TaskFinalStatus
	2, // 3: kfp_common.ValueOrRuntimeParameter.constant_value:type_name -> kfp_common.Value
	6, // 4: kfp_common.ValueOrRuntimeParameter.constant:type_name -> google.protobuf.Value
	5, // [5:5] is the sub-list for method output_type
	5, // [5:5] is the sub-list for method input_type
	5, // [5:5] is the sub-list for extension type_name
	5, // [5:5] is the sub-list for extension extendee
	0, // [0:5] is the sub-list for field type_name
}

func init() { file_common_common_proto_init() }
func file_common_common_proto_init() {
	if File_common_common_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_common_common_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*InputParameterSpec); i {
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
		file_common_common_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ValueOrRuntimeParameter); i {
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
		file_common_common_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Value); i {
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
		file_common_common_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TaskOutputParameterSpec); i {
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
		file_common_common_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*InputParameterSpec_TaskOutputParameterSpec); i {
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
		file_common_common_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*InputParameterSpec_TaskFinalStatus); i {
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
	file_common_common_proto_msgTypes[0].OneofWrappers = []interface{}{
		(*InputParameterSpec_TaskOutputParameter)(nil),
		(*InputParameterSpec_RuntimeValue)(nil),
		(*InputParameterSpec_ComponentInputParameter)(nil),
		(*InputParameterSpec_TaskFinalStatus_)(nil),
	}
	file_common_common_proto_msgTypes[1].OneofWrappers = []interface{}{
		(*ValueOrRuntimeParameter_ConstantValue)(nil),
		(*ValueOrRuntimeParameter_RuntimeParameter)(nil),
		(*ValueOrRuntimeParameter_Constant)(nil),
	}
	file_common_common_proto_msgTypes[2].OneofWrappers = []interface{}{
		(*Value_IntValue)(nil),
		(*Value_DoubleValue)(nil),
		(*Value_StringValue)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_common_common_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   6,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_common_common_proto_goTypes,
		DependencyIndexes: file_common_common_proto_depIdxs,
		MessageInfos:      file_common_common_proto_msgTypes,
	}.Build()
	File_common_common_proto = out.File
	file_common_common_proto_rawDesc = nil
	file_common_common_proto_goTypes = nil
	file_common_common_proto_depIdxs = nil
}
