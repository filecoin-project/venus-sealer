package proof_client

import (
	"context"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/venus-sealer/api"
	"github.com/ipfs-force-community/venus-gateway/proofevent"
	"github.com/ipfs-force-community/venus-gateway/types"
	"go.uber.org/fx"
)

type ProofEventClient struct {
	ResponseProofEvent func(ctx context.Context, resp *types.ResponseEvent) error
	ListenProofEvent   func(ctx context.Context, policy *proofevent.ProofRegisterPolicy) (chan *types.RequestEvent, error)
}

func NewProofEventClient(lc fx.Lifecycle, url, token string) (*ProofEventClient, error) {
	pvc := &ProofEventClient{}
	apiInfo := api.APIInfo{
		Addr:  url,
		Token: []byte(token),
	}
	addr, err := apiInfo.DialArgs()
	if err != nil {
		return nil, err
	}
	closer, err := jsonrpc.NewMergeClient(context.Background(), addr, "Gateway", []interface{}{pvc}, apiInfo.AuthHeader())
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
