// Copyright 2025 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package model

// MetricType represents the type of metric (input/output)
type MetricType int

const (
	MetricTypeInput  MetricType = 0
	MetricTypeOutput MetricType = 1
)

// MetricSchema represents the schema type for metrics
type MetricSchema string

const (
	MetricSchemaMetric                     MetricSchema = "Metric"
	MetricSchemaClassificationMetric       MetricSchema = "ClassificationMetric"
	MetricSchemaSlicedClassificationMetric MetricSchema = "SlicedClassificationMetric"
)

// RunMetric represents metrics stored for tasks, replacing MLMD metrics artifacts
type RunMetric struct {
	RunID string `gorm:"column:RunID; not null; primaryKey; type:varchar(191);"`
	// TaskID is nullable to support migration from V1 Metrics which did not have a Task associated with them.
	// In the future we can make this nullable by introducing synthetic "run-level task" that we can create
	// for old runs that don't have a task associated with them, and backfilling metrics' TaskID to satisfy
	// the non-nullable constraint.
	TaskID         string       `gorm:"column:TaskID; primaryKey; type:varchar(191);"`
	Name           string       `gorm:"column:Name; not null; primaryKey; type:varchar(128);"`
	NumberValue    *float64     `gorm:"column:NumberValue; default:null;"`
	Namespace      string       `gorm:"column:Namespace; not null; type:varchar(63);"`
	JsonValue      JSONData     `gorm:"column:JsonValue; type:json; default:null;"`
	CreatedAtInSec int64        `gorm:"column:CreatedAtInSec; not null; index:idx_run_metrics_created_timestamp;"`
	Type           MetricType   `gorm:"column:Type; not null;"`
	Schema         MetricSchema `gorm:"column:Schema; not null; type:varchar(64);"`

	// Relationships
	Task Task `gorm:"foreignKey:TaskID;references:UUID;constraint:fk_run_metrics_tasks,OnDelete:CASCADE,OnUpdate:CASCADE;"`
}

type RunMetricV1 struct {
	RunUUID     string
	NodeID      string
	Name        string
	NumberValue float64
	Format      string
	Payload     LargeText
}

func (rm RunMetric) PrimaryKeyColumnName() string {
	return "TaskID"
}

func (rm RunMetric) DefaultSortField() string {
	return "CreatedAtInSec"
}

func (rm RunMetric) APIToModelFieldMap() map[string]string {
	return runMetricAPIToModelFieldMap
}

func (rm RunMetric) GetModelName() string {
	return "metrics"
}

func (rm RunMetric) GetSortByFieldPrefix(s string) string {
	return "metrics."
}

func (rm RunMetric) GetKeyFieldPrefix() string {
	return "metrics."
}

var runMetricAPIToModelFieldMap = map[string]string{
	"run_id":       "RunID",
	"task_id":      "TaskID",
	"name":         "Name",
	"number_value": "NumberValue",
	"namespace":    "Namespace",
	"json_value":   "JsonValue",
	"created_at":   "CreatedAtInSec",
	"type":         "Type",
	"schema":       "Schema",
}

func (rm RunMetric) GetField(name string) (string, bool) {
	if field, ok := runMetricAPIToModelFieldMap[name]; ok {
		return field, true
	}
	return "", false
}

func (rm RunMetric) GetFieldValue(name string) interface{} {
	switch name {
	case "RunID":
		return rm.RunID
	case "TaskID":
		return rm.TaskID
	case "Name":
		return rm.Name
	case "NumberValue":
		return rm.NumberValue
	case "Namespace":
		return rm.Namespace
	case "JsonValue":
		return rm.JsonValue
	case "CreatedAtInSec":
		return rm.CreatedAtInSec
	case "Type":
		return rm.Type
	case "Schema":
		return rm.Schema
	default:
		return nil
	}
}
