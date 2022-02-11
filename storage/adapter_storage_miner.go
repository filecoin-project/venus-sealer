package storage

import (
	"bytes"
	"context"

	cbg "github.com/whyrusleeping/cbor-gen"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/go-state-types/dline"
	"github.com/filecoin-project/go-state-types/network"
	market2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/market"
	market5 "github.com/filecoin-project/specs-actors/v5/actors/builtin/market"
	api2 "github.com/filecoin-project/venus-market/api"
	types4 "github.com/filecoin-project/venus-market/types"
	"github.com/filecoin-project/venus-sealer/api"
	"github.com/filecoin-project/venus-sealer/constants"
	sealing "github.com/filecoin-project/venus-sealer/storage-sealing"
	types2 "github.com/filecoin-project/venus-sealer/types"
	chain2 "github.com/filecoin-project/venus/pkg/chain"
	constants2 "github.com/filecoin-project/venus/pkg/constants"
	"github.com/filecoin-project/venus/venus-shared/actors"
	"github.com/filecoin-project/venus/venus-shared/actors/builtin/market"
	"github.com/filecoin-project/venus/venus-shared/actors/builtin/miner"
	"github.com/filecoin-project/venus/venus-shared/types"
	types3 "github.com/filecoin-project/venus/venus-shared/types/messager"
	"github.com/ipfs/go-cid"
)

var _ sealing.SealingAPI = new(SealingAPIAdapter)

type SealingAPIAdapter struct {
	delegate  fullNodeFilteredAPI
	messager  api.IMessager
	marketAPI api2.MarketFullNode
}

func NewSealingAPIAdapter(api fullNodeFilteredAPI, messager api.IMessager, marketAPI api2.MarketFullNode) SealingAPIAdapter {
	return SealingAPIAdapter{delegate: api, messager: messager, marketAPI: marketAPI}
}

func (s SealingAPIAdapter) StateMinerSectorSize(ctx context.Context, maddr address.Address, tok types2.TipSetToken) (abi.SectorSize, error) {
	// TODO: update storage-fsm to just StateMinerInfo
	mi, err := s.StateMinerInfo(ctx, maddr, tok)
	if err != nil {
		return 0, err
	}
	return mi.SectorSize, nil
}

func (s SealingAPIAdapter) StateMinerPreCommitDepositForPower(ctx context.Context, a address.Address, pci miner.SectorPreCommitInfo, tok types2.TipSetToken) (big.Int, error) {
	tsk, err := types.TipSetKeyFromBytes(tok)
	if err != nil {
		return big.Zero(), xerrors.Errorf("failed to unmarshal TipSetToken to TipSetKey: %w", err)
	}

	return s.delegate.StateMinerPreCommitDepositForPower(ctx, a, pci, tsk)
}

func (s SealingAPIAdapter) StateMinerInitialPledgeCollateral(ctx context.Context, a address.Address, pci miner.SectorPreCommitInfo, tok types2.TipSetToken) (big.Int, error) {
	tsk, err := types.TipSetKeyFromBytes(tok)
	if err != nil {
		return big.Zero(), xerrors.Errorf("failed to unmarshal TipSetToken to TipSetKey: %w", err)
	}

	return s.delegate.StateMinerInitialPledgeCollateral(ctx, a, pci, tsk)
}

func (s SealingAPIAdapter) StateMinerInfo(ctx context.Context, maddr address.Address, tok types2.TipSetToken) (miner.MinerInfo, error) {
	tsk, err := types.TipSetKeyFromBytes(tok)
	if err != nil {
		return miner.MinerInfo{}, xerrors.Errorf("failed to unmarshal TipSetToken to TipSetKey: %w", err)
	}

	// TODO: update storage-fsm to just StateMinerInfo
	return s.delegate.StateMinerInfo(ctx, maddr, tsk)
}

func (s SealingAPIAdapter) StateMinerAvailableBalance(ctx context.Context, maddr address.Address, tok types2.TipSetToken) (big.Int, error) {
	tsk, err := types.TipSetKeyFromBytes(tok)
	if err != nil {
		return big.Zero(), xerrors.Errorf("failed to unmarshal TipSetToken to TipSetKey: %w", err)
	}

	return s.delegate.StateMinerAvailableBalance(ctx, maddr, tsk)
}

