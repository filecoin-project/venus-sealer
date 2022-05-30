package sealing

import (
	"context"
	"fmt"

	"github.com/filecoin-project/go-padreader"
	"github.com/filecoin-project/venus/venus-shared/types/market"

	"github.com/filecoin-project/venus-sealer/types"
)

func (m *Sealing) DealSector(ctx context.Context) ([]types.DealAssign, error) {
	if m.pieceStorageMrg == nil {
		return nil, fmt.Errorf("havn't configured piece storage")
	}
	m.startupWait.Wait()

	deals, err := m.api.GetUnPackedDeals(ctx, m.maddr, &market.GetDealSpec{MaxPiece: 50})
	if err != nil {
		return nil, err
	}
	log.Infof("got %d deals from venus-market", len(deals))
	//read from file

	var assigned []types.DealAssign
	for _, deal := range deals {
		pieceStorage, err := m.pieceStorageMrg.FindStorageForRead(ctx, deal.PieceCID.String())
		if err != nil {
			log.Errorf("failed to found piece storage %v", err)
			continue
		}
		r, err := pieceStorage.GetReaderCloser(ctx, deal.PieceCID.String())
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
