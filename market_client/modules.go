package market_client

import (
	"context"
	"github.com/filecoin-project/venus-market/piecestorage"
	"github.com/filecoin-project/venus-sealer/proof_client"
	gwapi0 "github.com/filecoin-project/venus/venus-shared/api/gateway/v0"
	"github.com/filecoin-project/venus/venus-shared/api/market"
	xerrors "github.com/pkg/errors"
	"go.uber.org/fx"

	"github.com/filecoin-project/venus-sealer/config"
	"github.com/filecoin-project/venus-sealer/sector-storage"
	"github.com/filecoin-project/venus-sealer/sector-storage/stores"
	"github.com/filecoin-project/venus-sealer/storage/sectorblocks"
	"github.com/filecoin-project/venus-sealer/types"
)

type MarketEventClientSets map[string]gwapi0.IMarketServiceProvider

func NewMarketEvents(gatewayEvents proof_client.GatewayClientSets,
	mrgCfg *config.RegisterMarketConfig,
	nodeConfig *config.MarketNodeConfig,
	marketNode market.IMarket) (MarketEventClientSets, error) {

	var marketClients = make(map[string]gwapi0.IMarketServiceProvider)

	if len(mrgCfg.Urls) == 0 { // RegisterMarket is not set in configurations, use all
		for url, client := range gatewayEvents {
			marketClients[url] = client
		}
		if marketNode != nil {
			marketClients[nodeConfig.Url] = marketNode
		}
	} else {
		for _, url := range mrgCfg.Urls {
			if client, exist := gatewayEvents[url]; exist && client != nil {
				marketClients[url] = client
				continue
			}
			if url == nodeConfig.Url {
				marketClients[url] = marketNode
				continue
			}
			log.Warnf("Don't kown endpoint :%s is a market node or gateway node", url)
		}
		if len(marketClients) == 0 {
			return nil, xerrors.Errorf("RegisterMarket is set in configurations, but its neither a 'market' nor a 'gateway'")
		}
	}

	return marketClients, nil
}

func StartMarketEvent(lc fx.Lifecycle, stores *stores.Remote,
	pieceStorage piecestorage.IPieceStorage, sectorBlocks *sectorblocks.SectorBlocks, storageMgr *sectorstorage.Manager,
	index stores.SectorIndex, evtClients MarketEventClientSets, mAddr types.MinerAddress) error {
	if len(evtClients) == 0 {
		log.Warnf("register market config is empty ...")
		return nil
	}

	for _, evtClient := range evtClients {
		marketEvent := MarketEvent{
			client:       evtClient,
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
