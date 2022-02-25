package storage

import (
	"context"
	"io"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/specs-storage/storage"

	"github.com/filecoin-project/venus/venus-shared/actors/builtin"

	"github.com/filecoin-project/venus-sealer/api"
	"github.com/filecoin-project/venus-sealer/constants"
	"github.com/filecoin-project/venus-sealer/sector-storage/storiface"
	"github.com/filecoin-project/venus-sealer/storage-sealing/sealiface"
	"github.com/filecoin-project/venus-sealer/types"
)

// TODO: refactor this to be direct somehow

func (m *Miner) Address() address.Address {
	return m.sealing.Address()
}

func (m *Miner) SectorAddPieceToAny(ctx context.Context, size abi.UnpaddedPieceSize, r io.Reader, d types.PieceDealInfo) (api.SectorOffset, error) {
	return m.sealing.SectorAddPieceToAny(ctx, size, r, d)
}

func (m *Miner) StartPackingSector(sectorNum abi.SectorNumber) error {
	return m.sealing.StartPacking(sectorNum)
}

func (m *Miner) ListSectors() ([]types.SectorInfo, error) {
	return m.sealing.ListSectors()
}

func (m *Miner) GetSectorInfo(sid abi.SectorNumber) (types.SectorInfo, error) {
	return m.sealing.GetSectorInfo(sid)
}

func (m *Miner) PledgeSector(ctx context.Context) (storage.SectorRef, error) {
	return m.sealing.PledgeSector(ctx)
}

func (m *Miner) CurrentSectorID(ctx context.Context) (abi.SectorNumber, error) {
	return m.sealing.CurrentSectorID(ctx)
}

func (m *Miner) DealSector(ctx context.Context) ([]types.DealAssign, error) {
	return m.sealing.DealSector(ctx)
}

func (m *Miner) RedoSector(ctx context.Context, rsi storiface.SectorRedoParams) error {
	return m.sealing.RedoSector(ctx, rsi)
}

func (m *Miner) ForceSectorState(ctx context.Context, id abi.SectorNumber, state types.SectorState) error {
	return m.sealing.ForceSectorState(ctx, id, state)
}

func (m *Miner) RemoveSector(ctx context.Context, id abi.SectorNumber) error {
	return m.sealing.Remove(ctx, id)
}

func (m *Miner) TerminateSector(ctx context.Context, id abi.SectorNumber) error {
	return m.sealing.Terminate(ctx, id)
}

func (m *Miner) TerminateFlush(ctx context.Context) (string, error) {
	return m.sealing.TerminateFlush(ctx)
}

func (m *Miner) TerminatePending(ctx context.Context) ([]abi.SectorID, error) {
	return m.sealing.TerminatePending(ctx)
}

func (m *Miner) SectorPreCommitFlush(ctx context.Context) ([]sealiface.PreCommitBatchRes, error) {
	return m.sealing.SectorPreCommitFlush(ctx)
}

func (m *Miner) SectorPreCommitPending(ctx context.Context) ([]abi.SectorID, error) {
	return m.sealing.SectorPreCommitPending(ctx)
}

func (m *Miner) CommitFlush(ctx context.Context) ([]sealiface.CommitBatchRes, error) {
	return m.sealing.CommitFlush(ctx)
}

func (m *Miner) CommitPending(ctx context.Context) ([]abi.SectorID, error) {
	return m.sealing.CommitPending(ctx)
}

func (m *Miner) SectorMatchPendingPiecesToOpenSectors(ctx context.Context) error {
	return m.sealing.MatchPendingPiecesToOpenSectors(ctx)
}
func (m *Miner) MarkForUpgrade(ctx context.Context, id abi.SectorNumber, snap bool) error {
	if snap {
		return m.sealing.MarkForSnapUpgrade(ctx, id)
	}
	return m.sealing.MarkForUpgrade(ctx, id)
}

func (m *Miner) IsMarkedForUpgrade(id abi.SectorNumber) bool {
	return m.sealing.IsMarkedForUpgrade(id)
}

func (m *Miner) SectorAbortUpgrade(sectorNum abi.SectorNumber) error {
	return m.sealing.AbortUpgrade(sectorNum)
}

func (s *Miner) MockWindowPoSt(ctx context.Context, sis []builtin.ExtendedSectorInfo, rand abi.PoStRandomness) error {
	mid, err := address.IDFromAddress(s.maddr)
	if err != nil {
		return err
	}

	tCtx := context.TODO()
	for {
		tsStart := constants.Clock.Now()

		_, _, err = s.sealer.GenerateWindowPoSt(tCtx, abi.ActorID(mid), sis, append(abi.PoStRandomness{}, rand...))
		if err != nil {
			log.Warnf("generate window post failed: %v", err.Error())
			continue
		}

		elapsed := time.Since(tsStart)
		log.Infow("mock generate window post", "elapsed", elapsed)
		break
	}

	return nil
}
