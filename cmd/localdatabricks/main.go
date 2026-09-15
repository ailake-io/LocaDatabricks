package main

import (
	"embed"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/logger"

	"github.com/ailake-io/LocaDatabricks/internal/api"
	"github.com/ailake-io/LocaDatabricks/internal/engine"
	"github.com/ailake-io/LocaDatabricks/internal/store"
)

//go:embed all:ui
var embeddedUI embed.FS

func main() {
	port := flag.String("port", "8080", "port to listen on")
	dataDir := flag.String("data-dir", ".", "root directory for dbfs_root, workspace_root, metadata.db")
	flag.Parse()

	dbfsRoot := filepath.Join(*dataDir, "dbfs_root")
	workspaceRoot := filepath.Join(*dataDir, "workspace_root")
	if err := os.MkdirAll(dbfsRoot, 0o755); err != nil {
		log.Fatalf("create dbfs_root: %v", err)
	}
	if err := os.MkdirAll(workspaceRoot, 0o755); err != nil {
		log.Fatalf("create workspace_root: %v", err)
	}

	s, err := store.New(filepath.Join(*dataDir, "metadata.db"))
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer s.Close()

	exec := engine.New()

	app := fiber.New(fiber.Config{
		AppName: "LocalDatabricks Emulator",
	})
	app.Use(logger.New())

	api.Register(app, s, exec, api.Config{
		WorkspaceRoot: workspaceRoot,
		DBFSRoot:      dbfsRoot,
	})

	subFS, err := fs.Sub(embeddedUI, "ui")
	if err != nil {
		log.Fatal(err)
	}
	app.Use("/", filesystem.New(filesystem.Config{
		Root:  http.FS(subFS),
		Index: "index.html",
	}))

	log.Printf("LocalDatabricks emulator listening on http://localhost:%s\n", *port)
	log.Fatal(app.Listen(":" + *port))
}