func (s SealingAPIAdapter) StateMinerWorkerAddress(ctx context.Context, maddr address.Address, tok types2.TipSetToken) (address.Address, error) {
	// TODO: update storage-fsm to just StateMinerInfo
	mi, err := s.StateMinerInfo(ctx, maddr, tok)
	if err != nil {
		return address.Undef, err
	}
	return mi.Worker, nil
}

func (s SealingAPIAdapter) StateMinerDeadlines(ctx context.Context, maddr address.Address, tok types2.TipSetToken) ([]types.Deadline, error) {
	tsk, err := types.TipSetKeyFromBytes(tok)
	if err != nil {
		return nil, xerrors.Errorf("failed to unmarshal TipSetToken to TipSetKey: %w", err)
	}

	return s.delegate.StateMinerDeadlines(ctx, maddr, tsk)
}

func (s SealingAPIAdapter) StateMinerSectorAllocated(ctx context.Context, maddr address.Address, sid abi.SectorNumber, tok types2.TipSetToken) (bool, error) {
	tsk, err := types.TipSetKeyFromBytes(tok)
	if err != nil {
		return false, xerrors.Errorf("failed to unmarshal TipSetToken to TipSetKey: %w", err)
	}

	return s.delegate.StateMinerSectorAllocated(ctx, maddr, sid, tsk)
}

func (s SealingAPIAdapter) StateMinerActiveSectors(ctx context.Context, maddr address.Address, tok types2.TipSetToken) ([]*miner.SectorOnChainInfo, error) {
	tsk, err := types.TipSetKeyFromBytes(tok)
	if err != nil {
		return nil, xerrors.Errorf("failed to unmarshal TipSetToken to TipSetKey: %w", err)
	}

	return s.delegate.StateMinerActiveSectors(ctx, maddr, tsk)
}

func (s SealingAPIAdapter) StateWaitMsg(ctx context.Context, mcid cid.Cid) (types2.MsgLookup, error) {
	wmsg, err := s.delegate.StateWaitMsg(ctx, mcid, constants.MessageConfidence, constants2.LookbackNoLimit, true)
	if err != nil {
		return types2.MsgLookup{}, err
	}

	return types2.MsgLookup{
		Receipt: types2.MessageReceipt{
			ExitCode: wmsg.Receipt.ExitCode,
			Return:   wmsg.Receipt.Return,
			GasUsed:  wmsg.Receipt.GasUsed,
		},
		TipSetTok: wmsg.TipSet.Bytes(),
		Height:    wmsg.Height,
	}, nil
}

func (s SealingAPIAdapter) StateSearchMsg(ctx context.Context, c cid.Cid) (*types2.MsgLookup, error) {
	wmsg, err := s.delegate.StateSearchMsg(ctx, types.EmptyTSK, c, constants2.LookbackNoLimit, true)
	if err != nil {
		return nil, err
	}

	if wmsg == nil {
		return nil, nil
	}

	return &types2.MsgLookup{
		Receipt: types2.MessageReceipt{
			ExitCode: wmsg.Receipt.ExitCode,
			Return:   wmsg.Receipt.Return,
			GasUsed:  wmsg.Receipt.GasUsed,
		},
		TipSetTok: wmsg.TipSet.Bytes(),
		Height:    wmsg.Height,
	}, nil
}

