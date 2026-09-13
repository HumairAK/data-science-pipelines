// Copyright 2026 The Kubeflow Authors
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

const (
	RuntimeMetadataMigrationName    = "mlmd_to_native_runtime"
	RuntimeMetadataMigrationVersion = 1

	RuntimeMetadataMigrationNotStarted = "NOT_STARTED"
	RuntimeMetadataMigrationRunning    = "RUNNING"
	RuntimeMetadataMigrationFailed     = "FAILED"
	RuntimeMetadataMigrationCompleted  = "COMPLETED"
)

// RuntimeMetadataMigration records the lifecycle of the MLMD-to-native runtime
// metadata migration. A completion record is written only after the migration
// tool has validated all migrated data.
type RuntimeMetadataMigration struct {
	Name                string    `gorm:"column:Name; not null; primaryKey; type:varchar(64);"`
	Version             int       `gorm:"column:Version; not null; primaryKey;"`
	Status              string    `gorm:"column:Status; not null; type:varchar(32);"`
	StartedAtInSec      int64     `gorm:"column:StartedAtInSec; not null; default:0;"`
	CompletedAtInSec    int64     `gorm:"column:CompletedAtInSec; not null; default:0;"`
	SourceCounts        LargeText `gorm:"column:SourceCounts; default:null;"`
	DestinationCounts   LargeText `gorm:"column:DestinationCounts; default:null;"`
	ValidationError     LargeText `gorm:"column:ValidationError; default:null;"`
	ToolVersion         string    `gorm:"column:ToolVersion; default:null; type:varchar(64);"`
	LeaseToken          string    `gorm:"column:LeaseToken; default:null; type:varchar(64);"`
	LeaseExpiresAtInSec int64     `gorm:"column:LeaseExpiresAtInSec; not null; default:0;"`
}

func (RuntimeMetadataMigration) TableName() string {
	return "runtime_metadata_migrations"
}
