package converter

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// runCmd executes an external command with a timeout, calling onProgress
// periodically with an estimated progress value (0.0–1.0).
func runCmd(timeout time.Duration, args []string, env []string, onProgress func(float64)) error {
	if len(args) == 0 {
		return fmt.Errorf("empty command")
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Env = append(os.Environ(), env...)

	// Send progress ticks while running
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	start := time.Now()
	elapsed := time.Duration(0)
	clamped := float64(0)

	for {
		select {
		case err := <-done:
			onProgress(1.0)
			if err != nil {
				return fmt.Errorf("command failed: %w", err)
			}
			return nil
		case <-ticker.C:
			elapsed = time.Since(start)
			// Estimate progress: linear up to 95% based on elapsed vs timeout,
			// capped at 0.95 until completion.
			if timeout > 0 {
				ratio := float64(elapsed) / float64(timeout)
				if ratio > 0.95 {
					ratio = 0.95
				}
				if ratio > clamped {
					clamped = ratio
				}
			}
			onProgress(clamped)
		case <-ctx.Done():
			return fmt.Errorf("command timed out after %v", timeout)
		}
	}
}