func (s SealingAPIAdapter) StateComputeDataCommitment(ctx context.Context, maddr address.Address, sectorType abi.RegisteredSealProof, deals []abi.DealID, tok types2.TipSetToken) (cid.Cid, error) {
	tsk, err := types.TipSetKeyFromBytes(tok)
	if err != nil {
		return cid.Undef, xerrors.Errorf("failed to unmarshal TipSetToken to TipSetKey: %w", err)
	}

	nv, err := s.delegate.StateNetworkVersion(ctx, tsk)
	if err != nil {
		return cid.Cid{}, err
	}

	var ccparams []byte
	if nv < network.Version13 {
		ccparams, err = actors.SerializeParams(&market2.ComputeDataCommitmentParams{
			DealIDs:    deals,
			SectorType: sectorType,
		})
	} else {
		ccparams, err = actors.SerializeParams(&market5.ComputeDataCommitmentParams{
			Inputs: []*market5.SectorDataSpec{
				{
					DealIDs:    deals,
					SectorType: sectorType,
				},
			},
		})
	}

	if err != nil {
		return cid.Undef, xerrors.Errorf("computing params for ComputeDataCommitment: %w", err)
	}

	ccmt := &types.Message{
		To:     market.Address,
		From:   maddr,
		Value:  types.NewInt(0),
		Method: market.Methods.ComputeDataCommitment,
		Params: ccparams,
	}
	r, err := s.delegate.StateCall(ctx, ccmt, tsk)
	if err != nil {
		return cid.Undef, xerrors.Errorf("calling ComputeDataCommitment: %w", err)
	}
	if r.MsgRct.ExitCode != 0 {
		return cid.Undef, xerrors.Errorf("receipt for ComputeDataCommitment had exit code %d", r.MsgRct.ExitCode)
	}

	if nv < network.Version13 {
		var c cbg.CborCid
		if err := c.UnmarshalCBOR(bytes.NewReader(r.MsgRct.Return)); err != nil {
			return cid.Undef, xerrors.Errorf("failed to unmarshal CBOR to CborCid: %w", err)
		}

		return cid.Cid(c), nil
	}

	var cr market5.ComputeDataCommitmentReturn
	if err := cr.UnmarshalCBOR(bytes.NewReader(r.MsgRct.Return)); err != nil {
		return cid.Undef, xerrors.Errorf("failed to unmarshal CBOR to CborCid: %w", err)
	}

	if len(cr.CommDs) != 1 {
		return cid.Undef, xerrors.Errorf("CommD output must have 1 entry")
	}

	return cid.Cid(cr.CommDs[0]), nil
}

func (s SealingAPIAdapter) StateSectorPreCommitInfo(ctx context.Context, maddr address.Address, sectorNumber abi.SectorNumber, tok types2.TipSetToken) (*miner.SectorPreCommitOnChainInfo, error) {
	tsk, err := types.TipSetKeyFromBytes(tok)
	if err != nil {
		return nil, xerrors.Errorf("failed to unmarshal TipSetToken to TipSetKey: %w", err)
	}

	act, err := s.delegate.StateGetActor(ctx, maddr, tsk)
	if err != nil {
		return nil, xerrors.Errorf("handleSealFailed(%d): temp error: %+v", sectorNumber, err)
	}

	stor := chain2.ActorStore(ctx, api.NewAPIBlockstore(s.delegate))

	state, err := miner.Load(stor, act)
	if err != nil {
		return nil, xerrors.Errorf("handleSealFailed(%d): temp error: loading miner state: %+v", sectorNumber, err)
	}

	pci, err := state.GetPrecommittedSector(sectorNumber)
	if err != nil {
		return nil, err
	}
	if pci == nil {
		set, err := state.IsAllocated(sectorNumber)
		if err != nil {
			return nil, xerrors.Errorf("checking if sector is allocated: %w", err)
		}
		if set {
			return nil, sealing.ErrSectorAllocated
		}

		return nil, nil
	}

	return pci, nil
}

func (s SealingAPIAdapter) StateSectorGetInfo(ctx context.Context, maddr address.Address, sectorNumber abi.SectorNumber, tok types2.TipSetToken) (*miner.SectorOnChainInfo, error) {
	tsk, err := types.TipSetKeyFromBytes(tok)
	if err != nil {
		return nil, xerrors.Errorf("failed to unmarshal TipSetToken to TipSetKey: %w", err)
	}

	return s.delegate.StateSectorGetInfo(ctx, maddr, sectorNumber, tsk)
}

func (s SealingAPIAdapter) StateSectorPartition(ctx context.Context, maddr address.Address, sectorNumber abi.SectorNumber, tok types2.TipSetToken) (*sealing.SectorLocation, error) {
	tsk, err := types.TipSetKeyFromBytes(tok)
	if err != nil {
		return nil, xerrors.Errorf("failed to unmarshal TipSetToken to TipSetKey: %w", err)
	}

	l, err := s.delegate.StateSectorPartition(ctx, maddr, sectorNumber, tsk)
	if err != nil {
		return nil, err
	}
	if l != nil {
		return &sealing.SectorLocation{
			Deadline:  l.Deadline,
			Partition: l.Partition,
		}, nil
	}

	return nil, nil // not found
}

