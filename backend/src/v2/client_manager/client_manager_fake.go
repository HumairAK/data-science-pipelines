package client_manager

import (
	v2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/v2/cacheutils"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata_provider"
	"k8s.io/client-go/kubernetes"
)

type FakeClientManager struct {
	k8sClient           kubernetes.Interface
	metadataClient      metadata.ClientInterface
	cacheClient         cacheutils.Client
	metadataRunProvider metadata_provider.RunProvider
	runServiceClient    v2beta1.RunServiceClient
}

// Ensure FakeClientManager implements ClientManagerInterface
var _ ClientManagerInterface = (*FakeClientManager)(nil)

func (f *FakeClientManager) K8sClient() kubernetes.Interface {
	return f.k8sClient
}

func (f *FakeClientManager) MetadataClient() metadata.ClientInterface {
	return f.metadataClient
}

func (f *FakeClientManager) CacheClient() cacheutils.Client {
	return f.cacheClient
}

func (f *FakeClientManager) MetadataRunProvider() metadata_provider.RunProvider {
	return f.metadataRunProvider
}

func (f *FakeClientManager) RunServiceClient() v2beta1.RunServiceClient {
	return f.runServiceClient
}

func NewFakeClientManager(k8sClient kubernetes.Interface, metadataClient metadata.ClientInterface, cacheClient cacheutils.Client) *FakeClientManager {
	return &FakeClientManager{
		k8sClient:      k8sClient,
		metadataClient: metadataClient,
		cacheClient:    cacheClient,
	}
}
