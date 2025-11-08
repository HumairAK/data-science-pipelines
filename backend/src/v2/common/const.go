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

package common

// KFP service account token configuration for authentication with API server
const (
	// KFPTokenExpirationSeconds is the expiration time for the projected service account token.
	// Set to 7200 seconds (2 hours) to provide enough buffer while kubelet auto-rotates tokens.
	KFPTokenExpirationSeconds = 7200
	// KFPTokenVolumeName is the name of the volume containing the KFP service account token
	KFPTokenVolumeName = "kfp-launcher-token"
	// KFPTokenMountPath is the path where the KFP token is mounted
	KFPTokenMountPath = "/var/run/secrets/kfp"
	// KFPTokenAudience is the audience for the projected service account token
	KFPTokenAudience = "pipelines.kubeflow.org"
)

// KFPTokenExpirationSecondsPtr returns a pointer to the KFP token expiration seconds constant.
// This is used for the ServiceAccountTokenProjection ExpirationSeconds field which requires *int64.
func KFPTokenExpirationSecondsPtr() *int64 {
	seconds := int64(KFPTokenExpirationSeconds)
	return &seconds
}
