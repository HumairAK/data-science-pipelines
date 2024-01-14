// Copyright 2021 The Kubeflow Authors
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

// This package contains helper methods for using object stores.
package objectstore

import (
	"context"
	"fmt"
	"github.com/kubeflow/pipelines/backend/src/v2/config"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/golang/glog"
	"gocloud.dev/blob"
	_ "gocloud.dev/blob/gcsblob"
	"gocloud.dev/blob/s3blob"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type Config struct {
	Scheme      string
	BucketName  string
	Prefix      string
	QueryString string
	// optional, bucket auth contains provider authentication configurations
	BucketAuth *config.BucketAuth
}

// The endpoint uses Kubernetes service DNS name with namespace:
// https://kubernetes.io/docs/concepts/services-networking/service/#dns
const defaultMinioEndpointInMultiUserMode = "minio-service.kubeflow:9000"
const minioArtifactSecretName = "mlpipeline-minio-artifact"
const minioArtifactSecretKeyKey = "secretkey"
const minioArtifactAccessKeyKey = "accesskey"

// return the first authconfig found with a matching prefix
func getBucketAuthByPrefix(authConfigs []config.AuthConfig, bucketName, prefix string) *config.AuthConfig {
	for _, authConfig := range authConfigs {
		if authConfig.BucketName == bucketName && strings.HasPrefix(prefix, authConfig.KeyPrefix) {
			return &authConfig
		}
	}
	return nil
}

func createBucketSession(ctx context.Context, provider, namespace string, bucketConfig *Config, client kubernetes.Interface) (*session.Session, error) {
	bucketAuth := bucketConfig.BucketAuth
	if bucketAuth == nil {
		// Use default minio if provider is minio, otherwise we default to executor env
		if provider == "minio" {
			return getDefaultMinioSession(ctx, client, namespace)
		} else {
			// If not using minio, and no other provider config is provided
			// rely on executor env (e.g. IRSA) for authenticating with provider
			return nil, nil
		}
	}

	var providerConfig *config.ProviderConfig
	switch provider {
	case "minio":
		providerConfig = bucketAuth.Minio
		break
	case "s3":
		providerConfig = bucketAuth.S3
		break
	case "gcs":
		providerConfig = bucketAuth.Minio
		break
	default:
		return nil, fmt.Errorf("Encountered unsupported provider in BucketAuth %s", provider)
	}

	if providerConfig == nil {
		if provider == "minio" {
			return getDefaultMinioSession(ctx, client, namespace)
		} else {
			return nil, nil
		}
	}

	endpoint := providerConfig.Endpoint
	if endpoint == "" {
		if provider == "minio" {
			endpoint = MinioDefaultEndpoint()
		} else {
			return nil, fmt.Errorf("Invalid bucket auth config, %s.defaultProviderSecretRef is required for this storage provider", provider)
		}
	}

	// DefaultProviderSecretRef takes precedent over other configs
	secretRef := providerConfig.DefaultProviderSecretRef
	if secretRef == nil {
		if provider == "minio" {
			secretRef = &config.SecretRef{
				SecretName:   minioArtifactSecretName,
				SecretKeyKey: minioArtifactSecretKeyKey,
				AccessKeyKey: minioArtifactAccessKeyKey,
			}
		} else {
			return nil, fmt.Errorf("Invalid bucket auth config, %s.defaultProviderSecretRef is required for this storage provider", provider)
		}
	}

	disableSSL := providerConfig.DisableSSL
	region := providerConfig.Region

	authConfig := getBucketAuthByPrefix(providerConfig.AuthConfigs, bucketConfig.BucketName, bucketConfig.Prefix)
	if authConfig != nil {
		if authConfig.SecretRef == nil {
			return nil, fmt.Errorf("Invalid bucket auth config, %s.AuthConfigs[].secretConfig is required", provider)
		}
		secretRef = authConfig.SecretRef
	}

	// AuthConfig.secretRef takes precedence over defaultProviderSecretRef
	creds, err := getBucketCredential(ctx, client, namespace, secretRef.SecretName, secretRef.SecretKeyKey, secretRef.AccessKeyKey)
	if err != nil {
		return nil, err
	}

	sess, err := session.NewSession(&aws.Config{
		Credentials:      creds,
		Region:           aws.String(region),
		Endpoint:         aws.String(endpoint),
		DisableSSL:       aws.Bool(disableSSL),
		S3ForcePathStyle: aws.Bool(true),
	})
	if err != nil {
		return nil, fmt.Errorf("Failed to create session to access minio: %v", err)
	}

	return sess, nil
}

func OpenBucket(ctx context.Context, k8sClient kubernetes.Interface, namespace string, bucketConfig *Config) (bucket *blob.Bucket, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("Failed to open bucket %q: %w", bucketConfig.BucketName, err)
		}
	}()

	provider := strings.TrimSuffix(bucketConfig.Scheme, "://")
	sess, err := createBucketSession(ctx, provider, namespace, bucketConfig, k8sClient)
	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve credentials for bucket %s: %w", bucketConfig.BucketName, err)
	}

	if sess != nil {
		openedBucket, err := s3blob.OpenBucket(ctx, sess, bucketConfig.BucketName, nil)
		if err != nil {
			return nil, err
		}
		// Directly calling s3blob.OpenBucket does not allow overriding prefix via bucketConfig.BucketURL().
		// Therefore, we need to explicitly configure the prefixed bucket.
		return blob.PrefixedBucket(openedBucket, bucketConfig.Prefix), nil
	}
	return blob.OpenBucket(ctx, bucketConfig.bucketURL())
}

