package objectstore

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_KeyFromURI(t *testing.T) {
	pipelineRoot := "s3://mlpipeline/v2/artifacts/"
	artifactUri := "s3://mlpipeline/v2/artifacts/create-dataset/5b9cda26-84a6-4360-ba4a-fe954f60d986/executor-logs"

	bucketConfig, err := ParseBucketPathToConfig(pipelineRoot)
	require.NoError(t, err)
	result, err := bucketConfig.KeyFromURI(artifactUri)
	require.NoError(t, err)
	require.Equal(t, "create-dataset/5b9cda26-84a6-4360-ba4a-fe954f60d986/executor-logs", result)
}

func Test_ParseBucketPathToConfig(t *testing.T) {
	pipelineRoot := "s3://mlpipeline/v2/artifacts/"
	result, err := ParseBucketPathToConfig(pipelineRoot)
	require.NoError(t, err)
	require.Equal(t, "s3://", result.Scheme)
	require.Equal(t, "v2/artifacts/", result.Prefix)
	require.Equal(t, "mlpipeline", result.BucketName)
}
