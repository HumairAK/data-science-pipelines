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
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	api "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	executor "github.com/kubeflow/pipelines/backend/src/apiserver/artifactstorage"
	"github.com/kubeflow/pipelines/backend/src/apiserver/artifactstorage/types"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	log "github.com/sirupsen/logrus"
	"io"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/utils/env"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strconv"
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

type Message struct {
	Id      int    `json:"id,omitempty"`
	Message string `json:"message,omitempty"`
}

const (
	ArtifactKey = "artifact_id"
)

func (s *ArtifactServer) DownloadArtifactHttp(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	artifactId, err := strconv.ParseInt(vars[ArtifactKey], 10, 64)
	if err != nil {
		s.writeErrorToResponse(w, http.StatusInternalServerError, fmt.Errorf("failed to parse artifact parameter in request: %v", err))
		return
	}
	ctx := context.Background()
	artifacts, err := s.resourceManager.GetArtifactById(ctx, []int64{artifactId})

	// note artifacts length will never be greater than one since artifact ID uniquely identifies one artifact
	// but we add a check for completeness
	if err != nil || artifacts == nil || len(artifacts) > 1 {
		s.writeErrorToResponse(w, http.StatusNotFound, fmt.Errorf("failed to find artifact with id %d: %v", artifactId, err))
		return
	}

	artifact := artifacts[0]
	sessionInfo, namespace, err := s.resourceManager.GetArtifactSessionInfo(ctx, artifact)
	if err != nil || sessionInfo == nil {
		s.writeErrorToResponse(w, http.StatusNotFound, fmt.Errorf("failed to retrieve session info for artifact id %d error: %v", artifactId, err))
		return
	}
	endpoint, _ := url.Parse(sessionInfo.Session.Endpoint)

	art := &types.Artifact{
		Name: artifact.CustomProperties["display_name"].GetStringValue(),
		ArtifactLocation: types.ArtifactLocation{
			S3: &types.S3Artifact{
				S3Bucket: types.S3Bucket{
					Bucket:   sessionInfo.BucketName,
					Region:   sessionInfo.Session.Region,
					Endpoint: endpoint.Host,
					Insecure: aws.Bool(sessionInfo.Session.DisableSSL),
					AccessKeySecret: &v1.SecretKeySelector{
						LocalObjectReference: v1.LocalObjectReference{Name: sessionInfo.Session.SecretName},
						Key:                  sessionInfo.Session.AccessKeyKey,
					},
					SecretKeySecret: &v1.SecretKeySelector{
						LocalObjectReference: v1.LocalObjectReference{Name: sessionInfo.Session.SecretName},
						Key:                  sessionInfo.Session.SecretKeyKey,
					},
				},
				Key: sessionInfo.Prefix,
			},
		},
	}
	driver, err := executor.NewDriver(ctx, art, s.resourceManager, namespace)
	if err != nil {
		s.writeErrorToResponse(w, http.StatusInternalServerError, fmt.Errorf("failed to create artifact driver for artifact id %d: %v", artifactId, err))
		return
	}

	stream, err := driver.OpenStream(art)
	if err != nil {
		s.writeErrorToResponse(w, http.StatusInternalServerError, fmt.Errorf("failed to stream artifact: %v", err))
		return
	}

	defer func() {
		if stream != nil {
			err = stream.Close()
			if err != nil {
				log.WithFields(log.Fields{"stream": stream}).WithError(err).Warning("Error closing stream")
			}
		}
	}()

	key, _ := art.GetKey()
	w.Header().Add("Content-Disposition", fmt.Sprintf(`filename="%s"`, path.Base(key)))
	w.Header().Add("Content-Type", mime.TypeByExtension(path.Ext(key)))
	w.Header().Add("Content-Security-Policy", env.GetString("ARTIFACT_CONTENT_SECURITY_POLICY", "sandbox; base-uri 'none'; default-src 'none'; img-src 'self'; style-src 'self' 'unsafe-inline'"))
	w.Header().Add("X-Frame-Options", env.GetString("ARTIFACT_X_FRAME_OPTIONS", "SAMEORIGIN"))

	_, err = io.Copy(w, stream)
	if err != nil {
		s.writeErrorToResponse(w, http.StatusInternalServerError, fmt.Errorf("failed to stream artifact: %v", err))
	} else {
		w.WriteHeader(http.StatusOK)
	}
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
