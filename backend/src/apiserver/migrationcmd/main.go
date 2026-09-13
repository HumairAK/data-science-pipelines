// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

// Command mlmd-to-native migrates historical MLMD runtime metadata into the
// native KFP runtime tables. Run it before starting the upgraded API server.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/kubeflow/pipelines/backend/src/apiserver/migration"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	var driver, dsn, metadataAddress, toolVersion string
	var pageSize int
	var dryRun bool
	flag.StringVar(&driver, "db-driver", "mysql", "native database driver: mysql or pgx")
	flag.StringVar(&dsn, "db-dsn", "", "native database DSN")
	flag.StringVar(&metadataAddress, "metadata-address", "metadata-grpc-service:8080", "MLMD gRPC address")
	flag.StringVar(&toolVersion, "tool-version", "mlmd-to-native/1", "operator tool version recorded in the migration ledger")
	flag.IntVar(&pageSize, "page-size", 500, "MLMD page size")
	flag.BoolVar(&dryRun, "dry-run", false, "load and validate conversion counts without writing native rows")
	flag.Parse()
	if dsn == "" {
		fatal("--db-dsn is required")
	}

	db, err := openDatabase(driver, dsn)
	if err != nil {
		fatal("open native database: %v", err)
	}
	if err := db.AutoMigrate(model.AllModels()...); err != nil {
		fatal("create native schema: %v", err)
	}
	conn, err := grpc.Dial(metadataAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		fatal("connect to MLMD at %q: %v", metadataAddress, err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 24*time.Hour)
	defer cancel()
	counts, err := migration.Run(ctx, db, migration.NewGRPCSource(ml_metadata.NewMetadataStoreServiceClient(conn)), int32(pageSize), toolVersion, dryRun, func(format string, args ...any) {
		fmt.Printf(format+"\n", args...)
	})
	if err != nil {
		fatal("migration failed: %v", err)
	}
	fmt.Printf("migration result: %+v\n", counts)
}

func openDatabase(driver, dsn string) (*gorm.DB, error) {
	switch driver {
	case "mysql":
		return gorm.Open(mysql.Open(dsn), &gorm.Config{})
	case "pgx", "postgres":
		return gorm.Open(postgres.Open(dsn), &gorm.Config{})
	default:
		return nil, fmt.Errorf("unsupported database driver %q", driver)
	}
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
