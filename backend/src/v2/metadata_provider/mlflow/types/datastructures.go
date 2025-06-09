package types

type Dataset struct {
	Name       string `json:"name"`
	Digest     int64  `json:"digest"`
	SourceType int64  `json:"source_type"`
	Source     string `json:"source"`
	Schema     string `json:"schema"`
	Profile    string `json:"profile"`
}

type DataSetInput struct {
	Tags    []RunTag `json:"tags"`
	Dataset Dataset  `json:"dataset"`
}

type Experiment struct {
	ExperimentID     string          `json:"experiment_id"`
	Name             string          `json:"name"`
	ArtifactLocation string          `json:"artifact_location"`
	LifecycleStage   string          `json:"lifecycle_stage"`
	LastUpdateTime   int64           `json:"last_update_time"`
	CreationTime     int64           `json:"creation_time"`
	Tags             []ExperimentTag `json:"tags"`
}

type ExperimentTag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type FileInfo struct {
	Path     string `json:"path"`
	IsDir    bool   `json:"is_dir"`
	FileSize int64  `json:"file_size"`
}

type InputTag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Metric struct {
	Key       string  `json:"key"`
	Value     float64 `json:"value"`
	Timestamp int64   `json:"timestamp"`
	Step      int64   `json:"step"`
}

type ModelVersion struct{}

type ModelVersionTag struct{}

type Param struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type RegisteredModel struct{}

type RegisteredModelAlias struct{}

type RegisteredModelTag struct{}

type Run struct {
	Info   RunInfo   `json:"info"`
	Data   RunData   `json:"data"`
	Inputs RunInputs `json:"inputs"`
}

type RunData struct {
	Metrics []Metric `json:"metrics"`
	Params  []Param  `json:"params"`
	Tags    []RunTag `json:"tags"`
}

type RunInfo struct {
	RunID          string    `json:"run_id"`
	RunUUID        string    `json:"run_uuid"`
	RunName        string    `json:"run_name"`
	ExperimentID   string    `json:"experiment_id"`
	UserID         string    `json:"user_id"` // deprecated, may be omitted
	Status         RunStatus `json:"status"`
	StartTime      int64     `json:"start_time"`
	EndTime        int64     `json:"end_time"`
	ArtifactURI    string    `json:"artifact_uri"`
	LifecycleStage string    `json:"lifecycle_stage"`
}

type RunInputs struct {
	DatasetInputs []DataSetInput `json:"dataset_inputs"`
}

type RunTag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type ModelVersionStatus struct{}

type RunStatus string

const (
	Running   RunStatus = "RUNNING"
	Scheduled RunStatus = "SCHEDULED"
	Finished  RunStatus = "FINISHED"
	Failed    RunStatus = "FAILED"
	Killed    RunStatus = "KILLED"
)

type ViewType string

const (
	ACTIVE_ONLY ViewType = "ACTIVE_ONLY"
	DELETE_ONLY ViewType = "DELETE_ONLY"
	ALL         ViewType = "ALL"
)
