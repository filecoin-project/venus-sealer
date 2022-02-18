package api

import (
	"context"
	"github.com/filecoin-project/venus-sealer/config"
	"github.com/filecoin-project/venus/venus-shared/api/market"
	"github.com/ipfs-force-community/venus-common-utils/apiinfo"
	xerrors "github.com/pkg/errors"
	"go.uber.org/fx"
)

func NewMarketNodeRPCAPIV0(lc fx.Lifecycle, mCfg *config.MarketNodeConfig) (market.IMarket, error) {
	if mCfg.Url == "" {
		log.Warnf("market node config is empty ...")
		return nil, nil
	}

	apiInfo := apiinfo.APIInfo{
		Addr:  mCfg.Url,
		Token: []byte(mCfg.Token),
	}

	addr, err := apiInfo.DialArgs("v0")
	if err != nil {
		return nil, err
	}
	marketNode, closer, err := market.NewIMarketRPC(context.TODO(), addr, apiInfo.AuthHeader())
	if err!=nil {
		return nil, xerrors.Errorf("create marketnode: %s failed:%w", addr, err)
	}
	lc.Append(fx.Hook{
		OnStop: func(_ context.Context) error {
			closer()
			return nil
		},
	})
	return marketNode, err
}
