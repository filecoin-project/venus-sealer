package market_client

import (
	"context"
	"github.com/modern-go/reflect2"

	"encoding/json"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/venus-sealer/sector-storage/stores"
	"github.com/filecoin-project/venus-sealer/sector-storage/storiface"
	types2 "github.com/filecoin-project/venus-sealer/types"
	"github.com/google/uuid"
	"github.com/ipfs-force-community/venus-gateway/marketevent"
	"github.com/ipfs-force-community/venus-gateway/types"
	logging "github.com/ipfs/go-log/v2"

	sectorstorage "github.com/filecoin-project/venus-sealer/sector-storage"
	"github.com/filecoin-project/venus-sealer/storage/sectorblocks"
	"golang.org/x/xerrors"
	"time"
)

var log = logging.Logger("proof_event")

type MarketEvent struct {
	client       *MarketEventClient
	mAddr        types2.MinerAddress
	stor         *stores.Remote
	sectorBlocks *sectorblocks.SectorBlocks
	storageMgr   *sectorstorage.Manager
}

func (e *MarketEvent) listenMarketRequest(ctx context.Context) {
	for {
		if err := e.listenMarketRequestOnce(ctx); err != nil {
			log.Errorf("listen market event errored: %s", err)
		} else {
			log.Warn("listenHeadChanges quit")
		}
		select {
		case <-time.After(time.Second):
		case <-ctx.Done():
			log.Warnf("not restarting listenHeadChanges: context error: %s", ctx.Err())
			return
		}

		log.Info("restarting market listen")
	}
}

func (e *MarketEvent) listenMarketRequestOnce(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	policy := &marketevent.MarketRegisterPolicy{
		Miner: address.Address(e.mAddr),
	}
	proofEventCh, err := e.client.ListenMarketEvent(ctx, policy)
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
		case "IsUnsealed":
			req := marketevent.IsUnsealRequest{}
			err := json.Unmarshal(proofEvent.Payload, &req)
			if err != nil {
				_ = e.client.ResponseMarketEvent(ctx, &types.ResponseEvent{
					Id:      proofEvent.Id,
					Payload: nil,
					Error:   err.Error(),
				})
				continue
			}
			e.processIsUnsealed(ctx, proofEvent.Id, req)
		case "SectorsUnsealPiece":
			req := marketevent.UnsealRequest{}
			err := json.Unmarshal(proofEvent.Payload, &req)
			if err != nil {
				_ = e.client.ResponseMarketEvent(ctx, &types.ResponseEvent{
					Id:      proofEvent.Id,
					Payload: nil,
					Error:   err.Error(),
				})
				continue
			}
			e.processSectorUnsealed(ctx, proofEvent.Id, req)
		default:
			log.Errorf("unexpect proof event type %s", proofEvent.Method)
		}
	}

	return nil
}

func (e *MarketEvent) processIsUnsealed(ctx context.Context, reqId uuid.UUID, req marketevent.IsUnsealRequest) {
	has, err := e.stor.CheckIsUnsealed(ctx, req.Sector, abi.PaddedPieceSize(req.Offset), req.Size)
	if err != nil {
		e.error(ctx, reqId, err)
		return
	}

	e.val(ctx, reqId, has)
}

func (e *MarketEvent) processSectorUnsealed(ctx context.Context, reqId uuid.UUID, req marketevent.UnsealRequest) {
	sectorInfo, err := e.sectorBlocks.GetSectorInfo(req.Sector.ID.Number)
	if err != nil {
		e.error(ctx, reqId, err)
		return
	}
	err = e.storageMgr.SectorsUnsealPiece(ctx, req.Sector, storiface.UnpaddedByteIndex(abi.PaddedPieceSize(req.Offset).Unpadded()), req.Size.Unpadded(), sectorInfo.TicketValue, sectorInfo.CommD)
	if err != nil {
		e.error(ctx, reqId, err)
		return
	}

	e.val(ctx, reqId, nil)
}

func (e *MarketEvent) error(ctx context.Context, reqId uuid.UUID, err error) {
	_ = e.client.ResponseMarketEvent(ctx, &types.ResponseEvent{
		Id:      reqId,
		Payload: nil,
		Error:   err.Error(),
	})
}
func (e *MarketEvent) val(ctx context.Context, reqId uuid.UUID, val interface{}) {

	var respBytes []byte
	if !reflect2.IsNil(val) {
		var err error
		respBytes, err = json.Marshal(val)
		if err != nil {
			e.error(ctx, reqId, err)
			return
		}
	}

	err := e.client.ResponseMarketEvent(ctx, &types.ResponseEvent{
		Id:      reqId,
		Payload: respBytes,
		Error:   "",
	})
	if err != nil {
		log.Errorf("response market event %s failed", reqId)
	}
}
