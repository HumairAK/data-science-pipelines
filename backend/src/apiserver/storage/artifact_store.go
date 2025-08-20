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
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

const artifactTableName = "artifacts"

var artifactColumns = []string{
	"UUID",
	"Namespace",
	"Type",
	"Uri",
	"Name",
	"CreatedAtInSec",
	"LastUpdateInSec",
	"Metadata",
}

type ArtifactStoreInterface interface {
	// Create an artifact entry in the database.
	CreateArtifact(artifact *model.Artifact) (*model.Artifact, error)

	// Update an existing artifact in the database.
	UpdateArtifact(artifact *model.Artifact) (*model.Artifact, error)

	// Fetches an artifact with a given id.
	GetArtifact(id string) (*model.Artifact, error)

	// Fetches artifacts for given filtering and listing options.
	ListArtifacts(filterContext *model.FilterContext, opts *list.Options) ([]*model.Artifact, int, string, error)
}

type ArtifactStore struct {
	db   *DB
	time util.TimeInterface
	uuid util.UUIDGeneratorInterface
}

// NewArtifactStore creates a new ArtifactStore.
func NewArtifactStore(db *DB, time util.TimeInterface, uuid util.UUIDGeneratorInterface) *ArtifactStore {
	return &ArtifactStore{
		db:   db,
		time: time,
		uuid: uuid,
	}
}

func (s *ArtifactStore) CreateArtifact(artifact *model.Artifact) (*model.Artifact, error) {
	// Set up UUID for artifact.
	newArtifact := *artifact
	id, err := s.uuid.NewRandom()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create an artifact id")
	}
	newArtifact.UUID = id.String()

	// Set creation timestamps
	now := s.time.Now().Unix()
	newArtifact.CreatedAtInSec = now
	newArtifact.LastUpdateInSec = now

	// Convert metadata to JSON string for storage
	metadataJSON, err := newArtifact.Metadata.Value()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to marshal artifact metadata")
	}

	sql, args, err := sq.
		Insert(artifactTableName).
		SetMap(
			sq.Eq{
				"UUID":            newArtifact.UUID,
				"Namespace":       newArtifact.Namespace,
				"Type":            newArtifact.Type,
				"Uri":             newArtifact.Uri,
				"Name":            newArtifact.Name,
				"CreatedAtInSec":  newArtifact.CreatedAtInSec,
				"LastUpdateInSec": newArtifact.LastUpdateInSec,
				"Metadata":        metadataJSON,
			},
		).
		ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create query to insert artifact to artifact table: %v",
			err.Error())
	}

	_, err = s.db.Exec(sql, args...)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to add artifact to artifact table: %v",
			err.Error())
	}

	return &newArtifact, nil
}

func (s *ArtifactStore) UpdateArtifact(artifact *model.Artifact) (*model.Artifact, error) {
	if artifact.UUID == "" {
		return nil, util.NewInvalidInputError("Artifact UUID is required for update")
	}

	// Update the last update timestamp
	updatedArtifact := *artifact
	updatedArtifact.LastUpdateInSec = s.time.Now().Unix()

	// Convert metadata to JSON string for storage
	metadataJSON, err := updatedArtifact.Metadata.Value()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to marshal artifact metadata")
	}

	sql, args, err := sq.
		Update(artifactTableName).
		SetMap(sq.Eq{
			"Namespace":       updatedArtifact.Namespace,
			"Type":            updatedArtifact.Type,
			"Uri":             updatedArtifact.Uri,
			"Name":            updatedArtifact.Name,
			"LastUpdateInSec": updatedArtifact.LastUpdateInSec,
			"Metadata":        metadataJSON,
		}).
		Where(sq.Eq{"UUID": artifact.UUID}).
		ToSql()

	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create query to update artifact: %v", err.Error())
	}

	result, err := s.db.Exec(sql, args...)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to update artifact: %v", err.Error())
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to get rows affected by update: %v", err.Error())
	}

	if rowsAffected == 0 {
		return nil, util.NewResourceNotFoundError("artifact", artifact.UUID)
	}

	return &updatedArtifact, nil
}

