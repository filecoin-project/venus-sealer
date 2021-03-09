package sqlite

import (
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/venus-sealer/models/repo"
	"github.com/filecoin-project/venus-sealer/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type workerCall struct {
	Id string `gorm:"column:id;type:varchar(36);primary_key;"json:"id"` // 主键
	//storiface.CallID
	WorkId   string `gorm:"uniqueIndex:call_id;column:work_id;type:varchar(36);"json:"work_id"`
	MinerID  uint64 `gorm:"uniqueIndex:call_id;column:miner_id;type:unsigned bigint;"json:"miner_id"`
	SectorId uint64 `gorm:"uniqueIndex:call_id;column:sector_id;type:unsigned bigint;"json:"sector_id"`
	Role     string `gorm:"uniqueIndex:call_id;column:role;type:unsigned bigint;"json:"role"`

	RetType string `gorm:"column:ret_type;type:varchar(256);"json:"ret_type"`

	State uint64 `gorm:"column:state;type:unsigned bigint;"json:"state"`

	//json byte
	Result []byte `gorm:"column:result;type:blob;"json:"result"`
}

func (workerCall *workerCall) TableName() string {
	return "worker_calls"
}

func (workerCall *workerCall) Call() (*types.Call, error) {
	uid, err := uuid.Parse(workerCall.WorkId)
	if err != nil {
		return nil, err
	}
	return &types.Call{
		ID: types.CallID{
			Sector: abi.SectorID{
				Miner:  abi.ActorID(workerCall.MinerID),
				Number: abi.SectorNumber(workerCall.SectorId),
			},
			ID: uid,
		},
		RetType: types.ReturnType(workerCall.RetType),
		State:   types.CallState(workerCall.State),
		Result:  types.NewManyBytes(workerCall.Result),
	}, nil
}

var _ repo.WorkerCallRepo = (*workerCallRepo)(nil)

type workerCallRepo struct {
	*gorm.DB
}

func newWorkerCallRepo(db *gorm.DB) *workerCallRepo {
	return &workerCallRepo{DB: db}
}

func (w *workerCallRepo) GetCallByCallID(role string, callId types.CallID) (*types.Call, error) {
	var workerCall workerCall
	err := w.DB.Table("worker_calls").
		First(&workerCall,
			"miner_id=? AND sector_id=? AND work_id=? AND role=?",
			callId.Sector.Miner,
			callId.Sector.Number,
			callId.ID,
			role).Error
	if err != nil {
		return nil, err
	}
	return workerCall.Call()
}

func (w *workerCallRepo) HasCall(role string, callId types.CallID) (bool, error) {
	var count int64
	err := w.DB.Table("worker_calls").
		Where("miner_id=? AND sector_id=? AND work_id=? AND role=?",
			callId.Sector.Miner,
			callId.Sector.Number,
			callId.ID,
			role).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (w *workerCallRepo) Save(role string, callId types.CallID, call *types.Call) error {
	workerCall := &workerCall{
		Id:       uuid.New().String(),
		WorkId:   callId.ID.String(),
		Role:     role,
		MinerID:  uint64(callId.Sector.Miner),
		SectorId: uint64(callId.Sector.Number),
		RetType:  string(call.RetType),
		State:    uint64(call.State),
		Result:   call.Result.Bytes(),
	}

	return w.DB.Create(&workerCall).Error
}

func (w *workerCallRepo) GetAllCall(role string) ([]*types.Call, error) {
	var workerCalls []*workerCall
	err := w.DB.Table("worker_calls").Find(&workerCalls, "role=?", role).Error
	if err != nil {
		return nil, err
	}
	result := make([]*types.Call, len(workerCalls))
	for index, st := range workerCalls {
		newSt, err := st.Call()
		if err != nil {
			return nil, err
		}
		result[index] = newSt
	}
	return result, nil
}

func (w *workerCallRepo) DeleteByCallID(role string, callId types.CallID) error {
	return w.DB.Delete(&workerCall{},
		"miner_id=? AND sector_id=? AND work_id=? AND role=?",
		callId.Sector.Miner,
		callId.Sector.Number,
		callId.ID,
		role).Error
}

func (w *workerCallRepo) UpdateCallByCallID(role string, call *types.Call, callId types.CallID) error {
	updateClause := map[string]interface{}{
		"ret_type": call.RetType,
		"state":    call.State,
		"result":   call.Result.Bytes(),
	}
	return w.DB.Table("worker_calls").
		Where("miner_id=? AND sector_id=? AND work_id=? AND role=?",
			callId.Sector.Miner,
			callId.Sector.Number,
			callId.ID,
			role).
		UpdateColumns(updateClause).Error
}
