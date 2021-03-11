package types

import (
	"sync"

	"github.com/filecoin-project/go-state-types/abi"
)

type StatSectorState int

const (
	SstSealing StatSectorState = iota
	SstFailed
	SstProving
	Nsst
)

type SectorStats struct {
	lk sync.Mutex

	BySector map[abi.SectorID]StatSectorState
	Totals   [Nsst]uint64
}

func (ss *SectorStats) UpdateSector(id abi.SectorID, st SectorState) {
	ss.lk.Lock()
	defer ss.lk.Unlock()

	oldst, found := ss.BySector[id]
	if found {
		ss.Totals[oldst]--
	}

	sst := toStatState(st)
	ss.BySector[id] = sst
	ss.Totals[sst]++
}

// return the number of sectors currently in the sealing pipeline
func (ss *SectorStats) CurSealing() uint64 {
	ss.lk.Lock()
	defer ss.lk.Unlock()

	return ss.Totals[SstSealing] + ss.Totals[SstFailed]
}
