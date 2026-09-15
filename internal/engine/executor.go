// Package engine runs job scripts as local subprocesses instead of on a
// real Spark cluster — see docs/API.md "Emulation scope by product".
package engine

import (
	"context"
	"os"
	"os/exec"
	"time"
)

// Executor runs a script and reports SUCCESS/FAILED to the given callback.
type Executor struct{}

func New() *Executor { return &Executor{} }

// RunAsync starts the script in a goroutine and calls onDone with the
// resulting life-cycle/result state once it exits. The caller (jobs.go)
// is responsible for confining scriptPath to the workspace root before
// this ever runs — this is the emulator's one real code-execution surface.
func (e *Executor) RunAsync(scriptPath string, onDone func(lifeCycle, result string)) {
	go func() {
		info, err := os.Stat(scriptPath)
		if err != nil || !info.Mode().IsRegular() {
			onDone("TERMINATED", "FAILED")
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		cmd := exec.CommandContext(ctx, "python3", scriptPath)
		// Minimal, explicit environment — job scripts don't inherit the
		// server process's full environment (which may hold unrelated
		// secrets from the host shell).
		cmd.Env = []string{
			"PATH=" + os.Getenv("PATH"),
			"HOME=" + os.Getenv("HOME"),
		}

		if err := cmd.Run(); err != nil {
			onDone("TERMINATED", "FAILED")
			return
		}
		onDone("TERMINATED", "SUCCESS")
	}()
}
