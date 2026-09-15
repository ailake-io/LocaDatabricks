// Package engine runs job scripts as local subprocesses instead of on a
// real Spark cluster — see docs/API.md "Emulation scope by product".
package engine

import (
	"context"
	"os/exec"
	"time"
)

// Executor runs a script and reports SUCCESS/FAILED to the given callback.
type Executor struct{}

func New() *Executor { return &Executor{} }

// RunAsync starts the script in a goroutine and calls onDone with the
// resulting life-cycle/result state once it exits.
func (e *Executor) RunAsync(scriptPath string, onDone func(lifeCycle, result string)) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		interpreter := "python3"
		cmd := exec.CommandContext(ctx, interpreter, scriptPath)

		if err := cmd.Run(); err != nil {
			onDone("TERMINATED", "FAILED")
			return
		}
		onDone("TERMINATED", "SUCCESS")
	}()
}
