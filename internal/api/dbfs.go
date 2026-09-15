package api

import (
	"io"
	"os"
	"path/filepath"

	"github.com/gofiber/fiber/v2"
)

func dbfsList(root string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqPath := c.Query("path", "/")
		dir, ok := resolveRootedPath(root, reqPath)
		if !ok {
			return dbxError(c, fiber.StatusBadRequest, "INVALID_PARAMETER_VALUE", "path escapes dbfs root")
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			return dbxError(c, fiber.StatusNotFound, "RESOURCE_DOES_NOT_EXIST", err.Error())
		}

		files := make([]fiber.Map, 0, len(entries))
		for _, e := range entries {
			info, err := e.Info()
			if err != nil {
				continue
			}
			files = append(files, fiber.Map{
				"path":      filepath.Join(reqPath, e.Name()),
				"is_dir":    e.IsDir(),
				"file_size": info.Size(),
			})
		}
		return c.JSON(fiber.Map{"files": files})
	}
}

// dbfsPut streams the request body directly to disk — never buffers the
// whole file in memory (see docs/DEVELOPMENT.md).
func dbfsPut(root string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqPath := c.Query("path")
		if reqPath == "" {
			return dbxError(c, fiber.StatusBadRequest, "INVALID_PARAMETER_VALUE", "path is required")
		}
		dest, ok := resolveRootedPath(root, reqPath)
		if !ok {
			return dbxError(c, fiber.StatusBadRequest, "INVALID_PARAMETER_VALUE", "path escapes dbfs root")
		}

		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return dbxError(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		}

		f, err := os.Create(dest)
		if err != nil {
			return dbxError(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		}
		defer f.Close()

		bodyReader := c.Context().RequestBodyStream()
		if bodyReader != nil {
			if _, err := io.Copy(f, bodyReader); err != nil {
				return dbxError(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
			}
		} else if _, err := f.Write(c.Body()); err != nil {
			return dbxError(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		}
		return c.JSON(fiber.Map{})
	}
}

func dbfsRead(root string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqPath := c.Query("path")
		src, ok := resolveRootedPath(root, reqPath)
		if !ok {
			return dbxError(c, fiber.StatusBadRequest, "INVALID_PARAMETER_VALUE", "path escapes dbfs root")
		}
		return c.SendFile(src, false)
	}
}
