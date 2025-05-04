package types

type CreateRunRequest struct {
	ExperimentId string   `json:"experiment_id"`
	UserId       string   `json:"user_id"`
	RunName      string   `json:"run_name"`
	StartTime    int64    `json:"start_time"`
	Tags         []RunTag `json:"tags"`
}

type CreateRunResponse struct {
	Run Run `json:"run"`
}

type DeleteRunRequest struct {
	RunID string `json:"run_id"`
}

type RestoreRunRequest struct {
	RunID string `json:"run_id"`
}

type GetRunRequest struct {
	RunID string `json:"run_id"`
}

type GetRunResponse struct {
	Run Run `json:"run"`
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

type SetTagsRequest struct {
	RunId   string `json:"run_id"`
	RunUUID string `json:"run_uuid"`
	Key     string `json:"key"`
	Value   string `json:"value"`
}

type DeleteTagsRequest struct {
	RunId string `json:"run_id"`
	Key   string `json:"key"`
}

type ListArtifactsRequest struct {
	RunId     string `json:"run_id"`
	RunUUID   string `json:"run_uuid"`
	Path      string `json:"path"`
	PageToken string `json:"page_token"`
}

type ListArtifactsResponse struct {
	RootURI       string     `json:"root_uri"`
	Files         []FileInfo `json:"files"`
	NextPageToken string     `json:"next_page_token"`
}
