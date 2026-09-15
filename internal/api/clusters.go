package api

import (
	"github.com/gofiber/fiber/v2"

	"github.com/ailake-io/LocaDatabricks/internal/store"
)

func createCluster(s *store.Store) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var body struct {
			ClusterName string `json:"cluster_name"`
		}
		if err := c.BodyParser(&body); err != nil {
			return dbxError(c, fiber.StatusBadRequest, "INVALID_PARAMETER_VALUE", err.Error())
		}
		cl := s.CreateCluster(body.ClusterName)
		return c.JSON(fiber.Map{"cluster_id": cl.ClusterID})
	}
}

func getCluster(s *store.Store) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Query("cluster_id")
		cl, ok := s.GetCluster(id)
		if !ok {
			return dbxError(c, fiber.StatusNotFound, "RESOURCE_DOES_NOT_EXIST", "cluster not found: "+id)
		}
		return c.JSON(cl)
	}
}

func deleteCluster(s *store.Store) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var body struct {
			ClusterID string `json:"cluster_id"`
		}
		if err := c.BodyParser(&body); err != nil {
			return dbxError(c, fiber.StatusBadRequest, "INVALID_PARAMETER_VALUE", err.Error())
		}
		if !s.DeleteCluster(body.ClusterID) {
			return dbxError(c, fiber.StatusNotFound, "RESOURCE_DOES_NOT_EXIST", "cluster not found: "+body.ClusterID)
		}
		return c.JSON(fiber.Map{})
	}
}
