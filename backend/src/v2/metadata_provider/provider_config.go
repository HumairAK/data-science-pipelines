package metadata_provider

// TODO: This should be the Proto value once defined in experiment.proto for createrequest
type ProviderRuntimeConfig map[string]interface{}

// MetadataProvider defines the type of metadata provider
type MetadataProvider string

const (
	MetadataProviderMLFlow MetadataProvider = "mlflow"
)
