package market_client

import (
	"context"

	"go.uber.org/fx"

	"github.com/filecoin-project/go-jsonrpc"

	"github.com/ipfs-force-community/venus-common-utils/apiinfo"
	"github.com/ipfs-force-community/venus-gateway/marketevent"
	types2 "github.com/ipfs-force-community/venus-gateway/types"

	"github.com/filecoin-project/venus-sealer/types"
)

type MarketEventClient struct {
	ResponseMarketEvent func(ctx context.Context, resp *types2.ResponseEvent) error
	ListenMarketEvent   func(ctx context.Context, policy *marketevent.MarketRegisterPolicy) (<-chan *types2.RequestEvent, error)
}

func NewMarketEventClient(lc fx.Lifecycle, mode types.MarketMode, url, token string) (*MarketEventClient, error) {
	pvc := &MarketEventClient{}
	apiInfo := apiinfo.APIInfo{
		Addr:  url,
		Token: []byte(token),
	}
	addr, err := apiInfo.DialArgs("v0")
	if err != nil {
		return nil, err
	}

	namespace := "VENUS_MARKET"
	if mode == types.MarketPool {
		namespace = "Gateway"
	} else if mode == types.MarketSolo {
		namespace = "VENUS_MARKET"
	}

	closer, err := jsonrpc.NewMergeClient(context.Background(), addr, namespace, []interface{}{pvc}, apiInfo.AuthHeader())
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
