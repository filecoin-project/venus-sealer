package storiface

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/specs-storage/storage"

	"github.com/filecoin-project/venus-sealer/types"
)

type WorkerID uuid.UUID // worker session UUID

func (w WorkerID) String() string {
	return uuid.UUID(w).String()
}

type WorkerInfo struct {
	Hostname string

	// IgnoreResources indicates whether the worker's available resources should
	// be used ignored (true) or used (false) for the purposes of scheduling and
	// task assignment. Only supported on local workers. Used for testing.
	// Default should be false (zero value, i.e. resources taken into account).
	IgnoreResources bool
	Resources       WorkerResources
}

type WorkerResources struct {
	MemPhysical uint64
	MemUsed     uint64
	MemSwap     uint64
	MemSwapUsed uint64

	CPUs uint64 // Logical cores
	GPUs []string

	// if nil use the default resource table
	Resources map[types.TaskType]map[abi.RegisteredSealProof]Resources
}

func (wr WorkerResources) ResourceSpec(spt abi.RegisteredSealProof, tt types.TaskType) Resources {
	res := ResourceTable[tt][spt]

	// if the worker specifies custom resource table, prefer that
	if wr.Resources != nil {
		tr, ok := wr.Resources[tt]
		if !ok {
			return res
		}

		r, ok := tr[spt]
		if ok {
			return r
		}
	}

	// otherwise, use the default resource table
	return res
}

type WorkerStats struct {
	Info    WorkerInfo
	Enabled bool

	MemUsedMin uint64
	MemUsedMax uint64
	GpuUsed    float64 // nolint
	CpuUse     uint64  // nolint
}

const (
	RWPrepared = 1
	RWRunning  = 0
	RWRetWait  = -1
	RWReturned = -2
	RWRetDone  = -3
)

type WorkerJob struct {
	ID     types.CallID
	Sector abi.SectorID
	Task   types.TaskType

	// 2+ - assigned
	// 1  - prepared
	// 0  - running
	// -1 - ret-wait
	// -2 - returned
	// -3 - ret-done
	RunWait int
	Start   time.Time

	Hostname string `json:",omitempty"` // optional, set for ret-wait jobs
}

type WorkerCalls interface {
	AddPiece(ctx context.Context, sector storage.SectorRef, pieceSizes []abi.UnpaddedPieceSize, newPieceSize abi.UnpaddedPieceSize, pieceData storage.Data) (types.CallID, error)
	SealPreCommit1(ctx context.Context, sector storage.SectorRef, ticket abi.SealRandomness, pieces []abi.PieceInfo) (types.CallID, error)
	SealPreCommit2(ctx context.Context, sector storage.SectorRef, pc1o storage.PreCommit1Out) (types.CallID, error)
	SealCommit1(ctx context.Context, sector storage.SectorRef, ticket abi.SealRandomness, seed abi.InteractiveSealRandomness, pieces []abi.PieceInfo, cids storage.SectorCids) (types.CallID, error)
	SealCommit2(ctx context.Context, sector storage.SectorRef, c1o storage.Commit1Out) (types.CallID, error)
	FinalizeSector(ctx context.Context, sector storage.SectorRef, keepUnsealed []storage.Range) (types.CallID, error)
	ReleaseUnsealed(ctx context.Context, sector storage.SectorRef, safeToFree []storage.Range) (types.CallID, error)
	ReplicaUpdate(ctx context.Context, sector storage.SectorRef, pieces []abi.PieceInfo) (types.CallID, error)
	ProveReplicaUpdate1(ctx context.Context, sector storage.SectorRef, sectorKey, newSealed, newUnsealed cid.Cid) (types.CallID, error)
	ProveReplicaUpdate2(ctx context.Context, sector storage.SectorRef, sectorKey, newSealed, newUnsealed cid.Cid, vanillaProofs storage.ReplicaVanillaProofs) (types.CallID, error)
	GenerateSectorKeyFromData(ctx context.Context, sector storage.SectorRef, commD cid.Cid) (types.CallID, error)
	MoveStorage(ctx context.Context, sector storage.SectorRef, types SectorFileType) (types.CallID, error)
	UnsealPiece(context.Context, storage.SectorRef, UnpaddedByteIndex, abi.UnpaddedPieceSize, abi.SealRandomness, cid.Cid) (types.CallID, error)
	Fetch(context.Context, storage.SectorRef, SectorFileType, PathType, AcquireMode) (types.CallID, error)
}

type ErrorCode int

const (
	ErrUnknown ErrorCode = iota
)

const (
	// Temp Errors
	ErrTempUnknown ErrorCode = iota + 100
	ErrTempWorkerRestart
	ErrTempAllocateSpace
)

type CallError struct {
	Code    ErrorCode
	Message string
	sub     error
}

func (c *CallError) Error() string {
	return fmt.Sprintf("storage call error %d: %s", c.Code, c.Message)
}

func (c *CallError) Unwrap() error {
	if c.sub != nil {
		return c.sub
	}

	return errors.New(c.Message)
}

func Err(code ErrorCode, sub error) *CallError {
	return &CallError{
		Code:    code,
		Message: sub.Error(),

		sub: sub,
	}
}

type WorkerReturn interface {
	ReturnAddPiece(ctx context.Context, callID types.CallID, pi abi.PieceInfo, err *CallError) error
	ReturnSealPreCommit1(ctx context.Context, callID types.CallID, p1o storage.PreCommit1Out, err *CallError) error
	ReturnSealPreCommit2(ctx context.Context, callID types.CallID, sealed storage.SectorCids, err *CallError) error
	ReturnSealCommit1(ctx context.Context, callID types.CallID, out storage.Commit1Out, err *CallError) error
	ReturnSealCommit2(ctx context.Context, callID types.CallID, proof storage.Proof, err *CallError) error
	ReturnFinalizeSector(ctx context.Context, callID types.CallID, err *CallError) error
	ReturnReleaseUnsealed(ctx context.Context, callID types.CallID, err *CallError) error
	ReturnReplicaUpdate(ctx context.Context, callID types.CallID, out storage.ReplicaUpdateOut, err *CallError) error
	ReturnProveReplicaUpdate1(ctx context.Context, callID types.CallID, proofs storage.ReplicaVanillaProofs, err *CallError) error
	ReturnProveReplicaUpdate2(ctx context.Context, callID types.CallID, proof storage.ReplicaUpdateProof, err *CallError) error
	ReturnGenerateSectorKeyFromData(ctx context.Context, callID types.CallID, err *CallError) error
	ReturnMoveStorage(ctx context.Context, callID types.CallID, err *CallError) error
	ReturnUnsealPiece(ctx context.Context, callID types.CallID, err *CallError) error
	ReturnReadPiece(ctx context.Context, callID types.CallID, ok bool, err *CallError) error
	ReturnFetch(ctx context.Context, callID types.CallID, err *CallError) error
}
