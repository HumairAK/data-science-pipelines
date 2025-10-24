package common

import (
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient/kfpapi"
)

// DriverAPI is deprecated. Use kfpapi.API instead.
// This type alias is provided for backward compatibility.
type DriverAPI = kfpapi.API

// NewDriverAPI is deprecated. Use kfpapi.New instead.
// This function is provided for backward compatibility.
func NewDriverAPI(c *apiclient.Client) DriverAPI {
	return kfpapi.New(c)
}
