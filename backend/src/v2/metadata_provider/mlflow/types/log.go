package types

type LogParamRequest struct {
	RunId   string `json:"run_id"`
	RunUUID string `json:"run_uuid"`
	Key     string `json:"key"`
	Value   string `json:"value"`
}

type LogMetricRequest struct {
	RunId     string  `json:"run_id"`
	RunUUID   string  `json:"run_uuid"`
	Key       string  `json:"key"`
	Value     float64 `json:"value"`
	Timestamp int64   `json:"timestamp"`
	Step      int64   `json:"step"`
}

type LogInputRequest struct {
	RunId    string         `json:"run_id"`
	Datasets []DataSetInput `json:"dataset"`
}
