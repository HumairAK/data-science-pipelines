package types

type CreateExperimentRequest struct {
	Name             string          `json:"name"`
	ArtifactLocation string          `json:"artifact_location"`
	Tags             []ExperimentTag `json:"tags"`
}

type CreateExperimentResponse struct {
	ExperimentId string `json:"experiment_id"`
}

type SearchExperimentRequest struct {
	MaxResults int64    `json:"max_results"`
	PageToken  string   `json:"page_token"`
	Filter     string   `json:"filter"`
	OrderBy    []string `json:"order_by"`
	ViewType   ViewType `json:"view_type"`
}

type SearchExperimentResponse struct {
	Experiments   []Experiment `json:"experiments"`
	NextPageToken string       `json:"next_page_token"`
}

type GetExperimentRequest struct {
	ExperimentId string `json:"experiment_id"`
}

type GetExperimentResponse struct {
	Experiment Experiment `json:"experiment"`
}

type GetExperimentByNameRequest struct {
	ExperimentName string `json:"experiment_name"`
}

type GetExperimentByNameResponse struct {
	Experiment Experiment `json:"experiment"`
}

type UpdateExperimentRequest struct {
	ExperimentId string `json:"experiment_id"`
	NewName      string `json:"new_name"`
}
