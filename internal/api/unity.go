package api

import (
	"github.com/gofiber/fiber/v2"

	"github.com/ailake-io/LocaDatabricks/internal/store"
)

func createCatalog(s *store.Store) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var body struct {
			Name    string `json:"name"`
			Comment string `json:"comment"`
		}
		if err := c.BodyParser(&body); err != nil {
			return dbxError(c, fiber.StatusBadRequest, "INVALID_PARAMETER_VALUE", err.Error())
		}
		if err := s.CreateCatalog(body.Name, body.Comment); err != nil {
			return dbxError(c, fiber.StatusConflict, "ALREADY_EXISTS", err.Error())
		}
		return c.JSON(fiber.Map{"name": body.Name, "comment": body.Comment})
	}
}

func createSchema(s *store.Store) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var body struct {
			CatalogName string `json:"catalog_name"`
			Name        string `json:"name"`
			Comment     string `json:"comment"`
		}
		if err := c.BodyParser(&body); err != nil {
			return dbxError(c, fiber.StatusBadRequest, "INVALID_PARAMETER_VALUE", err.Error())
		}
		if err := s.CreateSchema(body.CatalogName, body.Name, body.Comment); err != nil {
			return dbxError(c, fiber.StatusConflict, "ALREADY_EXISTS", err.Error())
		}
		return c.JSON(fiber.Map{"catalog_name": body.CatalogName, "name": body.Name})
	}
}

func createTable(s *store.Store) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var body struct {
			CatalogName      string `json:"catalog_name"`
			SchemaName       string `json:"schema_name"`
			Name             string `json:"name"`
			DataSourceFormat string `json:"data_source_format"`
			StorageLocation  string `json:"storage_location"`
		}
		if err := c.BodyParser(&body); err != nil {
			return dbxError(c, fiber.StatusBadRequest, "INVALID_PARAMETER_VALUE", err.Error())
		}
		if body.DataSourceFormat == "" {
			body.DataSourceFormat = "DELTA"
		}
		if err := s.CreateTable(body.CatalogName, body.SchemaName, body.Name, body.DataSourceFormat, body.StorageLocation); err != nil {
			return dbxError(c, fiber.StatusConflict, "ALREADY_EXISTS", err.Error())
		}
		return c.JSON(fiber.Map{
			"catalog_name": body.CatalogName,
			"schema_name":  body.SchemaName,
			"name":         body.Name,
		})
	}
}

func listTables(s *store.Store) fiber.Handler {
	return func(c *fiber.Ctx) error {
		catalog := c.Query("catalog_name")
		schema := c.Query("schema_name")
		tables, err := s.ListTables(catalog, schema)
		if err != nil {
			return dbxError(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		}
		return c.JSON(fiber.Map{"tables": tables})
	}
}
