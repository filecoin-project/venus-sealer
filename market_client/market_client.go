package market_client

import (
	"context"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/ipfs-force-community/venus-common-utils/apiinfo"
	"github.com/ipfs-force-community/venus-gateway/marketevent"
	"github.com/ipfs-force-community/venus-gateway/types"
	"go.uber.org/fx"
)

type MarketEventClient struct {
	ResponseMarketEvent func(ctx context.Context, resp *types.ResponseEvent) error
	ListenMarketEvent   func(ctx context.Context, policy *marketevent.MarketRegisterPolicy) (<-chan *types.RequestEvent, error)
}

func NewMarketEventClient(lc fx.Lifecycle, url, token string) (*MarketEventClient, error) {
	pvc := &MarketEventClient{}
	apiInfo := apiinfo.APIInfo{
		Addr:  url,
		Token: []byte(token),
	}
	addr, err := apiInfo.DialArgs("v0")
	if err != nil {
		return nil, err
	}
	closer, err := jsonrpc.NewMergeClient(context.Background(), addr, "VENUS_MARKET", []interface{}{pvc}, apiInfo.AuthHeader())
	if err != nil {
		return nil, err
	}
	lc.Append(fx.Hook{
		OnStop: func(_ context.Context) error {
			closer()
			return nil
		},
	})
	return pvc, nil
}
