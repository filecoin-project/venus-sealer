package market_client

import (
	"context"
	"github.com/filecoin-project/venus/venus-shared/types/gateway"
)

type IMarketEventClient interface {
	ResponseMarketEvent(ctx context.Context, resp *gateway.ResponseEvent) error                                        //perm:read
	ListenMarketEvent(ctx context.Context, policy *gateway.MarketRegisterPolicy) (<-chan *gateway.RequestEvent, error) //perm:read
}