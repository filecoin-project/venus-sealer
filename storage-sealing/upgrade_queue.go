package sealing

import (
	"context"

	"github.com/filecoin-project/go-address"

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
		return xerrors.Errorf("can't mark sectors not in the 'Proving' state for upgrade")
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

	active, err := m.api.StateMinerActiveSectors(ctx, m.maddr, tok)
	if err != nil {
		return xerrors.Errorf("failed to check active sectors: %w", err)
	}
	// Ensure the upgraded sector is active
	var found bool
	for _, si := range active {
		if si.SectorNumber == id {
			found = true
			break
		}
	}
	if !found {
		return xerrors.Errorf("cannot mark inactive sector for upgrade")
	}

	if onChainInfo.Expiration-head < market7.DealMinDuration {
		return xerrors.Errorf("pointless to upgrade sector %d, expiration %d is less than a min deal duration away from current epoch."+
			"Upgrade expiration before marking for upgrade", id, onChainInfo.Expiration)
	}

	return m.sectors.Send(uint64(id), SectorStartCCUpdate{})
}

func sectorActive(ctx context.Context, api SealingAPI, maddr address.Address, tok types.TipSetToken, sector abi.SectorNumber) (bool, error) {
	active, err := api.StateMinerActiveSectors(ctx, maddr, tok)
	if err != nil {
		return false, xerrors.Errorf("failed to check active sectors: %w", err)
	}

	// Ensure the upgraded sector is active
	var found bool
	for _, si := range active {
		if si.SectorNumber == sector {
			found = true
			break
		}
	}
	return found, nil
}
