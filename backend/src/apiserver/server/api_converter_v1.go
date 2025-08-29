package server

import (
	"regexp"

	apiv1beta1 "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/validation"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

// Converts API run metric to its internal representation.
// Supports both v1beta1 and v2beta1 API.
func toModelRunMetricV1(m interface{}, runId string) (*model.RunMetricV1, error) {
	var name, nodeId, format string
	var val float64
	switch apiRunMetric := m.(type) {
	case *apiv1beta1.RunMetric:
		name = apiRunMetric.GetName()
		nodeId = apiRunMetric.GetNodeId()
		val = apiRunMetric.GetNumberValue()
		format = apiRunMetric.GetFormat().String()
	default:
		return nil, util.NewUnknownApiVersionError("RunMetric", m)
	}
	modelMetric := &model.RunMetricV1{
		RunUUID:     runId,
		Name:        name,
		NodeID:      nodeId,
		NumberValue: val,
		Format:      format,
	}
	if err := validation.ValidateModel(modelMetric); err != nil {
		return nil, util.NewInternalServerError(err, "Failed to convert API run metric to internal representation")
	}
	return modelMetric, nil

}

// Converts internal run metric representation to its API counterpart.
// Supports v1beta1 API.
func toApiRunMetricV1(metric *model.RunMetricV1) *apiv1beta1.RunMetric {
	return &apiv1beta1.RunMetric{
		Name:   metric.Name,
		NodeId: metric.NodeID,
		Value: &apiv1beta1.RunMetric_NumberValue{
			NumberValue: metric.NumberValue,
		},
		Format: apiv1beta1.RunMetric_Format(apiv1beta1.RunMetric_Format_value[metric.Format]),
	}
}

// Converts an array of internal run metric representations to an array of their API counterparts.
// Supports v1beta1 API.
func toApiRunMetricsV1(m []*model.RunMetricV1) []*apiv1beta1.RunMetric {
	apiMetrics := make([]*apiv1beta1.RunMetric, 0)
	for _, metric := range m {
		apiMetrics = append(apiMetrics, toApiRunMetricV1(metric))
	}
	return apiMetrics
}

// Convert results of run metrics creation to API response.
// Supports v1beta1 API.
// Return nil if a parsing error occurs.
func toApiReportMetricsResultV1(metricName string, nodeId string, status string, message string) *apiv1beta1.ReportRunMetricsResponse_ReportRunMetricResult {
	apiResultV1 := &apiv1beta1.ReportRunMetricsResponse_ReportRunMetricResult{
		MetricName:   metricName,
		MetricNodeId: nodeId,
		Message:      message,
	}
	switch status {
	case "ok":
		apiResultV1.Status = apiv1beta1.ReportRunMetricsResponse_ReportRunMetricResult_OK
	case "internal":
		apiResultV1.Status = apiv1beta1.ReportRunMetricsResponse_ReportRunMetricResult_INTERNAL_ERROR
	case "invalid":
		apiResultV1.Status = apiv1beta1.ReportRunMetricsResponse_ReportRunMetricResult_INVALID_ARGUMENT
	case "duplicate":
		apiResultV1.Status = apiv1beta1.ReportRunMetricsResponse_ReportRunMetricResult_DUPLICATE_REPORTING
	default:
		return nil
	}
	return apiResultV1
}

// Validates a run metric fields from request.
func validateRunMetricV1(metric *model.RunMetricV1) error {
	matched, err := regexp.MatchString(metricNamePattern, metric.Name)
	if err != nil {
		// This should never happen.
		return util.NewInternalServerError(
			err, "failed to compile pattern '%s'", metricNamePattern)
	}
	if !matched {
		return util.NewInvalidInputError(
			"metric.name '%s' doesn't match with the pattern '%s'", metric.Name, metricNamePattern)
	}
	if metric.NodeID == "" {
		return util.NewInvalidInputError("metric.node_id must not be empty")
	}
	if len(metric.NodeID) > 128 {
		return util.NewInvalidInputError(
			"metric.node_id '%s' cannot be longer than 128 characters", metric.NodeID)
	}
	return nil
}

// Converts RunMetricV1 to RunMetric
func convertModelRunMetricToV2(metricV1 *model.RunMetricV1) *model.RunMetric {
	return &model.RunMetric{
		RunID:       metricV1.RunUUID,
		Name:        metricV1.Name,
		NumberValue: &metricV1.NumberValue,
		Type:        model.MetricTypeOutput,
		Schema:      model.MetricSchemaMetric,
	}
}

// Converts RunMetric to RunMetricV1
func convertModelRunMetricToV1(metric *model.RunMetric) *model.RunMetricV1 {
	var numberValue float64
	if metric.NumberValue != nil {
		numberValue = *metric.NumberValue
	}
	return &model.RunMetricV1{
		RunUUID:     metric.RunID,
		Name:        metric.Name,
		NumberValue: numberValue,
		Format:      "PERCENTAGE",
	}
}
