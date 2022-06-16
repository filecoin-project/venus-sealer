package types

import (
	"context"

	"github.com/filecoin-project/go-state-types/builtin/v8/market"
	"github.com/filecoin-project/go-state-types/builtin/v8/miner"
	"github.com/filecoin-project/venus/venus-shared/types/messager"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/specs-storage/storage"
)

type SectorPreCommitInfo = miner.SectorPreCommitInfo

type PieceWithDealInfo struct {
	Piece    abi.PieceInfo
	DealInfo PieceDealInfo
}

type DealSchedule struct {
	StartEpoch abi.ChainEpoch
	EndEpoch   abi.ChainEpoch
}

type PieceDealInfo struct {
	PublishCid   *cid.Cid
	DealID       abi.DealID
	DealProposal *market.DealProposal
	DealSchedule DealSchedule
	KeepUnsealed bool
}

// Piece is a tuple of piece info and optional deal
type Piece struct {
	Piece    abi.PieceInfo
	DealInfo *PieceDealInfo // nil for pieces which do not appear in deals (e.g. filler pieces)
}

type SectorPiece struct {
	Piece    abi.PieceInfo
	DealInfo *PieceDealInfo // nil for pieces which do not appear in deals (e.g. filler pieces)
}

type Log struct {
	Timestamp uint64
	Trace     string // for errors

	Message string

	// additional data (Event info)
	Kind string
}

type ReturnState string

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
	PreCommitMessage *cid.Cid
	PreCommitTipSet  TipSetToken

	PreCommit2Fails uint64

	// WaitSeed
	SeedValue abi.InteractiveSealRandomness
	SeedEpoch abi.ChainEpoch

	// Committing
	CommitMessage *cid.Cid
	InvalidProofs uint64 // failed proof computations (doesn't validate with proof inputs; can't compute)

	// CCUpdate
	CCUpdate             bool
	CCPieces             []Piece
	UpdateSealed         *cid.Cid
	UpdateUnsealed       *cid.Cid
	ReplicaUpdateProof   storage.ReplicaUpdateProof
	ReplicaUpdateMessage *cid.Cid

	// Faults
	FaultReportMsg *cid.Cid

	// Recovery
	Return ReturnState

	// Termination
	TerminateMessage *cid.Cid
	TerminatedAt     abi.ChainEpoch

	// Debug
	LastErr string

	Log []Log
}

type msgCidGetter interface {
	GetMessageByUid(context.Context, string) (*messager.Message, error)
}

func (t *SectorInfo) pieceInfos() []abi.PieceInfo {
	out := make([]abi.PieceInfo, len(t.Pieces))
	for i, p := range t.Pieces {
		out[i] = p.Piece
	}
	return out
}

func (t *SectorInfo) dealIDs() []abi.DealID {
	out := make([]abi.DealID, 0, len(t.Pieces))
	for _, p := range t.Pieces {
		if p.DealInfo == nil {
			continue
		}
		out = append(out, p.DealInfo.DealID)
	}
	return out
}

func (t *SectorInfo) existingPieceSizes() []abi.UnpaddedPieceSize {
	out := make([]abi.UnpaddedPieceSize, len(t.Pieces))
	for i, p := range t.Pieces {
		out[i] = p.Piece.Size.Unpadded()
	}
	return out
}

func (t *SectorInfo) hasDeals() bool {
	for _, piece := range t.Pieces {
		if piece.DealInfo != nil {
			return true
		}
	}
	return false
}

// Returns list of offset/length tuples of sector data ranges which clients
// requested to keep unsealed
func (t *SectorInfo) keepUnsealedRanges(invert, alwaysKeep bool) []storage.Range {
	var out []storage.Range

	var at abi.UnpaddedPieceSize
	for _, piece := range t.Pieces {
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

type TipSetToken []byte
