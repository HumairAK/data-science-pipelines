// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");

package migration

import (
	"testing"
	"time"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestMigrationLeaseRejectsConcurrentWorkerAndIgnoresStaleFailure(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:migration_lease_test?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(model.AllModels()...))

	now := time.Unix(1_700_000_000, 0)
	owner, completed, err := Start(db, "test", now)
	require.NoError(t, err)
	require.False(t, completed)
	require.NotEmpty(t, owner)

	_, _, err = Start(db, "other", now.Add(time.Minute))
	require.Error(t, err)

	staleErr := assertError("stale worker")
	require.Equal(t, staleErr, Fail(db, "wrong-owner", staleErr))
	var marker model.RuntimeMetadataMigration
	require.NoError(t, db.First(&marker).Error)
	require.Equal(t, model.RuntimeMetadataMigrationRunning, marker.Status)
	require.Equal(t, owner, marker.LeaseToken)
	require.NoError(t, Fail(db, owner, assertError("recorded failure")))
	require.NoError(t, db.First(&marker).Error)
	require.Equal(t, model.RuntimeMetadataMigrationFailed, marker.Status)
}

func TestMigrationLedgerUsesNameAndVersionAsCompositeIdentity(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:migration_ledger_version_test?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(model.AllModels()...))

	require.NoError(t, db.Create(&model.RuntimeMetadataMigration{
		Name:    model.RuntimeMetadataMigrationName,
		Version: model.RuntimeMetadataMigrationVersion + 1,
		Status:  model.RuntimeMetadataMigrationNotStarted,
	}).Error)

	var count int64
	require.NoError(t, db.Model(&model.RuntimeMetadataMigration{}).
		Where("Name = ?", model.RuntimeMetadataMigrationName).Count(&count).Error)
	require.EqualValues(t, 1, count)
}

func assertError(message string) error { return &migrationTestError{message: message} }

type migrationTestError struct{ message string }

func (e *migrationTestError) Error() string { return e.message }
