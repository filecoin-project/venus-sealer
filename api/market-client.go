package api

import (
	"context"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/venus-market/api"
	"github.com/filecoin-project/venus-sealer/config"
	"github.com/ipfs-force-community/venus-common-utils/apiinfo"
	"go.uber.org/fx"
)

func NewMarketRPC(lc fx.Lifecycle, marketCfg *config.MarketConfig) (api.MarketFullNode, error) {
	apiInfo := apiinfo.APIInfo{
		Addr:  marketCfg.Url,
		Token: []byte(marketCfg.Token),
	}

	addr, err := apiInfo.DialArgs("v0")
	if err != nil {
		return nil, err
	}
	var res api.MarketFullNodeStruct
	closer, err := jsonrpc.NewMergeClient(context.Background(), addr, "VENUS_MARKET",
		[]interface{}{
			&res.Internal,
		},
		apiInfo.AuthHeader(),
	)
	lc.Append(fx.Hook{
		OnStop: func(_ context.Context) error {
			closer()
			return nil
		},
	})
	return &res, err
}
