package api

import (
	"context"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/venus-market/api"
	"github.com/filecoin-project/venus-sealer/config"
	"github.com/ipfs-force-community/venus-common-utils/apiinfo"
	"go.uber.org/fx"
)

func NewMarketNodeRPCAPIV0(lc fx.Lifecycle, mCfg *config.MarketNodeConfig) (api.MarketFullNode, error) {
	if mCfg.Url == "" {
		log.Warnf("market node config is empty ...")
		return &api.MarketFullNodeStruct{}, nil
	}

	apiInfo := apiinfo.APIInfo{
		Addr:  mCfg.Url,
		Token: []byte(mCfg.Token),
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
