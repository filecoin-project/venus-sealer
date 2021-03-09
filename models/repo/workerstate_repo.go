package repo

import "github.com/filecoin-project/venus-sealer/types"

type WorkerStateRepo interface {
	GetWorkerStateByWorkID(workId types.WorkID) (*types.WorkState, error)
	HasState(workId types.WorkID) (bool, error)
	Save(workId types.WorkID, state *types.WorkState) error
	GetAllWorkState() ([]*types.WorkState, error)
	DeleteByWorkID(workId types.WorkID) error
	UpdateStateByWorkID(state *types.WorkState, workId types.WorkID) error
}
