package market_client

import (
	"context"
	"encoding/json"
	types3 "github.com/filecoin-project/venus/venus-shared/types"
	"github.com/filecoin-project/venus/venus-shared/types/gateway"
	"time"

	"github.com/modern-go/reflect2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/venus-sealer/sector-storage/stores"
	"github.com/filecoin-project/venus-sealer/sector-storage/storiface"
	types2 "github.com/filecoin-project/venus-sealer/types"

	"github.com/ipfs-force-community/venus-gateway/marketevent"
	"github.com/ipfs-force-community/venus-gateway/types"
	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/venus-market/piecestorage"

	sectorstorage "github.com/filecoin-project/venus-sealer/sector-storage"
	"github.com/filecoin-project/venus-sealer/sector-storage/fr32"
	"github.com/filecoin-project/venus-sealer/storage/sectorblocks"
)

var log = logging.Logger("market_event")

type MarketEvent struct {
	client       IMarketEventClient
	mAddr        types2.MinerAddress
	stor         *stores.Remote
	sectorBlocks *sectorblocks.SectorBlocks
	storageMgr   *sectorstorage.Manager
	index        stores.SectorIndex
	pieceStorage piecestorage.IPieceStorage
}

func (e *MarketEvent) listenMarketRequest(ctx context.Context) {
	log.Infof("start market event listening")
	for {
		if err := e.listenMarketRequestOnce(ctx); err != nil {
			log.Errorf("listen market event errored: %s", err)
		} else {
			log.Warn("list market quit")
		}
		select {
		case <-time.After(time.Second):
		case <-ctx.Done():
			log.Warnf("not restarting market listen: context error: %s", ctx.Err())
			return
		}

		log.Info("restarting market listen")
	}
}

func (e *MarketEvent) listenMarketRequestOnce(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	policy := &gateway.MarketRegisterPolicy{
		Miner: address.Address(e.mAddr),
	}
	marketEventCh, err := e.client.ListenMarketEvent(ctx, policy)
	if err != nil {
		// Retry is handled by caller
		return xerrors.Errorf("listen market event call failed: %w", err)
	}

	for marketEvent := range marketEventCh {
		switch marketEvent.Method {
		case "InitConnect":
			req := types.ConnectedCompleted{}
			err := json.Unmarshal(marketEvent.Payload, &req)
			if err != nil {
				return xerrors.Errorf("odd error in connect %v", err)
			}
			log.Infof("success to connect with market %s", req.ChannelId)
		case "IsUnsealed":
			req := marketevent.IsUnsealRequest{}
			err := json.Unmarshal(marketEvent.Payload, &req)
			if err != nil {
				_ = e.client.ResponseMarketEvent(ctx, &gateway.ResponseEvent{
					ID:      marketEvent.ID,
					Payload: nil,
					Error:   err.Error(),
				})
				continue
			}
			e.processIsUnsealed(ctx, marketEvent.ID, req)
		case "SectorsUnsealPiece":
			req := marketevent.UnsealRequest{}
			err := json.Unmarshal(marketEvent.Payload, &req)
			if err != nil {
				_ = e.client.ResponseMarketEvent(ctx, &gateway.ResponseEvent{
					ID:      marketEvent.ID,
					Payload: nil,
					Error:   err.Error(),
				})
				continue
			}
			e.processSectorUnsealed(ctx, marketEvent.ID, req)
		default:
			log.Errorf("unexpect market event type %s", marketEvent.Method)
		}
	}

	return nil
}

func (e *MarketEvent) processIsUnsealed(ctx context.Context, reqId types3.UUID, req marketevent.IsUnsealRequest) {
	has, err := e.stor.CheckIsUnsealed(ctx, req.Sector, abi.PaddedPieceSize(req.Offset), req.Size)
	if err != nil {
		e.error(ctx, reqId, err)
		return
	}

	e.val(ctx, reqId, has)
}

func (e *MarketEvent) processSectorUnsealed(ctx context.Context, reqId types3.UUID, req marketevent.UnsealRequest) {
	sectorInfo, err := e.sectorBlocks.GetSectorInfo(req.Sector.ID.Number)
	if err != nil {
		e.error(ctx, reqId, err)
		return
	}
	err = e.storageMgr.SectorsUnsealPiece(ctx, req.Sector, storiface.UnpaddedByteIndex(abi.PaddedPieceSize(req.Offset).Unpadded()), req.Size.Unpadded(), sectorInfo.TicketValue, sectorInfo.CommD)
	if err != nil {
		e.error(ctx, reqId, err)
		log.Debugf("unsealer piece file from sector %d %w", req.Sector.ID.Number, err)
		return
	}

	if err := e.index.StorageLock(ctx, req.Sector.ID, storiface.FTUnsealed, storiface.FTNone); err != nil {
		e.error(ctx, reqId, err)
		return
	}

	// Reader returns a reader for an unsealed piece at the given offset in the given sector.
	// The returned reader will be nil if none of the workers has an unsealed sector file containing
	// the unsealed piece.
	rg, err := e.stor.Reader(ctx, req.Sector, abi.PaddedPieceSize(req.Offset), req.Size)
	if err != nil {
		log.Debugf("did not get storage reader;sector=%+v, err:%s", req.Sector.ID, err)
		e.error(ctx, reqId, err)
		return
	}

	if rg == nil {
		return
	}

	r, err := rg(0) // TODO review ???
	if err != nil {
		log.Debugf("getting reader err: %s", err)
		e.error(ctx, reqId, err)
		return
	}

	upr, err := fr32.NewUnpadReader(r, req.Size)
	if err != nil {
		e.error(ctx, reqId, err)
		return
	}

	_, err = e.pieceStorage.SaveTo(ctx, req.Dest, upr)
	if err != nil {
		e.error(ctx, reqId, err)
		return
	}
	e.val(ctx, reqId, nil)
}

func (e *MarketEvent) error(ctx context.Context, reqId types3.UUID, err error) {
	_ = e.client.ResponseMarketEvent(ctx, &gateway.ResponseEvent{
		ID:      reqId,
		Payload: nil,
		Error:   err.Error(),
	})
}

func (e *MarketEvent) val(ctx context.Context, reqId types3.UUID, val interface{}) {
	var respBytes []byte
	if !reflect2.IsNil(val) {
		var err error
		respBytes, err = json.Marshal(val)
		if err != nil {
			e.error(ctx, reqId, err)
			return
		}
	}

	err := e.client.ResponseMarketEvent(ctx, &gateway.ResponseEvent{
		ID:      reqId,
		Payload: respBytes,
		Error:   "",
	})
	if err != nil {
		log.Errorf("response market event %s failed", reqId)
	}
}
