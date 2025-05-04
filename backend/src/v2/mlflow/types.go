package mlflow

type GetRunRequest struct {
	RunID string `json:"run_id"`
}

type CreateRunResponse struct {
	Run Run `json:"run"`
}

type GetRunResponse struct {
	Run Run `json:"run"`
}

type SearchExperimentRequest struct {
	MaxResults int64    `json:"max_results"`
	PageToken  string   `json:"page_token"`
	Filter     string   `json:"filter"`
	OrderBy    []string `json:"order_by"`
	ViewType   ViewType `json:"view_type"`
}

type SearchRunRequest struct {
	ExperimentIds []string `json:"experiment_ids"`
	Filter        string   `json:"filter"`
	RunViewType   ViewType `json:"run_view_type"`
	MaxResults    int64    `json:"max_results"`
	OrderBy       []string `json:"order_by"`
	PageToken     string   `json:"page_token"`
}

type UpdateRunRequest struct {
	RunId   string    `json:"run_id"`
	RunUUID string    `json:"run_uuid"`
	Status  RunStatus `json:"status"`
	EndTime int64     `json:"end_time"`
	RunName string    `json:"run_name"`
}

type UpdateRunResponse struct {
	RunInfo RunInfo `json:"run_info"`
}

type SearchRunResponse struct {
	Runs          []Run  `json:"runs"`
	NextPageToken string `json:"next_page_token"`
}

type SearchExperimentResponse struct {
	Experiment    []Experiment `json:"experiments"`
	NextPageToken string       `json:"next_page_token"`
}

type Run struct {
	Info   RunInfo   `json:"info"`
	Data   RunData   `json:"data"`
	Inputs RunInputs `json:"inputs"`
}

type ViewType string

const (
	ACTIVE_ONLY ViewType = "ACTIVE_ONLY"
	DELETE_ONLY ViewType = "DELETE_ONLY"
	ALL         ViewType = "ALL"
)

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

type RunStatus string

const (
	Running   RunStatus = "RUNNING"
	Scheduled RunStatus = "SCHEDULED"
	Finished  RunStatus = "FINISHED"
	Failed    RunStatus = "FAILED"
	Killed    RunStatus = "KILLED"
)

type RunData struct {
	Metrics []Metric `json:"metrics"`
	Params  []Param  `json:"params"`
	Tags    []RunTag `json:"tags"`
}

type Metric struct {
	Key       string  `json:"key"`
	Value     float64 `json:"value"`
	Timestamp int64   `json:"timestamp"`
	Step      int64   `json:"step"`
}

type Param struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type RunTag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type ExperimentTag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type RunInputs struct {
	DatasetInputs []DataSetInput `json:"dataset_inputs"`
}

type DataSetInput struct {
	Tags    []RunTag `json:"tags"`
	Dataset Dataset  `json:"dataset"`
}

type Dataset struct {
	Name       string `json:"name"`
	Digest     int64  `json:"digest"`
	SourceType int64  `json:"source_type"`
	Source     string `json:"source"`
	Schema     string `json:"schema"`
	Profile    string `json:"profile"`
}

type LogParamRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Experiment struct {
	ExperimentID     string          `json:"experiment_id"`
	Name             string          `json:"name"`
	ArtifactLocation string          `json:"artifact_location"`
	LifecycleStage   string          `json:"lifecycle_stage"`
	LastUpdateTime   string          `json:"last_update_time"`
	CreationTime     string          `json:"creation_time"`
	Tags             []ExperimentTag `json:"tags"`
}
