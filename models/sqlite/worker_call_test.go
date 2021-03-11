package sqlite

import (
	"bytes"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/venus-sealer/types"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"os"
	"testing"
)

func setupWorkerCallRef(suffix string, t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("./workcall_"+suffix), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info), // 日志配置
	})
	if err != nil {
		t.Error(err)
	}
	err = db.AutoMigrate(&workerCall{})
	if err != nil {
		t.Error(err)
	}
	return db
}

func cleanWorkerCall(suffix string, t *testing.T) {
	os.Remove("./workcall_" + suffix)
}

func Test_workerCallRepo_DeleteByCallIDButNotFound(t *testing.T) {
	db := setupWorkerCallRef("delete_by_call_id_notfound", t)
	defer cleanWorkerCall("delete_by_call_id_notfound", t)
	dRepo := newWorkerCallRepo(db)

	uid := uuid.New()
	wid := uuid.New()
	dRepo.Create(&workerCall{
		Id:       uid.String(),
		WorkId:   wid.String(),
		MinerID:  12,
		SectorId: 12,
		Role:     "worker",
		RetType:  "11",
		State:    0,
		Result:   []byte{1, 2, 3, 4},
	})

	uid2 := uuid.New()
	wid2 := uuid.New()
	dRepo.Create(&workerCall{
		Id:       uid2.String(),
		WorkId:   wid2.String(),
		MinerID:  13,
		SectorId: 13,
		Role:     "sealer",
		RetType:  "13",
		State:    0,
		Result:   []byte{2, 2, 2, 2},
	})

	err := dRepo.DeleteByCallID("sealer", types.CallID{
		Sector: abi.SectorID{
			Miner:  abi.ActorID(13),
			Number: 13,
		},
		ID: wid,
	})
	if err != nil {
		t.Error(err)
	}
	var calls []workerCall
	err = db.Find(&calls).Error
	if err != nil {
		t.Error(err)
	}

	if len(calls) != 2 {
		t.Errorf("delete should not work, expect %d calls but got %d", 2, len(calls))
	}
}

func Test_workerCallRepo_DeleteByCallID(t *testing.T) {
	db := setupWorkerCallRef("delete_by_call_id", t)
	defer cleanWorkerCall("delete_by_call_id", t)
	dRepo := newWorkerCallRepo(db)

	uid := uuid.New()
	wid := uuid.New()
	dRepo.Create(&workerCall{
		Id:       uid.String(),
		WorkId:   wid.String(),
		MinerID:  12,
		SectorId: 12,
		Role:     "worker",
		RetType:  "11",
		State:    0,
		Result:   []byte{1, 2, 3, 4},
	})

	uid2 := uuid.New()
	wid2 := uuid.New()
	dRepo.Create(&workerCall{
		Id:       uid2.String(),
		WorkId:   wid2.String(),
		MinerID:  13,
		SectorId: 13,
		Role:     "sealer",
		RetType:  "13",
		State:    0,
		Result:   []byte{2, 2, 2, 2},
	})

	err := dRepo.DeleteByCallID("sealer", types.CallID{
		Sector: abi.SectorID{
			Miner:  abi.ActorID(13),
			Number: 13,
		},
		ID: wid2,
	})
	if err != nil {
		t.Error(err)
	}
	var calls []workerCall
	err = db.Find(&calls).Error
	if err != nil {
		t.Error(err)
	}

	if len(calls) != 1 {
		t.Errorf("delete worked, expect %d calls but got %d", 2, len(calls))
	}
	if calls[0].Id != uid.String() {
		t.Errorf("delete worked, expect %s calls but got %s", uid.String(), calls[0].Id)
	}
}

func Test_workerCallRepo_GetAllCall(t *testing.T) {
	db := setupWorkerCallRef("get_all_call", t)
	defer cleanWorkerCall("get_all_call", t)
	dRepo := newWorkerCallRepo(db)

	uid := uuid.New()
	wid := uuid.New()
	dRepo.Create(&workerCall{
		Id:       uid.String(),
		WorkId:   wid.String(),
		MinerID:  12,
		SectorId: 12,
		Role:     "worker",
		RetType:  "11",
		State:    0,
		Result:   []byte{1, 2, 3, 4},
	})

	uid2 := uuid.New()
	wid2 := uuid.New()
	dRepo.Create(&workerCall{
		Id:       uid2.String(),
		WorkId:   wid2.String(),
		MinerID:  13,
		SectorId: 13,
		Role:     "sealer",
		RetType:  "13",
		State:    0,
		Result:   []byte{2, 2, 2, 2},
	})

	uid3 := uuid.New()
	wid3 := uuid.New()
	dRepo.Create(&workerCall{
		Id:       uid3.String(),
		WorkId:   wid3.String(),
		MinerID:  14,
		SectorId: 14,
		Role:     "sealer",
		RetType:  "14",
		State:    0,
		Result:   []byte{3, 3, 3, 3},
	})

	calls, err := dRepo.GetAllCall("sealer")
	if err != nil {
		t.Error(err)
	}

	if len(calls) != 2 {
		t.Errorf("expect %d calls but got %d", 2, len(calls))
	}
}

