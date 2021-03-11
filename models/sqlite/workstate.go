package sqlite

import (
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/venus-sealer/models/repo"
	"github.com/filecoin-project/venus-sealer/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type workerState struct {
	Id     string `gorm:"column:id;type:varchar(36);primary_key;" json:"id"` // 主键
	Method string `gorm:"uniqueIndex:method_params;column:method;type:varchar(256);" json:"method"`
	Params string `gorm:"uniqueIndex:method_params;column:params;type:text;" json:"params"`

	Status   string `gorm:"column:status;type:varchar(256);" json:"status"`
	WorkId   string `gorm:"column:work_id;type:varchar(36);" json:"work_id"`
	MinerID  uint64 `gorm:"column:miner_id;type:unsigned bigint;" json:"miner_id"`
	SectorId uint64 `gorm:"column:sector_id;type:unsigned bigint;" json:"sector_id"`

	WorkError string `gorm:"column:work_error;type:varchar(256);" json:"work_error"`

	WorkerHostname string `gorm:"column:worker_host_name;type:varchar(256);" json:"worker_host_name"`
	StartTime      int64  `gorm:"column:start_time;type:bigint;" json:"start_time"`
}

func (workerState *workerState) State() (*types.WorkState, error) {
	uid, err := uuid.Parse(workerState.WorkId)
	if err != nil {
		return nil, err
	}
	return &types.WorkState{
		ID: types.WorkID{
			Method: types.TaskType(workerState.Method),
			Params: workerState.Params,
		},
		Status: types.WorkStatus(workerState.Status),
		WorkerCall: types.CallID{
			Sector: abi.SectorID{
				Miner:  abi.ActorID(workerState.MinerID),
				Number: abi.SectorNumber(workerState.SectorId),
			},
			ID: uid,
		},
		WorkError:      workerState.WorkError,
		WorkerHostname: workerState.WorkerHostname,
		StartTime:      workerState.StartTime,
	}, nil
}

func (workerState *workerState) TableName() string {
	return "worker_states"
}

var _ repo.WorkerStateRepo = (*workerStateRepo)(nil)

type workerStateRepo struct {
	*gorm.DB
}

func newWorkerStateRepo(db *gorm.DB) *workerStateRepo {
	return &workerStateRepo{DB: db}
}

func (w *workerStateRepo) GetWorkerStateByWorkID(workId types.WorkID) (*types.WorkState, error) {
	var workState workerState
	err := w.DB.Table("worker_states").
		First(&workState, "method=? AND params=?", workId.Method, workId.Params).Error
	if err != nil {
		return nil, err
	}
	return workState.State()
}

func (w *workerStateRepo) HasState(workId types.WorkID) (bool, error) {
	var count int64
	err := w.DB.Table("worker_states").
		Where("method=? AND params=?", workId.Method, workId.Params).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (w *workerStateRepo) Save(workId types.WorkID, state *types.WorkState) error {
	workerState := workerState{
		Id:             uuid.New().String(),
		Method:         string(workId.Method),
		Params:         workId.Params,
		Status:         string(state.Status),
		WorkId:         state.WorkerCall.ID.String(),
		MinerID:        uint64(state.WorkerCall.Sector.Miner),
		SectorId:       uint64(state.WorkerCall.Sector.Number),
		WorkError:      state.WorkError,
		WorkerHostname: state.WorkerHostname,
		StartTime:      state.StartTime,
	}

	return w.DB.Create(&workerState).Error
}

func (w *workerStateRepo) GetAllWorkState() ([]*types.WorkState, error) {
	var workerStates []*workerState
	err := w.DB.Table("worker_states").Find(&workerStates).Error
	if err != nil {
		return nil, err
	}
	result := make([]*types.WorkState, len(workerStates))
	for index, st := range workerStates {
		newSt, err := st.State()
		if err != nil {
			return nil, err
		}
		result[index] = newSt
	}
	return result, nil
}

func (w *workerStateRepo) DeleteByWorkID(workId types.WorkID) error {
	return w.DB.Delete(&workerState{}, "method=? AND params=?", workId.Method, workId.Params).Error
}

func (w *workerStateRepo) UpdateStateByWorkID(state *types.WorkState, workId types.WorkID) error {
	updateClause := map[string]interface{}{
		"start_time":       state.StartTime,
		"worker_host_name": state.WorkerHostname,
		"work_error":       state.WorkError,
		"sector_id":        state.WorkerCall.Sector.Number,
		"work_id":          state.WorkerCall.ID.String(),
		"status":           state.Status,
		"miner_id":         state.WorkerCall.Sector.Miner,
	}
	return w.DB.Model(&workerState{}).
		Where("method=? AND params=?", workId.Method, workId.Params).
		UpdateColumns(updateClause).Error
}
