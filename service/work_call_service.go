package service

import (
	"github.com/filecoin-project/go-statestore"
	"github.com/filecoin-project/venus-sealer/models/repo"
	"github.com/filecoin-project/venus-sealer/types"
	cbg "github.com/whyrusleeping/cbor-gen"
	"golang.org/x/xerrors"
	"reflect"
)

var _ statestore.StateStore = (*WorkCallService)(nil)

type WorkCallService struct {
	repo.WorkerCallRepo
	role string //sealer/worker
}

func NewWorkCallService(repo repo.Repo, role string) *WorkCallService {
	return &WorkCallService{WorkerCallRepo: repo.WorkerCallRepo()}
}

//types.CallID
//types.Call
func (workCallService *WorkCallService) Begin(i interface{}, state interface{}) error {
	callId := i.(types.CallID)
	call := state.(*types.Call)
	call.ID = callId
	has, err := workCallService.WorkerCallRepo.HasCall(workCallService.role, callId)
	if err != nil {
		return err
	}
	if has {
		return xerrors.Errorf("already tracking state for %v", i)
	}

	return workCallService.WorkerCallRepo.Save(workCallService.role, callId, call)
}

func (workCallService *WorkCallService) Get(i interface{}) statestore.StoredState {
	return NewWorkerCallStoredState(workCallService.WorkerCallRepo, i.(types.CallID), workCallService.role)
}

func (workCallService *WorkCallService) Has(i interface{}) (bool, error) {
	return workCallService.WorkerCallRepo.HasCall(workCallService.role, i.(types.CallID))
}

func (workCallService *WorkCallService) List(out interface{}) error {
	calls, err := workCallService.WorkerCallRepo.GetAllCall(workCallService.role)
	if err != nil {
		return err
	}
	//todo check success
	outCalls := make([]types.Call, len(calls))
	for index, call := range calls {
		outCalls[index] = *call
	}
	reflect.ValueOf(out).Elem().Set(reflect.ValueOf(outCalls))
	return nil
}

type WorkerCallStoredState struct {
	repo.WorkerCallRepo
	callId types.CallID
	role   string
}

func NewWorkerCallStoredState(workerCallRepo repo.WorkerCallRepo, callId types.CallID, role string) *WorkerCallStoredState {
	return &WorkerCallStoredState{WorkerCallRepo: workerCallRepo, callId: callId, role: role}
}

func (w *WorkerCallStoredState) End() error {
	has, err := w.WorkerCallRepo.HasCall(w.role, w.callId)
	if err != nil {
		return err
	}
	if !has {
		return xerrors.Errorf("No state for %s", w.callId)
	}

	if err := w.WorkerCallRepo.DeleteByCallID(w.role, w.callId); err != nil {
		return xerrors.Errorf("removing state from datastore: %w", err)
	}
	return nil
}

func (w *WorkerCallStoredState) Get(out cbg.CBORUnmarshaler) error {
	val, err := w.WorkerCallRepo.GetCallByCallID(w.role, w.callId)
	if err != nil {
		return err
	}

	//todo check success
	rVal := reflect.ValueOf(out)
	if rVal.Kind() == reflect.Ptr {
		reflect.ValueOf(out).Elem().Set(reflect.ValueOf(*val))
	} else {
		reflect.ValueOf(out).Set(reflect.ValueOf(*val))
	}
	return nil
}

// mutator func(*T) error
func (w *WorkerCallStoredState) Mutate(mutator interface{}) error {
	has, err := w.WorkerCallRepo.HasCall(w.role, w.callId)
	if err != nil {
		return err
	}
	if !has {
		return xerrors.Errorf("No state for %s", w.callId)
	}

	cur, err := w.WorkerCallRepo.GetCallByCallID(w.role, w.callId)
	if err != nil {
		return err
	}

	rmut := reflect.ValueOf(mutator)
	out := rmut.Call([]reflect.Value{reflect.ValueOf(cur)})
	if err := out[0].Interface(); err != nil {
		return err.(error)
	}
	return w.WorkerCallRepo.UpdateCallByCallID(w.role, cur, w.callId)
}
