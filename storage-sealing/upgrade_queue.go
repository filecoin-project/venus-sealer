package sealing

import (
	"context"

	market7 "github.com/filecoin-project/specs-actors/v7/actors/builtin/market"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/venus-sealer/types"
)

func (m *Sealing) MarkForSnapUpgrade(ctx context.Context, id abi.SectorNumber) error {
	cfg, err := m.getConfig()
	if err != nil {
		return xerrors.Errorf("getting storage config: %w", err)
	}

	curStaging := m.stats.CurStaging()
	if cfg.MaxWaitDealsSectors > 0 && curStaging >= cfg.MaxWaitDealsSectors {
		return xerrors.Errorf("already waiting for deals in %d >= %d (cfg.MaxWaitDealsSectors) sectors, no free resources to wait for deals in another",
			curStaging, cfg.MaxWaitDealsSectors)
	}

	si, err := m.GetSectorInfo(id)
	if err != nil {
		return xerrors.Errorf("getting sector info: %w", err)
	}

	if si.State != types.Proving {
		return xerrors.Errorf("unable to snap-up sectors not in the 'Proving' state")
	}

	if len(si.Pieces) != 1 {
		return xerrors.Errorf("not a committed-capacity sector, expected 1 piece")
	}

	if si.Pieces[0].DealInfo != nil {
		return xerrors.Errorf("not a committed-capacity sector, has deals")
	}

	tok, head, err := m.api.ChainHead(ctx)
	if err != nil {
		return xerrors.Errorf("couldnt get chain head: %w", err)
	}
	onChainInfo, err := m.api.StateSectorGetInfo(ctx, m.maddr, id, tok)
	if err != nil {
		return xerrors.Errorf("failed to read sector on chain info: %w", err)
	}

	active, err := m.sectorActive(ctx, tok, id)
	if err != nil {
		return xerrors.Errorf("failed to check active sectors: %w", err)
	}

	if !active {
		return xerrors.Errorf("cannot mark inactive sector for upgrade")
	}

	if onChainInfo.Expiration-head < market7.DealMinDuration {
		return xerrors.Errorf("pointless to upgrade sector %d, expiration %d is less than a min deal duration away from current epoch."+
			"Upgrade expiration before marking for upgrade", id, onChainInfo.Expiration)
	}

	return m.sectors.Send(uint64(id), SectorMarkForUpdate{})
}

func (m *Sealing) sectorActive(ctx context.Context, tok types.TipSetToken, sector abi.SectorNumber) (bool, error) {
	active, err := m.api.StateMinerActiveSectors(ctx, m.maddr, tok)
	if err != nil {
		return false, xerrors.Errorf("failed to check active sectors: %w", err)
	}

	return active.IsSet(uint64(sector))
}
