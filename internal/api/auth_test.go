package api

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func newTestApp(token string) *fiber.App {
	app := fiber.New()
	app.Use(RequireToken(token))
	app.Get("/probe", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })
	return app
}

func TestRequireToken_Missing(t *testing.T) {
	app := newTestApp("secret")
	req := httptest.NewRequest("GET", "/probe", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("got %d, want 401", resp.StatusCode)
	}
}

func TestRequireToken_Wrong(t *testing.T) {
	app := newTestApp("secret")
	req := httptest.NewRequest("GET", "/probe", nil)
	req.Header.Set("Authorization", "Bearer nope")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("got %d, want 401", resp.StatusCode)
	}
}

func TestRequireToken_Correct(t *testing.T) {
	app := newTestApp("secret")
	req := httptest.NewRequest("GET", "/probe", nil)
	req.Header.Set("Authorization", "Bearer secret")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("got %d, want 200", resp.StatusCode)
	}
}
