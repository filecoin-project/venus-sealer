package sectorblocks

import (
	"context"
	"encoding/binary"
	"errors"
	"github.com/filecoin-project/venus-sealer/service"
	"github.com/filecoin-project/venus-sealer/types"
	"io"
	"sync"

	"github.com/ipfs/go-datastore"
	dshelp "github.com/ipfs/go-ipfs-ds-help"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/venus-sealer/storage"
)

type SealSerialization uint8

const (
	SerializationUnixfs0 SealSerialization = 'u'
)

var dsPrefix = datastore.NewKey("/sealedblocks")

var ErrNotFound = errors.New("not found")

func DealIDToDsKey(dealID abi.DealID) datastore.Key {
	buf := make([]byte, binary.MaxVarintLen64)
	size := binary.PutUvarint(buf, uint64(dealID))
	return dshelp.NewKeyFromBinary(buf[:size])
}

func DsKeyToDealID(key datastore.Key) (uint64, error) {
	buf, err := dshelp.BinaryFromDsKey(key)
	if err != nil {
		return 0, err
	}
	dealID, _ := binary.Uvarint(buf)
	return dealID, nil
}

type SectorBlocks struct {
	*storage.Miner

	keys  types.DealRef
	keyLk sync.Mutex
}

func NewSectorBlocks(miner *storage.Miner, ds *service.DealRefService) *SectorBlocks {
	sbc := &SectorBlocks{
		Miner: miner,
		keys:  ds,
	}

	return sbc
}

func (st *SectorBlocks) AddPiece(ctx context.Context, size abi.UnpaddedPieceSize, r io.Reader, d types.DealInfo) (abi.SectorNumber, abi.PaddedPieceSize, error) {
	sn, offset, err := st.Miner.AddPieceToAnySector(ctx, size, r, d)
	if err != nil {
		return 0, 0, err
	}

	// TODO: DealID has very low finality here
	st.keyLk.Lock() // TODO: make this multithreaded
	defer st.keyLk.Unlock()

	//todo save more db to database
	err = st.keys.Save(uint64(d.DealID), types.SealedRef{
		SectorID: sn,
		Offset:   offset,
		Size:     size,
	}, d.DealProposal) // TODO: batch somehow
	if err != nil {
		return 0, 0, xerrors.Errorf("writeRef: %w", err)
	}

	return sn, offset, nil
}

func (st *SectorBlocks) List() (map[uint64][]types.SealedRef, error) {
	return st.keys.List()
}

func (st *SectorBlocks) GetRefs(dealID abi.DealID) ([]types.SealedRef, error) { // TODO: track local sectors
	ent, err := st.keys.Get(uint64(dealID))
	if err != nil {
		return nil, err
	}

	return ent.Refs, nil
}

func (st *SectorBlocks) GetSize(dealID abi.DealID) (uint64, error) {
	refs, err := st.GetRefs(dealID)
	if err != nil {
		return 0, err
	}

	return uint64(refs[0].Size), nil
}

func (st *SectorBlocks) Has(dealID abi.DealID) (bool, error) {
	// TODO: ensure sector is still there
	return st.keys.Has(uint64(dealID))
}
