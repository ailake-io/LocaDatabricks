# LocalDatabricks Emulator

Lightweight local emulator for Databricks Control Plane/Data Plane APIs, written in Go. Lets developers test Terraform configs, `databricks-sdk-go`/Python SDK scripts, and Delta Lake pipelines without a real Databricks workspace.

## Stack

- **Language:** Go, HTTP routing via Fiber (`gofiber/fiber/v2`)
- **Metadata store:** embedded SQLite (or DuckDB) — `metadata.db`
- **File storage:** local directories (`./dbfs_root`, `./workspace_root`)
- **UI:** single embedded HTML file (Tailwind CDN + Alpine.js), bundled into the binary via `//go:embed ui/*`
- **Constraint:** must idle under ~100MB RAM; single-binary distribution

## Project layout

```
cmd/localdatabricks/main.go   # entrypoint, routes, embedded UI
cmd/localdatabricks/ui/       # dashboard (must stay next to main.go — go:embed scope)
internal/api/                 # clusters.go, jobs.go, dbfs.go, workspace.go, unity.go, router.go, errors.go
internal/engine/executor.go   # subprocess runner (job scripts, not real Spark)
internal/store/db.go          # SQLite metastore + in-memory cluster/job state
```

Full diagram: [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

## Running

```bash
export DATABRICKS_HOST=http://localhost:8080
export DATABRICKS_TOKEN=dapi-local-mock-token
go run cmd/localdatabricks/main.go
```

Serves API + UI on `:8080`.

## Rules

- Stream DBFS payloads (`io.Reader`/`io.Writer`) — never buffer large files into RAM.
- Response JSON must match `databricks-sdk-go` / Terraform provider field names exactly.
- Errors use Databricks' shape: `{"error_code": ..., "message": ...}`, not generic HTTP 500s.
- New UI assets go under `ui/` so `//go:embed ui/*` keeps picking them up.

## Docs

- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) — component diagram, project structure, storage model
- [docs/API.md](docs/API.md) — full emulated endpoint list (clusters, jobs, DBFS, workspace, Unity Catalog)
- [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md) — dev guidelines, local testing, reference code skeleton
