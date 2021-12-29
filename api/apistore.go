package api

import (
	"context"

	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/venus-sealer/lib/blockstore"
)

type ChainIO interface {
	ChainReadObj(context.Context, cid.Cid) ([]byte, error)
	ChainHasObj(context.Context, cid.Cid) (bool, error)
}

type apiBStore struct {
	api ChainIO
}

func NewAPIBlockstore(cio ChainIO) blockstore.Blockstore {
	return &apiBStore{
		api: cio,
	}
}

func (a *apiBStore) DeleteBlock(context.Context, cid.Cid) error {
	return xerrors.New("not supported")
}

func (a *apiBStore) Has(ctx context.Context, c cid.Cid) (bool, error) {
	return a.api.ChainHasObj(ctx, c)
}

func (a *apiBStore) Get(ctx context.Context, c cid.Cid) (blocks.Block, error) {
	bb, err := a.api.ChainReadObj(ctx, c)
	if err != nil {
		return nil, err
	}
	return blocks.NewBlockWithCid(bb, c)
}

func (a *apiBStore) GetSize(ctx context.Context, c cid.Cid) (int, error) {
	bb, err := a.api.ChainReadObj(ctx, c)
	if err != nil {
		return 0, err
	}
	return len(bb), nil
}

func (a *apiBStore) Put(context.Context, blocks.Block) error {
	return xerrors.New("not supported")
}

func (a *apiBStore) PutMany(context.Context, []blocks.Block) error {
	return xerrors.New("not supported")
}

func (a *apiBStore) AllKeysChan(ctx context.Context) (<-chan cid.Cid, error) {
	return nil, xerrors.New("not supported")
}

func (a *apiBStore) HashOnRead(enabled bool) {
}

var _ blockstore.Blockstore = &apiBStore{}
