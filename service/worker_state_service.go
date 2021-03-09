package service

import (
	"github.com/filecoin-project/go-statestore"
	"github.com/filecoin-project/venus-sealer/models/repo"
	"github.com/filecoin-project/venus-sealer/types"
	cbg "github.com/whyrusleeping/cbor-gen"
	"golang.org/x/xerrors"
	"reflect"
)

var _ statestore.StateStore = (*WorkerStateService)(nil)
var _ statestore.StoredState = (*WorkerStoreStoredState)(nil)

type WorkerStateService struct {
	repo.WorkerStateRepo
}

func NewWorkStateService(repo repo.Repo) *WorkerStateService {
	return &WorkerStateService{WorkerStateRepo: repo.WorkerStateRepo()}
}

//i WorkID
//state WorkState
func (w WorkerStateService) Begin(i interface{}, state interface{}) error {
	key := i.(types.WorkID)
	st := state.(*types.WorkState)
	st.ID = key
	has, err := w.WorkerStateRepo.HasState(key)
	if err != nil {
		return err
	}
	if has {
		return xerrors.Errorf("already tracking state for %v", i)
	}

	return w.WorkerStateRepo.Save(key, st)
}

//i WorkID
func (w WorkerStateService) Get(i interface{}) statestore.StoredState {
	return NewWorkerStoreStoredState(w.WorkerStateRepo, i.(types.WorkID))
}

//i WorkID
func (w WorkerStateService) Has(i interface{}) (bool, error) {
	return w.WorkerStateRepo.HasState(i.(types.WorkID))
}

func (w WorkerStateService) List(out interface{}) error {
	states, err := w.WorkerStateRepo.GetAllWorkState()
	if err != nil {
		return err
	}
	//todo check success
	outStates := make([]types.WorkState, len(states))
	for index, state := range states {
		outStates[index] = *state
	}
	reflect.ValueOf(out).Elem().Set(reflect.ValueOf(outStates))
	return nil
}

type WorkerStoreStoredState struct {
	repo.WorkerStateRepo
	workeId types.WorkID
}

func NewWorkerStoreStoredState(workerStateRepo repo.WorkerStateRepo, workeId types.WorkID) *WorkerStoreStoredState {
	return &WorkerStoreStoredState{WorkerStateRepo: workerStateRepo, workeId: workeId}
}

func (w *WorkerStoreStoredState) End() error {
	has, err := w.WorkerStateRepo.HasState(w.workeId)
	if err != nil {
		return err
	}
	if !has {
		return xerrors.Errorf("No state for %s", w.workeId.String())
	}

	if err := w.WorkerStateRepo.DeleteByWorkID(w.workeId); err != nil {
		return xerrors.Errorf("removing state from datastore: %w", err)
	}
	return nil
}

func (w *WorkerStoreStoredState) Get(out cbg.CBORUnmarshaler) error {
	val, err := w.WorkerStateRepo.GetWorkerStateByWorkID(w.workeId)
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
func (st *WorkerStoreStoredState) Mutate(mutator interface{}) error {
	has, err := st.WorkerStateRepo.HasState(st.workeId)
	if err != nil {
		return err
	}
	if !has {
		return xerrors.Errorf("No state for %s", st.workeId)
	}

	cur, err := st.WorkerStateRepo.GetWorkerStateByWorkID(st.workeId)
	if err != nil {
		return err
	}

	rmut := reflect.ValueOf(mutator)
	out := rmut.Call([]reflect.Value{reflect.ValueOf(cur)})
	if err := out[0].Interface(); err != nil {
		return err.(error)
	}
	return st.WorkerStateRepo.UpdateStateByWorkID(cur, st.workeId)
}
