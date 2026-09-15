# Development guidelines

## Rules for code generation

- **Low memory footprint** — avoid loading large payloads into RAM; stream files for DBFS endpoints using Go's `io.Reader`/`io.Writer` (see `internal/api/dbfs.go`).
- **Standard SDK compatibility** — response JSON keys must match what `databricks-sdk-go` and the Databricks Terraform provider expect.
- **Error handling** — return standard Databricks API error format (`error_code`, `message`) via `internal/api/errors.go:dbxError`, never a generic HTTP 500 page.
- **Path safety** — any endpoint that maps a request path to disk (DBFS, workspace, job scripts) must confine it under its root via `resolveRootedPath` (`internal/api/pathsafe.go`) before touching the filesystem or `exec.Command`. It checks both lexically (`..` traversal) and by resolving symlinks on the nearest existing ancestor — a lexical-only check can be bypassed by a symlink planted under the root that points outside it.
- **Auth** — every `/api/*` route sits behind `RequireToken` (`internal/api/auth.go`), a bearer-token check comparable to real Databricks' `DATABRICKS_TOKEN`. `jobs/run-now` executes real subprocesses, so an unauthenticated instance reachable by others is an RCE surface, not just a mock. The server also binds to `127.0.0.1` by default for the same reason — only pass `--bind` to something else deliberately.
- **Single-binary integrity** — UI assets live at `cmd/localdatabricks/ui/`, next to `main.go` — `go:embed` only sees files in its own source file's subtree, so the UI can't live at the repo root.
- **No external CDN in the UI** — the dashboard is plain HTML/CSS/JS with zero third-party script tags, so it works fully offline and carries no supply-chain surface.

## Build requirements

The SQL warehouse uses `go-duckdb` (cgo) — building requires a C compiler (`gcc`/`clang`) on `PATH`. Everything else in the stack is pure Go. CI (`.github/workflows/ci.yml`) runs on `ubuntu-latest`, which has one preinstalled.

## Local testing

```bash
go run ./cmd/localdatabricks --port 8080 --data-dir .
```

Or against a built binary:

```bash
go build -o bin/localdatabricks ./cmd/localdatabricks
./bin/localdatabricks --port 8080
```

Startup prints a generated bearer token (skip that by passing `--token` or setting `LOCALDATABRICKS_TOKEN`). Point the real Databricks CLI/SDK/Terraform provider at it:

```bash
export DATABRICKS_HOST=http://localhost:8080
export DATABRICKS_TOKEN=<token printed on startup>
```

Open http://localhost:8080 for the embedded web management console.

## Where things live

| Concern | File |
|---|---|
| Route wiring | `internal/api/router.go` |
| Clusters API | `internal/api/clusters.go` |
| Jobs/Runs API | `internal/api/jobs.go` |
| DBFS API | `internal/api/dbfs.go` |
| Workspace API | `internal/api/workspace.go` |
| Unity Catalog registry | `internal/api/unity.go` |
| SQLite metastore + in-memory state | `internal/store/db.go` |
| DuckDB-backed SQL statement execution | `internal/store/warehouse.go` |
| SQL Statement Execution API | `internal/api/sql.go` |
| Job script execution (subprocess, not Spark) | `internal/engine/executor.go` |
| Embedded UI | `cmd/localdatabricks/ui/index.html` |

Endpoint-by-endpoint spec: [API.md](API.md). Emulation scope and the governance layering plan (ACL, RLS, masking, lineage, audit) are also in API.md — implement in that order when extending Unity Catalog beyond the current plain registry.
