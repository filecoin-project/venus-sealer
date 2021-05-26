package proof_client

import (
	"context"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/venus-sealer/api"
	"github.com/filecoin-project/venus-sealer/config"
	"github.com/ipfs-force-community/venus-gateway/types"
	"net/http"
)

type ProofEventClient struct {
	ResponseProofEvent func(ctx context.Context, resp *types.ResponseEvent) error
	ListenProofEvent   func(ctx context.Context, mAddr address.Address) (chan *types.RequestEvent, error)
}

func NewProofEventClient(ctx context.Context, cfg *config.ProofConfig) (*ProofEventClient, jsonrpc.ClientCloser, error) {
	headers := http.Header{}
	headers.Add("Authorization", "Bearer "+cfg.Token)
	pvc := &ProofEventClient{}
	apiInfo := api.APIInfo{
		Addr:  cfg.Url,
		Token: []byte(cfg.Token),
	}
	addr, err := apiInfo.DialArgs()
	if err != nil {
		return nil, nil, err
	}
	closer, err := jsonrpc.NewMergeClient(ctx, addr, "Filecoin", []interface{}{pvc}, apiInfo.AuthHeader())
	if err != nil {
		return nil, nil, err
	}

	return pvc, closer, nil
}
