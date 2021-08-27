package api

import (
	"context"

	"github.com/google/uuid"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/specs-storage/storage"

	"github.com/filecoin-project/venus-sealer/constants"
	"github.com/filecoin-project/venus-sealer/sector-storage/stores"
	"github.com/filecoin-project/venus-sealer/sector-storage/storiface"
	"github.com/filecoin-project/venus-sealer/types"
)

type WorkerAPI interface {
	Version(context.Context) (constants.Version, error)
	// TODO: Info() (name, ...) ?

	TaskTypes(context.Context) (map[types.TaskType]struct{}, error) // TaskType -> Weight
	Paths(context.Context) ([]stores.StoragePath, error)
	Info(context.Context) (storiface.WorkerInfo, error)

	TaskNumbers(context.Context) (string, error)

	SectorExists(context.Context, types.TaskType, storage.SectorRef) (bool, error)

	storiface.WorkerCalls

	TaskDisable(ctx context.Context, tt types.TaskType) error
	TaskEnable(ctx context.Context, tt types.TaskType) error

	// Storage / Other
	Remove(ctx context.Context, sector abi.SectorID) error

	StorageAddLocal(ctx context.Context, path string) error

	// SetEnabled marks the worker as enabled/disabled. Not that this setting
	// may take a few seconds to propagate to task scheduler
	SetEnabled(ctx context.Context, enabled bool) error

	Enabled(ctx context.Context) (bool, error)

	// WaitQuiet blocks until there are no tasks running
	WaitQuiet(ctx context.Context) error

	// returns a random UUID of worker session, generated randomly when worker
	// process starts
	ProcessSession(context.Context) (uuid.UUID, error)

	// Like ProcessSession, but returns an error when worker is disabled
	Session(context.Context) (uuid.UUID, error)
}
