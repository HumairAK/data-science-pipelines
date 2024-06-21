package executor

import (
	"context"
	"fmt"
	"github.com/kubeflow/pipelines/backend/src/apiserver/artifactstorage/logging"
	"github.com/kubeflow/pipelines/backend/src/apiserver/artifactstorage/s3"
	"github.com/kubeflow/pipelines/backend/src/apiserver/artifactstorage/types"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
)

var ErrUnsupportedDriver = fmt.Errorf("unsupported artifact driver")

type NewDriverFunc func(ctx context.Context, art *types.Artifact, rm resource.ResourceManager, namespace string) (types.ArtifactDriver, error)

// NewDriver initializes an instance of an artifact driver
func NewDriver(ctx context.Context, art *types.Artifact, rm *resource.ResourceManager, namespace string) (types.ArtifactDriver, error) {
	drv, err := newDriver(ctx, art, rm, namespace)
	if err != nil {
		return nil, err
	}
	return logging.New(drv), nil

}
func newDriver(ctx context.Context, art *types.Artifact, rm *resource.ResourceManager, namespace string) (types.ArtifactDriver, error) {
	if art.S3 != nil {
		var accessKey string
		var secretKey string
		var serverSideCustomerKey string
		var kmsKeyId string
		var kmsEncryptionContext string
		var enableEncryption bool
		var caKey string

		if art.S3.AccessKeySecret != nil && art.S3.AccessKeySecret.Name != "" {
			accessKeyBytes, err := rm.GetSecret(ctx, namespace, art.S3.AccessKeySecret.Name, art.S3.AccessKeySecret.Key)
			if err != nil {
				return nil, err
			}
			accessKey = accessKeyBytes
			secretKeyBytes, err := rm.GetSecret(ctx, namespace, art.S3.SecretKeySecret.Name, art.S3.SecretKeySecret.Key)
			if err != nil {
				return nil, err
			}
			secretKey = secretKeyBytes
		}

		if art.S3.EncryptionOptions != nil {
			if art.S3.EncryptionOptions.ServerSideCustomerKeySecret != nil {
				if art.S3.EncryptionOptions.KmsKeyId != "" {
					return nil, fmt.Errorf("serverSideCustomerKeySecret and kmsKeyId cannot be set together")
				}

				serverSideCustomerKeyBytes, err := rm.GetSecret(ctx, namespace, art.S3.EncryptionOptions.ServerSideCustomerKeySecret.Name, art.S3.EncryptionOptions.ServerSideCustomerKeySecret.Key)
				if err != nil {
					return nil, err
				}
				serverSideCustomerKey = serverSideCustomerKeyBytes
			}

			enableEncryption = art.S3.EncryptionOptions.EnableEncryption
			kmsKeyId = art.S3.EncryptionOptions.KmsKeyId
			kmsEncryptionContext = art.S3.EncryptionOptions.KmsEncryptionContext
		}

		if art.S3.CASecret != nil && art.S3.CASecret.Name != "" {
			caBytes, err := rm.GetSecret(ctx, namespace, art.S3.CASecret.Name, art.S3.CASecret.Key)
			if err != nil {
				return nil, err
			}
			caKey = caBytes
		}

		driver := s3.ArtifactDriver{
			Endpoint:              art.S3.Endpoint,
			AccessKey:             accessKey,
			SecretKey:             secretKey,
			Secure:                art.S3.Insecure == nil || !*art.S3.Insecure,
			TrustedCA:             caKey,
			Region:                art.S3.Region,
			RoleARN:               art.S3.RoleARN,
			UseSDKCreds:           art.S3.UseSDKCreds,
			KmsKeyId:              kmsKeyId,
			KmsEncryptionContext:  kmsEncryptionContext,
			EnableEncryption:      enableEncryption,
			ServerSideCustomerKey: serverSideCustomerKey,
		}

		return &driver, nil
	}

	return nil, ErrUnsupportedDriver
}
