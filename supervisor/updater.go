package supervisor

import (
	"context"

	"storj.io/common/version"
	"storj.io/storj/private/version/checker"
)

type Updater struct {
	checker *checker.Client
}

// NewUpdater creates a new updater.
func NewUpdater(checker *checker.Client) *Updater {
	return &Updater{
		checker: checker,
	}
}

// ShouldUpdate checks if the service should be updated.
func (u *Updater) ShouldUpdate(ctx context.Context, process *Process) (bool, error) {
	all, err := u.checker.All(ctx)
	if err != nil {
		return false, err
	}

	version.ShouldUpdate()
}
