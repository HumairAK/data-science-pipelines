package client_manager

import (
	"fmt"

	"github.com/kubeflow/pipelines/backend/src/v2/apiclient"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient/kfpapi"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type ClientManagerInterface interface {
	K8sClient() kubernetes.Interface
	DriverAPI() kfpapi.API
}

// Ensure ClientManager implements ClientManagerInterface
var _ ClientManagerInterface = (*ClientManager)(nil)

// ClientManager is a container for various service clients.
type ClientManager struct {
	k8sClient kubernetes.Interface
	driverAPI kfpapi.API
}

// NewClientManager creates and Init a new instance of ClientManager.
func NewClientManager() (*ClientManager, error) {
	clientManager := &ClientManager{}
	err := clientManager.init()
	if err != nil {
		return nil, err
	}

	return clientManager, nil
}

func (cm *ClientManager) K8sClient() kubernetes.Interface {
	return cm.k8sClient
}

func (cm *ClientManager) DriverAPI() kfpapi.API {
	return cm.driverAPI
}

func (cm *ClientManager) init() error {
	k8sClient, err := initK8sClient()
	if err != nil {
		return err
	}
	cm.k8sClient = k8sClient

	// Initialize connection to new KFP v2beta1 API server (Tasks/Artifacts)
	apiCfg := apiclient.FromEnv()
	kfpAPIClient, apiErr := apiclient.New(apiCfg)
	if apiErr != nil {
		return fmt.Errorf("failed to init KFP API client: %w", apiErr)
	}
	defer kfpAPIClient.Close()
	var driverAPI kfpapi.API
	driverAPI = kfpapi.New(kfpAPIClient)
	cm.driverAPI = driverAPI

	return nil
}

func initK8sClient() (kubernetes.Interface, error) {
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize kubernetes client: %w", err)
	}
	k8sClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize kubernetes client set: %w", err)
	}
	return k8sClient, nil
}
