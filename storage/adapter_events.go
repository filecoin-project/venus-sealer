package storage

import (
	"context"

	"github.com/filecoin-project/go-state-types/abi"

	sealing "github.com/filecoin-project/venus-sealer/storage-sealing"
	"github.com/filecoin-project/venus/pkg/events"
	"github.com/filecoin-project/venus/pkg/types"
)

var _ sealing.Events = new(EventsAdapter)

type EventsAdapter struct {
	delegate *events.Events
}

func NewEventsAdapter(api *events.Events) EventsAdapter {
	return EventsAdapter{delegate: api}
}

func (e EventsAdapter) ChainAt(hnd sealing.HeightHandler, rev sealing.RevertHandler, confidence int, h abi.ChainEpoch) error {
	return e.delegate.ChainAt(func(ctx context.Context, ts *types.TipSet, curH abi.ChainEpoch) error {
		return hnd(ctx, ts.Key().Bytes(), curH)
	}, func(ctx context.Context, ts *types.TipSet) error {
		return rev(ctx, ts.Key().Bytes())
	}, confidence, h)
}
