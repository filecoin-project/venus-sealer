package proof_client

import (
	"context"
	"encoding/json"
	"time"

	logging "github.com/ipfs/go-log/v2"
	"golang.org/x/xerrors"

	"github.com/google/uuid"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/venus-sealer/storage"
	types2 "github.com/filecoin-project/venus-sealer/types"

	"github.com/ipfs-force-community/venus-gateway/proofevent"
	"github.com/ipfs-force-community/venus-gateway/types"
)

var log = logging.Logger("proof_event")

type ProofEvent struct {
	prover storage.WinningPoStProver
	client *ProofEventClient
	mAddr  types2.MinerAddress
}

func (e *ProofEvent) listenProofRequest(ctx context.Context) {
	log.Infof("start proof event listening")
	for {
		if err := e.listenProofRequestOnce(ctx); err != nil {
			log.Errorf("listen head changes errored: %s", err)
		} else {
			log.Warn("listenHeadChanges quit")
		}
		select {
		case <-time.After(time.Second):
		case <-ctx.Done():
			log.Warnf("not restarting listenHeadChanges: context error: %s", ctx.Err())
			return
		}

		log.Info("restarting listenHeadChanges")
	}
}

func (e *ProofEvent) listenProofRequestOnce(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	policy := &proofevent.ProofRegisterPolicy{
		MinerAddress: address.Address(e.mAddr),
	}
	proofEventCh, err := e.client.ListenProofEvent(ctx, policy)
	if err != nil {
		// Retry is handled by caller
		return xerrors.Errorf("listenHeadChanges ChainNotify call failed: %w", err)
	}

	for proofEvent := range proofEventCh {
		switch proofEvent.Method {
		case "InitConnect":
			req := types.ConnectedCompleted{}
			err := json.Unmarshal(proofEvent.Payload, &req)
			if err != nil {
				return xerrors.Errorf("odd error in connect %v", err)
			}
			log.Infof("success to connect with proof %s", req.ChannelId)
		case "ComputeProof":
			req := types.ComputeProofRequest{}
			err := json.Unmarshal(proofEvent.Payload, &req)
			if err != nil {
				_ = e.client.ResponseProofEvent(ctx, &types.ResponseEvent{
					Id:      proofEvent.Id,
					Payload: nil,
					Error:   err.Error(),
				})
				continue
			}
			e.processComputeProof(ctx, proofEvent.Id, req)
		default:
			log.Errorf("unexpect proof event type %s", proofEvent.Method)
		}
	}

	return nil
}

// context.Context, []builtin.ExtendedSectorInfo, abi.PoStRandomness, abi.ChainEpoch, network.Version
func (e *ProofEvent) processComputeProof(ctx context.Context, reqId uuid.UUID, req types.ComputeProofRequest) {
	proof, err := e.prover.ComputeProof(ctx, req.SectorInfos, req.Rand, req.Height, req.NWVersion)
	if err != nil {
		_ = e.client.ResponseProofEvent(ctx, &types.ResponseEvent{
			Id:      reqId,
			Payload: nil,
			Error:   err.Error(),
		})
		return
	}

	proofBytes, err := json.Marshal(proof)
	if err != nil {
		_ = e.client.ResponseProofEvent(ctx, &types.ResponseEvent{
			Id:      reqId,
			Payload: nil,
			Error:   err.Error(),
		})
		return
	}

	err = e.client.ResponseProofEvent(ctx, &types.ResponseEvent{
		Id:      reqId,
		Payload: proofBytes,
		Error:   "",
	})
	if err != nil {
		log.Errorf("response proof event %s failed", reqId)
	}
}
