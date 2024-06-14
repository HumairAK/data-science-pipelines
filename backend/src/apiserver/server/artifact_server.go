// Copyright 2024 The Kubeflow Authors
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

package server

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/golang/glog"
	api "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"gocloud.dev/blob/s3blob"
	"io"
	"k8s.io/apimachinery/pkg/util/json"
	"net/http"
	"os"
)

type ArtifactServer struct {
	resourceManager *resource.ResourceManager
}

func (s *ArtifactServer) ListArtifacts(context.Context, *apiv2beta1.ListArtifactRequest) (*apiv2beta1.Artifact, error) {
	return &apiv2beta1.Artifact{ArtifactId: "foo1", ArtifactType: "blah1"}, nil
}
func (s *ArtifactServer) GetArtifacts(context.Context, *apiv2beta1.GetArtifactRequest) (*apiv2beta1.Artifact, error) {

	return &apiv2beta1.Artifact{ArtifactId: "foo2", ArtifactType: "blah2"}, nil
}
func (s *ArtifactServer) DownloadArtifact(request *apiv2beta1.DownloadArtifactRequest, stream apiv2beta1.ArtifactService_DownloadArtifactServer) error {
	ctx := context.Background()
	config := &aws.Config{
		Credentials:      credentials.NewStaticCredentials(os.Getenv("AWS_ID"), os.Getenv("AWS_SECRET"), ""),
		Region:           aws.String("minio"),
		DisableSSL:       aws.Bool(true),
		S3ForcePathStyle: aws.Bool(true),
		Endpoint:         aws.String("https://minio-dspa1.apps.hukhan-3.dev.datahub.redhat.com"),
	}
	sess, err := session.NewSession(config)
	if err != nil {
		return err
	}
	bucketName := "mlpipeline"
	openedBucket, err := s3blob.OpenBucket(ctx, sess, bucketName, nil)
	defer openedBucket.Close()
	reader, err := openedBucket.NewReader(ctx, "samplejson.json", nil)
	if err != nil {
		glog.Error(err)
		return err
	}
	defer reader.Close()

	buffer := make([]byte, 1024) // Create a buffer of 1KB

	for {
		n, err := reader.Read(buffer)
		if err != nil && err != io.EOF {
			glog.Fatal("Failed to send stream buffer data")
		}
		if n > 0 {
			body := &apiv2beta1.FileResponse{
				//ContentType: "application/octet-stream",
				Chunk: buffer[:n],
			}
			if err := stream.Send(body); err != nil {
				return err
			}
			fmt.Print(string(buffer[:n]))
		}
		if err == io.EOF {
			break
		}
	}
	glog.Info("Ending stream.")
	return nil
}

type Message struct {
	Id      int    `json:"id,omitempty"`
	Message string `json:"message,omitempty"`
}

func (s *ArtifactServer) DownloadArtifactHttp(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	config := &aws.Config{
		Credentials:      credentials.NewStaticCredentials(os.Getenv("AWS_ID"), os.Getenv("AWS_SECRET"), ""),
		Region:           aws.String("minio"),
		DisableSSL:       aws.Bool(true),
		S3ForcePathStyle: aws.Bool(true),
		Endpoint:         aws.String("https://minio-dspa1.apps.hukhan-3.dev.datahub.redhat.com"),
	}
	sess, err := session.NewSession(config)
	if err != nil {
		s.writeErrorToResponse(w, http.StatusInternalServerError, err)
	}
	bucketName := "mlpipeline"
	openedBucket, err := s3blob.OpenBucket(ctx, sess, bucketName, nil)
	defer openedBucket.Close()
	reader, err := openedBucket.NewReader(ctx, "samplejson.json", nil)
	if err != nil {
		glog.Error(err)
		s.writeErrorToResponse(w, http.StatusInternalServerError, err)
	}
	defer reader.Close()

	cn, ok := w.(http.CloseNotifier)
	if !ok {
		http.NotFound(w, r)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.NotFound(w, r)
		return
	}

	// Send the initial headers saying we're gonna stream the response.
	w.Header().Set("Transfer-Encoding", "chunked")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	buffer := make([]byte, 1024) // Create a buffer of 1KB
	for {
		select {
		case <-cn.CloseNotify():
			glog.Infof("Client stopped listening")
			return
		default:
			n, err := reader.Read(buffer)
			if err != nil && err != io.EOF {
				glog.Fatal("Failed to send stream buffer data")
				return
			}
			if n > 0 {
				// Send some data.
				_, err := w.Write(buffer[:n])
				if err != nil {
					glog.Fatal(err)
					return
				}
				flusher.Flush()
				fmt.Print(string(buffer[:n]))
			}
			if err == io.EOF {
				glog.Info("Reached EOF.")
				return
			}
		}
	}

	//glog.Info("Ending stream.")
}

func (s *ArtifactServer) writeErrorToResponse(w http.ResponseWriter, code int, err error) {
	glog.Errorf("Failed to read artifact. Error: %+v", err)
	w.WriteHeader(code)
	errorResponse := &api.Error{ErrorMessage: err.Error(), ErrorDetails: fmt.Sprintf("%+v", err)}
	errBytes, err := json.Marshal(errorResponse)
	if err != nil {
		w.Write([]byte("Error reading artifact"))
	}
	w.Write(errBytes)
}

func NewArtifactServer(resourceManager *resource.ResourceManager) *ArtifactServer {
	return &ArtifactServer{resourceManager: resourceManager}
}
