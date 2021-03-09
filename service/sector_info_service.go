package service

import (
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-statestore"
	"github.com/filecoin-project/venus-sealer/models/repo"
	"github.com/filecoin-project/venus-sealer/types"
	cbg "github.com/whyrusleeping/cbor-gen"
	"golang.org/x/xerrors"
	"reflect"
)

var _ statestore.StateStore = (*SectorInfoService)(nil)

type SectorInfoService struct {
	repo.SectorInfoRepo
	repo.LogRepo
}

func NewSectorInfoService(repo repo.Repo) *SectorInfoService {
	return &SectorInfoService{
		SectorInfoRepo: repo.SectorInfoRepo(),
		LogRepo:        repo.LogRepo(),
	}
}

//i abi.SectorId
//state *types.SectorInfo
func (sectorInfoService *SectorInfoService) Begin(i interface{}, state interface{}) error {
	sectorId := i.(uint64)
	sectorInfo := state.(*types.SectorInfo)
	has, err := sectorInfoService.SectorInfoRepo.HasSectorInfo(sectorId)
	if err != nil {
		return err
	}
	if has {
		return xerrors.Errorf("already tracking state for %v", i)
	}
	sectorInfo.SectorNumber = abi.SectorNumber(sectorId)
	return sectorInfoService.SectorInfoRepo.Save(sectorInfo)
}

func (sectorInfoService *SectorInfoService) Get(i interface{}) statestore.StoredState {
	return NewSectorInfoStoredState(sectorInfoService.SectorInfoRepo, sectorInfoService.LogRepo, i.(uint64))
}

func (sectorInfoService *SectorInfoService) Has(i interface{}) (bool, error) {
	return sectorInfoService.SectorInfoRepo.HasSectorInfo(i.(uint64))
}

func (sectorInfoService *SectorInfoService) List(out interface{}) error {
	sectors, err := sectorInfoService.SectorInfoRepo.GetAllSectorInfos()
	if err != nil {
		return err
	}
	//todo check success
	outSectors := make([]types.SectorInfo, len(sectors))
	for index, sector := range sectors {
		outSectors[index] = *sector
	}
	reflect.ValueOf(out).Elem().Set(reflect.ValueOf(outSectors))
	return nil
}

type SectorInfoStoredState struct {
	repo.SectorInfoRepo
	repo.LogRepo
	sectorId uint64
}

func NewSectorInfoStoredState(sectorInfoRepo repo.SectorInfoRepo, logRepo repo.LogRepo, sectorId uint64) *SectorInfoStoredState {
	return &SectorInfoStoredState{SectorInfoRepo: sectorInfoRepo, LogRepo: logRepo, sectorId: sectorId}
}

func (w *SectorInfoStoredState) End() error {
	has, err := w.SectorInfoRepo.HasSectorInfo(w.sectorId)
	if err != nil {
		return err
	}
	if !has {
		return xerrors.Errorf("No state for %s", w.sectorId)
	}

	if err := w.SectorInfoRepo.DeleteBySectorId(w.sectorId); err != nil {
		return xerrors.Errorf("removing state from datastore: %w", err)
	}
	if err := w.LogRepo.DelLogs(w.sectorId); err != nil {
		return xerrors.Errorf("removing logs from datastore: %w", err)
	}
	return nil
}

func (w *SectorInfoStoredState) Get(out cbg.CBORUnmarshaler) error {
	val, err := w.SectorInfoRepo.GetSectorInfoByID(w.sectorId)
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
func (st *SectorInfoStoredState) Mutate(mutator interface{}) error {
	has, err := st.SectorInfoRepo.HasSectorInfo(st.sectorId)
	if err != nil {
		return err
	}
	if !has {
		return xerrors.Errorf("No state for %s", st.sectorId)
	}

	cur, err := st.SectorInfoRepo.GetSectorInfoByID(st.sectorId)
	if err != nil {
		return err
	}

	rmut := reflect.ValueOf(mutator)
	out := rmut.Call([]reflect.Value{reflect.ValueOf(cur)})
	if err := out[0].Interface(); err != nil {
		return err.(error)
	}
	return st.SectorInfoRepo.UpdateSectorInfoBySectorId(cur, st.sectorId)
}