func (s SealingAPIAdapter) StateMinerPartitions(ctx context.Context, maddr address.Address, dlIdx uint64, tok types2.TipSetToken) ([]types.Partition, error) {
	tsk, err := types.TipSetKeyFromBytes(tok)
	if err != nil {
		return nil, xerrors.Errorf("failed to unmarshal TipSetToken to TipSetKey: %w", err)
	}

	return s.delegate.StateMinerPartitions(ctx, maddr, dlIdx, tsk)
}

func (s SealingAPIAdapter) StateLookupID(ctx context.Context, addr address.Address, tok types2.TipSetToken) (address.Address, error) {
	tsk, err := types.TipSetKeyFromBytes(tok)
	if err != nil {
		return address.Undef, err
	}

	return s.delegate.StateLookupID(ctx, addr, tsk)
}

func (s SealingAPIAdapter) StateMarketStorageDeal(ctx context.Context, dealID abi.DealID, tok types2.TipSetToken) (*types.MarketDeal, error) {
	tsk, err := types.TipSetKeyFromBytes(tok)
	if err != nil {
		return nil, err
	}

	return s.delegate.StateMarketStorageDeal(ctx, dealID, tsk)
}

func (s SealingAPIAdapter) StateMarketStorageDealProposal(ctx context.Context, dealID abi.DealID, tok types2.TipSetToken) (market.DealProposal, error) {
	tsk, err := types.TipSetKeyFromBytes(tok)
	if err != nil {
		return market.DealProposal{}, err
	}

	deal, err := s.delegate.StateMarketStorageDeal(ctx, dealID, tsk)
	if err != nil {
		return market.DealProposal{}, err
	}

	return deal.Proposal, nil
}

func (s SealingAPIAdapter) StateNetworkVersion(ctx context.Context, tok types2.TipSetToken) (network.Version, error) {
	tsk, err := types.TipSetKeyFromBytes(tok)
	if err != nil {
		return network.VersionMax, err
	}

	return s.delegate.StateNetworkVersion(ctx, tsk)
}

func (s SealingAPIAdapter) StateMinerProvingDeadline(ctx context.Context, maddr address.Address, tok types2.TipSetToken) (*dline.Info, error) {
	tsk, err := types.TipSetKeyFromBytes(tok)
	if err != nil {
		return nil, err
	}

	return s.delegate.StateMinerProvingDeadline(ctx, maddr, tsk)
}

func (s SealingAPIAdapter) SendMsg(ctx context.Context, from, to address.Address, method abi.MethodNum, value, maxFee abi.TokenAmount, params []byte) (cid.Cid, error) {
	msg := types.Message{
		To:     to,
		From:   from,
		Value:  value,
		Method: method,
		Params: params,
	}

	smsg, err := s.delegate.MpoolPushMessage(ctx, &msg, &types.MessageSendSpec{MaxFee: maxFee})
	if err != nil {
		return cid.Undef, err
	}

	return smsg.Cid(), nil
}

func (s SealingAPIAdapter) ChainHead(ctx context.Context) (types2.TipSetToken, abi.ChainEpoch, error) {
	head, err := s.delegate.ChainHead(ctx)
	if err != nil {
		return nil, 0, err
	}

	return head.Key().Bytes(), head.Height(), nil
}

func (s SealingAPIAdapter) ChainBaseFee(ctx context.Context, tok types2.TipSetToken) (abi.TokenAmount, error) {
	tsk, err := types.TipSetKeyFromBytes(tok)
	if err != nil {
		return big.Zero(), err
	}

	ts, err := s.delegate.ChainGetTipSet(ctx, tsk)
	if err != nil {
		return big.Zero(), err
	}

	return ts.Blocks()[0].ParentBaseFee, nil
}

