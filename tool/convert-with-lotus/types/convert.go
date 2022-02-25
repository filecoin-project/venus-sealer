package types

import (
	"context"
	"github.com/filecoin-project/venus-sealer/types"
	"github.com/ipfs/go-cid"
	u "github.com/ipfs/go-ipfs-util"
)

func (p *PieceDealInfo) SetWithVenus(deal *types.PieceDealInfo) *PieceDealInfo {
	if deal == nil {
		return nil
	}
	p.PublishCid = deal.PublishCid
	p.DealID = deal.DealID
	p.DealProposal = &(*deal.DealProposal)
	p.DealSchedule.EndEpoch = deal.DealSchedule.EndEpoch
	p.DealSchedule.StartEpoch = deal.DealSchedule.StartEpoch
	p.KeepUnsealed = deal.KeepUnsealed
	return p
}

func (p *PieceDealInfo) ToVenus() *types.PieceDealInfo {
	if p == nil {
		return nil
	}
	return &types.PieceDealInfo{
		PublishCid:   p.PublishCid,
		DealID:       p.DealID,
		DealProposal: &(*p.DealProposal),
		DealSchedule: types.DealSchedule{
			StartEpoch: p.DealSchedule.StartEpoch,
			EndEpoch:   p.DealSchedule.EndEpoch,
		},
		KeepUnsealed: p.KeepUnsealed,
	}
}

func (t *SectorInfo) FromVenus(vSector *types.SectorInfo, getter msgCidGetter) *SectorInfo {
	var toChainCid = func(id string) *cid.Cid {
		undef := cid.NewCidV0(u.Hash([]byte("undef")))
		if len(id) == 0 || getter == nil {
			return &undef
		}
		if msg, err := getter.GetMessageByUid(context.TODO(), id); err == nil {
			return msg.SignedCid
		}
		return &undef
	}
	var copyPieces = func(vPieces []types.Piece) []Piece {
		if len(vPieces) == 0 {
			return nil
		}
		pieces := make([]Piece, len(vPieces))
		for idx, piece := range vPieces {
			pieces[idx].Piece = piece.Piece
			pieces[idx].DealInfo = (&PieceDealInfo{}).SetWithVenus(piece.DealInfo)
		}
		return pieces
	}
	t.State = SectorState(vSector.State)
	t.SectorNumber = vSector.SectorNumber
	t.SectorType = vSector.SectorType
	t.CreationTime = vSector.CreationTime
	t.Pieces = copyPieces(vSector.Pieces)
	// PreCommit1
	t.TicketValue = vSector.TicketValue
	t.TicketEpoch = vSector.TicketEpoch
	t.PreCommit1Out = vSector.PreCommit1Out
	// PreCommit2
	t.CommD = vSector.CommD
	t.CommR = vSector.CommR
	t.Proof = vSector.Proof
	t.PreCommitInfo = vSector.PreCommitInfo
	t.PreCommitDeposit = vSector.PreCommitDeposit
	t.PreCommitMessage = toChainCid(vSector.PreCommitMessage)
	t.PreCommitTipSet = TipSetToken(vSector.PreCommitTipSet)
	t.PreCommit2Fails = vSector.PreCommit2Fails
	// WaitSeed
	t.SeedValue = vSector.SeedValue
	t.SeedEpoch = vSector.SeedEpoch
	// Committing
	t.CommitMessage = toChainCid(vSector.CommitMessage)
	t.InvalidProofs = vSector.InvalidProofs
	// CCUpdate
	t.CCUpdate = vSector.CCUpdate
	t.CCPieces = copyPieces(vSector.CCPieces)
	t.UpdateSealed = vSector.UpdateSealed
	t.UpdateUnsealed = vSector.UpdateUnsealed
	t.ReplicaUpdateProof = vSector.ReplicaUpdateProof
	t.ReplicaUpdateMessage = toChainCid(vSector.ReplicaUpdateMessage)
	// Faults
	t.FaultReportMsg = toChainCid(vSector.FaultReportMsg)
	// Recovery
	t.Return = ReturnState(vSector.Return)
	// Termination
	t.TerminateMessage = toChainCid(vSector.TerminateMessage)
	t.TerminatedAt = vSector.TerminatedAt
	// Debug
	t.LastErr = vSector.LastErr
	//t.Log = copyLogs(vSector.Logs)
	return t
}

func (t *SectorInfo) ToVenus() *types.SectorInfo {
	var v types.SectorInfo

	var copyPieces = func(pieces []Piece) []types.Piece {
		if len(pieces) == 0 {
			return nil
		}
		vPieces := make([]types.Piece, len(pieces))
		for idx, piece := range pieces {
			vPieces[idx].Piece = piece.Piece
			vPieces[idx].DealInfo = piece.DealInfo.ToVenus()
		}
		return vPieces
	}

	v.State = types.SectorState(t.State)
	v.SectorNumber = t.SectorNumber
	v.SectorType = t.SectorType
	// Packing
	v.CreationTime = t.CreationTime
	v.Pieces = copyPieces(t.Pieces)
	// PreCommit1
	v.TicketValue = t.TicketValue
	v.TicketEpoch = t.TicketEpoch
	v.PreCommit1Out = t.PreCommit1Out
	// PreCommit2
	v.CommD = t.CommD
	v.CommR = t.CommR
	v.Proof = t.Proof
	v.PreCommitInfo = t.PreCommitInfo
	v.PreCommitDeposit = t.PreCommitDeposit
	//v.PreCommitMessage = t.PreCommitMessage
	//v.PreCommitTipSet = t.PreCommitTipSet
	v.PreCommit2Fails = t.PreCommit2Fails
	// WaitSeed
	v.SeedValue = t.SeedValue
	v.SeedEpoch = t.SeedEpoch
	// Committing
	//v.CommitMessage = t.CommitMessage
	v.InvalidProofs = t.InvalidProofs
	// CCUpdate
	v.CCUpdate = t.CCUpdate
	v.CCPieces = copyPieces(t.CCPieces)
	v.UpdateSealed = t.UpdateSealed
	v.UpdateUnsealed = t.UpdateUnsealed
	v.ReplicaUpdateProof = t.ReplicaUpdateProof
	//v.ReplicaUpdateMessage = t.ReplicaUpdateMessage
	// Faults
	//v.FaultReportMsg = t.FaultReportMsg
	// Recovery
	v.Return = types.ReturnState(t.Return)
	// Termination
	//v.TerminateMessage = t.TerminateMessage
	v.TerminatedAt = t.TerminatedAt
	// Debug
	v.LastErr = t.LastErr
	return &v
}
