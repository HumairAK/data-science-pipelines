// Copyright 2021-2023 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package metadata contains types to record/retrieve metadata stored in MLMD
// for individual pipeline steps.
package config

import (
	"context"
	"fmt"
	"github.com/kubeflow/pipelines/backend/src/v2/objectstore"
	"io/ioutil"
	"reflect"
	"sigs.k8s.io/yaml"
	"strconv"
	"strings"

	"github.com/golang/glog"
	k8errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	configMapName                = "kfp-launcher"
	defaultPipelineRoot          = "minio://mlpipeline/v2/artifacts"
	configKeyDefaultPipelineRoot = "defaultPipelineRoot"
	configBucketProviders        = "providers"
	minioArtifactSecretName      = "mlpipeline-minio-artifact"
	minioArtifactSecretKeyKey    = "secretkey"
	minioArtifactAccessKeyKey    = "accesskey"
)

type BucketProviders struct {
	Minio *MinioProviderConfig `json:"minio"`
	S3    *S3ProviderConfig    `json:"s3"`
	GCS   *GCSProviderConfig   `json:"gs"`
}

type SessionInfoProvider interface {
	ProvideSessionInfo(bucketName, bucketPrefix string) (objectstore.SessionInfo, error)
}

// Config is the KFP runtime configuration.
type Config struct {
	data map[string]string
}

// FromConfigMap loads config from a kfp-launcher Kubernetes config map.
func FromConfigMap(ctx context.Context, clientSet kubernetes.Interface, namespace string) (*Config, error) {
	config, err := clientSet.CoreV1().ConfigMaps(namespace).Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil {
		if k8errors.IsNotFound(err) {
			glog.Infof("cannot find launcher configmap: name=%q namespace=%q, will use default config", configMapName, namespace)
			// LauncherConfig is optional, so ignore not found error.
			return nil, nil
		}
		return nil, err
	}
	return &Config{data: config.Data}, nil
}

// DefaultPipelineRoot gets the configured default pipeline root.
func (c *Config) DefaultPipelineRoot() string {
	// The key defaultPipelineRoot is optional in launcher config.
	if c == nil || c.data[configKeyDefaultPipelineRoot] == "" {
		return defaultPipelineRoot
	}
	return c.data[configKeyDefaultPipelineRoot]
}

// InPodNamespace gets current namespace from inside a Kubernetes Pod.
func InPodNamespace() (string, error) {
	// The path is available in Pods.
	// https://kubernetes.io/docs/tasks/run-application/access-api-from-pod/#directly-accessing-the-rest-api
	ns, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return "", fmt.Errorf("failed to get namespace in Pod: %w", err)
	}
	return string(ns), nil
}

// InPodName gets the pod name from inside a Kubernetes Pod.
func InPodName() (string, error) {
	podName, err := ioutil.ReadFile("/etc/hostname")
	if err != nil {
		return "", fmt.Errorf("failed to get pod name in Pod: %w", err)
	}
	name := string(podName)
	return strings.TrimSuffix(name, "\n"), nil
}

func (c *Config) GetBucketSessionInfo(path string) (objectstore.SessionInfo, error) {
	bucketConfig, err := objectstore.ParseBucketPathToConfig(path)
	if err != nil {
		return objectstore.SessionInfo{}, err
	}
	bucketName := bucketConfig.BucketName
	bucketPrefix := bucketConfig.Prefix
	provider := strings.TrimSuffix(bucketConfig.Scheme, "://")
	bucketProviders, err := c.getBucketProviders()
	if err != nil {
		return objectstore.SessionInfo{}, err
	}

	if bucketProviders == nil {
		// Use default minio if provider is minio, otherwise we default to executor env
		if provider == "minio" {
			sess, sessErr := getDefaultMinioSessionInfo()
			if sessErr != nil {
				return objectstore.SessionInfo{}, nil
			}
			return sess, nil
		} else {
			// If not using minio, and no other provider config is provided
			// rely on executor env (e.g. IRSA) for authenticating with provider
			return objectstore.SessionInfo{}, nil
		}
	}

	var sessProvider SessionInfoProvider

	switch provider {
	case "minio":
		sessProvider = bucketProviders.Minio
		break
	case "s3":
		sessProvider = bucketProviders.S3
		break
	case "gs":
		sessProvider = bucketProviders.GCS
		break
	default:
		return objectstore.SessionInfo{}, fmt.Errorf("Encountered unsupported provider in BucketProviders %s", provider)
	}

	if sessProvider == nil || reflect.ValueOf(sessProvider).IsNil() {
		return objectstore.SessionInfo{}, fmt.Errorf("Encountered unsupported provider in provider config %s", provider)
	}

	sess, err := sessProvider.ProvideSessionInfo(bucketName, bucketPrefix)
	if err != nil {
		return objectstore.SessionInfo{}, err
	}
	return sess, nil
}

// getBucketProviders gets the provider configuration
func (c *Config) getBucketProviders() (*BucketProviders, error) {
	if c == nil || c.data[configBucketProviders] == "" {
		return nil, nil
	}
	bucketProviders := &BucketProviders{}
	configAuth := c.data[configBucketProviders]
	err := yaml.Unmarshal([]byte(configAuth), bucketProviders)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshall kfp bucket providers, ensure that providers config is well formed: %w", err)
	}
	return bucketProviders, nil
}

func getDefaultMinioSessionInfo() (objectstore.SessionInfo, error) {
	sess := objectstore.SessionInfo{
		Provider: "minio",
		Params: map[string]string{
			"region":       "minio",
			"endpoint":     objectstore.MinioDefaultEndpoint(),
			"disableSsl":   strconv.FormatBool(true),
			"fromEnv":      strconv.FormatBool(false),
			"secretName":   minioArtifactSecretName,
			"accessKeyKey": minioArtifactAccessKeyKey,
			"secretKeyKey": minioArtifactSecretKeyKey,
		},
	}
	return sess, nil
}