func (s *ArtifactStore) scanRows(rows *sql.Rows) ([]*model.Artifact, error) {
	var artifacts []*model.Artifact
	for rows.Next() {
		var uuid, namespace, uri string
		var name sql.NullString
		var artifactType int32
		var createdAtInSec, lastUpdateInSec int64
		var metadataBytes []byte

		err := rows.Scan(
			&uuid,
			&namespace,
			&artifactType,
			&uri,
			&name,
			&createdAtInSec,
			&lastUpdateInSec,
			&metadataBytes,
		)
		if err != nil {
			return artifacts, err
		}

		// Parse metadata JSON
		var metadata model.JSONData
		if metadataBytes != nil {
			err = metadata.Scan(metadataBytes)
			if err != nil {
				return artifacts, util.NewInternalServerError(err, "Failed to parse artifact metadata")
			}
		}

		artifact := &model.Artifact{
			UUID:            uuid,
			Namespace:       namespace,
			Type:            artifactType,
			Uri:             uri,
			Name:            name.String,
			CreatedAtInSec:  createdAtInSec,
			LastUpdateInSec: lastUpdateInSec,
			Metadata:        metadata,
		}
		artifacts = append(artifacts, artifact)
	}
	return artifacts, nil
}

func (s *ArtifactStore) ListArtifacts(filterContext *model.FilterContext, opts *list.Options) ([]*model.Artifact, int, string, error) {
	errorF := func(err error) ([]*model.Artifact, int, string, error) {
		return nil, 0, "", util.NewInternalServerError(err, "Failed to list artifacts: %v", err)
	}

	// SQL for getting the filtered and paginated rows
	sqlBuilder := sq.Select(artifactColumns...).From(artifactTableName)

	// Apply namespace filtering if provided
	if filterContext != nil && filterContext.ReferenceKey != nil {
		switch filterContext.ReferenceKey.Type {
		case model.NamespaceResourceType:
			sqlBuilder = sqlBuilder.Where(sq.Eq{"Namespace": filterContext.ReferenceKey.ID})
		}
	}

	sqlBuilder = opts.AddFilterToSelect(sqlBuilder)

	rowsSql, rowsArgs, err := opts.AddPaginationToSelect(sqlBuilder).ToSql()
	if err != nil {
		return errorF(err)
	}

	// SQL for getting total size
	countBuilder := sq.Select("count(*)").From(artifactTableName)
	if filterContext != nil && filterContext.ReferenceKey != nil {
		switch filterContext.ReferenceKey.Type {
		case model.NamespaceResourceType:
			countBuilder = countBuilder.Where(sq.Eq{"Namespace": filterContext.ReferenceKey.ID})
		}
	}
	sizeSql, sizeArgs, err := opts.AddFilterToSelect(countBuilder).ToSql()
	if err != nil {
		return errorF(err)
	}

	// Use a transaction to make sure we're returning the total_size of the same rows queried
	tx, err := s.db.Begin()
	if err != nil {
		glog.Errorf("Failed to start transaction to list artifacts")
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
	artifacts, err := s.scanRows(rows)
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
		glog.Errorf("Failed to commit transaction to list artifacts")
		return errorF(err)
	}

	if len(artifacts) <= opts.PageSize {
		return artifacts, total_size, "", nil
	}

	npt, err := opts.NextPageToken(artifacts[opts.PageSize])
	return artifacts[:opts.PageSize], total_size, npt, err
}

func (s *ArtifactStore) GetArtifact(id string) (*model.Artifact, error) {
	sql, args, err := sq.
		Select(artifactColumns...).
		From(artifactTableName).
		Where(sq.Eq{"UUID": id}).
		Limit(1).ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create query to get artifact: %v", err.Error())
	}

	r, err := s.db.Query(sql, args...)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to get artifact: %v", err.Error())
	}
	defer r.Close()

	artifacts, err := s.scanRows(r)
	if err != nil || len(artifacts) > 1 {
		return nil, util.NewInternalServerError(err, "Failed to get artifact: %v", err.Error())
	}
	if len(artifacts) == 0 {
		return nil, util.NewResourceNotFoundError("artifact", fmt.Sprint(id))
	}

	return artifacts[0], nil
}
