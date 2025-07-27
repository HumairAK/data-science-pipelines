package proto_tests

import (
	"fmt"
	"path/filepath"
	"testing"

	pb "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
)

// This is the commit that contains the proto generated files
// that were used to generate the test data.
const commit = "1791485"

func generatePath(path string) string {
	return filepath.Join(fmt.Sprintf("generated-%s", commit), path)
}

func TestRuns(t *testing.T) {
	testOBJ(t, caseOpts[*pb.Run]{
		message:          completedRun,
		expectedPBPath:   generatePath("run_completed.pb"),
		expectedJSONPath: generatePath("run_completed.json"),
	})

	testOBJ(t, caseOpts[*pb.Run]{
		message:          completedRunWithPipelineSpec,
		expectedPBPath:   generatePath("run_completed_with_spec.pb"),
		expectedJSONPath: generatePath("run_completed_with_spec.json"),
	})

	testOBJ(t, caseOpts[*pb.Run]{
		message:          failedRun,
		expectedPBPath:   generatePath("run_failed.pb"),
		expectedJSONPath: generatePath("run_failed.json"),
	})

	testOBJ(t, caseOpts[*pb.Pipeline]{
		message:          pipeline,
		expectedPBPath:   generatePath("pipeline.pb"),
		expectedJSONPath: generatePath("pipeline.json"),
	})
	testOBJ(t, caseOpts[*pb.PipelineVersion]{
		message:          pipelineVersion,
		expectedPBPath:   generatePath("pipeline_version.pb"),
		expectedJSONPath: generatePath("pipeline_version.json"),
	})
	testOBJ(t, caseOpts[*pb.Experiment]{
		message:          experiment,
		expectedPBPath:   generatePath("experiment.pb"),
		expectedJSONPath: generatePath("experiment.json"),
	})
}
