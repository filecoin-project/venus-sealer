package sealing

import (
	"context"

	"github.com/filecoin-project/go-state-types/abi"

	"github.com/filecoin-project/venus-sealer/types"
)

// `curH`-`ts.Height` = `confidence`
type HeightHandler func(ctx context.Context, tok types.TipSetToken, curH abi.ChainEpoch) error
type RevertHandler func(ctx context.Context, tok types.TipSetToken) error

type Events interface {
	ChainAt(hnd HeightHandler, rev RevertHandler, confidence int, h abi.ChainEpoch) error
}
