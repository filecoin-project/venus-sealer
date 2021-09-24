package market_client

import (
	"context"
	"github.com/filecoin-project/venus-sealer/config"
	sectorstorage "github.com/filecoin-project/venus-sealer/sector-storage"
	"github.com/filecoin-project/venus-sealer/sector-storage/stores"
	"github.com/filecoin-project/venus-sealer/storage/sectorblocks"
	types2 "github.com/filecoin-project/venus-sealer/types"
	"go.uber.org/fx"
)

func StartMarketEvent(lc fx.Lifecycle, stor *stores.Remote, sectorBlocks *sectorblocks.SectorBlocks, storageMgr *sectorstorage.Manager, cfg *config.RegisterProofConfig, mAddr types2.MinerAddress) error {
	for _, addr := range cfg.Urls {
		client, err := NewProofEventClient(lc, addr, cfg.Token)
		if err != nil {
			return err
		}
		proofEvent := MarketEvent{
			client:       client,
			mAddr:        mAddr,
			stor:         stor,
			sectorBlocks: sectorBlocks,
			storageMgr:   storageMgr,
		}
		go proofEvent.listenMarketRequest(context.Background())
	}

	return nil
}
