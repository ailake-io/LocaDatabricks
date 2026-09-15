# Architecture

## Component mapping

```
                       +---------------------------------------+
                       |   Databricks CLI / SDK / Terraform    |
                       +---------------------------------------+
                                           |
                                    HTTP / REST API
                                           v
                       +---------------------------------------+
                       |    LocalDatabricks Go Server (8080)   |
                       |  (Fiber API + Embedded Web UI (HTML)) |
                       +---------------------------------------+
                          |                 |                |
             +------------+           ------+-------+        +------------+
             |                                      |                     |
             v                                      v                     v
   +--------------------+                 +--------------------+   +-------------------+
   |  Workspace & Jobs  |                 |  Unity Catalog     |   |   DBFS / Storage  |
   |  (In-memory State) |                 |  (SQLite / DuckDB) |   |  (Local Directory)|
   +--------------------+                 +--------------------+   +-------------------+
             |
             v
   +-----------------------------------------------------------+
   |             Spark Connect / Local Subprocess              |
   |              (Executes PySpark / Delta-RS)                |
   +-----------------------------------------------------------+
```

## Project structure

```
local-databricks/
├── cmd/
│   └── localdatabricks/
│       ├── main.go          # Entrypoint, CLI flag parsing, API routes & embedded UI
│       └── ui/
│           └── index.html   # Dashboard (plain HTML/CSS/JS, no CDN deps) — must live
│                             # next to main.go: go:embed only sees its own subtree
├── internal/
│   ├── api/
│   │   ├── router.go        # Fiber routing setup
│   │   ├── clusters.go      # Cluster handlers
│   │   ├── jobs.go          # Job execution handlers
│   │   ├── dbfs.go          # File system endpoints
│   │   └── unity.go         # Unity catalog endpoints
│   ├── engine/
│   │   └── executor.go      # Subprocess runner for PySpark / Spark Connect
│   └── store/
│       └── db.go            # SQLite / in-memory state management
├── go.mod
└── go.sum
```

## Storage model

| Concern | Backing |
|---|---|
| Workspace & job state | in-memory |
| Unity Catalog / metastore | embedded SQLite (`metadata.db`) — pure Go, no cgo |
| SQL statement execution | embedded DuckDB (`warehouse.duckdb`) — cgo, real query engine |
| DBFS | local directory (`./dbfs_root`) |
| Workspace files/notebooks | local directory (`./workspace_root`) |

The metastore and the warehouse are separate databases today (see docs/API.md's "Known gap" note) — UC's table registry doesn't yet drive what's queryable in DuckDB.

See [API.md](API.md) for the endpoint surface and [DEVELOPMENT.md](DEVELOPMENT.md) for implementation rules and the reference code skeleton.
