package client_manager

import (
	"fmt"

	v2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/cacheutils"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata_provider"
	md "github.com/kubeflow/pipelines/backend/src/v2/metadata_provider/manager"
	k8score "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

type ClientManagerInterface interface {
	K8sClient() kubernetes.Interface
	MetadataClient() metadata.ClientInterface
	CacheClient() cacheutils.Client
	MetadataRunProvider() metadata_provider.RunProvider
	RunServiceClient() v2beta1.RunServiceClient
	MetadataArtifactProvider() metadata_provider.MetadataArtifactProvider
	MetadataNestedRunSupport() bool
}

// Ensure ClientManager implements ClientManagerInterface
var _ ClientManagerInterface = (*ClientManager)(nil)

// ClientManager is a container for various service clients.
type ClientManager struct {
	k8sClient                kubernetes.Interface
	metadataClient           metadata.ClientInterface
	cacheClient              cacheutils.Client
	metadataRunProvider      metadata_provider.RunProvider
	runServiceClient         v2beta1.RunServiceClient
	metadataArtifactProvider metadata_provider.MetadataArtifactProvider
	metadataNestedRunSupport bool
	metadataEnv              []k8score.EnvVar
}

type Options struct {
	MLMDServerAddress         string
	MLMDServerPort            string
	CacheDisabled             bool
	MetadatRunProviderConfig  string
	MLPipelineServiceName     string
	MLPipelineServiceGRPCPort string
	MLPipelineTLSEnabled      bool
	CaCertPath                string
}

// NewClientManager creates and Init a new instance of ClientManager.
func NewClientManager(options *Options) (*ClientManager, error) {
	clientManager := &ClientManager{}
	err := clientManager.init(options)
	if err != nil {
		return nil, err
	}

	return clientManager, nil
}

func (cm *ClientManager) K8sClient() kubernetes.Interface {
	return cm.k8sClient
}

func (cm *ClientManager) MetadataClient() metadata.ClientInterface {
	return cm.metadataClient
}

func (cm *ClientManager) CacheClient() cacheutils.Client {
	return cm.cacheClient
}

func (cm *ClientManager) MetadataRunProvider() metadata_provider.RunProvider {
	return cm.metadataRunProvider
}

func (cm *ClientManager) MetadataArtifactProvider() metadata_provider.MetadataArtifactProvider {
	return cm.metadataArtifactProvider
}

func (cm *ClientManager) MetadataNestedRunSupport() bool {
	return cm.metadataNestedRunSupport
}

func (cm *ClientManager) MetadataEnv() []k8score.EnvVar {
	return cm.metadataEnv
}

func (cm *ClientManager) RunServiceClient() v2beta1.RunServiceClient {
	return cm.runServiceClient
}

func (cm *ClientManager) init(opts *Options) error {
	k8sClient, err := initK8sClient()
	if err != nil {
		return err
	}
	metadataClient, err := initMetadataClient(opts.MLMDServerAddress, opts.MLMDServerPort, opts.MLPipelineTLSEnabled, opts.CaCertPath)
	if err != nil {
		return err
	}
	cacheClient, err := initCacheClient(opts.CacheDisabled, opts.MLPipelineTLSEnabled)
	if err != nil {
		return err
	}
	cm.k8sClient = k8sClient
	cm.metadataClient = metadataClient
	cm.cacheClient = cacheClient

	// TODO(humair): just have CM hold the metadata manager, instead of a million of these field
	if opts.MetadatRunProviderConfig != "" {
		metadataProvider, err := md.NewProviderFromJSON(opts.MetadatRunProviderConfig)
		if err != nil {
			return fmt.Errorf("failed to parse metadata provider config: %w", err)
		}
		metadatRunProvider, err := metadataProvider.NewRunProvider()
		if err != nil {
			return fmt.Errorf("failed to create metadata provider: %w", err)
		}
		cm.metadataRunProvider = metadatRunProvider

		artifactProvider, err := metadataProvider.NewMetadataArtifactProvider()
		if err != nil {
			return fmt.Errorf("failed to create metadata artifact provider: %w", err)
		}
		cm.metadataArtifactProvider = artifactProvider
		cm.metadataNestedRunSupport = metadataProvider.SupportNestedRuns()
		cm.metadataEnv = metadataProvider.GetEnv()
	}

	connection, err := util.GetRpcConnection(
		fmt.Sprintf(
			"%s:%s",
			opts.MLPipelineServiceName,
			opts.MLPipelineServiceGRPCPort,
		),
		opts.MLPipelineTLSEnabled,
		opts.CaCertPath,
	)
	if err != nil {
		return fmt.Errorf("failed to connect to ML Pipeline GRPC server: %w", err)
	}
	cm.runServiceClient = v2beta1.NewRunServiceClient(connection)

	return nil
}

func initK8sClient() (kubernetes.Interface, error) {
	restConfig, err := util.GetKubernetesConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize kubernetes client: %w", err)
	}
	k8sClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize kubernetes client set: %w", err)
	}
	return k8sClient, nil
}

func initMetadataClient(address string, port string, tlsEnabled bool, caCertPath string) (metadata.ClientInterface, error) {
	return metadata.NewClient(address, port, tlsEnabled, caCertPath)
}

func initCacheClient(cacheDisabled bool, tlsEnabled bool) (cacheutils.Client, error) {
	return cacheutils.NewClient(cacheDisabled, tlsEnabled)
}
