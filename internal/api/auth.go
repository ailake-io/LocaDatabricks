package api

import (
	"crypto/subtle"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// RequireToken enforces a bearer token on every request, mirroring real
// Databricks' DATABRICKS_TOKEN auth — without it, anyone who can reach the
// port (e.g. on a shared LAN) could create clusters, write DBFS, or run
// arbitrary scripts via jobs/run-now.
func RequireToken(token string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		const prefix = "Bearer "
		h := c.Get(fiber.HeaderAuthorization)
		if !strings.HasPrefix(h, prefix) {
			return dbxError(c, fiber.StatusUnauthorized, "PERMISSION_DENIED", "missing bearer token")
		}
		got := strings.TrimPrefix(h, prefix)
		if subtle.ConstantTimeCompare([]byte(got), []byte(token)) != 1 {
			return dbxError(c, fiber.StatusUnauthorized, "PERMISSION_DENIED", "invalid token")
		}
		return c.Next()
	}
}
