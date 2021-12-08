package proof_client

import (
	"context"

	"go.uber.org/fx"

	"github.com/filecoin-project/venus-sealer/config"
	"github.com/filecoin-project/venus-sealer/storage"
	"github.com/filecoin-project/venus-sealer/types"
)

func StartProofEvent(lc fx.Lifecycle, cfg *config.RegisterProofConfig, prover storage.WinningPoStProver, mAddr types.MinerAddress) error {
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
