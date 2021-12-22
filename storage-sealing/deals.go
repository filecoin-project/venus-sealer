package sealing

import (
	"context"
	"github.com/filecoin-project/go-padreader"

	types2 "github.com/filecoin-project/venus-market/types"
	"github.com/filecoin-project/venus-sealer/types"
)

func (m *Sealing) DealSector(ctx context.Context) ([]types.DealAssign, error) {
	m.startupWait.Wait()

	deals, err := m.api.GetUnPackedDeals(ctx, m.maddr, &types2.GetDealSpec{MaxPiece: 50})
	if err != nil {
		return nil, err
	}
	log.Infof("got %d deals from venus-market", len(deals))
	//read from file

	var assigned []types.DealAssign
	for _, deal := range deals {
		r, err := m.pieceStorage.Read(ctx, deal.PieceCID.String())
		if err != nil {
			log.Errorf("read piece from piece storage %v", err)
			continue
		}

		padR, err := padreader.NewInflator(r, uint64(deal.PayloadSize), deal.PieceSize.Unpadded())
		if err != nil {
			return nil, err
		}

		so, err := m.SectorAddPieceToAny(ctx, deal.Length.Unpadded(), padR, types.PieceDealInfo{
			PublishCid:   &deal.PublishCid,
			DealID:       deal.DealID,
			DealProposal: &deal.DealProposal,
			DealSchedule: types.DealSchedule{StartEpoch: deal.StartEpoch, EndEpoch: deal.EndEpoch},
			KeepUnsealed: deal.FastRetrieval,
		})
		_ = r.Close()
		if err != nil {
			log.Errorf("add piece to sector %v", err)
			continue
		}

		err = m.api.UpdateDealOnPacking(ctx, m.maddr, deal.DealID, so.Sector, so.Offset)
		if err != nil {
			log.Errorf("update deal status on chain ", err)
			//if error how to fix this problems
			continue
		}
		assigned = append(assigned, types.DealAssign{
			DealId:   deal.DealID,
			SectorId: so.Sector,
			PieceCid: deal.PieceCID,
			Offset:   so.Offset,
			Size:     deal.PieceSize,
		})
	}
	return assigned, err
}
