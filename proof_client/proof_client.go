package proof_client

import (
	"context"
	gwapi0 "github.com/filecoin-project/venus/venus-shared/api/gateway/v0"
	"github.com/ipfs-force-community/venus-common-utils/apiinfo"
	xerrors "github.com/pkg/errors"
	"go.uber.org/fx"
)

func newGateway(lc fx.Lifecycle, ctx context.Context, url, token string) (gwapi0.IGateway, error) {
	apiInfo := apiinfo.APIInfo{
		Addr:  url,
		Token: []byte(token),
	}
	addr, err := apiInfo.DialArgs("v0")
	if err != nil {
		return nil, err
	}
	client, closer, err := gwapi0.NewIGatewayRPC(ctx, addr, apiInfo.AuthHeader())
	if err != nil {
		return nil, xerrors.Errorf("create gateway fullnode:%s failed:%w", addr, err)
	}
	lc.Append(fx.Hook{
		OnStop: func(_ context.Context) error {
			closer()
			return nil
		},
	})
	return client, nil
}