func Test_workerCallRepo_GetCallByCallID(t *testing.T) {
	db := setupWorkerCallRef("get_call_by_call_id", t)
	defer cleanWorkerCall("get_call_by_call_id", t)
	dRepo := newWorkerCallRepo(db)

	uid := uuid.New()
	wid := uuid.New()
	dRepo.Create(&workerCall{
		Id:       uid.String(),
		WorkId:   wid.String(),
		MinerID:  12,
		SectorId: 12,
		Role:     "worker",
		RetType:  "11",
		State:    0,
		Result:   []byte{1, 2, 3, 4},
	})

	uid2 := uuid.New()
	wid2 := uuid.New()
	dRepo.Create(&workerCall{
		Id:       uid2.String(),
		WorkId:   wid2.String(),
		MinerID:  13,
		SectorId: 13,
		Role:     "sealer",
		RetType:  "13",
		State:    0,
		Result:   []byte{2, 2, 2, 2},
	})

	call, err := dRepo.GetCallByCallID("worker", types.CallID{
		Sector: abi.SectorID{
			Miner:  abi.ActorID(12),
			Number: 12,
		},
		ID: wid,
	})

	if err != nil {
		t.Error(err)
	}

	if !bytes.Equal(call.Result.Bytes(), []byte{1, 2, 3, 4}) {
		t.Errorf("query fail, expect [2,2,2,2] but got %v", call.Result.Bytes())
	}
}

func Test_workerCallRepo_HasCall(t *testing.T) {
	db := setupWorkerCallRef("has_call", t)
	defer cleanWorkerCall("has_call", t)
	dRepo := newWorkerCallRepo(db)

	uid := uuid.New()
	wid := uuid.New()
	dRepo.Create(&workerCall{
		Id:       uid.String(),
		WorkId:   wid.String(),
		MinerID:  12,
		SectorId: 12,
		Role:     "worker",
		RetType:  "11",
		State:    0,
		Result:   []byte{1, 2, 3, 4},
	})

	uid2 := uuid.New()
	wid2 := uuid.New()
	dRepo.Create(&workerCall{
		Id:       uid2.String(),
		WorkId:   wid2.String(),
		MinerID:  13,
		SectorId: 13,
		Role:     "sealer",
		RetType:  "13",
		State:    0,
		Result:   []byte{2, 2, 2, 2},
	})

	{
		has, err := dRepo.HasCall("worker", types.CallID{
			Sector: abi.SectorID{
				Miner:  abi.ActorID(12),
				Number: 12,
			},
			ID: wid,
		})

		if err != nil {
			t.Error(err)
		}

		if !has {
			t.Errorf("should find call %s, but not ", uid.String())
		}
	}

	{
		has, err := dRepo.HasCall("sealer", types.CallID{
			Sector: abi.SectorID{
				Miner:  abi.ActorID(13),
				Number: 13,
			},
			ID: wid2,
		})

		if err != nil {
			t.Error(err)
		}

		if !has {
			t.Errorf("should find call %s, but not ", uid2.String())
		}
	}

	{
		has, err := dRepo.HasCall("worker", types.CallID{
			Sector: abi.SectorID{
				Miner:  abi.ActorID(13),
				Number: 13,
			},
			ID: uid2,
		})

		if err != nil {
			t.Error(err)
		}

		if has {
			t.Errorf("expect not find call %s, but not ", uid.String())
		}
	}
}

func Test_workerCallRepo_Save(t *testing.T) {
	db := setupWorkerCallRef("save_call", t)
	defer cleanWorkerCall("save_call", t)
	dRepo := newWorkerCallRepo(db)

	wid := uuid.New()
	callId := types.CallID{
		Sector: abi.SectorID{
			Miner:  20,
			Number: 20,
		},
		ID: wid,
	}

	err := dRepo.Save("worker", callId, &types.Call{
		ID:      callId,
		RetType: "xxxxxxx",
		State:   2,
		Result:  types.NewManyBytes([]byte{2, 2, 2, 2, 2}),
	})

	if err != nil {
		t.Error(err)
	}

	var calls []workerCall
	err = db.Find(&calls).Error
	if err != nil {
		t.Error(err)
	}

	if len(calls) != 1 {
		t.Errorf("expect calls %d, but got %d", 1, len(calls))
	}
	if !(calls[0].Role == "worker" &&
		bytes.Equal(calls[0].Result, []byte{2, 2, 2, 2, 2}) &&
		calls[0].MinerID == 20) {
		t.Errorf("save call incorrectly")
	}
}

func Test_workerCallRepo_UpdateCallByCallID(t *testing.T) {
	db := setupWorkerCallRef("update_call_by_call_id", t)
	defer cleanWorkerCall("update_call_by_call_id", t)
	dRepo := newWorkerCallRepo(db)

	wid := uuid.New()
	callId := types.CallID{
		Sector: abi.SectorID{
			Miner:  12,
			Number: 12,
		},
		ID: wid,
	}
	err := dRepo.Save("worker", callId, &types.Call{
		ID:      callId,
		RetType: "xxxxxxx",
		State:   2,
		Result:  types.NewManyBytes([]byte{2, 2, 2, 2, 2}),
	})
	if err != nil {
		t.Error(err)
	}
	err = dRepo.UpdateCallByCallID("worker", &types.Call{
		ID:      callId,
		RetType: "yyyyyy",
		State:   3,
		Result:  types.NewManyBytes([]byte{3, 3, 3, 3, 3}),
	}, callId)
	if err != nil {
		t.Error(err)
	}

	var calls []workerCall
	err = db.Find(&calls).Error
	if err != nil {
		t.Error(err)
	}

	if len(calls) != 1 {
		t.Errorf("expect calls %d, but got %d", 1, len(calls))
	}
	if !(calls[0].Role == "worker" &&
		bytes.Equal(calls[0].Result, []byte{3, 3, 3, 3, 3}) &&
		calls[0].RetType == "yyyyyy") {
		t.Errorf("update call incorrectly")
	}
}
