package api

import "github.com/gofiber/fiber/v2"

// dbxError writes a Databricks-shaped error body: {"error_code", "message"}.
// See docs/DEVELOPMENT.md: never return a bare HTTP 500 page.
func dbxError(c *fiber.Ctx, status int, code, message string) error {
	return c.Status(status).JSON(fiber.Map{
		"error_code": code,
		"message":    message,
	})
}
