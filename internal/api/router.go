// Package api wires the emulated Databricks REST endpoints (see
// docs/API.md) onto a Fiber app.
package api

import (
	"github.com/gofiber/fiber/v2"

	"github.com/ailake-io/LocaDatabricks/internal/engine"
	"github.com/ailake-io/LocaDatabricks/internal/store"
)

type Config struct {
	WorkspaceRoot string
	DBFSRoot      string
	Token         string
}

// Register mounts all /api/2.0 and /api/2.1 routes behind bearer-token auth.
func Register(app *fiber.App, s *store.Store, exec *engine.Executor, cfg Config) {
	app.Use("/api", RequireToken(cfg.Token))

	v20 := app.Group("/api/2.0")

	v20.Post("/workspace/mkdirs", workspaceMkdirs(cfg.WorkspaceRoot))
	v20.Post("/workspace/import", workspaceImport(cfg.WorkspaceRoot))
	v20.Get("/workspace/list", workspaceList(cfg.WorkspaceRoot))

	v20.Post("/clusters/create", createCluster(s))
	v20.Get("/clusters/get", getCluster(s))
	v20.Post("/clusters/delete", deleteCluster(s))

	v20.Post("/jobs/create", createJob(s, cfg.WorkspaceRoot))
	v20.Post("/jobs/run-now", runNow(s, exec))
	v20.Get("/jobs/runs/get", getRun(s))

	v20.Get("/dbfs/list", dbfsList(cfg.DBFSRoot))
	v20.Post("/dbfs/put", dbfsPut(cfg.DBFSRoot))
	v20.Get("/dbfs/read", dbfsRead(cfg.DBFSRoot))

	uc := app.Group("/api/2.1/unity-catalog")
	uc.Post("/catalogs", createCatalog(s))
	uc.Post("/schemas", createSchema(s))
	uc.Post("/tables", createTable(s))
	uc.Get("/tables", listTables(s))
}
