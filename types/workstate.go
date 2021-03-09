package types

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"golang.org/x/xerrors"
)

type WorkID struct {
	Method TaskType
	Params string // json [...params]
}

func (w WorkID) String() string {
	return fmt.Sprintf("%s(%s)", w.Method, w.Params)
}

var _ fmt.Stringer = &WorkID{}

type WorkStatus string

const (
	WsStarted WorkStatus = "started" // task started, not scheduled/running on a worker yet
	WsRunning WorkStatus = "running" // task running on a worker, waiting for worker return
	WsDone    WorkStatus = "done"    // task returned from the worker, results available
)

type WorkState struct {
	ID WorkID

	Status WorkStatus

	WorkerCall CallID // Set when entering WsRunning
	WorkError  string // Status = WsDone, set when failed to start work

	WorkerHostname string // hostname of last worker handling this job
	StartTime      int64  // unix seconds
}

func NewWorkID(method TaskType, params ...interface{}) (WorkID, error) {
	pb, err := json.Marshal(params)
	if err != nil {
		return WorkID{}, xerrors.Errorf("marshaling work params: %w", err)
	}

	if len(pb) > 256 {
		s := sha256.Sum256(pb)
		pb = []byte(hex.EncodeToString(s[:]))
	}

	return WorkID{
		Method: method,
		Params: string(pb),
	}, nil
}
