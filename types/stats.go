package types

import (
	"sync"

	"github.com/filecoin-project/go-state-types/abi"

	"github.com/filecoin-project/venus-sealer/storage-sealing/sealiface"
)

type StatSectorState int

const (
	SstStaging StatSectorState = iota
	SstSealing
	SstFailed
	SstProving
	Nsst
)

type SectorStats struct {
	lk sync.Mutex

	BySector map[abi.SectorID]StatSectorState
	Totals   [Nsst]uint64
}

func (ss *SectorStats) UpdateSector(cfg sealiface.Config, id abi.SectorID, st SectorState) (updateInput bool) {
	ss.lk.Lock()
	defer ss.lk.Unlock()

	preSealing := ss.curSealingLocked()
	preStaging := ss.curStagingLocked()

	// update totals
	oldst, found := ss.BySector[id]
	if found {
		ss.Totals[oldst]--
	}

	sst := toStatState(st, cfg.FinalizeEarly)
	ss.BySector[id] = sst
	ss.Totals[sst]++

	// check if we may need be able to process more deals
	sealing := ss.curSealingLocked()
	staging := ss.curStagingLocked()

	if cfg.MaxSealingSectorsForDeals > 0 && // max sealing deal sector limit set
		preSealing >= cfg.MaxSealingSectorsForDeals && // we were over limit
		sealing < cfg.MaxSealingSectorsForDeals { // and we're below the limit now
		updateInput = true
	}

	if cfg.MaxWaitDealsSectors > 0 && // max waiting deal sector limit set
		preStaging >= cfg.MaxWaitDealsSectors && // we were over limit
		staging < cfg.MaxWaitDealsSectors { // and we're below the limit now
		updateInput = true
	}

	return updateInput
}

func (ss *SectorStats) curSealingLocked() uint64 {
	return ss.Totals[SstStaging] + ss.Totals[SstSealing] + ss.Totals[SstFailed]
}

func (ss *SectorStats) curStagingLocked() uint64 {
	return ss.Totals[SstStaging]
}

// return the number of sectors currently in the sealing pipeline
func (ss *SectorStats) CurSealing() uint64 {
	ss.lk.Lock()
	defer ss.lk.Unlock()

	return ss.curSealingLocked()
}

// return the number of sectors waiting to enter the sealing pipeline
func (ss *SectorStats) CurStaging() uint64 {
	ss.lk.Lock()
	defer ss.lk.Unlock()

	return ss.curStagingLocked()
}
