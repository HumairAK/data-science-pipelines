// Copyright 2025 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package storage

import (
	"database/sql"
	"database/sql/driver"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

const runMetricTableName = "run_metrics"

var runMetricColumns = []string{
	"TaskID",
	"Name",
	"NumberValue",
	"Namespace",
	"JsonValue",
	"CreatedAtInSec",
	"Type",
	"Schema",
}

type RunMetricStoreInterface interface {
	// CreateRunMetric Create a metric entry in the database.
	CreateRunMetric(metric *model.RunMetric) (*model.RunMetric, error)

	CreateRunMetrics(metrics []*model.RunMetric) ([]*model.RunMetric, error)

	// GetRunMetric Fetches a metric with a given task ID and name.
	GetRunMetric(taskID, name string) (*model.RunMetric, error)

	// ListRunMetrics Fetches metrics for given filtering and listing options.
	// filterContexts supports multiple filters: namespace is always supported,
	// plus either TaskID or RunUUID filtering
	ListRunMetrics(filterContexts []*model.FilterContext, opts *list.Options) ([]*model.RunMetric, int, string, error)
}

type RunMetricStore struct {
	db   *DB
	time util.TimeInterface
}

// NewRunMetricStore creates a new RunMetricStore.
func NewRunMetricStore(db *DB, time util.TimeInterface) *RunMetricStore {
	return &RunMetricStore{
		db:   db,
		time: time,
	}
}

func (s *RunMetricStore) CreateRunMetric(metric *model.RunMetric) (*model.RunMetric, error) {
	metrics, err := s.CreateRunMetrics([]*model.RunMetric{metric})
	if err != nil {
		return nil, err
	}
	if len(metrics) == 0 {
		return nil, util.NewInternalServerError(nil, "Failed to create a metric")
	}
	return metrics[0], nil
}

func (s *RunMetricStore) scanRows(rows *sql.Rows) ([]*model.RunMetric, error) {
	var metrics []*model.RunMetric
	for rows.Next() {
		var taskID, name, namespace, metricType, schema string
		var numberValue sql.NullFloat64
		var createdAtInSec int64
		var jsonValueBytes []byte

		err := rows.Scan(
			&taskID,
			&name,
			&numberValue,
			&namespace,
			&jsonValueBytes,
			&createdAtInSec,
			&metricType,
			&schema,
		)
		if err != nil {
			return metrics, err
		}

		// Parse JSON value if present
		var jsonValue model.JSONData
		if jsonValueBytes != nil {
			err = jsonValue.Scan(jsonValueBytes)
			if err != nil {
				return metrics, util.NewInternalServerError(err, "Failed to parse metric JSON value")
			}
		}

		// Convert string types back to enum types
		var typeEnum model.MetricType
		switch metricType {
		case "0":
			typeEnum = model.MetricTypeInput
		case "1":
			typeEnum = model.MetricTypeOutput
		default:
			typeEnum = model.MetricTypeInput
		}

		var numberPtr *float64
		if numberValue.Valid {
			numberPtr = &numberValue.Float64
		}

		metric := &model.RunMetric{
			TaskID:         taskID,
			Name:           name,
			NumberValue:    numberPtr,
			Namespace:      namespace,
			JsonValue:      jsonValue,
			CreatedAtInSec: createdAtInSec,
			Type:           typeEnum,
			Schema:         model.MetricSchema(schema),
		}
		metrics = append(metrics, metric)
	}
	return metrics, nil
}

// applyFilterContextsToQuery applies multiple filter contexts to the query builder
func (s *RunMetricStore) applyFilterContextsToQuery(sqlBuilder sq.SelectBuilder, filterContexts []*model.FilterContext) sq.SelectBuilder {
	var hasRunFilter bool

	for _, filterContext := range filterContexts {
		if filterContext == nil || filterContext.ReferenceKey == nil {
			continue
		}

		switch filterContext.ReferenceKey.Type {
		case model.NamespaceResourceType:
			sqlBuilder = sqlBuilder.Where(sq.Eq{"run_metrics.Namespace": filterContext.ReferenceKey.ID})
		case model.TaskResourceType:
			sqlBuilder = sqlBuilder.Where(sq.Eq{"run_metrics.TaskID": filterContext.ReferenceKey.ID})
		case model.RunResourceType:
			// Need to join with tasks table to filter by run
			if !hasRunFilter {
				sqlBuilder = sqlBuilder.Join("tasks ON run_metrics.TaskID = tasks.UUID")
				hasRunFilter = true
			}
			sqlBuilder = sqlBuilder.Where(sq.Eq{"tasks.RunUUID": filterContext.ReferenceKey.ID})
		}
	}

	return sqlBuilder
}

func (s *RunMetricStore) ListRunMetrics(filterContexts []*model.FilterContext, opts *list.Options) ([]*model.RunMetric, int, string, error) {
	errorF := func(err error) ([]*model.RunMetric, int, string, error) {
		return nil, 0, "", util.NewInternalServerError(err, "Failed to list metrics: %v", err)
	}

	// SQL for getting the filtered and paginated rows
	sqlBuilder := sq.Select(runMetricColumns...).From(runMetricTableName)
	sqlBuilder = s.applyFilterContextsToQuery(sqlBuilder, filterContexts)
	sqlBuilder = opts.AddFilterToSelect(sqlBuilder)

	rowsSql, rowsArgs, err := opts.AddPaginationToSelect(sqlBuilder).ToSql()
	if err != nil {
		return errorF(err)
	}

	// SQL for getting total size
	countBuilder := sq.Select("count(*)").From(runMetricTableName)
	countBuilder = s.applyFilterContextsToQuery(countBuilder, filterContexts)
	sizeSql, sizeArgs, err := opts.AddFilterToSelect(countBuilder).ToSql()
	if err != nil {
		return errorF(err)
	}

	// Use a transaction to make sure we're returning the total_size of the same rows queried
	tx, err := s.db.Begin()
	if err != nil {
		glog.Errorf("Failed to start transaction to list metrics")
		return errorF(err)
	}

	rows, err := tx.Query(rowsSql, rowsArgs...)
	if err != nil {
		tx.Rollback()
		return errorF(err)
	}
	if err := rows.Err(); err != nil {
		tx.Rollback()
		return errorF(err)
	}
	metrics, err := s.scanRows(rows)
	if err != nil {
		tx.Rollback()
		return errorF(err)
	}
	defer rows.Close()

	sizeRow, err := tx.Query(sizeSql, sizeArgs...)
	if err != nil {
		tx.Rollback()
		return errorF(err)
	}
	if err := sizeRow.Err(); err != nil {
		tx.Rollback()
		return errorF(err)
	}
	total_size, err := list.ScanRowToTotalSize(sizeRow)
	if err != nil {
		tx.Rollback()
		return errorF(err)
	}
	defer sizeRow.Close()

	err = tx.Commit()
	if err != nil {
		glog.Errorf("Failed to commit transaction to list metrics")
		return errorF(err)
	}

	if len(metrics) <= opts.PageSize {
		return metrics, total_size, "", nil
	}

	npt, err := opts.NextPageToken(metrics[opts.PageSize])
	return metrics[:opts.PageSize], total_size, npt, err
}

func (s *RunMetricStore) CreateRunMetrics(metrics []*model.RunMetric) ([]*model.RunMetric, error) {
	if len(metrics) == 0 {
		return nil, nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to start transaction to create metrics")
	}

	createdMetrics := make([]*model.RunMetric, 0, len(metrics))
	for _, metric := range metrics {
		newMetric := *metric
		if newMetric.CreatedAtInSec == 0 {
			newMetric.CreatedAtInSec = s.time.Now().Unix()
		}

		var jsonValue driver.Value
		var err error
		if newMetric.JsonValue != nil {
			jsonValue, err = newMetric.JsonValue.Value()
			if err != nil {
				tx.Rollback()
				return nil, util.NewInternalServerError(err, "Failed to marshal metric JSON value")
			}
		}

		sql, args, err := sq.
			Insert(runMetricTableName).
			SetMap(sq.Eq{
				"TaskID":         newMetric.TaskID,
				"Name":           newMetric.Name,
				"NumberValue":    newMetric.NumberValue,
				"Namespace":      newMetric.Namespace,
				"JsonValue":      jsonValue,
				"CreatedAtInSec": newMetric.CreatedAtInSec,
				"Type":           newMetric.Type,
				"Schema":         newMetric.Schema,
			}).
			ToSql()
		if err != nil {
			tx.Rollback()
			return nil, util.NewInternalServerError(err, "Failed to create query to insert metric")
		}

		_, err = tx.Exec(sql, args...)
		if err != nil {
			tx.Rollback()
			return nil, util.NewInternalServerError(err, "Failed to insert metric")
		}

		createdMetrics = append(createdMetrics, &newMetric)
	}

	err = tx.Commit()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to commit transaction")
	}

	return createdMetrics, nil
}

func (s *RunMetricStore) GetRunMetric(taskID, name string) (*model.RunMetric, error) {
	sql, args, err := sq.
		Select(runMetricColumns...).
		From(runMetricTableName).
		Where(sq.Eq{"TaskID": taskID, "Name": name}).
		Limit(1).ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create query to get metric: %v", err.Error())
	}

	r, err := s.db.Query(sql, args...)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to get metric: %v", err.Error())
	}
	defer r.Close()

	metrics, err := s.scanRows(r)
	if err != nil || len(metrics) > 1 {
		return nil, util.NewInternalServerError(err, "Failed to get metric: %v", err.Error())
	}
	if len(metrics) == 0 {
		return nil, util.NewResourceNotFoundError("metric", fmt.Sprintf("%s/%s", taskID, name))
	}

	return metrics[0], nil
}
