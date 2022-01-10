// blockstore contains all the basic blockstore constructors used by venus. Any
// blockstores not ultimately constructed out of the building blocks in this
// package may not work properly.
//
//  * This package correctly wraps blockstores with the IdBlockstore. This blockstore:
//    * Filters out all puts for blocks with CIDs using the "identity" hash function.
//    * Extracts inlined blocks from CIDs using the identity hash function and
//      returns them on get/has, ignoring the contents of the blockstore.
//  * In the future, this package may enforce additional restrictions on block
//    sizes, CID validity, etc.
//
// To make auditing for misuse of blockstores tractable, this package re-exports
// parts of the go-ipfs-blockstore package such that no other package needs to
// import it directly.
package blockstore

import (
	"context"

	cid "github.com/ipfs/go-cid"
	ds "github.com/ipfs/go-datastore"
	logging "github.com/ipfs/go-log/v2"

	blockstore "github.com/ipfs/go-ipfs-blockstore"
)

var log = logging.Logger("blockstore")

var ErrNotFound = blockstore.ErrNotFound

// Blockstore is the blockstore interface used by Lotus. It is the union
// of the basic go-ipfs blockstore, with other capabilities required by Lotus,
// e.g. View or Sync.
type Blockstore interface {
	blockstore.Blockstore
	blockstore.Viewer
	BatchDeleter
}

// BasicBlockstore is an alias to the original IPFS Blockstore.
type BasicBlockstore = blockstore.Blockstore

type Viewer = blockstore.Viewer


type BatchDeleter interface {
	DeleteMany(ctx context.Context, cids []cid.Cid) error
}

// WrapIDStore wraps the underlying blockstore in an "identity" blockstore.
// The ID store filters out all puts for blocks with CIDs using the "identity"
// hash function. It also extracts inlined blocks from CIDs using the identity
// hash function and returns them on get/has, ignoring the contents of the
// blockstore.
func WrapIDStore(bstore blockstore.Blockstore) Blockstore {
	if is, ok := bstore.(*idstore); ok {
		// already wrapped
		return is
	}

	if bs, ok := bstore.(Blockstore); ok {
		// we need to wrap our own because we don't want to neuter the DeleteMany method
		// the underlying blockstore has implemented an (efficient) DeleteMany
		return NewIDStore(bs)
	}

	// The underlying blockstore does not implement DeleteMany, so we need to shim it.
	// This is less efficient as it'll iterate and perform single deletes.
	return NewIDStore(Adapt(bstore))
}

// FromDatastore creates a new blockstore backed by the given datastore.
func FromDatastore(dstore ds.Batching) Blockstore {
	return WrapIDStore(blockstore.NewBlockstore(dstore))
}

type adaptedBlockstore struct {
	blockstore.Blockstore
}

var _ Blockstore = (*adaptedBlockstore)(nil)

func (a *adaptedBlockstore) View(ctx context.Context, cid cid.Cid, callback func([]byte) error) error {
	blk, err := a.Get(ctx, cid)
	if err != nil {
		return err
	}
	return callback(blk.RawData())
}

func (a *adaptedBlockstore) DeleteMany(ctx context.Context, cids []cid.Cid) error {
	for _, cid := range cids {
		err := a.DeleteBlock(ctx, cid)
		if err != nil {
			return err
		}
	}

	return nil
}

// Adapt adapts a standard blockstore to a Lotus blockstore by
// enriching it with the extra methods that Lotus requires (e.g. View, Sync).
//
// View proxies over to Get and calls the callback with the value supplied by Get.
// Sync noops.
func Adapt(bs blockstore.Blockstore) Blockstore {
	if ret, ok := bs.(Blockstore); ok {
		return ret
	}
	return &adaptedBlockstore{bs}
}
