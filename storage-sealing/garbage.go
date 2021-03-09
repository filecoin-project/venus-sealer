package sealing

import (
	"context"
	"github.com/filecoin-project/venus-sealer/lib/reader"
	"github.com/filecoin-project/venus-sealer/types"

	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/specs-storage/storage"
)

func (m *Sealing) pledgeSector(ctx context.Context, sectorID storage.SectorRef, existingPieceSizes []abi.UnpaddedPieceSize, sizes ...abi.UnpaddedPieceSize) ([]abi.PieceInfo, error) {
	if len(sizes) == 0 {
		return nil, nil
	}

	log.Infof("Pledge %d, contains %+v", sectorID, existingPieceSizes)

	out := make([]abi.PieceInfo, len(sizes))
	for i, size := range sizes {
		ppi, err := m.sealer.AddPiece(ctx, sectorID, existingPieceSizes, size, reader.NewNullReader(size))
		if err != nil {
			return nil, xerrors.Errorf("add piece: %w", err)
		}

		existingPieceSizes = append(existingPieceSizes, size)

		out[i] = ppi
	}

	return out, nil
}

func (m *Sealing) PledgeSector() error {
	cfg, err := m.getConfig()
	if err != nil {
		return xerrors.Errorf("getting config: %w", err)
	}

	if cfg.MaxSealingSectors > 0 {
		if m.stats.CurSealing() >= cfg.MaxSealingSectors {
			return xerrors.Errorf("too many sectors sealing (curSealing: %d, max: %d)", m.stats.CurSealing(), cfg.MaxSealingSectors)
		}
	}

	go func() {
		ctx := context.TODO() // we can't use the context from command which invokes
		// this, as we run everything here async, and it's cancelled when the
		// command exits

		spt, err := m.currentSealProof(ctx)
		if err != nil {
			log.Errorf("%+v", err)
			return
		}

		size, err := spt.SectorSize()
		if err != nil {
			log.Errorf("%+v", err)
			return
		}

		sid, err := m.sc.Next()
		if err != nil {
			log.Errorf("%+v", err)
			return
		}
		sectorID := m.minerSector(spt, sid)
		err = m.sealer.NewSector(ctx, sectorID)
		if err != nil {
			log.Errorf("%+v", err)
			return
		}

		pieces, err := m.pledgeSector(ctx, sectorID, []abi.UnpaddedPieceSize{}, abi.PaddedPieceSize(size).Unpadded())
		if err != nil {
			log.Errorf("%+v", err)
			return
		}

		ps := make([]types.Piece, len(pieces))
		for idx := range ps {
			ps[idx] = types.Piece{
				Piece:    pieces[idx],
				DealInfo: nil,
			}
		}

		if err := m.newSectorCC(ctx, sid, ps); err != nil {
			log.Errorf("%+v", err)
			return
		}
	}()
	return nil
}
