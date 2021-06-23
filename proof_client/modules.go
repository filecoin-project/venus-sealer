package proof_client

import (
	"context"
	"github.com/filecoin-project/venus-sealer/config"
	"github.com/filecoin-project/venus-sealer/storage"
	types2 "github.com/filecoin-project/venus-sealer/types"
	"go.uber.org/fx"
)

func StartProofEvent(prover storage.WinningPoStProver, lc fx.Lifecycle, cfg *config.RegisterProofConfig, mAddr types2.MinerAddress) error {
	for _, addr := range cfg.Urls {
		client, err := NewProofEventClient(lc, addr, cfg.Token)
		if err != nil {
			return err
		}
		proofEvent := ProofEvent{
			prover: prover,
			client: client,
			mAddr:  mAddr,
		}
		go proofEvent.listenProofRequest(context.Background())
	}

	return nil
}