func (s SealingAPIAdapter) ChainGetMessage(ctx context.Context, mc cid.Cid) (*types.Message, error) {
	return s.delegate.ChainGetMessage(ctx, mc)
}

func (s SealingAPIAdapter) StateGetRandomnessFromBeacon(ctx context.Context, personalization crypto.DomainSeparationTag, randEpoch abi.ChainEpoch, entropy []byte, tok types2.TipSetToken) (abi.Randomness, error) {
	tsk, err := types.TipSetKeyFromBytes(tok)
	if err != nil {
		return nil, err
	}

	return s.delegate.StateGetRandomnessFromBeacon(ctx, personalization, randEpoch, entropy, tsk)
}

func (s SealingAPIAdapter) StateGetRandomnessFromTickets(ctx context.Context, personalization crypto.DomainSeparationTag, randEpoch abi.ChainEpoch, entropy []byte, tok types2.TipSetToken) (abi.Randomness, error) {
	tsk, err := types.TipSetKeyFromBytes(tok)
	if err != nil {
		return nil, err
	}

	return s.delegate.StateGetRandomnessFromTickets(ctx, personalization, randEpoch, entropy, tsk)
}

func (s SealingAPIAdapter) ChainReadObj(ctx context.Context, ocid cid.Cid) ([]byte, error) {
	return s.delegate.ChainReadObj(ctx, ocid)
}

//todo MsgLookup use types in venus
func (s SealingAPIAdapter) MessagerWaitMsg(ctx context.Context, uuid string) (types2.MsgLookup, error) {
	msg, err := s.messager.WaitMessage(ctx, uuid, constants.MessageConfidence)
	if err != nil {
		return types2.MsgLookup{}, err
	}

	return types2.MsgLookup{
		Receipt: types2.MessageReceipt{
			ExitCode: msg.Receipt.ExitCode,
			Return:   msg.Receipt.Return,
			GasUsed:  msg.Receipt.GasUsed,
		},
		TipSetTok: msg.TipSetKey.Bytes(),
		Height:    abi.ChainEpoch(msg.Height),
	}, nil
}

func (s SealingAPIAdapter) MessagerSearchMsg(ctx context.Context, uuid string) (*types2.MsgLookup, error) {
	msg, err := s.messager.GetMessageByUid(ctx, uuid)
	if err != nil {
		return nil, err
	}
	if msg.State != types3.FillMsg {
		return nil, nil
	}
	return &types2.MsgLookup{
		Receipt: types2.MessageReceipt{
			ExitCode: msg.Receipt.ExitCode,
			Return:   msg.Receipt.Return,
			GasUsed:  msg.Receipt.GasUsed,
		},
		TipSetTok: msg.TipSetKey.Bytes(),
		Height:    abi.ChainEpoch(msg.Height),
	}, nil
}

func (s SealingAPIAdapter) MessagerSendMsg(ctx context.Context, from, to address.Address, method abi.MethodNum, value, maxFee abi.TokenAmount, params []byte) (string, error) {
	return s.messager.PushMessage(ctx, &types.Message{
		Version: 0,
		To:      to,
		From:    from,
		Value:   value,
		Method:  method,
		Params:  params,
	}, &types3.SendSpec{
		ExpireEpoch:       0,
		GasOverEstimation: 0,
		//todo give a maxFee or for nil
		MaxFee: maxFee,
		//MaxFeeCap:        big.NewInt(10000000000000),
	})
}

func (s SealingAPIAdapter) GetUnPackedDeals(ctx context.Context, miner address.Address, spec *types4.GetDealSpec) ([]*types4.DealInfoIncludePath, error) {
	return s.marketAPI.GetUnPackedDeals(ctx, miner, spec)
}

func (s SealingAPIAdapter) MarkDealsAsPacking(ctx context.Context, miner address.Address, deals []abi.DealID) error {
	return s.marketAPI.MarkDealsAsPacking(ctx, miner, deals)
}

func (s SealingAPIAdapter) UpdateDealOnPacking(ctx context.Context, miner address.Address, dealId abi.DealID, sectorid abi.SectorNumber, offset abi.PaddedPieceSize) error {
	return s.marketAPI.UpdateDealOnPacking(ctx, miner, dealId, sectorid, offset)
}