func (b *Config) bucketURL() string {
	u := b.Scheme + b.BucketName

	// append prefix=b.prefix to existing queryString
	q := b.QueryString
	if len(b.Prefix) > 0 {
		if len(q) > 0 {
			q = q + "&prefix=" + b.Prefix
		} else {
			q = "?prefix=" + b.Prefix
		}
	}

	u = u + q
	return u
}

func (b *Config) PrefixedBucket() string {
	return b.Scheme + path.Join(b.BucketName, b.Prefix)
}

func (b *Config) KeyFromURI(uri string) (string, error) {
	prefixedBucket := b.PrefixedBucket()
	if !strings.HasPrefix(uri, prefixedBucket) {
		return "", fmt.Errorf("URI %q does not have expected bucket prefix %q", uri, prefixedBucket)
	}

	key := strings.TrimLeft(strings.TrimPrefix(uri, prefixedBucket), "/")
	if len(key) == 0 {
		return "", fmt.Errorf("URI %q has empty key given prefixed bucket %q", uri, prefixedBucket)
	}
	return key, nil
}

func (b *Config) UriFromKey(blobKey string) string {
	return b.Scheme + path.Join(b.BucketName, b.Prefix, blobKey)
}

func UploadBlob(ctx context.Context, bucket *blob.Bucket, localPath, blobPath string) error {
	fileInfo, err := os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("unable to stat local filepath %q: %w", localPath, err)
	}

	if !fileInfo.IsDir() {
		return uploadFile(ctx, bucket, localPath, blobPath)
	}

	// localPath is a directory.
	files, err := ioutil.ReadDir(localPath)
	if err != nil {
		return fmt.Errorf("unable to list local directory %q: %w", localPath, err)
	}

	for _, f := range files {
		if f.IsDir() {
			err = UploadBlob(ctx, bucket, filepath.Join(localPath, f.Name()), blobPath+"/"+f.Name())
			if err != nil {
				return err
			}
		} else {
			blobFilePath := filepath.Join(blobPath, filepath.Base(f.Name()))
			localFilePath := filepath.Join(localPath, f.Name())
			if err := uploadFile(ctx, bucket, localFilePath, blobFilePath); err != nil {
				return err
			}
		}

	}

	return nil
}

func DownloadBlob(ctx context.Context, bucket *blob.Bucket, localDir, blobDir string) error {
	iter := bucket.List(&blob.ListOptions{Prefix: blobDir})
	for {
		obj, err := iter.Next(ctx)
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to list objects in remote storage %q: %w", blobDir, err)
		}
		if obj.IsDir {
			// TODO: is this branch possible?

			// Object stores list all files with the same prefix,
			// there is no need to recursively list each folder.
			continue
		} else {
			relativePath, err := filepath.Rel(blobDir, obj.Key)
			if err != nil {
				return fmt.Errorf("unexpected object key %q when listing %q: %w", obj.Key, blobDir, err)
			}
			if err := downloadFile(ctx, bucket, obj.Key, filepath.Join(localDir, relativePath)); err != nil {
				return err
			}
		}
	}
	return nil
}

var bucketPattern = regexp.MustCompile(`(^[a-z][a-z0-9]+:///?)([^/?]+)(/[^?]*)?(\?.+)?$`)

func ParseBucketConfig(path string) (*Config, error) {
	ms := bucketPattern.FindStringSubmatch(path)
	if ms == nil || len(ms) != 5 {
		return nil, fmt.Errorf("parse bucket config failed: unrecognized pipeline root format: %q", path)
	}

	// TODO: Verify/add support for file:///.
	if ms[1] != "gs://" && ms[1] != "s3://" && ms[1] != "minio://" {
		return nil, fmt.Errorf("parse bucket config failed: unsupported Cloud bucket: %q", path)
	}

	prefix := strings.TrimPrefix(ms[3], "/")
	if len(prefix) > 0 && !strings.HasSuffix(prefix, "/") {
		prefix = prefix + "/"
	}

	return &Config{
		Scheme:      ms[1],
		BucketName:  ms[2],
		Prefix:      prefix,
		QueryString: ms[4],
	}, nil
}

func ParseBucketConfigForArtifactURI(uri string) (*Config, error) {
	ms := bucketPattern.FindStringSubmatch(uri)
	if ms == nil || len(ms) != 5 {
		return nil, fmt.Errorf("parse bucket config failed: unrecognized uri format: %q", uri)
	}

	// TODO: Verify/add support for file:///.
	if ms[1] != "gs://" && ms[1] != "s3://" && ms[1] != "minio://" {
		return nil, fmt.Errorf("parse bucket config failed: unsupported Cloud bucket: %q", uri)
	}

	return &Config{
		Scheme:     ms[1],
		BucketName: ms[2],
	}, nil
}

