# MLMD-to-native runtime migration

Upgrades from an MLMD-backed KFP release must run the migration before the API
server is started. Take and verify a database backup, keep MLMD available, and
run the command with administrative database credentials.

From the repository root:

```bash
go run ./backend/src/apiserver/migrationcmd \
  --db-driver=mysql \
  --db-dsn='user:password@tcp(host:3306)/mlpipeline?parseTime=true' \
  --metadata-address=metadata-grpc-service:8080
```

For PostgreSQL use `--db-driver=pgx` and a PostgreSQL DSN. A dry run loads MLMD
and reports the conversion inventory without writing native rows:

```bash
go run ./backend/src/apiserver/migrationcmd --dry-run \
  --db-driver=pgx --db-dsn="$KFP_DATABASE_DSN"
```

The command is safe to resume after interruption. It records `RUNNING`,
`FAILED`, or `COMPLETED` in `runtime_metadata_migrations`; deterministic source
identities, conflict-safe inserts, and an expiring migration lease prevent
duplicate native rows or concurrent operators from overwriting one another. The
completion marker is written only after the destination references validate.
Skipped or unsupported source records fail the migration instead of being
silently accepted; resolve them and rerun.

Do not start the upgraded API server until the command reports completion. If
the command fails, inspect its error and the migration ledger, restore the
verified backup if required, then rerun the command. The API server startup gate
intentionally remains closed for failed or incomplete migrations.

## Backup, rollback, and operator procedure

Before the first write, take a transactionally consistent backup of the native
database and verify that it can be restored in a disposable database. Keep the
MLMD service and the previous API-server deployment available until the new
deployment has passed verification. Run the command with the same database
credentials and namespace scope used by the API server; do not run it from an
API-server startup hook.

Quiesce pipeline writes and MLMD mutations while the migration runs. The
current implementation is restart-safe and idempotent, but its source reader
materializes the MLMD inventory and does not provide transaction-level MLMD
snapshot isolation.

The command reports `skipped` and `unsupported` source records in dry-run and
completion progress. Treat non-zero counts as an operator review item. A
successful completion marker is the handoff point: only then should the
upgraded API server be rolled out.

If validation fails before completion, leave the API server stopped, inspect
the ledger and report, and rerun the command after correcting the source or
destination issue. If native rows have been written and the migration cannot
be completed, stop the upgraded API server, restore the verified native
database backup, keep MLMD available, and restart the previous deployment.
Do not delete individual migrated rows or manually mark the ledger completed.
After rollback, take a fresh backup before retrying.

The repository's upgrade workflow is intentionally opt-in through the
`KFP_ENABLE_MLMD_UPGRADE_TESTS` repository variable. Enabling it runs the
production-like migration fixture in CI before the disposable Kubernetes
upgrade test; the deployed-cluster fixture remains operator-controlled because
it mutates a real database and restarts the API server.

## Real Kubernetes/MLMD fixture

The repository includes a hermetic production-like fixture test and an opt-in
test for a deployed cluster. The real test requires a preloaded MLMD/database
fixture containing two namespaces, loop and condition executions, an exit
handler, metrics, cacheable and ineligible executions, and historical
unnamespaced artifact URIs. It is deliberately disabled by default because it
writes the database and restarts the API-server deployment.

Run it from `backend` only against a disposable upgrade environment:

```powershell
$env:KFP_RUN_MLMD_UPGRADE_E2E = "true"
$env:KFP_MLMD_UPGRADE_DB_DRIVER = "pgx"
$env:KFP_MLMD_UPGRADE_DB_DSN = $env:KFP_DATABASE_DSN
$env:KFP_MLMD_UPGRADE_METADATA_ADDRESS = "metadata-grpc-service:8080"
$env:KFP_MLMD_UPGRADE_API_DEPLOYMENT = "ml-pipeline"
$env:KFP_MLMD_UPGRADE_K8S_NAMESPACE = "kubeflow"
$env:KFP_MLMD_UPGRADE_HEALTH_URL = "http://localhost:8888/apis/v2beta1/healthz"
go test ./src/apiserver/migration -run TestRealKubernetesMLMDUpgrade -count=1
```

The test runs the same migration orchestration as the operator command, waits
for the API-server rollout, and verifies the health endpoint after migration.
