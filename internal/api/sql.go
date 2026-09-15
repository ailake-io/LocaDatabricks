package api

import (
	"github.com/gofiber/fiber/v2"

	"github.com/ailake-io/LocaDatabricks/internal/store"
)

func statementResultJSON(r *store.StatementResult) fiber.Map {
	out := fiber.Map{
		"statement_id": r.StatementID,
		"status":       fiber.Map{"state": r.State},
	}
	if r.State == "FAILED" {
		out["status"] = fiber.Map{
			"state": r.State,
			"error": fiber.Map{"message": r.ErrorMessage},
		}
		return out
	}

	columns := make([]fiber.Map, len(r.Columns))
	for i, c := range r.Columns {
		columns[i] = fiber.Map{"name": c.Name, "type_text": c.TypeText}
	}
	out["manifest"] = fiber.Map{"schema": fiber.Map{"columns": columns}}
	out["result"] = fiber.Map{"data_array": r.Rows}
	return out
}

func executeStatement(s *store.Store) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var body struct {
			Statement string `json:"statement"`
		}
		if err := c.BodyParser(&body); err != nil {
			return dbxError(c, fiber.StatusBadRequest, "INVALID_PARAMETER_VALUE", err.Error())
		}
		if body.Statement == "" {
			return dbxError(c, fiber.StatusBadRequest, "INVALID_PARAMETER_VALUE", "statement is required")
		}

		result, err := s.ExecuteStatement(body.Statement)
		if err != nil {
			return dbxError(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		}
		return c.JSON(statementResultJSON(result))
	}
}

func getStatement(s *store.Store) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")
		result, ok := s.GetStatement(id)
		if !ok {
			return dbxError(c, fiber.StatusNotFound, "RESOURCE_DOES_NOT_EXIST", "statement not found: "+id)
		}
		return c.JSON(statementResultJSON(result))
	}
}
