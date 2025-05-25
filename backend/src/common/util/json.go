// Copyright 2018 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"bytes"
	"fmt"
	"github.com/golang/protobuf/jsonpb"
	"github.com/kubeflow/pipelines/kubernetes_platform/go/kubernetesplatform"

	"encoding/json"
	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	"strings"
)

func UnmarshalJsonOrFail(data string, v interface{}) {
	err := json.Unmarshal([]byte(data), v)
	if err != nil {
		glog.Fatalf("Failed to unmarshal the object: %v", data)
	}
}

func MarshalJsonOrFail(v interface{}) []byte {
	bytes, err := json.Marshal(v)
	if err != nil {
		glog.Fatalf("Failed to marshal the object: %+v", v)
	}
	return bytes
}

// Converts an object into []byte array
func MarshalJsonWithError(v interface{}) ([]byte, error) {
	bytes, err := json.Marshal(v)
	if err != nil {
		return nil, Wrapf(err, "Failed to marshal the object: %+v", v)
	}
	return bytes, nil
}

// Converts a []byte array into an interface
func UnmarshalJsonWithError(data interface{}, v *interface{}) error {
	var bytes []byte
	switch data := data.(type) {
	case string:
		bytes = []byte(data)
	case []byte:
		bytes = data
	case *[]byte:
		bytes = *data
	default:
		return NewInvalidInputError("Unmarshalling %T is not implemented.", data)
	}
	err := json.Unmarshal(bytes, v)
	if err != nil {
		return Wrapf(err, "Failed to unmarshal the object: %+v", v)
	}
	return nil
}

// UnmarshalString unmarshals a JSON object from s into m.
// Allows unknown fields
func UnmarshalString(s string, m proto.Message) error {
	unmarshaler := jsonpb.Unmarshaler{AllowUnknownFields: true}
	return unmarshaler.Unmarshal(strings.NewReader(s), m)
}

func ParseExecConfigJson(k8sExecConfigJson *string) (*kubernetesplatform.KubernetesExecutorConfig, error) {
	var k8sExecCfg *kubernetesplatform.KubernetesExecutorConfig
	if *k8sExecConfigJson != "" {
		glog.Infof("input kubernetesConfig:%s\n", PrettyPrint(*k8sExecConfigJson))
		k8sExecCfg = &kubernetesplatform.KubernetesExecutorConfig{}
		if err := UnmarshalString(*k8sExecConfigJson, k8sExecCfg); err != nil {
			return nil, fmt.Errorf("failed to unmarshal Kubernetes config, error: %w\nKubernetesConfig: %v", err, k8sExecConfigJson)
		}
	}
	return k8sExecCfg, nil
}

// TODO: use this everywhere in cli driver?
func PrettyPrint(jsonStr string) string {
	var prettyJSON bytes.Buffer
	err := json.Indent(&prettyJSON, []byte(jsonStr), "", "  ")
	if err != nil {
		return jsonStr
	}
	return prettyJSON.String()
}
