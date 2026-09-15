package main

import (
	"crypto/rand"
	"embed"
	"encoding/hex"
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
	bind := flag.String("bind", "127.0.0.1", "address to listen on — only change this to expose the emulator beyond localhost")
	port := flag.String("port", "8080", "port to listen on")
	dataDir := flag.String("data-dir", ".", "root directory for dbfs_root, workspace_root, metadata.db")
	token := flag.String("token", os.Getenv("LOCALDATABRICKS_TOKEN"), "bearer token clients must send; a random one is generated and printed if omitted")
	flag.Parse()

	if *token == "" {
		generated, err := randomToken()
		if err != nil {
			log.Fatalf("generate token: %v", err)
		}
		*token = generated
	}

	dbfsRoot := filepath.Join(*dataDir, "dbfs_root")
	workspaceRoot := filepath.Join(*dataDir, "workspace_root")
	if err := os.MkdirAll(dbfsRoot, 0o755); err != nil {
		log.Fatalf("create dbfs_root: %v", err)
	}
	if err := os.MkdirAll(workspaceRoot, 0o755); err != nil {
		log.Fatalf("create workspace_root: %v", err)
	}

	s, err := store.New(filepath.Join(*dataDir, "metadata.db"), filepath.Join(*dataDir, "warehouse.duckdb"))
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
		Token:         *token,
	})

	subFS, err := fs.Sub(embeddedUI, "ui")
	if err != nil {
		log.Fatal(err)
	}
	app.Use("/", filesystem.New(filesystem.Config{
		Root:  http.FS(subFS),
		Index: "index.html",
	}))

	if *bind != "127.0.0.1" && *bind != "localhost" {
		log.Printf("WARNING: binding to %s exposes an unauthenticated-by-default RCE surface (jobs/run-now executes scripts) to anyone who can reach this address\n", *bind)
	}
	log.Printf("LocalDatabricks emulator listening on http://%s:%s\n", *bind, *port)
	log.Printf("Bearer token: %s (send as 'Authorization: Bearer %s')\n", *token, *token)
	log.Fatal(app.Listen(*bind + ":" + *port))
}

func randomToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
