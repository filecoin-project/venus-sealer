package sealiface

import (
	"github.com/filecoin-project/go-state-types/abi"
)

type CommitBatchRes struct {
	Sectors []abi.SectorNumber

	FailedSectors map[abi.SectorNumber]string

	Msg   string
	Error string // if set, means that all sectors are failed, implies Msg==nil
}

type PreCommitBatchRes struct {
	Sectors []abi.SectorNumber

	Msg   string
	Error string // if set, means that all sectors are failed, implies Msg==nil
}
