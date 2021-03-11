package types

import (
	"fmt"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/google/uuid"
)

type CallID struct {
	Sector abi.SectorID
	ID     uuid.UUID
}

func (c CallID) String() string {
	return fmt.Sprintf("%d-%d-%s", c.Sector.Miner, c.Sector.Number, c.ID)
}

var _ fmt.Stringer = &CallID{}

var UndefCall CallID
