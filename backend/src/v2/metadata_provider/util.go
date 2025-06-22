package metadata_provider

import (
	"fmt"
	"google.golang.org/protobuf/encoding/protojson"
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
		parameters = append(parameters, RunParameter{Name: k, Value: structPBtoString(v)})
	}
	return parameters
}

// In the case of list/struct we return json values
func structPBtoString(v *structpb.Value) string {
	switch kind := v.Kind.(type) {
	case *structpb.Value_NullValue:
		return "null"
	case *structpb.Value_NumberValue:
		return fmt.Sprintf("%v", kind.NumberValue)
	case *structpb.Value_StringValue:
		return kind.StringValue
	case *structpb.Value_BoolValue:
		return fmt.Sprintf("%v", kind.BoolValue)
	case *structpb.Value_StructValue:
		b, _ := protojson.Marshal(kind.StructValue)
		return string(b)
	case *structpb.Value_ListValue:
		b, _ := protojson.Marshal(kind.ListValue)
		return string(b)
	default:
		return "<unknown>"
	}
}
