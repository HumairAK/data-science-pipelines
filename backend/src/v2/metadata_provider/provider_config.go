package metadata_provider

import "google.golang.org/protobuf/types/known/structpb"

// TODO: This should be the Proto value once defined in experiment.proto for createrequest
type ProviderRuntimeConfig map[string]interface{}

func ConvertStructToConfig(s *structpb.Struct) ProviderRuntimeConfig {
	return s.AsMap()
}

// MetadataProvider defines the type of metadata provider
type MetadataProvider string

const (
	MetadataProviderMLFlow MetadataProvider = "mlflow"
)