// TODO(neuromage): Move these helper functions to a storage package and add tests.
func uploadFile(ctx context.Context, bucket *blob.Bucket, localFilePath, blobFilePath string) error {
	errorF := func(err error) error {
		return fmt.Errorf("uploadFile(): unable to complete copying %q to remote storage %q: %w", localFilePath, blobFilePath, err)
	}

	w, err := bucket.NewWriter(ctx, blobFilePath, nil)
	if err != nil {
		return errorF(fmt.Errorf("unable to open writer for bucket: %w", err))
	}

	r, err := os.Open(localFilePath)
	if err != nil {
		return errorF(fmt.Errorf("unable to open local file %q for reading: %w", localFilePath, err))
	}
	defer r.Close()

	if _, err = io.Copy(w, r); err != nil {
		return errorF(fmt.Errorf("unable to complete copying: %w", err))
	}

	if err = w.Close(); err != nil {
		return errorF(fmt.Errorf("failed to close Writer for bucket: %w", err))
	}

	glog.Infof("uploadFile(localFilePath=%q, blobFilePath=%q)", localFilePath, blobFilePath)
	return nil
}

func downloadFile(ctx context.Context, bucket *blob.Bucket, blobFilePath, localFilePath string) (err error) {
	errorF := func(err error) error {
		return fmt.Errorf("downloadFile(): unable to complete copying %q to local storage %q: %w", blobFilePath, localFilePath, err)
	}

	r, err := bucket.NewReader(ctx, blobFilePath, nil)
	if err != nil {
		return errorF(fmt.Errorf("unable to open reader for bucket: %w", err))
	}
	defer r.Close()

	localDir := filepath.Dir(localFilePath)
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return errorF(fmt.Errorf("failed to create local directory %q: %w", localDir, err))
	}

	w, err := os.Create(localFilePath)
	if err != nil {
		return errorF(fmt.Errorf("unable to open local file %q for writing: %w", localFilePath, err))
	}
	defer func() {
		errClose := w.Close()
		if err == nil && errClose != nil {
			// override named return value "err" when there's a close error
			err = errorF(errClose)
		}
	}()

	if _, err = io.Copy(w, r); err != nil {
		return errorF(fmt.Errorf("unable to complete copying: %w", err))
	}

	return nil
}

func MinioDefaultEndpoint() string {
	// Discover minio-service in the same namespace by env var.
	// https://kubernetes.io/docs/concepts/services-networking/service/#environment-variables
	minioHost := os.Getenv("MINIO_SERVICE_SERVICE_HOST")
	minioPort := os.Getenv("MINIO_SERVICE_SERVICE_PORT")
	if minioHost != "" && minioPort != "" {
		// If there is a minio-service Kubernetes service in the same namespace,
		// MINIO_SERVICE_SERVICE_HOST and MINIO_SERVICE_SERVICE_PORT env vars should
		// exist by default, so we use it as default.
		return minioHost + ":" + minioPort
	}
	// If the env vars do not exist, we guess that we are running in KFP multi user mode, so default minio service should be `minio-service.kubeflow:9000`.
	glog.Infof("Cannot detect minio-service in the same namespace, default to %s as MinIO endpoint.", defaultMinioEndpointInMultiUserMode)
	return defaultMinioEndpointInMultiUserMode
}

func getDefaultMinioSession(ctx context.Context, clientSet kubernetes.Interface, namespace string) (cred *session.Session, err error) {
	creds, err := getBucketCredential(ctx, clientSet, namespace, minioArtifactSecretName, minioArtifactSecretKeyKey,
		minioArtifactAccessKeyKey)
	if err != nil {
		return nil, err
	}
	sess, err := session.NewSession(&aws.Config{
		Credentials:      creds,
		Region:           aws.String("minio"),
		Endpoint:         aws.String(MinioDefaultEndpoint()),
		DisableSSL:       aws.Bool(true),
		S3ForcePathStyle: aws.Bool(true),
	})
	if err != nil {
		return nil, err
	}
	return sess, nil
}

func getBucketCredential(
	ctx context.Context,
	clientSet kubernetes.Interface,
	namespace string,
	secretName string,
	bucketSecretKeyKey string,
	bucketAccessKeyKey string,
) (cred *credentials.Credentials, err error) {
	defer func() {
		if err != nil {
			// wrap error before returning
			err = fmt.Errorf("Failed to get Bucket credentials from secret name=%q namespace=%q: %w", secretName, namespace, err)
		}
	}()
	secret, err := clientSet.CoreV1().Secrets(namespace).Get(
		ctx,
		secretName,
		metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	accessKey := string(secret.Data[bucketAccessKeyKey])
	secretKey := string(secret.Data[bucketSecretKeyKey])

	if accessKey != "" && secretKey != "" {
		cred = credentials.NewStaticCredentials(accessKey, secretKey, "")
		return cred, err
	}
	return nil, fmt.Errorf("does not have 'accesskey' or 'secretkey' key")
}
