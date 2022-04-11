package types

import (
	"bytes"
	"context"

	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-state-types/exitcode"
	"github.com/filecoin-project/specs-storage/storage"

	"github.com/filecoin-project/venus/venus-shared/actors/builtin/miner"
	"github.com/filecoin-project/venus/venus-shared/actors/policy"

	"github.com/filecoin-project/specs-actors/v2/actors/builtin/market"

	"github.com/filecoin-project/venus-sealer/storage-sealing/sealiface"
)

var DealSectorPriority = 1024
var MaxTicketAge = policy.MaxPreCommitRandomnessLookback

// Context is a go-statemachine context
type Context interface {
	Context() context.Context
	Send(evt interface{}) error
}

// DealInfo is a tuple of deal identity and its schedule
type PieceDealInfo struct {
	PublishCid   *cid.Cid
	DealID       abi.DealID
	DealProposal *market.DealProposal
	DealSchedule DealSchedule
	KeepUnsealed bool
}

// DealSchedule communicates the time interval of a storage deal. The deal must
// appear in a sealed (proven) sector no later than StartEpoch, otherwise it
// is invalid.
type DealSchedule struct {
	StartEpoch abi.ChainEpoch
	EndEpoch   abi.ChainEpoch
}

// Piece is a tuple of piece and deal info
type PieceWithDealInfo struct {
	Piece    abi.PieceInfo
	DealInfo PieceDealInfo
}

// Piece is a tuple of piece info and optional deal
type Piece struct {
	Piece    abi.PieceInfo
	DealInfo *PieceDealInfo // nil for pieces which do not appear in deals (e.g. filler pieces)
}

type Log struct {
	SectorNumber abi.SectorNumber
	Timestamp    uint64
	Trace        string // for errors

	Message string

	// additional data (Event info)
	Kind string
}

type ReturnState string

const (
	RetPreCommit1      = ReturnState(PreCommit1)
	RetPreCommitting   = ReturnState(PreCommitting)
	RetPreCommitFailed = ReturnState(PreCommitFailed)
	RetCommitFailed    = ReturnState(CommitFailed)
)

type SectorInfo struct {
	State        SectorState
	SectorNumber abi.SectorNumber

	SectorType abi.RegisteredSealProof

	// Packing
	CreationTime int64 // unix seconds
	Pieces       []Piece

	// PreCommit1
	TicketValue   abi.SealRandomness
	TicketEpoch   abi.ChainEpoch
	PreCommit1Out storage.PreCommit1Out

	// PreCommit2
	CommD *cid.Cid
	CommR *cid.Cid // SectorKey
	Proof []byte

	PreCommitInfo    *miner.SectorPreCommitInfo
	PreCommitDeposit big.Int
	PreCommitMessage string
	PreCommitTipSet  TipSetToken

	PreCommit2Fails uint64

	// WaitSeed
	SeedValue abi.InteractiveSealRandomness
	SeedEpoch abi.ChainEpoch

	// Committing
	CommitMessage string
	InvalidProofs uint64 // failed proof computations (doesn't validate with proof inputs; can't compute)

	// CCUpdate
	CCUpdate             bool
	CCPieces             []Piece
	UpdateSealed         *cid.Cid
	UpdateUnsealed       *cid.Cid
	ReplicaUpdateProof   storage.ReplicaUpdateProof
	ReplicaUpdateMessage string

	// Faults
	FaultReportMsg string

	// Recovery
	Return ReturnState

	// Termination
	TerminateMessage string
	TerminatedAt     abi.ChainEpoch

	// Debug
	LastErr string
}

func (t *SectorInfo) PieceInfos() []abi.PieceInfo {
	out := make([]abi.PieceInfo, len(t.Pieces))
	for i, p := range t.Pieces {
		out[i] = p.Piece
	}
	return out
}

func (t *SectorInfo) DealIDs() []abi.DealID {
	out := make([]abi.DealID, 0, len(t.Pieces))
	for _, p := range t.Pieces {
		if p.DealInfo == nil {
			continue
		}
		out = append(out, p.DealInfo.DealID)
	}
	return out
}

func (t *SectorInfo) ExistingPieceSizes() []abi.UnpaddedPieceSize {
	out := make([]abi.UnpaddedPieceSize, len(t.Pieces))
	for i, p := range t.Pieces {
		out[i] = p.Piece.Size.Unpadded()
	}
	return out
}

func (t *SectorInfo) HasDeals() bool {
	for _, piece := range t.Pieces {
		if piece.DealInfo != nil {
			return true
		}
	}

	return false
}

func (t *SectorInfo) hasDeals() bool {
	for _, piece := range t.Pieces {
		if piece.DealInfo != nil {
			return true
		}
	}

	return false
}

func (t *SectorInfo) SealingCtx(ctx context.Context) context.Context {
	// TODO: can also take start epoch into account to give priority to sectors
	//  we need sealed sooner

	if t.HasDeals() {
		return WithPriority(ctx, DealSectorPriority)
	}

	return ctx
}

// Returns list of offset/length tuples of sector data ranges which clients
// requested to keep unsealed
func (t *SectorInfo) KeepUnsealedRanges(pieces []Piece, invert, alwaysKeep bool) []storage.Range {
	var out []storage.Range

	var at abi.UnpaddedPieceSize
	for _, piece := range pieces {
		psize := piece.Piece.Size.Unpadded()
		at += psize

		if piece.DealInfo == nil {
			continue
		}

		keep := piece.DealInfo.KeepUnsealed || alwaysKeep

		if keep == invert {
			continue
		}

		out = append(out, storage.Range{
			Offset: at - psize,
			Size:   psize,
		})
	}

	return out
}

type SectorIDCounter interface {
	Next() (abi.SectorNumber, error)
	GetStorageCounter() (abi.SectorNumber, error)
}

type TipSetToken []byte

type MsgLookup struct {
	Receipt   MessageReceipt
	TipSetTok TipSetToken
	Height    abi.ChainEpoch
}

type MessageReceipt struct {
	ExitCode exitcode.ExitCode
	Return   []byte
	GasUsed  int64
}

type GetSealingConfigFunc func() (sealiface.Config, error)

func (mr *MessageReceipt) Equals(o *MessageReceipt) bool {
	return mr.ExitCode == o.ExitCode && bytes.Equal(mr.Return, o.Return) && mr.GasUsed == o.GasUsed
}

type DealAssign struct {
	DealId   abi.DealID
	SectorId abi.SectorNumber
	PieceCid cid.Cid
	Offset   abi.PaddedPieceSize
	Size     abi.PaddedPieceSize
}
