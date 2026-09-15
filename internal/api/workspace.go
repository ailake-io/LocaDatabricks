package api

import (
	"encoding/base64"
	"os"
	"path/filepath"

	"github.com/gofiber/fiber/v2"
)

func workspaceList(root string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqPath := c.Query("path", "/")
		dir, ok := resolveRootedPath(root, reqPath)
		if !ok {
			return dbxError(c, fiber.StatusBadRequest, "INVALID_PARAMETER_VALUE", "path escapes workspace root")
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			return dbxError(c, fiber.StatusNotFound, "RESOURCE_DOES_NOT_EXIST", err.Error())
		}

		objects := make([]fiber.Map, 0, len(entries))
		for _, e := range entries {
			objType := "FILE"
			if e.IsDir() {
				objType = "DIRECTORY"
			}
			objects = append(objects, fiber.Map{
				"path":        filepath.Join(reqPath, e.Name()),
				"object_type": objType,
			})
		}
		return c.JSON(fiber.Map{"objects": objects})
	}
}

func workspaceMkdirs(root string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var body struct {
			Path string `json:"path"`
		}
		if err := c.BodyParser(&body); err != nil {
			return dbxError(c, fiber.StatusBadRequest, "INVALID_PARAMETER_VALUE", err.Error())
		}
		dir, ok := resolveRootedPath(root, body.Path)
		if !ok {
			return dbxError(c, fiber.StatusBadRequest, "INVALID_PARAMETER_VALUE", "path escapes workspace root")
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return dbxError(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		}
		return c.JSON(fiber.Map{})
	}
}

// workspaceImport writes a base64-encoded notebook to disk (matches the
// real Databricks workspace/import content field).
func workspaceImport(root string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var body struct {
			Path    string `json:"path"`
			Content string `json:"content"`
			Format  string `json:"format"`
		}
		if err := c.BodyParser(&body); err != nil {
			return dbxError(c, fiber.StatusBadRequest, "INVALID_PARAMETER_VALUE", err.Error())
		}
		dest, ok := resolveRootedPath(root, body.Path)
		if !ok {
			return dbxError(c, fiber.StatusBadRequest, "INVALID_PARAMETER_VALUE", "path escapes workspace root")
		}

		decoded, err := base64.StdEncoding.DecodeString(body.Content)
		if err != nil {
			return dbxError(c, fiber.StatusBadRequest, "INVALID_PARAMETER_VALUE", "content must be base64")
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return dbxError(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		}
		if err := os.WriteFile(dest, decoded, 0o644); err != nil {
			return dbxError(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		}
		return c.JSON(fiber.Map{})
	}
}
