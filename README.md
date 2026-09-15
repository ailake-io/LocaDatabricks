# LocalDatabricks

Lightweight local emulator for core Databricks Control Plane APIs, written in Go. Point the real Databricks CLI, `databricks-sdk-go`/Python SDK, or Terraform provider at it and test against a local mock instead of a real workspace — no cluster costs, no network dependency, idle footprint targeted under ~100MB.

## Quick start

```bash
go run ./cmd/localdatabricks --port 8080
```

Binds to `127.0.0.1` by default and prints a generated bearer token on startup:

```
LocalDatabricks emulator listening on http://127.0.0.1:8080
Bearer token: <random hex> (send as 'Authorization: Bearer <random hex>')
```

Point a client at it:

```bash
export DATABRICKS_HOST=http://localhost:8080
export DATABRICKS_TOKEN=<token printed on startup>
```

Open http://localhost:8080 for the embedded dashboard.

## What's emulated

Clusters (fake state machine), Jobs/Runs (real local subprocess execution), DBFS, Workspace, a Unity Catalog registry (SQLite-backed catalogs/schemas/tables), and SQL statement execution against a real embedded DuckDB warehouse:

```bash
curl -s http://localhost:8080/api/2.0/sql/statements \
  -H "Authorization: Bearer $DATABRICKS_TOKEN" \
  -d '{"statement": "SELECT 1 + 1 AS answer"}'
```

Full endpoint list and the roadmap for adding UC governance (ACL, row-level security, column masking, lineage, audit) are in [docs/API.md](docs/API.md).

## Docs

- [CLAUDE.md](CLAUDE.md) — quick orientation for working in this repo
- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) — component diagram, project layout, storage model
- [docs/API.md](docs/API.md) — emulated endpoints, emulation-scope-by-product, UC governance plan
- [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md) — dev guidelines, local testing, where things live

## Building

Requires a C compiler on `PATH` (the SQL warehouse uses DuckDB via cgo):

```bash
go build -o bin/localdatabricks ./cmd/localdatabricks
```

## Testing

```bash
go vet ./...
go test ./...
```

## Security notes

- Every `/api/*` route requires the bearer token printed at startup (or set via `--token` / `LOCALDATABRICKS_TOKEN`).
- Binds to loopback by default. `jobs/run-now` executes real local subprocesses — only pass `--bind` to expose it beyond localhost if you've reviewed who else can reach that address.
- Paths from clients (DBFS, workspace, job scripts) are confined to their root directory, including against symlink-based escapes — see `internal/api/pathsafe.go`.
