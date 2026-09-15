package api

import (
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/ailake-io/LocaDatabricks/internal/engine"
	"github.com/ailake-io/LocaDatabricks/internal/store"
)

// resolveWorkspacePath confines a job's script path to workspaceRoot,
// rejecting any traversal outside of it before exec.Command ever sees it.
func resolveWorkspacePath(workspaceRoot, rel string) (string, bool) {
	full := filepath.Join(workspaceRoot, filepath.Clean("/"+rel))
	absRoot, err := filepath.Abs(workspaceRoot)
	if err != nil {
		return "", false
	}
	absFull, err := filepath.Abs(full)
	if err != nil {
		return "", false
	}
	if absFull != absRoot && !strings.HasPrefix(absFull, absRoot+string(filepath.Separator)) {
		return "", false
	}
	return absFull, true
}

func createJob(s *store.Store, workspaceRoot string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var body struct {
			Name            string `json:"name"`
			SparkPythonTask struct {
				PythonFile string `json:"python_file"`
			} `json:"spark_python_task"`
		}
		if err := c.BodyParser(&body); err != nil {
			return dbxError(c, fiber.StatusBadRequest, "INVALID_PARAMETER_VALUE", err.Error())
		}
		scriptPath, ok := resolveWorkspacePath(workspaceRoot, body.SparkPythonTask.PythonFile)
		if !ok {
			return dbxError(c, fiber.StatusBadRequest, "INVALID_PARAMETER_VALUE", "python_file must resolve within the workspace root")
		}
		job := s.CreateJob(body.Name, scriptPath)
		return c.JSON(fiber.Map{"job_id": job.JobID})
	}
}

func runNow(s *store.Store, exec *engine.Executor) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var body struct {
			JobID int64 `json:"job_id"`
		}
		if err := c.BodyParser(&body); err != nil {
			return dbxError(c, fiber.StatusBadRequest, "INVALID_PARAMETER_VALUE", err.Error())
		}
		job, ok := s.GetJob(body.JobID)
		if !ok {
			return dbxError(c, fiber.StatusNotFound, "RESOURCE_DOES_NOT_EXIST", "job not found: "+strconv.FormatInt(body.JobID, 10))
		}

		run := s.NewRun()
		exec.RunAsync(job.ScriptPath, func(lifeCycle, result string) {
			s.SetRunResult(run.RunID, lifeCycle, result)
		})
		return c.JSON(fiber.Map{"run_id": run.RunID})
	}
}

func getRun(s *store.Store) fiber.Handler {
	return func(c *fiber.Ctx) error {
		runID, err := strconv.ParseInt(c.Query("run_id"), 10, 64)
		if err != nil {
			return dbxError(c, fiber.StatusBadRequest, "INVALID_PARAMETER_VALUE", "run_id must be an integer")
		}
		run, ok := s.GetRun(runID)
		if !ok {
			return dbxError(c, fiber.StatusNotFound, "RESOURCE_DOES_NOT_EXIST", "run not found")
		}
		return c.JSON(fiber.Map{
			"run_id": run.RunID,
			"state": fiber.Map{
				"life_cycle_state": run.LifeCycleState,
				"result_state":     run.ResultState,
			},
		})
	}
}
