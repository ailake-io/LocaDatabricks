# Emulated API surface

Targets the Databricks REST API 2.0/2.1 endpoints needed by the Databricks CLI, SDK, and Terraform provider.

## A. Authentication & Workspace (`/api/2.0`)

- `GET /api/2.0/workspace/list` — lists notebooks/folders from local root (`./workspace_root`)
- `POST /api/2.0/workspace/mkdirs` — creates directories locally
- `POST /api/2.0/workspace/import` — writes a notebook file locally

## B. Clusters API (`/api/2.0/clusters`)

- `POST /api/2.0/clusters/create` — registers a virtual cluster profile in-memory, returns mock cluster ID (`local-cluster-xxx`)
- `GET /api/2.0/clusters/get` — returns running status for the mock cluster
- `POST /api/2.0/clusters/delete` — terminates the mock cluster state

## C. Jobs & Runs API (`/api/2.0/jobs`)

- `POST /api/2.0/jobs/create` — defines a scheduled task
- `POST /api/2.0/jobs/run-now` — triggers local execution of a script/Python file as a background goroutine subprocess
- `GET /api/2.0/jobs/runs/get` — checks status (`SUCCESS`, `RUNNING`, `FAILED`)

## D. DBFS API (`/api/2.0/dbfs`)

- `GET /api/2.0/dbfs/list` — maps to local directory `./dbfs_root`
- `POST /api/2.0/dbfs/put` — streams file bytes into `./dbfs_root`
- `GET /api/2.0/dbfs/read` — reads file contents from `./dbfs_root`

## E. Unity Catalog & Metastore (`/api/2.1/unity-catalog`)

- Manages catalogs, schemas, and tables using an embedded SQLite database (`metadata.db`)
- Supports registering Delta Lake paths associated with table names

## F. SQL Statement Execution (`/api/2.0/sql/statements`)

- `POST /api/2.0/sql/statements` — executes a SQL statement synchronously against the embedded DuckDB warehouse (`internal/store/warehouse.go`) and returns the result immediately, cached under a generated `statement_id`
- `GET /api/2.0/sql/statements/{id}` — re-fetches a previously executed statement's result by ID

Response shape matches the real Statement Execution API's essentials: `status.state` (`SUCCEEDED`/`FAILED`), `manifest.schema.columns`, `result.data_array`. Execution is synchronous (no real cluster to wait on), so there's no `PENDING`/`RUNNING` state — a request returns already resolved.

## Error format

Return standard Databricks API error shape (`error_code`, `message`) instead of generic HTTP 500 pages where possible.

## Emulation scope by product (low-resource local target)

No distributed Spark, no JVM. Engine substitute: **DuckDB** (SQL/warehouse, Delta tables) + **delta-rs** (Delta format read/write) cover most test cases without a real cluster.

| Product | Emulation strategy | Resource cost |
|---|---|---|
| Workspace (notebooks/folders) | Mock, maps to local filesystem | trivial |
| DBFS / Volumes | Mock, local filesystem | trivial |
| Clusters API | Fake state machine (create/get/delete), never spins up real Spark | trivial |
| Jobs/Workflows | Runs script as local subprocess/goroutine, no real Spark | low |
| Unity Catalog (registry) | SQLite for catalogs/schemas/tables — metadata only | low |
| Secrets API | KV store (SQLite/file) | trivial |
| Repos (git) | Thin wrapper over local git | trivial |
| Cluster policies | Static JSON validation | trivial |
| Delta Lake tables | `delta-rs` or DuckDB's Delta extension — real Delta format, no Spark (not yet wired to the catalog registry — see below) | low-medium |
| Databricks SQL / Warehouses | **Implemented** — `/api/2.0/sql/statements` runs against an embedded DuckDB (`internal/store/warehouse.go`); adds cgo to the build and ~60MB to the binary | low-medium |
| MLflow Tracking/Registry | Run real MLflow OSS server (lightweight) — not an emulation | low |
| Model Serving | Mock: endpoint returns canned/fixture prediction | trivial |
| Lakeview/Dashboards | Mock API, returns static JSON (no real rendering) | trivial |

**Not worth emulating locally** (effort >> payoff, or needs a heavy runtime):
- Distributed Spark (multi-node) — skip; if a real transform is unavoidable, `local[1]` only as last resort (JVM+PySpark costs hundreds of MB, breaks the <100MB idle target)
- Delta Live Tables (DLT) — complex orchestration; at most mock the API + run tasks sequentially as plain scripts
- Feature Store — shallow mock (tables tagged in SQLite), no real point-in-time joins
- Full Unity Catalog enforcement (RLS, masking, lineage) — see governance section below for a layered approach instead of skipping entirely

## Known gap: UC registry and SQL warehouse are not yet linked

`unity-catalog/tables` (metadata registry, SQLite) and `sql/statements` (real execution, DuckDB) are currently independent: creating a table via the UC API doesn't create anything queryable in DuckDB, and `CREATE TABLE`/`INSERT` via SQL statements doesn't register anything in the UC registry. Wiring them together (e.g. UC `CREATE TABLE` issuing the matching DDL against the warehouse, and/or `catalog.schema.table`-qualified names in SQL resolving through the registry) is natural follow-up work, and a prerequisite for the RLS/masking rewrite described below — that rewrite needs to know, per query, which UC-registered table a name refers to.

## Unity Catalog governance (layered emulation)

Governance can be emulated in layers on top of the plain UC registry, still without Spark — enforcement happens at the emulator's query-rewrite layer in front of DuckDB.

| Feature | Emulation approach | Cost | Complexity |
|---|---|---|---|
| GRANT/REVOKE (ACL) | SQLite table `(principal, securable, privilege)`; Go middleware checks before each API call | trivial | low |
| Principals (users/groups/SPs) | No real auth — custom header (e.g. `X-Test-Principal`) simulates "acting as X" | trivial | low |
| Row-level security | Policy stored in metadata; emulator rewrites query as a DuckDB **VIEW** injecting the policy's `WHERE` for the active principal | low-medium | medium |
| Column masking | Same mechanism: view swaps raw column for `mask_fn(column)` (hash/regexp_replace) per grant | low-medium | medium |
| Tags / classification (PII, etc.) | Simple KV `(securable, tag, value)` in SQLite | trivial | low |
| Audit log | Append-only table `(ts, principal, action, securable, result)` | trivial | low |
| Lineage (table-level) | Captured via declared `inlets/outlets` on jobs (no SQL parsing) — simple graph in SQLite | low | low |
| Lineage (column-level) | Needs a real SQL parser (e.g. sqlglot-equivalent) — dev-time cost, not runtime | medium | high |
| ABAC (tag-based policy) | Layer over ACL: rule references a tag instead of an explicit securable | medium | medium |
| Delta Sharing / cross-workspace | Out of scope for local emulation | — | — |

Suggested build order: ACL → tags → audit → table-level lineage → RLS/masking → ABAC → column-level lineage (last — the only item that's actually expensive).
