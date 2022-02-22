package proof_client

import (
	"context"
	"go.uber.org/fx"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/venus-sealer/config"
	"github.com/filecoin-project/venus-sealer/storage"
	"github.com/filecoin-project/venus-sealer/types"
	gwapi0 "github.com/filecoin-project/venus/venus-shared/api/gateway/v0"
)

type GatewayClientSets map[string]gwapi0.IGateway

func NewGatewayFullnodes(lc fx.Lifecycle, cfg *config.RegisterProofConfig) (GatewayClientSets, error) {
	clients := make(map[string]gwapi0.IGateway)
	var err error
	var ctx = context.TODO()
	for _, addr := range cfg.Urls {
		if clients[addr], err = newGateway(lc, ctx, addr, cfg.Token); err != nil {
			return nil, err
		}
	}

	if len(clients) == 0 {
		return nil, xerrors.Errorf("must have a GateWayNode, check 'RegisterProof' configuration")
	}
	return clients, nil
}

func StartProofEvent(lc fx.Lifecycle, clients GatewayClientSets, prover storage.WinningPoStProver, mAddr types.MinerAddress) error {
	for _, client := range clients {
		proofEvent := ProofEvent{
			prover: prover,
			client: client,
			mAddr:  mAddr,
		}
		go proofEvent.listenProofRequest(context.Background())
	}
	return nil
}
