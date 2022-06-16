package sealing

import (
	"bytes"
	"context"
	"fmt"

	market8 "github.com/filecoin-project/go-state-types/builtin/v8/market"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/exitcode"
	"github.com/filecoin-project/go-state-types/network"

	"github.com/ipfs/go-cid"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/venus/pkg/constants"
	"github.com/filecoin-project/venus/venus-shared/actors/builtin/market"
	"github.com/filecoin-project/venus/venus-shared/types"

	types2 "github.com/filecoin-project/venus-sealer/types"
)

type CurrentDealInfoAPI interface {
	ChainGetMessage(context.Context, cid.Cid) (*types.Message, error)
	StateLookupID(context.Context, address.Address, types2.TipSetToken) (address.Address, error)
	StateMarketStorageDeal(context.Context, abi.DealID, types2.TipSetToken) (*types.MarketDeal, error)
	StateSearchMsg(context.Context, cid.Cid) (*types2.MsgLookup, error)
	StateNetworkVersion(ctx context.Context, tok types2.TipSetToken) (network.Version, error)
}

type CurrentDealInfo struct {
	DealID           abi.DealID
	MarketDeal       *types.MarketDeal
	PublishMsgTipSet types2.TipSetToken
}

type CurrentDealInfoManager struct {
	CDAPI CurrentDealInfoAPI
}

// GetCurrentDealInfo gets the current deal state and deal ID.
// Note that the deal ID is assigned when the deal is published, so it may
// have changed if there was a reorg after the deal was published.
func (mgr *CurrentDealInfoManager) GetCurrentDealInfo(ctx context.Context, tok types2.TipSetToken, proposal *market.DealProposal, publishCid cid.Cid) (CurrentDealInfo, error) {
	// Lookup the deal ID by comparing the deal proposal to the proposals in
	// the publish deals message, and indexing into the message return value
	dealID, pubMsgTok, err := mgr.dealIDFromPublishDealsMsg(ctx, tok, proposal, publishCid)
	if err != nil {
		return CurrentDealInfo{}, err
	}

	// Lookup the deal state by deal ID
	marketDeal, err := mgr.CDAPI.StateMarketStorageDeal(ctx, dealID, tok)
	if err == nil && proposal != nil {
		// Make sure the retrieved deal proposal matches the target proposal
		equal, err := mgr.CheckDealEquality(ctx, tok, *proposal, marketDeal.Proposal)
		if err != nil {
			return CurrentDealInfo{}, err
		}
		if !equal {
			return CurrentDealInfo{}, xerrors.Errorf("Deal proposals for publish message %s did not match", publishCid)
		}
	}
	return CurrentDealInfo{DealID: dealID, MarketDeal: marketDeal, PublishMsgTipSet: pubMsgTok}, err
}

// dealIDFromPublishDealsMsg looks up the publish deals message by cid, and finds the deal ID
// by looking at the message return value
func (mgr *CurrentDealInfoManager) dealIDFromPublishDealsMsg(ctx context.Context, tok types2.TipSetToken, proposal *market.DealProposal, publishCid cid.Cid) (abi.DealID, types2.TipSetToken, error) {
	dealID := abi.DealID(0)

	// Get the return value of the publish deals message
	lookup, err := mgr.CDAPI.StateSearchMsg(ctx, publishCid)
	if err != nil {
		return dealID, nil, xerrors.Errorf("looking for publish deal message %s: search msg failed: %w", publishCid, err)
	}

	if lookup == nil {
		return dealID, nil, xerrors.Errorf("looking for publish deal message %s: not found", publishCid)
	}

	if lookup.Receipt.ExitCode != exitcode.Ok {
		return dealID, nil, xerrors.Errorf("looking for publish deal message %s: non-ok exit code: %s", publishCid, lookup.Receipt.ExitCode)
	}

	nv, err := mgr.CDAPI.StateNetworkVersion(ctx, lookup.TipSetTok)
	if err != nil {
		return dealID, nil, xerrors.Errorf("getting network version: %w", err)
	}

	retval, err := market.DecodePublishStorageDealsReturn(lookup.Receipt.Return, nv)
	if err != nil {
		return dealID, nil, xerrors.Errorf("looking for publish deal message %s: decoding message return: %w", publishCid, err)
	}

	dealIDs, err := retval.DealIDs()
	if err != nil {
		return dealID, nil, xerrors.Errorf("looking for publish deal message %s: getting dealIDs: %w", publishCid, err)
	}

	// TODO: Can we delete this? We're well past the point when we first introduced the proposals into sealing deal info
	// Previously, publish deals messages contained a single deal, and the
	// deal proposal was not included in the sealing deal info.
	// So check if the proposal is nil and check the number of deals published
	// in the message.
	if proposal == nil {
		if len(dealIDs) > 1 {
			return dealID, nil, xerrors.Errorf(
				"getting deal ID from publish deal message %s: "+
					"no deal proposal supplied but message return value has more than one deal (%d deals)",
				publishCid, len(dealIDs))
		}

		// There is a single deal in this publish message and no deal proposal
		// was supplied, so we have nothing to compare against. Just assume
		// the deal ID is correct and that it was valid
		return dealIDs[0], lookup.TipSetTok, nil
	}

	// Get the parameters to the publish deals message
	pubmsg, err := mgr.CDAPI.ChainGetMessage(ctx, publishCid)
	if err != nil {
		return dealID, nil, xerrors.Errorf("getting publish deal message %s: %w", publishCid, err)
	}

	var pubDealsParams market8.PublishStorageDealsParams
	if err := pubDealsParams.UnmarshalCBOR(bytes.NewReader(pubmsg.Params)); err != nil {
		return dealID, nil, xerrors.Errorf("unmarshalling publish deal message params for message %s: %w", publishCid, err)
	}

	// Scan through the deal proposals in the message parameters to find the
	// index of the target deal proposal
	dealIdx := -1
	for i, paramDeal := range pubDealsParams.Deals {
		eq, err := mgr.CheckDealEquality(ctx, tok, *proposal, paramDeal.Proposal)
		if err != nil {
			return dealID, nil, xerrors.Errorf("comparing publish deal message %s proposal to deal proposal: %w", publishCid, err)
		}
		if eq {
			dealIdx = i
			break
		}
	}
	fmt.Printf("found dealIdx %d\n", dealIdx)

	if dealIdx == -1 {
		return dealID, nil, xerrors.Errorf("could not find deal in publish deals message %s", publishCid)
	}

	if dealIdx >= len(pubDealsParams.Deals) {
		return dealID, nil, xerrors.Errorf(
			"deal index %d out of bounds of deal proposals (len %d) in publish deals message %s",
			dealIdx, len(dealIDs), publishCid)
	}

	valid, outIdx, err := retval.IsDealValid(uint64(dealIdx))
	if err != nil {
		return dealID, nil, xerrors.Errorf("determining deal validity: %w", err)
	}

	if !valid {
		return dealID, nil, xerrors.New("deal was invalid at publication")
	}

	// final check against for invalid return value output
	// should not be reachable from onchain output, only pathological test cases
	if outIdx >= len(dealIDs) {
		return dealID, nil, xerrors.Errorf("invalid publish storage deals ret marking %d as valid while only returning %d valid deals in publish deal message %s", outIdx, len(dealIDs), publishCid)
	}
	return dealIDs[outIdx], lookup.TipSetTok, nil
}

