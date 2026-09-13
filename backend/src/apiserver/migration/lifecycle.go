// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package migration

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Start claims the migration lease. The bool result is true when the durable
// marker is already COMPLETED; callers must treat that as a no-op rather than
// as ownership of the old lease token. Failed and expired RUNNING migrations
// may be resumed by a later invocation.
const migrationLeaseDuration = time.Hour

// Renew extends the lease only when the caller still owns the RUNNING marker.
// This prevents a paused or stale worker from extending a lease acquired by a
// newer migration invocation.
func Renew(db *gorm.DB, token string, now time.Time) error {
	if db == nil {
		return fmt.Errorf("migration lease renewal requires a database")
	}
	result := db.Model(&model.RuntimeMetadataMigration{}).
		Where(map[string]interface{}{
			"Name": model.RuntimeMetadataMigrationName, "Version": model.RuntimeMetadataMigrationVersion,
			"Status": model.RuntimeMetadataMigrationRunning, "LeaseToken": token,
		}).
		Updates(map[string]interface{}{"LeaseExpiresAtInSec": now.Add(migrationLeaseDuration).Unix()})
	if result.Error != nil {
		return fmt.Errorf("renew migration lease: %w", result.Error)
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("migration lease is no longer owned by this process")
	}
	return nil
}

func EnsureLease(db *gorm.DB, token string, now time.Time) error {
	var marker model.RuntimeMetadataMigration
	if err := db.Where(map[string]interface{}{
		"Name": model.RuntimeMetadataMigrationName, "Version": model.RuntimeMetadataMigrationVersion,
	}).First(&marker).Error; err != nil {
		return fmt.Errorf("read migration lease: %w", err)
	}
	if marker.Status != model.RuntimeMetadataMigrationRunning || marker.LeaseToken != token || marker.LeaseExpiresAtInSec <= now.Unix() {
		return fmt.Errorf("migration lease is no longer owned by this process")
	}
	return nil
}

func Start(db *gorm.DB, toolVersion string, now time.Time) (string, bool, error) {
	if db == nil {
		return "", false, fmt.Errorf("migration start requires a database")
	}
	token, err := newLeaseToken()
	if err != nil {
		return "", false, err
	}
	record := model.RuntimeMetadataMigration{
		Name:           model.RuntimeMetadataMigrationName,
		Version:        model.RuntimeMetadataMigrationVersion,
		Status:         model.RuntimeMetadataMigrationRunning,
		StartedAtInSec: now.Unix(),
		ToolVersion:    toolVersion,
	}
	alreadyCompleted := false
	if err := db.Transaction(func(tx *gorm.DB) error {
		var existing model.RuntimeMetadataMigration
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where(map[string]interface{}{"Name": record.Name, "Version": record.Version}).First(&existing).Error
		if err == nil {
			if existing.Status == model.RuntimeMetadataMigrationCompleted {
				alreadyCompleted = true
				return nil
			}
			if existing.Status == model.RuntimeMetadataMigrationRunning && existing.LeaseExpiresAtInSec > now.Unix() {
				return fmt.Errorf("MLMD-to-native migration is already running until %s", time.Unix(existing.LeaseExpiresAtInSec, 0).UTC().Format(time.RFC3339))
			}
			if existing.StartedAtInSec == 0 {
				existing.StartedAtInSec = now.Unix()
			}
			return tx.Model(&existing).Updates(map[string]interface{}{
				"Status":              model.RuntimeMetadataMigrationRunning,
				"StartedAtInSec":      existing.StartedAtInSec,
				"ToolVersion":         toolVersion,
				"ValidationError":     nil,
				"LeaseToken":          token,
				"LeaseExpiresAtInSec": now.Add(migrationLeaseDuration).Unix(),
			}).Error
		} else if err != gorm.ErrRecordNotFound {
			return fmt.Errorf("read migration ledger: %w", err)
		}
		record.LeaseToken = token
		record.LeaseExpiresAtInSec = now.Add(migrationLeaseDuration).Unix()
		return tx.Save(&record).Error
	}); err != nil {
		return "", false, err
	}
	if alreadyCompleted {
		return "", true, nil
	}
	var existing model.RuntimeMetadataMigration
	if err := db.Where(map[string]interface{}{"Name": model.RuntimeMetadataMigrationName, "Version": model.RuntimeMetadataMigrationVersion}).First(&existing).Error; err != nil {
		return "", false, err
	}
	return existing.LeaseToken, false, nil
}

// Fail records the error that stopped the operator job. It returns nil when
// the FAILED marker was durably written. If the lease is stale, the original
// error is returned and the newer owner is left untouched.
func Fail(db *gorm.DB, token string, err error) error {
	if db == nil {
		return fmt.Errorf("migration failure requires a database: %w", err)
	}
	update := map[string]interface{}{
		"Status":          model.RuntimeMetadataMigrationFailed,
		"ValidationError": err.Error(),
	}
	result := db.Model(&model.RuntimeMetadataMigration{}).
		Where(map[string]interface{}{"Name": model.RuntimeMetadataMigrationName, "Version": model.RuntimeMetadataMigrationVersion, "Status": model.RuntimeMetadataMigrationRunning, "LeaseToken": token}).
		Updates(update)
	if result.Error != nil {
		updateErr := result.Error
		return fmt.Errorf("record migration failure: %w (original error: %v)", updateErr, err)
	}
	if result.RowsAffected != 1 {
		return err
	}
	return nil
}

// Complete validates the migrated destination and writes the completion
// marker in the same transaction. The validator must use tx for all reads;
// this is what prevents a successful marker from being committed for a
// partially validated destination.
func Complete(db *gorm.DB, token string, sourceCounts, destinationCounts map[string]int64, toolVersion string, now time.Time, validate func(tx *gorm.DB) error) error {
	if db == nil {
		return fmt.Errorf("migration completion requires a database")
	}
	sourceJSON, err := json.Marshal(sourceCounts)
	if err != nil {
		return fmt.Errorf("encode source counts: %w", err)
	}
	destinationJSON, err := json.Marshal(destinationCounts)
	if err != nil {
		return fmt.Errorf("encode destination counts: %w", err)
	}
	if err := db.Transaction(func(tx *gorm.DB) error {
		var current model.RuntimeMetadataMigration
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where(map[string]interface{}{"Name": model.RuntimeMetadataMigrationName, "Version": model.RuntimeMetadataMigrationVersion}).First(&current).Error; err != nil {
			return err
		}
		if current.Status == model.RuntimeMetadataMigrationCompleted {
			return nil
		}
		if current.Status != model.RuntimeMetadataMigrationRunning || current.LeaseToken != token {
			return fmt.Errorf("migration lease is no longer owned by this process")
		}
		if validate != nil {
			if err := validate(tx); err != nil {
				return fmt.Errorf("validate migrated runtime metadata: %w", err)
			}
		}
		result := tx.Model(&model.RuntimeMetadataMigration{}).
			Where(map[string]interface{}{"Name": model.RuntimeMetadataMigrationName, "Version": model.RuntimeMetadataMigrationVersion}).
			Updates(map[string]interface{}{
				"Status":              model.RuntimeMetadataMigrationCompleted,
				"CompletedAtInSec":    now.Unix(),
				"SourceCounts":        model.LargeText(sourceJSON),
				"DestinationCounts":   model.LargeText(destinationJSON),
				"ValidationError":     nil,
				"ToolVersion":         toolVersion,
				"LeaseExpiresAtInSec": 0,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return fmt.Errorf("migration completion marker is missing; call Start before Complete")
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func newLeaseToken() (string, error) {
	bytes := make([]byte, 24)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate migration lease token: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}
