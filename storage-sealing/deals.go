package sealing

import (
	"context"
	"github.com/filecoin-project/venus-market/piece"
	"github.com/filecoin-project/venus-sealer/types"
)

func (m *Sealing) DealSector(ctx context.Context) ([]types.DealAssign, error) {
	m.startupWait.Wait()

	deals, err := m.api.GetUnPackedDeals(ctx, m.maddr, &piece.GetDealSpec{MaxPiece: 50})
	if err != nil {
		return nil, err
	}
	log.Infof("got %d deals from venus-market", len(deals))
	//read from file
	var assigned []types.DealAssign
	for _, deal := range deals {
		r, err := piece.Read(deal.PieceStorage)
		if err != nil {
			log.Errorf("read piece from piece storage %v", err)
			continue
		}

		sid, offset, err := m.AddPieceToAnySector(ctx, deal.Length.Unpadded(), r, types.DealInfo{
			PublishCid:   &deal.PublishCid,
			DealID:       deal.DealId,
			DealProposal: &deal.Proposal,
			DealSchedule: types.DealSchedule{StartEpoch: deal.Proposal.StartEpoch, EndEpoch: deal.Proposal.EndEpoch},
			KeepUnsealed: deal.FastRetrieval,
		})
		_ = r.Close()
		if err != nil {
			log.Errorf("add piece to sector %v", err)
			continue
		}

		err = m.api.UpdateDealOnPacking(ctx, m.maddr, deal.Proposal.PieceCID, deal.DealId, sid, offset)
		if err != nil {
			log.Errorf("update deal status on chain ", err)
			//if error how to fix this problems
			continue
		}
		assigned = append(assigned, types.DealAssign{
			DealId:   deal.DealId,
			SectorId: deal.SectorID,
			PieceCid: deal.Proposal.PieceCID,
			Offset:   offset,
			Size:     deal.Proposal.PieceSize,
		})
	}
	return assigned, err
}
