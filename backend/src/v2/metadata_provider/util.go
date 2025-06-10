package metadata_provider

import (
	"google.golang.org/protobuf/types/known/structpb"
	"sync"
)

var (
	registryMu sync.RWMutex
	registry   = make(map[string]ProviderFactory)
)

// Register is called by each provider package at init time
func Register(name string, factory ProviderFactory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[name] = factory
}

// Lookup returns a registered factory
func Lookup(name string) (ProviderFactory, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	f, ok := registry[name]
	return f, ok
}

func PBParamsToRunParameters(inputParams map[string]*structpb.Value) []RunParameter {
	var parameters []RunParameter
	for k, v := range inputParams {
		parameters = append(parameters, RunParameter{Name: k, Value: v.String()})
	}
	return parameters
}
