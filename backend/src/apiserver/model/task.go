// Copyright 2018 The Kubeflow Authors
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

import (
	"database/sql/driver"
	"encoding/json"
)

// JSONData represents JSON data stored in database columns
type JSONData map[string]interface{}

// Scan implements sql.Scanner interface for JSONData
func (j *JSONData) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, j)
	case string:
		return json.Unmarshal([]byte(v), j)
	default:
		return nil
	}
}

// Value implements driver.Valuer interface for JSONData
func (j JSONData) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// PodNames represents JSON array of pod names
type PodNames []string

// Scan implements sql.Scanner interface for PodNames
func (p *PodNames) Scan(value interface{}) error {
	if value == nil {
		*p = nil
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, p)
	case string:
		return json.Unmarshal([]byte(v), p)
	default:
		return nil
	}
}

// Value implements driver.Valuer interface for PodNames
func (p *PodNames) Value() (driver.Value, error) {
	if p == nil {
		return nil, nil
	}
	return json.Marshal(p)
}

type Task struct {
	UUID           string   `gorm:"column:UUID; not null; primaryKey; type:varchar(191);"`
	Namespace      string   `gorm:"column:Namespace; not null; type:varchar(63);"`
	PipelineName   string   `gorm:"column:PipelineName; not null; type:varchar(128); index:idx_pipeline_name;"`
	RunUUID        string   `gorm:"column:RunUUID; type:varchar(191); not null; index:idx_parent_run,priority:1;"`
	Run            Run      `gorm:"foreignKey:RunUUID;references:UUID;constraint:tasks_RunUUID_run_details_UUID_foreign,OnDelete:CASCADE,OnUpdate:CASCADE;"`
	PodNames       PodNames `gorm:"column:PodNames; not null; type:json;"`
	CreatedAtInSec int64    `gorm:"column:CreatedAtInSec; not null; index:idx_created_timestamp;"`

	StartedInSec     int64    `gorm:"column:StartedInSec; default:0; index:idx_started_timestamp;"`
	FinishedInSec    int64    `gorm:"column:FinishedInSec; default:0; index:idx_finished_timestamp;"`
	Fingerprint      string   `gorm:"column:Fingerprint; not null; type:varchar(255);"`
	Name             string   `gorm:"column:Name; type:varchar(128); default:null;"`
	DisplayName      string   `gorm:"column:DisplayName; type:varchar(128); default:null;"`
	ParentTaskUUID   string   `gorm:"column:ParentTaskUUID; type:varchar(191); default:null; index:idx_parent_task_uuid; index:idx_parent_run,priority:2;"`
	ParentTask       *Task    `gorm:"foreignKey:ParentTaskUUID;references:UUID;constraint:fk_tasks_parent_task,OnDelete:CASCADE,OnUpdate:CASCADE;"`
	Status           int32    `gorm:"column:Status; not null;"`
	StatusMetadata   JSONData `gorm:"column:StatusMetadata; type:json; default:null;"`
	StateHistory     JSONData `gorm:"column:StateHistory; type:json;"`
	InputParameters  JSONData `gorm:"column:InputParameters; type:json;"`
	OutputParameters JSONData `gorm:"column:OutputParameters; type:json;"`
	Type             int32    `gorm:"column:Type; not null; type:varchar(64); index:idx_task_type;"`
	TypeAttrs        JSONData `gorm:"column:TypeAttrs; not null; type:json;"`
}

func (t Task) ToString() string {
	task, err := json.Marshal(t)
	if err != nil {
		return ""
	} else {
		return string(task)
	}
}

func (t Task) PrimaryKeyColumnName() string {
	return "UUID"
}

func (t Task) DefaultSortField() string {
	return "CreatedAtInSec"
}

func (t Task) APIToModelFieldMap() map[string]string {
	return taskAPIToModelFieldMap
}

func (t Task) GetModelName() string {
	return "tasks"
}

func (t Task) GetSortByFieldPrefix(s string) string {
	return "tasks."
}

func (t Task) GetKeyFieldPrefix() string {
	return "tasks."
}

var taskAPIToModelFieldMap = map[string]string{
	"task_id":           "UUID", // v2beta1 API
	"id":                "UUID", // v1beta1 API
	"namespace":         "Namespace",
	"name":              "Name",           // v2beta1 API
	"display_name":      "DisplayName",    // v2beta1 API
	"pipeline_name":     "PipelineName",   // v2beta1 API
	"pipelineName":      "PipelineName",   // v1beta1 API
	"run_id":            "RunUUID",        // v2beta1 API
	"runId":             "RunUUID",        // v1beta1 API
	"create_time":       "CreatedAtInSec", // v2beta1 API
	"start_time":        "StartedInSec",   // v2beta1 API
	"end_time":          "FinishedInSec",  // v2beta1 API
	"fingerprint":       "Fingerprint",
	"status":            "Status",         // v2beta1 API
	"status_metadata":   "StatusMetadata", // v2beta1 API
	"state_history":     "StateHistory",   // v2beta1 API
	"parent_task_id":    "ParentTaskUUID", // v2beta1 API
	"created_at":        "CreatedAtInSec", // v1beta1 API
	"finished_at":       "FinishedInSec",  // v1beta1 API
	"input_parameters":  "InputParameters",
	"output_parameters": "OutputParameters",
	"type":              "Type",
	"type_attrs":        "TypeAttrs",
}

func (t Task) GetField(name string) (string, bool) {
	if field, ok := taskAPIToModelFieldMap[name]; ok {
		return field, true
	}
	return "", false
}

func (t Task) GetFieldValue(name string) interface{} {
	switch name {
	case "UUID":
		return t.UUID
	case "Namespace":
		return t.Namespace
	case "PipelineName":
		return t.PipelineName
	case "RunUUID":
		return t.RunUUID
	case "CreatedAtInSec":
		return t.CreatedAtInSec
	case "StartedInSec":
		return t.StartedInSec
	case "FinishedInSec":
		return t.FinishedInSec
	case "Fingerprint":
		return t.Fingerprint
	case "ParentTaskUUID":
		return t.ParentTaskUUID
	case "Status":
		return t.Status
	case "StatusMetadata":
		return t.StatusMetadata
	case "StateHistory":
		return t.StateHistory
	case "Name":
		return t.Name
	case "DisplayName":
		return t.DisplayName
	case "InputParameters":
		return t.InputParameters
	case "OutputParameters":
		return t.OutputParameters
	case "Type":
		return t.Type
	case "TypeAttrs":
		return t.TypeAttrs
	default:
		return nil
	}
}
