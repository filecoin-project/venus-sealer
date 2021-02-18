package storage

import (
	"context"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/build"
	"github.com/filecoin-project/lotus/chain/gen"
	proof2 "github.com/filecoin-project/specs-actors/v2/actors/runtime/proof"
	"github.com/filecoin-project/specs-storage/storage"
	"github.com/filecoin-project/venus-sealer/api"
	"github.com/filecoin-project/venus-sealer/dtypes"
	"github.com/filecoin-project/venus-sealer/extern/sector-storage/ffiwrapper"
	"github.com/filecoin-project/venus/pkg/specactors/builtin"
	"github.com/filecoin-project/venus/pkg/types"
	"golang.org/x/xerrors"
	"time"
)

var ValidWpostForTesting = []proof2.PoStProof{{
	ProofBytes: []byte("valid proof"),
}}

type WinningPoStProver interface {
	GenerateCandidates(context.Context, abi.PoStRandomness, uint64) ([]uint64, error)
	ComputeProof(context.Context, []proof2.SectorInfo, abi.PoStRandomness) ([]proof2.PoStProof, error)
}

type wppProvider struct{}

func (wpp *wppProvider) GenerateCandidates(ctx context.Context, _ abi.PoStRandomness, _ uint64) ([]uint64, error) {
	return []uint64{0}, nil
}

func (wpp *wppProvider) ComputeProof(context.Context, []proof2.SectorInfo, abi.PoStRandomness) ([]proof2.PoStProof, error) {
	return ValidWpostForTesting, nil
}

type StorageWpp struct {
	prover   storage.Prover
	verifier ffiwrapper.Verifier
	miner    abi.ActorID
	winnRpt  abi.RegisteredPoStProof
}

func NewWinningPoStProver(api api.FullNode, prover storage.Prover, verifier ffiwrapper.Verifier, miner dtypes.MinerID) (*StorageWpp, error) {
	ma, err := address.NewIDAddress(uint64(miner))
	if err != nil {
		return nil, err
	}

	mi, err := api.StateMinerInfo(context.TODO(), ma, types.EmptyTSK)
	if err != nil {
		return nil, xerrors.Errorf("getting sector size: %w", err)
	}

	if build.InsecurePoStValidation {
		log.Warn("*****************************************************************************")
		log.Warn(" Generating fake PoSt proof! You should only see this while running tests! ")
		log.Warn("*****************************************************************************")
	}

	return &StorageWpp{prover, verifier, abi.ActorID(miner), mi.WindowPoStProofType}, nil
}

var _ gen.WinningPoStProver = (*StorageWpp)(nil)

func (wpp *StorageWpp) GenerateCandidates(ctx context.Context, randomness abi.PoStRandomness, eligibleSectorCount uint64) ([]uint64, error) {
	start := build.Clock.Now()

	cds, err := wpp.verifier.GenerateWinningPoStSectorChallenge(ctx, wpp.winnRpt, wpp.miner, randomness, eligibleSectorCount)
	if err != nil {
		return nil, xerrors.Errorf("failed to generate candidates: %w", err)
	}
	log.Infof("Generate candidates took %s (C: %+v)", time.Since(start), cds)
	return cds, nil
}

func (wpp *StorageWpp) ComputeProof(ctx context.Context, ssi []builtin.SectorInfo, rand abi.PoStRandomness) ([]builtin.PoStProof, error) {
	if build.InsecurePoStValidation {
		return []builtin.PoStProof{{ProofBytes: []byte("valid proof")}}, nil
	}

	log.Infof("Computing WinningPoSt ;%+v; %v", ssi, rand)

	start := build.Clock.Now()
	proof, err := wpp.prover.GenerateWinningPoSt(ctx, wpp.miner, ssi, rand)
	if err != nil {
		return nil, err
	}
	log.Infof("GenerateWinningPoSt took %s", time.Since(start))
	return proof, nil
}
