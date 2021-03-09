package sealing_test

import (
	"context"
	"github.com/filecoin-project/venus-sealer/constants"
	"github.com/filecoin-project/venus-sealer/types"
	"testing"

	"github.com/filecoin-project/go-state-types/network"

	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	commcid "github.com/filecoin-project/go-fil-commcid"
	"github.com/filecoin-project/go-state-types/abi"

	sealing "github.com/filecoin-project/venus-sealer/storage-sealing"
)

type fakeChain struct {
	h abi.ChainEpoch
}

func (f *fakeChain) StateNetworkVersion(ctx context.Context, tok types.TipSetToken) (network.Version, error) {
	return constants.NewestNetworkVersion, nil
}

func (f *fakeChain) ChainHead(ctx context.Context) (types.TipSetToken, abi.ChainEpoch, error) {
	return []byte{1, 2, 3}, f.h, nil
}

func fakePieceCid(t *testing.T) cid.Cid {
	comm := [32]byte{1, 2, 3}
	fakePieceCid, err := commcid.ReplicaCommitmentV1ToCID(comm[:])
	require.NoError(t, err)
	return fakePieceCid
}

func TestBasicPolicyEmptySector(t *testing.T) {
	policy := sealing.NewBasicPreCommitPolicy(&fakeChain{
		h: abi.ChainEpoch(55),
	}, 10, 0)

	exp, err := policy.Expiration(context.Background())
	require.NoError(t, err)

	assert.Equal(t, 2879, int(exp))
}

func TestBasicPolicyMostConstrictiveSchedule(t *testing.T) {
	policy := sealing.NewBasicPreCommitPolicy(&fakeChain{
		h: abi.ChainEpoch(55),
	}, 100, 11)

	pieces := []types.Piece{
		{
			Piece: abi.PieceInfo{
				Size:     abi.PaddedPieceSize(1024),
				PieceCID: fakePieceCid(t),
			},
			DealInfo: &types.DealInfo{
				DealID: abi.DealID(42),
				DealSchedule: types.DealSchedule{
					StartEpoch: abi.ChainEpoch(70),
					EndEpoch:   abi.ChainEpoch(75),
				},
			},
		},
		{
			Piece: abi.PieceInfo{
				Size:     abi.PaddedPieceSize(1024),
				PieceCID: fakePieceCid(t),
			},
			DealInfo: &types.DealInfo{
				DealID: abi.DealID(43),
				DealSchedule: types.DealSchedule{
					StartEpoch: abi.ChainEpoch(80),
					EndEpoch:   abi.ChainEpoch(100),
				},
			},
		},
	}

	exp, err := policy.Expiration(context.Background(), pieces...)
	require.NoError(t, err)

	assert.Equal(t, 2890, int(exp))
}

func TestBasicPolicyIgnoresExistingScheduleIfExpired(t *testing.T) {
	policy := sealing.NewBasicPreCommitPolicy(&fakeChain{
		h: abi.ChainEpoch(55),
	}, 100, 0)

	pieces := []types.Piece{
		{
			Piece: abi.PieceInfo{
				Size:     abi.PaddedPieceSize(1024),
				PieceCID: fakePieceCid(t),
			},
			DealInfo: &types.DealInfo{
				DealID: abi.DealID(44),
				DealSchedule: types.DealSchedule{
					StartEpoch: abi.ChainEpoch(1),
					EndEpoch:   abi.ChainEpoch(10),
				},
			},
		},
	}

	exp, err := policy.Expiration(context.Background(), pieces...)
	require.NoError(t, err)

	assert.Equal(t, 2879, int(exp))
}

func TestMissingDealIsIgnored(t *testing.T) {
	policy := sealing.NewBasicPreCommitPolicy(&fakeChain{
		h: abi.ChainEpoch(55),
	}, 100, 11)

	pieces := []types.Piece{
		{
			Piece: abi.PieceInfo{
				Size:     abi.PaddedPieceSize(1024),
				PieceCID: fakePieceCid(t),
			},
			DealInfo: &types.DealInfo{
				DealID: abi.DealID(44),
				DealSchedule: types.DealSchedule{
					StartEpoch: abi.ChainEpoch(1),
					EndEpoch:   abi.ChainEpoch(10),
				},
			},
		},
		{
			Piece: abi.PieceInfo{
				Size:     abi.PaddedPieceSize(1024),
				PieceCID: fakePieceCid(t),
			},
			DealInfo: nil,
		},
	}

	exp, err := policy.Expiration(context.Background(), pieces...)
	require.NoError(t, err)

	assert.Equal(t, 2890, int(exp))
}
