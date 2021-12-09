package market_client

import (
	"context"

	"go.uber.org/fx"

	"github.com/filecoin-project/venus-market/piecestorage"

	"github.com/filecoin-project/venus-sealer/config"
	"github.com/filecoin-project/venus-sealer/sector-storage"
	"github.com/filecoin-project/venus-sealer/sector-storage/stores"
	"github.com/filecoin-project/venus-sealer/storage/sectorblocks"
	"github.com/filecoin-project/venus-sealer/types"
)

func StartMarketEvent(lc fx.Lifecycle, stores *stores.Remote, pieceStorage piecestorage.IPieceStorage, sectorBlocks *sectorblocks.SectorBlocks, storageMgr *sectorstorage.Manager, index stores.SectorIndex, mode types.MarketMode, cfg *config.RegisterMarketConfig, mAddr types.MinerAddress) error {
	if len(cfg.Urls) == 0 {
		log.Warnf("register market config is empty ...")
		return nil
	}

	for _, url := range cfg.Urls {
		client, err := NewMarketEventClient(lc, mode, url, cfg.Token)
		if err != nil {
			return err
		}

		marketEvent := MarketEvent{
			client:       client,
			mAddr:        mAddr,
			stor:         stores,
			sectorBlocks: sectorBlocks,
			storageMgr:   storageMgr,
			index:        index,
			pieceStorage: pieceStorage,
		}
		go marketEvent.listenMarketRequest(context.Background())
	}

	return nil
}
