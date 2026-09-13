// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package migration

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	mlmd "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// TestRealKubernetesMLMDUpgrade is an opt-in upgrade test for a deployed KFP
// environment. The fixture cluster must already contain production-like MLMD
// data: loop/condition/exit-handler executions, metrics, cacheable and
// ineligible executions, historical unnamespaced artifacts, and two namespaces.
// It is skipped by default because it mutates a real database and restarts the
// configured API-server deployment.
func TestRealKubernetesMLMDUpgrade(t *testing.T) {
	if os.Getenv("KFP_RUN_MLMD_UPGRADE_E2E") != "true" {
		t.Skip("set KFP_RUN_MLMD_UPGRADE_E2E=true to run against a real Kubernetes/MLMD fixture")
	}
	dsn := requireEnv(t, "KFP_MLMD_UPGRADE_DB_DSN")
	driver := requireEnv(t, "KFP_MLMD_UPGRADE_DB_DRIVER")
	metadataAddress := requireEnv(t, "KFP_MLMD_UPGRADE_METADATA_ADDRESS")
	healthURL := requireEnv(t, "KFP_MLMD_UPGRADE_HEALTH_URL")
	deployment := requireEnv(t, "KFP_MLMD_UPGRADE_API_DEPLOYMENT")
	namespace := os.Getenv("KFP_MLMD_UPGRADE_K8S_NAMESPACE")
	if namespace == "" {
		namespace = "kubeflow"
	}

	db, err := openUpgradeDatabase(driver, dsn)
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(model.AllModels()...))
	conn, err := grpc.Dial(metadataAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	counts, err := Run(ctx, db, NewGRPCSource(mlmd.NewMetadataStoreServiceClient(conn)), 500, "real-e2e/1", false, func(format string, args ...any) {
		t.Logf(format, args...)
	})
	require.NoError(t, err)
	require.NotZero(t, counts.Tasks)
	require.NotZero(t, counts.Artifacts)

	cmd := exec.CommandContext(ctx, "kubectl", "rollout", "restart", "deployment/"+deployment, "-n", namespace)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "restart API server: %s", output)
	cmd = exec.CommandContext(ctx, "kubectl", "rollout", "status", "deployment/"+deployment, "-n", namespace, "--timeout=10m")
	output, err = cmd.CombinedOutput()
	require.NoError(t, err, "wait for API server rollout: %s", output)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
	require.NoError(t, err)
	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	defer response.Body.Close()
	require.Less(t, response.StatusCode, http.StatusInternalServerError)
}

func requireEnv(t *testing.T, name string) string {
	t.Helper()
	value := os.Getenv(name)
	if value == "" {
		t.Fatalf("%s is required for the real MLMD upgrade fixture", name)
	}
	return value
}

func openUpgradeDatabase(driver, dsn string) (*gorm.DB, error) {
	switch driver {
	case "mysql":
		return gorm.Open(mysql.Open(dsn), &gorm.Config{})
	case "pgx", "postgres":
		return gorm.Open(postgres.Open(dsn), &gorm.Config{})
	default:
		return nil, fmt.Errorf("unsupported upgrade test database driver %q", driver)
	}
}
