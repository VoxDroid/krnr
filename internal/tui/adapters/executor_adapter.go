package adapters

import (
	"bufio"
	"context"
	"fmt"
	"io"

	"github.com/VoxDroid/krnr/internal/executor"
)

// executorAdapter implements ExecutorAdapter using an executor.Runner.
type executorAdapter struct{ runner executor.Runner }

// NewExecutorAdapter constructs an ExecutorAdapter backed by the provided Runner.
func NewExecutorAdapter(r executor.Runner) ExecutorAdapter { return &executorAdapter{runner: r} }

func (e *executorAdapter) Run(ctx context.Context, _ string, commands []string) (RunHandle, error) {
	ctx, cancel := context.WithCancel(ctx)
	rchan := make(chan RunEvent)

	// run commands sequentially; stop on first error or context cancellation
	go func() {
		defer close(rchan)
		for _, cmdText := range commands {
			// announce command
			rchan <- RunEvent{Line: fmt.Sprintf("-> %s", cmdText)}

			// pipe stdout/stderr
			r, w := io.Pipe()
			// exec in a goroutine
			execErr := make(chan error, 1)
			go func(cmdToRun string) {
				execErr <- e.runner.Execute(ctx, cmdToRun, "", w, w)
				_ = w.Close()
			}(cmdText)

			// read lines
			s := bufio.NewScanner(r)
			for s.Scan() {
				select {
				case <-ctx.Done():
					_ = r.Close()
					return
				default:
					rchan <- RunEvent{Line: s.Text()}
				}
			}
			if err := s.Err(); err != nil {
				rchan <- RunEvent{Err: err}
				_ = r.Close()
				return
			}
			// wait for command to finish and check its error
			if err := <-execErr; err != nil {
				rchan <- RunEvent{Err: fmt.Errorf("exec: %w", err)}
				_ = r.Close()
				return
			}
			_ = r.Close()
		}
	}()

	return &runHandleImpl{ch: rchan, cancel: cancel}, nil
}

type runHandleImpl struct {
	ch     <-chan RunEvent
	cancel context.CancelFunc
}

func (r *runHandleImpl) Events() <-chan RunEvent { return r.ch }
func (r *runHandleImpl) Cancel()                 { r.cancel() }
