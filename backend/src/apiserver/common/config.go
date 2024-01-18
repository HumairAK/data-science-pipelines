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

package common

import (
	"strconv"
	"time"

	"github.com/golang/glog"
	"github.com/spf13/viper"
)

const (
	MultiUserMode                           string = "MULTIUSER"
	MultiUserModeSharedReadAccess           string = "MULTIUSER_SHARED_READ"
	PodNamespace                            string = "POD_NAMESPACE"
	CacheEnabled                            string = "CacheEnabled"
	DefaultPipelineRunnerServiceAccountFlag string = "DEFAULTPIPELINERUNNERSERVICEACCOUNT"
	KubeflowUserIDHeader                    string = "KUBEFLOW_USERID_HEADER"
	KubeflowUserIDPrefix                    string = "KUBEFLOW_USERID_PREFIX"
	UpdatePipelineVersionByDefault          string = "AUTO_UPDATE_PIPELINE_DEFAULT_VERSION"
	TokenReviewAudience                     string = "TOKEN_REVIEW_AUDIENCE"

	// Discover ml-pipeline in the same namespace by env var.
	// https://kubernetes.io/docs/concepts/services-networking/service/#environment-variables
	// If there is a ml-pipeline Kubernetes service in the same namespace,
	// ML_PIPELINE_SERVICE_HOST and ML_PIPELINE_SERVICE_PORT env vars should
	// exist by default, so we use it as default. If using a service name that is not ml-pipeline
	// these env vars should be manually provided.
	mlPipelineServiceAddress  string = "ML_PIPELINE_SERVICE_HOST"
	mlPipelineServiceGRPCPort string = "ML_PIPELINE_SERVICE_PORT_GRPC"
)

func IsPipelineVersionUpdatedByDefault() bool {
	return GetBoolConfigWithDefault(UpdatePipelineVersionByDefault, true)
}

func GetStringConfig(configName string) string {
	if !viper.IsSet(configName) {
		glog.Fatalf("Please specify flag %s", configName)
	}
	return viper.GetString(configName)
}

func GetStringConfigWithDefault(configName, value string) string {
	if !viper.IsSet(configName) {
		return value
	}
	return viper.GetString(configName)
}

func GetMapConfig(configName string) map[string]string {
	if !viper.IsSet(configName) {
		glog.Infof("Config %s not specified, skipping", configName)
		return nil
	}
	return viper.GetStringMapString(configName)
}

func GetBoolConfigWithDefault(configName string, value bool) bool {
	if !viper.IsSet(configName) {
		return value
	}
	value, err := strconv.ParseBool(viper.GetString(configName))
	if err != nil {
		glog.Fatalf("Failed converting string to bool %s", viper.GetString(configName))
	}
	return value
}

func GetFloat64ConfigWithDefault(configName string, value float64) float64 {
	if !viper.IsSet(configName) {
		return value
	}
	return viper.GetFloat64(configName)
}

func GetIntConfigWithDefault(configName string, value int) int {
	if !viper.IsSet(configName) {
		return value
	}
	return viper.GetInt(configName)
}

func GetDurationConfig(configName string) time.Duration {
	if !viper.IsSet(configName) {
		glog.Fatalf("Please specify flag %s", configName)
	}
	return viper.GetDuration(configName)
}

func IsMultiUserMode() bool {
	return GetBoolConfigWithDefault(MultiUserMode, false)
}

func IsMultiUserSharedReadMode() bool {
	return GetBoolConfigWithDefault(MultiUserModeSharedReadAccess, false)
}

func GetPodNamespace() string {
	return GetStringConfig(PodNamespace)
}

func GetBoolFromStringWithDefault(value string, defaultValue bool) bool {
	boolVal, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return boolVal
}

func IsCacheEnabled() string {
	return GetStringConfigWithDefault(CacheEnabled, "true")
}

func GetKubeflowUserIDHeader() string {
	return GetStringConfigWithDefault(KubeflowUserIDHeader, GoogleIAPUserIdentityHeader)
}

func GetKubeflowUserIDPrefix() string {
	return GetStringConfigWithDefault(KubeflowUserIDPrefix, GoogleIAPUserIdentityPrefix)
}

func GetTokenReviewAudience() string {
	return GetStringConfigWithDefault(TokenReviewAudience, DefaultTokenReviewAudience)
}

func GetMLPipelineServiceAddress() string {
	return GetStringConfigWithDefault(mlPipelineServiceAddress, DefaultMLPipelineAddress)
}

func GetMlPipelineServiceGRPCPort() string {
	return GetStringConfigWithDefault(mlPipelineServiceGRPCPort, DefaultMLPipelineGRPCPort)
}
