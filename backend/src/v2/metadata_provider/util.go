package metadata_provider

import (
	"fmt"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"strings"
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

const prefix = "pipelinechannel--"

// extractPipelineChannelName will extract the input name if input is a parameter channel
func extractPipelineChannelName(inputKey string) string {
	if len(inputKey) > len(prefix) && inputKey[:len(prefix)] == prefix {
		return inputKey[len(prefix):]
	}
	return inputKey
}

func isPipelineChannel(inputKey string) bool {
	return strings.HasPrefix(inputKey, prefix)
}

func SanitizeTaskName(name string, index *int) string {
	// in the case of a loop give a more meaningful name that includes the loop iteration
	if strings.HasPrefix(name, "for-loop") && index != nil {
		return fmt.Sprintf("iteration-%d", *index)
	}
	return name
}

func PBParamsToRunParameters(inputParams map[string]*structpb.Value) []RunParameter {
	var parameters []RunParameter
	for k, v := range inputParams {

		// In the case of for loops the naming is a bit unfortunate for the loop parameters, consider the case:
		//     parameters = [
		//        {'top_k': k, 'temperature': t}
		//        for k in top_k_values
		//        for t in temperature_values
		//    ]
		//
		//    with dsl.ParallelFor(parameters) as param:
		//        # Execute RAG pipeline
		//        rag_execute_task = rag_pipeline_execute(
		//            query=query,
		//            top_k=param.top_k,
		//            temperature=param.temperature,
		//            documents=fetch_docs_task.outputs["documents"]
		//        ).set_caching_options(False)
		//
		// In this case the inputParams would be
		// {
		//  "pipelinechannel--ground_truth": "The Eiffel Tower is a famous landmark in Paris.",
		//  "pipelinechannel--loop-item-param-1": {
		//    "temperature": "0.5",
		//    "top_k": "11"
		//  },
		//  "pipelinechannel--query": "Tell me about the Eiffel Tower."
		//}
		// Note that documents is an artifact, so it's not listed.
		// In the cause of "ground_truth" and "query" the solution is trivial, we can just extract these values and
		// remove the pipelinechannel prefix.
		// In the case of the loop param, the naming is done by the compiler in such a way to avoid name collisions,
		// so we retain this name and add the param name as a suffix. So the result is:
		// [
		//   { "loop-item-param-1.temperature": "0.5" },
		//   { "loop-item-param-1.top_k": "11" }
		//   { "ground_truth": "The Eiffel Tower is a famous landmark in Paris." },
		//   { "query": "Tell me about the Eiffel Tower." }
		// ]
		extractedKey := extractPipelineChannelName(k)
		if isPipelineChannel(k) {
			if innerStructValue, ok := v.Kind.(*structpb.Value_StructValue); ok {
				for innerKey, innerValue := range innerStructValue.StructValue.Fields {
					parameters = append(parameters, RunParameter{Name: extractedKey + "." + innerKey, Value: structPBtoString(innerValue)})
				}
			} else {
				parameters = append(parameters, RunParameter{Name: extractedKey, Value: structPBtoString(v)})
			}
		} else {
			parameters = append(parameters, RunParameter{Name: k, Value: structPBtoString(v)})
		}

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
