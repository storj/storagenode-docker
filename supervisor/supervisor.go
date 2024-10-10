package supervisor

import (
	"context"
	"log/slog"
	"time"

	"github.com/zeebo/errs"
	"golang.org/x/sync/errgroup"

	"storj.io/common/sync2"
)

const maxRetries = 3

var errSupervisor = errs.Class("supervisor")

// Supervisor is a process manager for the storagenode.
// It manages only one storagenode process.
type Supervisor struct {
	updater *Updater

	process *Process

	updaterLoop *sync2.Cycle
}

// New creates a new supervisor.
func New(updater *Updater, process *Process, checkInterval time.Duration) *Supervisor {
	return &Supervisor{
		updater:     updater,
		process:     process,
		updaterLoop: sync2.NewCycle(checkInterval),
	}
}

// Run starts the supervisor
func (s *Supervisor) Run(ctx context.Context) error {
	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() error {
		// run the updater loop.
		// most of the errors are logged and ignored, so we don't exit the supervisor.
		return s.updaterLoop.Run(ctx, func(ctx context.Context) error {
			currentVersion, err := s.process.Version(ctx)
			if err != nil {
				slog.Error("Failed to get current version", "error", err)
				return nil
			}

			updated, err := s.updater.Update(ctx, s.process, currentVersion)
			if err != nil {
				slog.Error("Failed to update process", "error", err)
				return nil
			}

			if updated {
				return errSupervisor.Wrap(s.process.exit())
			}

			return nil
		})
	})

	group.Go(func() error {
		// let's check if the process exits multiple times in a row in a short period
		// of time, if so, we should stop trying to restart it and exit the supervisor
		var retries int

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			slog.Info("Restarting process")
			err := s.runProcess(ctx)
			if err != nil {
				// let's log the error and restart the process
				slog.Error("Process exited with error", "error", err)
			}
			retries++
			if retries >= maxRetries {
				slog.Info("Too many retries, exiting supervisor")
				return s.close()
			}
		}
	})

	return group.Wait()
}

func (s *Supervisor) runProcess(ctx context.Context) error {
	if err := s.process.start(ctx); err != nil {
		return err
	}

	return s.process.wait()
}

// close stops all processes managed by the supervisor including the updater.
func (s *Supervisor) close() error {
	s.updaterLoop.Close()
	return s.process.exit()
}
