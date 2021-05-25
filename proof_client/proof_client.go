package proof_client

import (
	"context"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc"
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
	closer, err := jsonrpc.NewMergeClient(ctx, cfg.Url, "Filecoin", []interface{}{pvc}, headers)
	if err != nil {
		return nil, nil, err
	}

	return pvc, closer, nil
}
