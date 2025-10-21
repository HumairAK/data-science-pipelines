package client_manager

import (
	"github.com/kubeflow/pipelines/backend/src/v2/driver/common"
	"k8s.io/client-go/kubernetes"
)

type FakeClientManager struct {
	k8sClient kubernetes.Interface
	driverAPI common.DriverAPI
}

// Ensure FakeClientManager implements ClientManagerInterface
var _ ClientManagerInterface = (*FakeClientManager)(nil)

func (f *FakeClientManager) K8sClient() kubernetes.Interface {
	return f.k8sClient
}

func (f *FakeClientManager) DriverAPI() common.DriverAPI {
	return f.driverAPI
}

func NewFakeClientManager(k8sClient kubernetes.Interface, driverAPI common.DriverAPI) *FakeClientManager {
	return &FakeClientManager{
		k8sClient: k8sClient,
		driverAPI: driverAPI,
	}
}
