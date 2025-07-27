package proto_tests

import (
	"os"
	"path/filepath"
	"time"

	"github.com/golang/glog"
	pb "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/server"
	"google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const mockPipelineID = "9b187b86-7c0a-42ae-a0bc-2a746b6eb7a3"
const mockPipelineVersionID = "e15dc3ec-b45e-4cc7-bb07-e76b5dbce99a"
const pipelineSpecYamlPath = "pipelinespec.yaml"

func mockPipelineSpec() *structpb.Struct {
	yamlContent, err := os.ReadFile(filepath.Join("testdata", pipelineSpecYamlPath))
	if err != nil {
		glog.Fatal(err)
	}
	spec, err := server.YamlStringToPipelineSpecStruct(string(yamlContent))
	if err != nil {
		glog.Fatal(err)
	}
	return spec
}

func fixedTimestamp() *timestamppb.Timestamp {
	return timestamppb.New(time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC))
}

var completedRun = &pb.Run{
	RunId:          "completed-run-123",
	DisplayName:    "Production Pipeline Run",
	ExperimentId:   "exp-456",
	StorageState:   pb.Run_AVAILABLE,
	Description:    "Production pipeline execution for data processing",
	ServiceAccount: "sa1",
	CreatedAt:      fixedTimestamp(),
	ScheduledAt:    fixedTimestamp(),
	FinishedAt:     fixedTimestamp(),
	RecurringRunId: "recurring-schedule-001",
	State:          pb.RuntimeState_SUCCEEDED,
	PipelineSource: &pb.Run_PipelineVersionReference{
		PipelineVersionReference: &pb.PipelineVersionReference{
			PipelineId:        mockPipelineID,
			PipelineVersionId: mockPipelineVersionID,
		},
	},
	RuntimeConfig: &pb.RuntimeConfig{
		Parameters: map[string]*structpb.Value{
			"batch_size":    structpb.NewNumberValue(1000),
			"learning_rate": structpb.NewStringValue("foo"),
		},
	},
}

var completedRunWithPipelineSpec = &pb.Run{
	RunId:          "completed-run-123",
	DisplayName:    "Production Pipeline Run",
	ExperimentId:   "exp-456",
	StorageState:   pb.Run_AVAILABLE,
	Description:    "Production pipeline execution for data processing",
	ServiceAccount: "sa1",
	CreatedAt:      fixedTimestamp(),
	ScheduledAt:    fixedTimestamp(),
	FinishedAt:     fixedTimestamp(),
	RecurringRunId: "recurring-schedule-001",
	State:          pb.RuntimeState_SUCCEEDED,
	PipelineSource: &pb.Run_PipelineSpec{
		PipelineSpec: mockPipelineSpec(),
	},
	RuntimeConfig: &pb.RuntimeConfig{
		Parameters: map[string]*structpb.Value{
			"batch_size":    structpb.NewNumberValue(1000),
			"learning_rate": structpb.NewStringValue("foo"),
		},
	},
}

var failedRun = &pb.Run{
	RunId:          "failed-run-456",
	DisplayName:    "Data Processing Pipeline",
	ExperimentId:   "exp-789",
	StorageState:   pb.Run_AVAILABLE,
	Description:    "Failed attempt to process customer data",
	ServiceAccount: "sa2",
	CreatedAt:      fixedTimestamp(),
	ScheduledAt:    fixedTimestamp(),
	FinishedAt:     fixedTimestamp(),
	State:          pb.RuntimeState_FAILED,
	Error: &status.Status{
		Code:    1,
		Message: "This was a Failed Run.",
	},
}

var pipeline = &pb.Pipeline{
	PipelineId:  mockPipelineID,
	DisplayName: "Production Data Processing Pipeline",
	Name:        "pipeline1",
	Description: "Pipeline for processing and analyzing production data",
	CreatedAt:   fixedTimestamp(),
	Namespace:   "namespace1",
	Error: &status.Status{
		Code:    0,
		Message: "This a successful pipeline.",
	},
}

var pipelineVersion = &pb.PipelineVersion{
	PipelineVersionId: mockPipelineVersionID,
	PipelineId:        mockPipelineID,
	DisplayName:       "v1.0.0 Production Data Processing Pipeline",
	Name:              "pipelineversion1",
	Description:       "First stable version of the production data processing pipeline",
	CreatedAt:         fixedTimestamp(),
	PipelineSpec:      mockPipelineSpec(),
	PackageUrl:        &pb.Url{PipelineUrl: "gs://my-bucket/pipelines/pipeline1-v1.0.0.yaml"},
	CodeSourceUrl:     "https://github.com/org/repo/pipeline1/tree/v1.0.0",
	Error: &status.Status{
		Code:    0,
		Message: "This is a successful pipeline version.",
	},
}

var experiment = &pb.Experiment{
	ExperimentId:     "exp-456",
	DisplayName:      "Production Data Processing Experiment",
	Description:      "Experiment for testing production data processing pipeline",
	CreatedAt:        fixedTimestamp(),
	Namespace:        "namespace1",
	StorageState:     pb.Experiment_AVAILABLE,
	LastRunCreatedAt: fixedTimestamp(),
}
