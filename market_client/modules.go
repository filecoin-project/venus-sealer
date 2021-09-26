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

func StartMarketEvent(lc fx.Lifecycle, stor *stores.Remote, sectorBlocks *sectorblocks.SectorBlocks, storageMgr *sectorstorage.Manager, mCfg *config.MarketConfig, cfg *config.RegisterMarketConfig, mAddr types2.MinerAddress) error {
	if len(cfg.Urls) == 0 {
		cfg.Urls = []string{mCfg.Url}
		cfg.Token = mCfg.Token
	}
	for _, addr := range cfg.Urls {
		client, err := NewMarketEventClient(lc, addr, cfg.Token)
		if err != nil {
			return err
		}
		marketEvent := MarketEvent{
			client:       client,
			mAddr:        mAddr,
			stor:         stor,
			sectorBlocks: sectorBlocks,
			storageMgr:   storageMgr,
		}
		go marketEvent.listenMarketRequest(context.Background())
	}

	return nil
}