func (mgr *CurrentDealInfoManager) CheckDealEquality(ctx context.Context, tok types2.TipSetToken, p1, p2 market.DealProposal) (bool, error) {
	p1ClientID, err := mgr.CDAPI.StateLookupID(ctx, p1.Client, tok)
	if err != nil {
		return false, err
	}
	p2ClientID, err := mgr.CDAPI.StateLookupID(ctx, p2.Client, tok)
	if err != nil {
		return false, err
	}
	return p1.PieceCID.Equals(p2.PieceCID) &&
		p1.PieceSize == p2.PieceSize &&
		p1.VerifiedDeal == p2.VerifiedDeal &&
		p1.Label.Equals(p2.Label) &&
		p1.StartEpoch == p2.StartEpoch &&
		p1.EndEpoch == p2.EndEpoch &&
		p1.StoragePricePerEpoch.Equals(p2.StoragePricePerEpoch) &&
		p1.ProviderCollateral.Equals(p2.ProviderCollateral) &&
		p1.ClientCollateral.Equals(p2.ClientCollateral) &&
		p1.Provider == p2.Provider &&
		p1ClientID == p2ClientID, nil
}

type CurrentDealInfoTskAPI interface {
	ChainGetMessage(ctx context.Context, mc cid.Cid) (*types.Message, error)
	StateLookupID(context.Context, address.Address, types.TipSetKey) (address.Address, error)
	StateMarketStorageDeal(context.Context, abi.DealID, types.TipSetKey) (*types.MarketDeal, error)
	StateNetworkVersion(ctx context.Context, tok types.TipSetKey) (network.Version, error)
	StateSearchMsg(ctx context.Context, from types.TipSetKey, msg cid.Cid, limit abi.ChainEpoch, allowReplaced bool) (*types.MsgLookup, error)
}

type CurrentDealInfoAPIAdapter struct {
	CurrentDealInfoTskAPI
}

func (c *CurrentDealInfoAPIAdapter) StateLookupID(ctx context.Context, a address.Address, tok types2.TipSetToken) (address.Address, error) {
	tsk, err := types.TipSetKeyFromBytes(tok)
	if err != nil {
		return address.Undef, xerrors.Errorf("failed to unmarshal TipSetToken to TipSetKey: %w", err)
	}

	return c.CurrentDealInfoTskAPI.StateLookupID(ctx, a, tsk)
}

func (c *CurrentDealInfoAPIAdapter) StateMarketStorageDeal(ctx context.Context, dealID abi.DealID, tok types2.TipSetToken) (*types.MarketDeal, error) {
	tsk, err := types.TipSetKeyFromBytes(tok)
	if err != nil {
		return nil, xerrors.Errorf("failed to unmarshal TipSetToken to TipSetKey: %w", err)
	}

	return c.CurrentDealInfoTskAPI.StateMarketStorageDeal(ctx, dealID, tsk)
}

func (c *CurrentDealInfoAPIAdapter) StateSearchMsg(ctx context.Context, k cid.Cid) (*types2.MsgLookup, error) {
	wmsg, err := c.CurrentDealInfoTskAPI.StateSearchMsg(ctx, types.EmptyTSK, k, constants.LookbackNoLimit, true)
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

func (c *CurrentDealInfoAPIAdapter) StateNetworkVersion(ctx context.Context, tok types2.TipSetToken) (network.Version, error) {
	tsk, err := types.TipSetKeyFromBytes(tok)
	if err != nil {
		return network.VersionMax, xerrors.Errorf("failed to unmarshal TipSetToken to TipSetKey: %w", err)
	}

	return c.CurrentDealInfoTskAPI.StateNetworkVersion(ctx, tsk)
}

var _ CurrentDealInfoAPI = (*CurrentDealInfoAPIAdapter)(nil)
