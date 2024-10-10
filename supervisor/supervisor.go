package supervisor

import (
	"context"
	"time"

	"golang.org/x/sync/errgroup"

	"storj.io/common/sync2"
)

type processState int

const (
	stateRunning processState = iota
	stateRestarting
	stateExited
	stateError
)

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
		updater:       updater,
		shouldRestart: make(chan *Process),
		process:       process,
		updaterLoop:   sync2.NewCycle(checkInterval),
	}
}

// Run starts the supervisor
func (s *Supervisor) Run(ctx context.Context) error {
	var group errgroup.Group

	group.Go(func() error {
		return s.updaterLoop.Run(ctx, func(ctx context.Context) error {
			shouldUpdate, err := s.updater.ShouldUpdate(ctx, s.process)
			if err != nil {
				return err
			}

		})
	})

	group.Go(func() error {
		for {
			err := s.runProcess(ctx)
			if err != nil {
				return err
			}
			select {
			case <-ctx.Done():
				return nil
			default:
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

// Close stops the supervisor
func (s *Supervisor) Close() error {
	return s.process.exit()
}
