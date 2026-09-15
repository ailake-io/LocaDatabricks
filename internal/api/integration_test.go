package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"

	"github.com/ailake-io/LocaDatabricks/internal/api"
	"github.com/ailake-io/LocaDatabricks/internal/engine"
	"github.com/ailake-io/LocaDatabricks/internal/store"
)

const testToken = "test-token"

func newTestApp(t *testing.T) *fiber.App {
	t.Helper()
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { s.Close() })

	app := fiber.New()
	api.Register(app, s, engine.New(), api.Config{
		WorkspaceRoot: t.TempDir(),
		DBFSRoot:      t.TempDir(),
		Token:         testToken,
	})
	return app
}

func do(t *testing.T, app *fiber.App, method, path, body string) *http.Response {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testToken)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request %s %s: %v", method, path, err)
	}
	return resp
}

func TestUnauthenticatedRequestIsRejected(t *testing.T) {
	app := newTestApp(t)
	req := httptest.NewRequest("POST", "/api/2.0/clusters/create", strings.NewReader(`{"cluster_name":"dev"}`))
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

func TestClusterCreateGetDelete(t *testing.T) {
	app := newTestApp(t)

	req := httptest.NewRequest("POST", "/api/2.0/clusters/create", strings.NewReader(`{"cluster_name":"dev"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testToken)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("create status = %d, want 200", resp.StatusCode)
	}

	var created struct {
		ClusterID string `json:"cluster_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.ClusterID == "" {
		t.Fatal("cluster_id was empty")
	}

	getReq := httptest.NewRequest("GET", "/api/2.0/clusters/get?cluster_id="+created.ClusterID, nil)
	getReq.Header.Set("Authorization", "Bearer "+testToken)
	getResp, err := app.Test(getReq)
	if err != nil {
		t.Fatal(err)
	}
	if getResp.StatusCode != fiber.StatusOK {
		t.Fatalf("get status = %d, want 200", getResp.StatusCode)
	}
}

func TestDBFSPathTraversalRejected(t *testing.T) {
	app := newTestApp(t)

	req := httptest.NewRequest("GET", "/api/2.0/dbfs/read?path=../../../../etc/passwd", nil)
	req.Header.Set("Authorization", "Bearer "+testToken)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode == fiber.StatusOK {
		t.Fatal("expected traversal request to be rejected, got 200")
	}
}

func TestUnityCatalogFlow(t *testing.T) {
	app := newTestApp(t)

	do(t, app, "POST", "/api/2.1/unity-catalog/catalogs", `{"name":"main"}`)
	do(t, app, "POST", "/api/2.1/unity-catalog/schemas", `{"catalog_name":"main","name":"default"}`)
	resp := do(t, app, "POST", "/api/2.1/unity-catalog/tables", `{"catalog_name":"main","schema_name":"default","name":"orders"}`)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("create table status = %d, want 200", resp.StatusCode)
	}

	listReq := httptest.NewRequest("GET", "/api/2.1/unity-catalog/tables?catalog_name=main&schema_name=default", nil)
	listReq.Header.Set("Authorization", "Bearer "+testToken)
	listResp, err := app.Test(listReq)
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		Tables []struct {
			Name string `json:"name"`
		} `json:"tables"`
	}
	if err := json.NewDecoder(listResp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if len(out.Tables) != 1 || out.Tables[0].Name != "orders" {
		t.Fatalf("tables = %+v, want one table named orders", out.Tables)
	}
}
