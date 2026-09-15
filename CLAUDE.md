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
go run ./cmd/localdatabricks --port 8080
```

Binds to `127.0.0.1` by default and prints a generated bearer token on startup (or pass `--token`/`LOCALDATABRICKS_TOKEN`). Point clients at it:

```bash
export DATABRICKS_HOST=http://localhost:8080
export DATABRICKS_TOKEN=<token printed on startup>
```

## Rules

- Stream DBFS payloads (`io.Reader`/`io.Writer`) — never buffer large files into RAM.
- Response JSON must match `databricks-sdk-go` / Terraform provider field names exactly.
- Errors use Databricks' shape: `{"error_code": ..., "message": ...}`, not generic HTTP 500s.
- New UI assets go under `cmd/localdatabricks/ui/` so `go:embed` keeps picking them up.
- Every `/api/*` route sits behind `api.RequireToken` — don't add a route outside the `/api` group unless it's genuinely meant to be public.
- Any handler that maps a request path to disk (DBFS, workspace, job scripts) must go through `resolveRootedPath` — it's symlink-aware, not just a lexical `..` check.
- `--bind` defaults to loopback on purpose: `jobs/run-now` executes real subprocesses, so exposing the port beyond localhost without also reviewing auth is a real RCE surface.

## Docs

- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) — component diagram, project structure, storage model
- [docs/API.md](docs/API.md) — full emulated endpoint list (clusters, jobs, DBFS, workspace, Unity Catalog)
- [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md) — dev guidelines, local testing, reference code skeleton
