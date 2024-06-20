package s3

import (
	"context"
	"crypto/x509"
	"fmt"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	argos3 "github.com/argoproj/pkg/s3"
	"github.com/golang/glog"
	artifactStoreTypes "github.com/kubeflow/pipelines/backend/src/apiserver/artifactstorage/types"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
)

// ArtifactDriver is a driver for AWS S3
type ArtifactDriver struct {
	Endpoint              string
	Region                string
	Secure                bool
	TrustedCA             string
	AccessKey             string
	SecretKey             string
	RoleARN               string
	UseSDKCreds           bool
	Context               context.Context
	KmsKeyId              string
	KmsEncryptionContext  string
	EnableEncryption      bool
	ServerSideCustomerKey string
}

var _ artifactStoreTypes.ArtifactDriver = &ArtifactDriver{}

// newS3Client instantiates a new S3 client object.
func (s3Driver *ArtifactDriver) newS3Client(ctx context.Context) (argos3.S3Client, error) {
	opts := argos3.S3ClientOpts{
		Endpoint:    s3Driver.Endpoint,
		Region:      s3Driver.Region,
		Secure:      s3Driver.Secure,
		AccessKey:   s3Driver.AccessKey,
		SecretKey:   s3Driver.SecretKey,
		RoleARN:     s3Driver.RoleARN,
		Trace:       os.Getenv(common.EnvVarArgoTrace) == "1",
		UseSDKCreds: s3Driver.UseSDKCreds,
		EncryptOpts: argos3.EncryptOpts{
			KmsKeyId:              s3Driver.KmsKeyId,
			KmsEncryptionContext:  s3Driver.KmsEncryptionContext,
			Enabled:               s3Driver.EnableEncryption,
			ServerSideCustomerKey: s3Driver.ServerSideCustomerKey,
		},
	}

	if tr, err := argos3.GetDefaultTransport(opts); err == nil {
		if s3Driver.Secure && s3Driver.TrustedCA != "" {
			// Trust only the provided root CA
			pool := x509.NewCertPool()
			pool.AppendCertsFromPEM([]byte(s3Driver.TrustedCA))
			tr.TLSClientConfig.RootCAs = pool
		}
		opts.Transport = tr
	}

	return argos3.NewS3Client(ctx, opts)
}

// OpenStream opens a stream reader for an artifact from S3 compliant storage
func (s3Driver *ArtifactDriver) OpenStream(inputArtifact *artifactStoreTypes.Artifact) (io.ReadCloser, error) {
	log.Infof("S3 OpenStream: key: %s", inputArtifact.S3.Key)
	s3cli, err := s3Driver.newS3Client(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("failed to create new S3 client: %v", err)
	}

	return streamS3Artifact(s3cli, inputArtifact)

}

func streamS3Artifact(s3cli argos3.S3Client, inputArtifact *artifactStoreTypes.Artifact) (io.ReadCloser, error) {
	b, err := s3cli.KeyExists(inputArtifact.S3.Bucket, inputArtifact.S3.Key)
	if err != nil {
		return nil, err
	}
	if !b {
		return nil, fmt.Errorf("Key does not exist")
	} else {
		glog.Info("Key Exists")
	}

	stream, origErr := s3cli.OpenFile(inputArtifact.S3.Bucket, inputArtifact.S3.Key)
	if origErr == nil {
		return stream, nil
	}
	if !argos3.IsS3ErrCode(origErr, "NoSuchKey") {
		return nil, fmt.Errorf("failed to get file: %v", origErr)
	}
	// If we get here, the error was a NoSuchKey. The key might be an s3 "directory"
	isDir, err := s3cli.IsDirectory(inputArtifact.S3.Bucket, inputArtifact.S3.Key)
	if err != nil {
		return nil, fmt.Errorf("failed to test if %s is a directory: %v", inputArtifact.S3.Key, err)
	}
	if !isDir {
		// It's neither a file, nor a directory. Return the original NoSuchKey error
		return nil, fmt.Errorf("Could not find Artifact Key, error: %v", err)
	}
	// directory case:
	// todo: make a .tgz file which can be streamed to user
	return nil, fmt.Errorf("Directory Stream capability currently unimplemented for S3")
}
