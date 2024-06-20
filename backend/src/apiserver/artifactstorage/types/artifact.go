package types

import (
	"fmt"
	apiv1 "k8s.io/api/core/v1"
)

// Artifact indicates an artifact to place at a specified path
type Artifact struct {
	// name of the artifact. must be unique within a template's inputs/outputs.
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`

	// ArtifactLocation contains the location of the artifact
	ArtifactLocation `json:",inline" protobuf:"bytes,5,opt,name=artifactLocation"`
}

// ArtifactLocation describes a location for a single or multiple artifacts.
// It is used as single artifact in the context of inputs/outputs (e.g. outputs.artifacts.artname).
// It is also used to describe the location of multiple artifacts such as the archive location
// of a single workflow step, which the executor will use as a default location to store its files.
type ArtifactLocation struct {
	// S3 contains S3 artifact location details
	S3 *S3Artifact `json:"s3,omitempty" protobuf:"bytes,2,opt,name=s3"`
}

// S3Artifact is the location of an S3 artifact
type S3Artifact struct {
	S3Bucket `json:",inline" protobuf:"bytes,1,opt,name=s3Bucket"`

	// Key is the key in the bucket where the artifact resides
	Key string `json:"key,omitempty" protobuf:"bytes,2,opt,name=key"`
}

// S3Bucket contains the access information required for interfacing with an S3 bucket
type S3Bucket struct {
	// Endpoint is the hostname of the bucket endpoint
	Endpoint string `json:"endpoint,omitempty" protobuf:"bytes,1,opt,name=endpoint"`

	// Bucket is the name of the bucket
	Bucket string `json:"bucket,omitempty" protobuf:"bytes,2,opt,name=bucket"`

	// Region contains the optional bucket region
	Region string `json:"region,omitempty" protobuf:"bytes,3,opt,name=region"`

	// Insecure will connect to the service with TLS
	Insecure *bool `json:"insecure,omitempty" protobuf:"varint,4,opt,name=insecure"`

	// AccessKeySecret is the secret selector to the bucket's access key
	AccessKeySecret *apiv1.SecretKeySelector `json:"accessKeySecret,omitempty" protobuf:"bytes,5,opt,name=accessKeySecret"`

	// SecretKeySecret is the secret selector to the bucket's secret key
	SecretKeySecret *apiv1.SecretKeySelector `json:"secretKeySecret,omitempty" protobuf:"bytes,6,opt,name=secretKeySecret"`

	// RoleARN is the Amazon Resource Name (ARN) of the role to assume.
	RoleARN string `json:"roleARN,omitempty" protobuf:"bytes,7,opt,name=roleARN"`

	// UseSDKCreds tells the driver to figure out credentials based on sdk defaults.
	UseSDKCreds bool `json:"useSDKCreds,omitempty" protobuf:"varint,8,opt,name=useSDKCreds"`

	// CreateBucketIfNotPresent tells the driver to attempt to create the S3 bucket for output artifacts, if it doesn't exist. Setting Enabled Encryption will apply either SSE-S3 to the bucket if KmsKeyId is not set or SSE-KMS if it is.
	CreateBucketIfNotPresent *CreateS3BucketOptions `json:"createBucketIfNotPresent,omitempty" protobuf:"bytes,9,opt,name=createBucketIfNotPresent"`

	EncryptionOptions *S3EncryptionOptions `json:"encryptionOptions,omitempty" protobuf:"bytes,10,opt,name=encryptionOptions"`

	// CASecret specifies the secret that contains the CA, used to verify the TLS connection
	CASecret *apiv1.SecretKeySelector `json:"caSecret,omitempty" protobuf:"bytes,11,opt,name=caSecret"`
}

// CreateS3BucketOptions options used to determine automatic bucket-creation process
type CreateS3BucketOptions struct {
	// ObjectLocking Enable object locking
	ObjectLocking bool `json:"objectLocking,omitempty" protobuf:"varint,3,opt,name=objectLocking"`
}

// S3EncryptionOptions used to determine encryption options during s3 operations
type S3EncryptionOptions struct {
	// KMSKeyId tells the driver to encrypt the object using the specified KMS Key.
	KmsKeyId string `json:"kmsKeyId,omitempty" protobuf:"bytes,1,opt,name=kmsKeyId"`

	// KmsEncryptionContext is a json blob that contains an encryption context. See https://docs.aws.amazon.com/kms/latest/developerguide/concepts.html#encrypt_context for more information
	KmsEncryptionContext string `json:"kmsEncryptionContext,omitempty" protobuf:"bytes,2,opt,name=kmsEncryptionContext"`

	// EnableEncryption tells the driver to encrypt objects if set to true. If kmsKeyId and serverSideCustomerKeySecret are not set, SSE-S3 will be used
	EnableEncryption bool `json:"enableEncryption,omitempty" protobuf:"varint,3,opt,name=enableEncryption"`

	// ServerSideCustomerKeySecret tells the driver to encrypt the output artifacts using SSE-C with the specified secret.
	ServerSideCustomerKeySecret *apiv1.SecretKeySelector `json:"serverSideCustomerKeySecret,omitempty" protobuf:"bytes,4,opt,name=serverSideCustomerKeySecret"`
}

func (a *ArtifactLocation) Get() (ArtifactLocationType, error) {
	if a == nil {
		return nil, fmt.Errorf("key unsupported: cannot get key for artifact location, because it is invalid")
	} else if a.S3 != nil {
		return a.S3, nil
	}
	return nil, fmt.Errorf("You need to configure artifact storage.")
}

type ArtifactLocationType interface {
	HasLocation() bool
	GetKey() (string, error)
	SetKey(key string) error
}

func (a *ArtifactLocation) GetKey() (string, error) {
	v, err := a.Get()
	if err != nil {
		return "", err
	}
	return v.GetKey()
}

func (s *S3Artifact) GetKey() (string, error) {
	return s.Key, nil
}
func (s *S3Artifact) SetKey(key string) error {
	s.Key = key
	return nil
}

func (s *S3Artifact) HasLocation() bool {
	return s != nil && s.Endpoint != "" && s.Bucket != "" && s.Key != ""
}
