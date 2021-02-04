package api

import (
	"context"
	"github.com/filecoin-project/go-address"
	datatransfer "github.com/filecoin-project/go-data-transfer"
	"github.com/filecoin-project/go-fil-markets/piecestore"
	"github.com/filecoin-project/go-fil-markets/retrievalmarket"
	"github.com/filecoin-project/go-fil-markets/storagemarket"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-storage/storage"
	"github.com/filecoin-project/venus-sealer/config"
	"github.com/filecoin-project/venus-sealer/extern/sector-storage/fsutil"
	"github.com/filecoin-project/venus-sealer/extern/sector-storage/stores"
	"github.com/filecoin-project/venus-sealer/extern/sector-storage/storiface"
	"github.com/google/uuid"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/peer"
	"time"
)

// StorageMiner is a low-level interface to the Filecoin network storage miner node
type StorageMiner interface {
	Common

	NetParamsConfig(ctx context.Context) (*config.NetParamsConfig, error)

	ActorAddress(context.Context) (address.Address, error)

	ActorSectorSize(context.Context, address.Address) (abi.SectorSize, error)
	ActorAddressConfig(ctx context.Context) (AddressConfig, error)

	// Temp api for testing
	PledgeSector(context.Context) error

	// Get the status of a given sector by ID
	SectorsStatus(ctx context.Context, sid abi.SectorNumber, showOnChainInfo bool) (SectorInfo, error)

	// List all staged sectors
	SectorsList(context.Context) ([]abi.SectorNumber, error)

	// Get summary info of sectors
	SectorsSummary(ctx context.Context) (map[SectorState]int, error)

	// List sectors in particular states
	SectorsListInStates(context.Context, []SectorState) ([]abi.SectorNumber, error)

	SectorsRefs(context.Context) (map[string][]SealedRef, error)

	// SectorStartSealing can be called on sectors in Empty or WaitDeals states
	// to trigger sealing early
	SectorStartSealing(context.Context, abi.SectorNumber) error
	// SectorSetSealDelay sets the time that a newly-created sector
	// waits for more deals before it starts sealing
	SectorSetSealDelay(context.Context, time.Duration) error
	// SectorGetSealDelay gets the time that a newly-created sector
	// waits for more deals before it starts sealing
	SectorGetSealDelay(context.Context) (time.Duration, error)
	// SectorSetExpectedSealDuration sets the expected time for a sector to seal
	SectorSetExpectedSealDuration(context.Context, time.Duration) error
	// SectorGetExpectedSealDuration gets the expected time for a sector to seal
	SectorGetExpectedSealDuration(context.Context) (time.Duration, error)
	SectorsUpdate(context.Context, abi.SectorNumber, SectorState) error
	// SectorRemove removes the sector from storage. It doesn't terminate it on-chain, which can
	// be done with SectorTerminate. Removing and not terminating live sectors will cause additional penalties.
	SectorRemove(context.Context, abi.SectorNumber) error
	// SectorTerminate terminates the sector on-chain (adding it to a termination batch first), then
	// automatically removes it from storage
	SectorTerminate(context.Context, abi.SectorNumber) error
	// SectorTerminateFlush immediately sends a terminate message with sectors batched for termination.
	// Returns null if message wasn't sent
	SectorTerminateFlush(ctx context.Context) (*cid.Cid, error)
	// SectorTerminatePending returns a list of pending sector terminations to be sent in the next batch message
	SectorTerminatePending(ctx context.Context) ([]abi.SectorID, error)
	SectorMarkForUpgrade(ctx context.Context, id abi.SectorNumber) error

	StorageList(ctx context.Context) (map[stores.ID][]stores.Decl, error)
	StorageLocal(ctx context.Context) (map[stores.ID]string, error)
	StorageStat(ctx context.Context, id stores.ID) (fsutil.FsStat, error)

	// WorkerConnect tells the node to connect to workers RPC
	WorkerConnect(context.Context, string) error
	WorkerStats(context.Context) (map[uuid.UUID]storiface.WorkerStats, error)
	WorkerJobs(context.Context) (map[uuid.UUID][]storiface.WorkerJob, error)
	storiface.WorkerReturn

	// SealingSchedDiag dumps internal sealing scheduler state
	SealingSchedDiag(ctx context.Context, doSched bool) (interface{}, error)
	SealingAbort(ctx context.Context, call storiface.CallID) error

	stores.SectorIndex

	MarketImportDealData(ctx context.Context, propcid cid.Cid, path string) error
	MarketListDeals(ctx context.Context) ([]MarketDeal, error)
	MarketListRetrievalDeals(ctx context.Context) ([]retrievalmarket.ProviderDealState, error)
	MarketGetDealUpdates(ctx context.Context) (<-chan storagemarket.MinerDeal, error)
	MarketListIncompleteDeals(ctx context.Context) ([]storagemarket.MinerDeal, error)
	MarketSetAsk(ctx context.Context, price types.BigInt, verifiedPrice types.BigInt, duration abi.ChainEpoch, minPieceSize abi.PaddedPieceSize, maxPieceSize abi.PaddedPieceSize) error
	MarketGetAsk(ctx context.Context) (*storagemarket.SignedStorageAsk, error)
	MarketSetRetrievalAsk(ctx context.Context, rask *retrievalmarket.Ask) error
	MarketGetRetrievalAsk(ctx context.Context) (*retrievalmarket.Ask, error)
	MarketListDataTransfers(ctx context.Context) ([]DataTransferChannel, error)
	MarketDataTransferUpdates(ctx context.Context) (<-chan DataTransferChannel, error)
	// MinerRestartDataTransfer attempts to restart a data transfer with the given transfer ID and other peer
	MarketRestartDataTransfer(ctx context.Context, transferID datatransfer.TransferID, otherPeer peer.ID, isInitiator bool) error
	// ClientCancelDataTransfer cancels a data transfer with the given transfer ID and other peer
	MarketCancelDataTransfer(ctx context.Context, transferID datatransfer.TransferID, otherPeer peer.ID, isInitiator bool) error

	DealsImportData(ctx context.Context, dealPropCid cid.Cid, file string) error
	DealsList(ctx context.Context) ([]MarketDeal, error)
	DealsConsiderOnlineStorageDeals(context.Context) (bool, error)
	DealsSetConsiderOnlineStorageDeals(context.Context, bool) error
	DealsConsiderOnlineRetrievalDeals(context.Context) (bool, error)
	DealsSetConsiderOnlineRetrievalDeals(context.Context, bool) error
	DealsPieceCidBlocklist(context.Context) ([]cid.Cid, error)
	DealsSetPieceCidBlocklist(context.Context, []cid.Cid) error
	DealsConsiderOfflineStorageDeals(context.Context) (bool, error)
	DealsSetConsiderOfflineStorageDeals(context.Context, bool) error
	DealsConsiderOfflineRetrievalDeals(context.Context) (bool, error)
	DealsSetConsiderOfflineRetrievalDeals(context.Context, bool) error
	DealsConsiderVerifiedStorageDeals(context.Context) (bool, error)
	DealsSetConsiderVerifiedStorageDeals(context.Context, bool) error
	DealsConsiderUnverifiedStorageDeals(context.Context) (bool, error)
	DealsSetConsiderUnverifiedStorageDeals(context.Context, bool) error

	StorageAddLocal(ctx context.Context, path string) error

	PiecesListPieces(ctx context.Context) ([]cid.Cid, error)
	PiecesListCidInfos(ctx context.Context) ([]cid.Cid, error)
	PiecesGetPieceInfo(ctx context.Context, pieceCid cid.Cid) (*piecestore.PieceInfo, error)
	PiecesGetCIDInfo(ctx context.Context, payloadCid cid.Cid) (*piecestore.CIDInfo, error)

	// CreateBackup creates node backup onder the specified file name. The
	// method requires that the lotus-miner is running with the
	// LOTUS_BACKUP_BASE_PATH environment variable set to some path, and that
	// the path specified when calling CreateBackup is within the base path
	CreateBackup(ctx context.Context, fpath string) error

	CheckProvable(ctx context.Context, pp abi.RegisteredPoStProof, sectors []storage.SectorRef, expensive bool) (map[abi.SectorNumber]string, error)
}

// StorageMinerStruct
type StorageMinerStruct struct {
	CommonStruct
	Internal struct {
		ActorAddress       func(context.Context) (address.Address, error)                 `perm:"read"`
		ActorSectorSize    func(context.Context, address.Address) (abi.SectorSize, error) `perm:"read"`
		ActorAddressConfig func(ctx context.Context) (AddressConfig, error)               `perm:"read"`
		NetParamsConfig    func(ctx context.Context) (*config.NetParamsConfig, error)     `perm:"read"`

		MarketImportDealData      func(context.Context, cid.Cid, string) error                                                                                                                                 `perm:"write"`
		MarketListDeals           func(ctx context.Context) ([]MarketDeal, error)                                                                                                                              `perm:"read"`
		MarketListRetrievalDeals  func(ctx context.Context) ([]retrievalmarket.ProviderDealState, error)                                                                                                       `perm:"read"`
		MarketGetDealUpdates      func(ctx context.Context) (<-chan storagemarket.MinerDeal, error)                                                                                                            `perm:"read"`
		MarketListIncompleteDeals func(ctx context.Context) ([]storagemarket.MinerDeal, error)                                                                                                                 `perm:"read"`
		MarketSetAsk              func(ctx context.Context, price types.BigInt, verifiedPrice types.BigInt, duration abi.ChainEpoch, minPieceSize abi.PaddedPieceSize, maxPieceSize abi.PaddedPieceSize) error `perm:"admin"`
		MarketGetAsk              func(ctx context.Context) (*storagemarket.SignedStorageAsk, error)                                                                                                           `perm:"read"`
		MarketSetRetrievalAsk     func(ctx context.Context, rask *retrievalmarket.Ask) error                                                                                                                   `perm:"admin"`
		MarketGetRetrievalAsk     func(ctx context.Context) (*retrievalmarket.Ask, error)                                                                                                                      `perm:"read"`
		MarketListDataTransfers   func(ctx context.Context) ([]DataTransferChannel, error)                                                                                                                     `perm:"write"`
		MarketDataTransferUpdates func(ctx context.Context) (<-chan DataTransferChannel, error)                                                                                                                `perm:"write"`
		MarketRestartDataTransfer func(ctx context.Context, transferID datatransfer.TransferID, otherPeer peer.ID, isInitiator bool) error                                                                     `perm:"read"`
		MarketCancelDataTransfer  func(ctx context.Context, transferID datatransfer.TransferID, otherPeer peer.ID, isInitiator bool) error                                                                     `perm:"read"`

		PledgeSector func(context.Context) error `perm:"write"`

		SectorsStatus                 func(ctx context.Context, sid abi.SectorNumber, showOnChainInfo bool) (SectorInfo, error) `perm:"read"`
		SectorsList                   func(context.Context) ([]abi.SectorNumber, error)                                         `perm:"read"`
		SectorsListInStates           func(context.Context, []SectorState) ([]abi.SectorNumber, error)                          `perm:"read"`
		SectorsSummary                func(ctx context.Context) (map[SectorState]int, error)                                    `perm:"read"`
		SectorsRefs                   func(context.Context) (map[string][]SealedRef, error)                                     `perm:"read"`
		SectorStartSealing            func(context.Context, abi.SectorNumber) error                                             `perm:"write"`
		SectorSetSealDelay            func(context.Context, time.Duration) error                                                `perm:"write"`
		SectorGetSealDelay            func(context.Context) (time.Duration, error)                                              `perm:"read"`
		SectorSetExpectedSealDuration func(context.Context, time.Duration) error                                                `perm:"write"`
		SectorGetExpectedSealDuration func(context.Context) (time.Duration, error)                                              `perm:"read"`
		SectorsUpdate                 func(context.Context, abi.SectorNumber, SectorState) error                                `perm:"admin"`
		SectorRemove                  func(context.Context, abi.SectorNumber) error                                             `perm:"admin"`
		SectorTerminate               func(context.Context, abi.SectorNumber) error                                             `perm:"admin"`
		SectorTerminateFlush          func(ctx context.Context) (*cid.Cid, error)                                               `perm:"admin"`
		SectorTerminatePending        func(ctx context.Context) ([]abi.SectorID, error)                                         `perm:"admin"`
		SectorMarkForUpgrade          func(ctx context.Context, id abi.SectorNumber) error                                      `perm:"admin"`

		WorkerConnect func(context.Context, string) error                                `perm:"admin" retry:"true"` // TODO: worker perm
		WorkerStats   func(context.Context) (map[uuid.UUID]storiface.WorkerStats, error) `perm:"admin"`
		WorkerJobs    func(context.Context) (map[uuid.UUID][]storiface.WorkerJob, error) `perm:"admin"`

		ReturnAddPiece        func(ctx context.Context, callID storiface.CallID, pi abi.PieceInfo, err *storiface.CallError) error          `perm:"admin" retry:"true"`
		ReturnSealPreCommit1  func(ctx context.Context, callID storiface.CallID, p1o storage.PreCommit1Out, err *storiface.CallError) error `perm:"admin" retry:"true"`
		ReturnSealPreCommit2  func(ctx context.Context, callID storiface.CallID, sealed storage.SectorCids, err *storiface.CallError) error `perm:"admin" retry:"true"`
		ReturnSealCommit1     func(ctx context.Context, callID storiface.CallID, out storage.Commit1Out, err *storiface.CallError) error    `perm:"admin" retry:"true"`
		ReturnSealCommit2     func(ctx context.Context, callID storiface.CallID, proof storage.Proof, err *storiface.CallError) error       `perm:"admin" retry:"true"`
		ReturnFinalizeSector  func(ctx context.Context, callID storiface.CallID, err *storiface.CallError) error                            `perm:"admin" retry:"true"`
		ReturnReleaseUnsealed func(ctx context.Context, callID storiface.CallID, err *storiface.CallError) error                            `perm:"admin" retry:"true"`
		ReturnMoveStorage     func(ctx context.Context, callID storiface.CallID, err *storiface.CallError) error                            `perm:"admin" retry:"true"`
		ReturnUnsealPiece     func(ctx context.Context, callID storiface.CallID, err *storiface.CallError) error                            `perm:"admin" retry:"true"`
		ReturnReadPiece       func(ctx context.Context, callID storiface.CallID, ok bool, err *storiface.CallError) error                   `perm:"admin" retry:"true"`
		ReturnFetch           func(ctx context.Context, callID storiface.CallID, err *storiface.CallError) error                            `perm:"admin" retry:"true"`

		SealingSchedDiag func(context.Context, bool) (interface{}, error)       `perm:"admin"`
		SealingAbort     func(ctx context.Context, call storiface.CallID) error `perm:"admin"`

		StorageList          func(context.Context) (map[stores.ID][]stores.Decl, error)                                                                                   `perm:"admin"`
		StorageLocal         func(context.Context) (map[stores.ID]string, error)                                                                                          `perm:"admin"`
		StorageStat          func(context.Context, stores.ID) (fsutil.FsStat, error)                                                                                      `perm:"admin"`
		StorageAttach        func(context.Context, stores.StorageInfo, fsutil.FsStat) error                                                                               `perm:"admin"`
		StorageDeclareSector func(context.Context, stores.ID, abi.SectorID, storiface.SectorFileType, bool) error                                                         `perm:"admin"`
		StorageDropSector    func(context.Context, stores.ID, abi.SectorID, storiface.SectorFileType) error                                                               `perm:"admin"`
		StorageFindSector    func(context.Context, abi.SectorID, storiface.SectorFileType, abi.SectorSize, bool) ([]stores.SectorStorageInfo, error)                      `perm:"admin"`
		StorageInfo          func(context.Context, stores.ID) (stores.StorageInfo, error)                                                                                 `perm:"admin"`
		StorageBestAlloc     func(ctx context.Context, allocate storiface.SectorFileType, ssize abi.SectorSize, sealing storiface.PathType) ([]stores.StorageInfo, error) `perm:"admin"`
		StorageReportHealth  func(ctx context.Context, id stores.ID, report stores.HealthReport) error                                                                    `perm:"admin"`
		StorageLock          func(ctx context.Context, sector abi.SectorID, read storiface.SectorFileType, write storiface.SectorFileType) error                          `perm:"admin"`
		StorageTryLock       func(ctx context.Context, sector abi.SectorID, read storiface.SectorFileType, write storiface.SectorFileType) (bool, error)                  `perm:"admin"`

		DealsImportData                        func(ctx context.Context, dealPropCid cid.Cid, file string) error `perm:"write"`
		DealsList                              func(ctx context.Context) ([]MarketDeal, error)                   `perm:"read"`
		DealsConsiderOnlineStorageDeals        func(context.Context) (bool, error)                               `perm:"read"`
		DealsSetConsiderOnlineStorageDeals     func(context.Context, bool) error                                 `perm:"admin"`
		DealsConsiderOnlineRetrievalDeals      func(context.Context) (bool, error)                               `perm:"read"`
		DealsSetConsiderOnlineRetrievalDeals   func(context.Context, bool) error                                 `perm:"admin"`
		DealsConsiderOfflineStorageDeals       func(context.Context) (bool, error)                               `perm:"read"`
		DealsSetConsiderOfflineStorageDeals    func(context.Context, bool) error                                 `perm:"admin"`
		DealsConsiderOfflineRetrievalDeals     func(context.Context) (bool, error)                               `perm:"read"`
		DealsSetConsiderOfflineRetrievalDeals  func(context.Context, bool) error                                 `perm:"admin"`
		DealsConsiderVerifiedStorageDeals      func(context.Context) (bool, error)                               `perm:"read"`
		DealsSetConsiderVerifiedStorageDeals   func(context.Context, bool) error                                 `perm:"admin"`
		DealsConsiderUnverifiedStorageDeals    func(context.Context) (bool, error)                               `perm:"read"`
		DealsSetConsiderUnverifiedStorageDeals func(context.Context, bool) error                                 `perm:"admin"`
		DealsPieceCidBlocklist                 func(context.Context) ([]cid.Cid, error)                          `perm:"read"`
		DealsSetPieceCidBlocklist              func(context.Context, []cid.Cid) error                            `perm:"admin"`

		StorageAddLocal func(ctx context.Context, path string) error `perm:"admin"`

		PiecesListPieces   func(ctx context.Context) ([]cid.Cid, error)                               `perm:"read"`
		PiecesListCidInfos func(ctx context.Context) ([]cid.Cid, error)                               `perm:"read"`
		PiecesGetPieceInfo func(ctx context.Context, pieceCid cid.Cid) (*piecestore.PieceInfo, error) `perm:"read"`
		PiecesGetCIDInfo   func(ctx context.Context, payloadCid cid.Cid) (*piecestore.CIDInfo, error) `perm:"read"`

		CreateBackup func(ctx context.Context, fpath string) error `perm:"admin"`

		CheckProvable func(ctx context.Context, pp abi.RegisteredPoStProof, sectors []storage.SectorRef, expensive bool) (map[abi.SectorNumber]string, error) `perm:"admin"`
	}
}

func (c *StorageMinerStruct) NetParamsConfig(ctx context.Context) (*config.NetParamsConfig, error) {
	return c.Internal.NetParamsConfig(ctx)
}

func (c *StorageMinerStruct) ActorAddress(ctx context.Context) (address.Address, error) {
	return c.Internal.ActorAddress(ctx)
}

func (c *StorageMinerStruct) ActorSectorSize(ctx context.Context, addr address.Address) (abi.SectorSize, error) {
	return c.Internal.ActorSectorSize(ctx, addr)
}

func (c *StorageMinerStruct) ActorAddressConfig(ctx context.Context) (AddressConfig, error) {
	return c.Internal.ActorAddressConfig(ctx)
}

func (c *StorageMinerStruct) PledgeSector(ctx context.Context) error {
	return c.Internal.PledgeSector(ctx)
}

// Get the status of a given sector by ID
func (c *StorageMinerStruct) SectorsStatus(ctx context.Context, sid abi.SectorNumber, showOnChainInfo bool) (SectorInfo, error) {
	return c.Internal.SectorsStatus(ctx, sid, showOnChainInfo)
}

// List all staged sectors
func (c *StorageMinerStruct) SectorsList(ctx context.Context) ([]abi.SectorNumber, error) {
	return c.Internal.SectorsList(ctx)
}

func (c *StorageMinerStruct) SectorsListInStates(ctx context.Context, states []SectorState) ([]abi.SectorNumber, error) {
	return c.Internal.SectorsListInStates(ctx, states)
}

func (c *StorageMinerStruct) SectorsSummary(ctx context.Context) (map[SectorState]int, error) {
	return c.Internal.SectorsSummary(ctx)
}

func (c *StorageMinerStruct) SectorsRefs(ctx context.Context) (map[string][]SealedRef, error) {
	return c.Internal.SectorsRefs(ctx)
}

func (c *StorageMinerStruct) SectorStartSealing(ctx context.Context, number abi.SectorNumber) error {
	return c.Internal.SectorStartSealing(ctx, number)
}

func (c *StorageMinerStruct) SectorSetSealDelay(ctx context.Context, delay time.Duration) error {
	return c.Internal.SectorSetSealDelay(ctx, delay)
}

func (c *StorageMinerStruct) SectorGetSealDelay(ctx context.Context) (time.Duration, error) {
	return c.Internal.SectorGetSealDelay(ctx)
}

func (c *StorageMinerStruct) SectorSetExpectedSealDuration(ctx context.Context, delay time.Duration) error {
	return c.Internal.SectorSetExpectedSealDuration(ctx, delay)
}

func (c *StorageMinerStruct) SectorGetExpectedSealDuration(ctx context.Context) (time.Duration, error) {
	return c.Internal.SectorGetExpectedSealDuration(ctx)
}

func (c *StorageMinerStruct) SectorsUpdate(ctx context.Context, id abi.SectorNumber, state SectorState) error {
	return c.Internal.SectorsUpdate(ctx, id, state)
}

func (c *StorageMinerStruct) SectorRemove(ctx context.Context, number abi.SectorNumber) error {
	return c.Internal.SectorRemove(ctx, number)
}

func (c *StorageMinerStruct) SectorTerminate(ctx context.Context, number abi.SectorNumber) error {
	return c.Internal.SectorTerminate(ctx, number)
}

func (c *StorageMinerStruct) SectorTerminateFlush(ctx context.Context) (*cid.Cid, error) {
	return c.Internal.SectorTerminateFlush(ctx)
}

func (c *StorageMinerStruct) SectorTerminatePending(ctx context.Context) ([]abi.SectorID, error) {
	return c.Internal.SectorTerminatePending(ctx)
}

func (c *StorageMinerStruct) SectorMarkForUpgrade(ctx context.Context, number abi.SectorNumber) error {
	return c.Internal.SectorMarkForUpgrade(ctx, number)
}

func (c *StorageMinerStruct) WorkerConnect(ctx context.Context, url string) error {
	return c.Internal.WorkerConnect(ctx, url)
}

func (c *StorageMinerStruct) WorkerStats(ctx context.Context) (map[uuid.UUID]storiface.WorkerStats, error) {
	return c.Internal.WorkerStats(ctx)
}

func (c *StorageMinerStruct) WorkerJobs(ctx context.Context) (map[uuid.UUID][]storiface.WorkerJob, error) {
	return c.Internal.WorkerJobs(ctx)
}

func (c *StorageMinerStruct) ReturnAddPiece(ctx context.Context, callID storiface.CallID, pi abi.PieceInfo, err *storiface.CallError) error {
	return c.Internal.ReturnAddPiece(ctx, callID, pi, err)
}

func (c *StorageMinerStruct) ReturnSealPreCommit1(ctx context.Context, callID storiface.CallID, p1o storage.PreCommit1Out, err *storiface.CallError) error {
	return c.Internal.ReturnSealPreCommit1(ctx, callID, p1o, err)
}

func (c *StorageMinerStruct) ReturnSealPreCommit2(ctx context.Context, callID storiface.CallID, sealed storage.SectorCids, err *storiface.CallError) error {
	return c.Internal.ReturnSealPreCommit2(ctx, callID, sealed, err)
}

func (c *StorageMinerStruct) ReturnSealCommit1(ctx context.Context, callID storiface.CallID, out storage.Commit1Out, err *storiface.CallError) error {
	return c.Internal.ReturnSealCommit1(ctx, callID, out, err)
}

func (c *StorageMinerStruct) ReturnSealCommit2(ctx context.Context, callID storiface.CallID, proof storage.Proof, err *storiface.CallError) error {
	return c.Internal.ReturnSealCommit2(ctx, callID, proof, err)
}

func (c *StorageMinerStruct) ReturnFinalizeSector(ctx context.Context, callID storiface.CallID, err *storiface.CallError) error {
	return c.Internal.ReturnFinalizeSector(ctx, callID, err)
}

func (c *StorageMinerStruct) ReturnReleaseUnsealed(ctx context.Context, callID storiface.CallID, err *storiface.CallError) error {
	return c.Internal.ReturnReleaseUnsealed(ctx, callID, err)
}

func (c *StorageMinerStruct) ReturnMoveStorage(ctx context.Context, callID storiface.CallID, err *storiface.CallError) error {
	return c.Internal.ReturnMoveStorage(ctx, callID, err)
}

func (c *StorageMinerStruct) ReturnUnsealPiece(ctx context.Context, callID storiface.CallID, err *storiface.CallError) error {
	return c.Internal.ReturnUnsealPiece(ctx, callID, err)
}

func (c *StorageMinerStruct) ReturnReadPiece(ctx context.Context, callID storiface.CallID, ok bool, err *storiface.CallError) error {
	return c.Internal.ReturnReadPiece(ctx, callID, ok, err)
}

func (c *StorageMinerStruct) ReturnFetch(ctx context.Context, callID storiface.CallID, err *storiface.CallError) error {
	return c.Internal.ReturnFetch(ctx, callID, err)
}

func (c *StorageMinerStruct) SealingSchedDiag(ctx context.Context, doSched bool) (interface{}, error) {
	return c.Internal.SealingSchedDiag(ctx, doSched)
}

func (c *StorageMinerStruct) SealingAbort(ctx context.Context, call storiface.CallID) error {
	return c.Internal.SealingAbort(ctx, call)
}

func (c *StorageMinerStruct) StorageAttach(ctx context.Context, si stores.StorageInfo, st fsutil.FsStat) error {
	return c.Internal.StorageAttach(ctx, si, st)
}

func (c *StorageMinerStruct) StorageDeclareSector(ctx context.Context, storageId stores.ID, s abi.SectorID, ft storiface.SectorFileType, primary bool) error {
	return c.Internal.StorageDeclareSector(ctx, storageId, s, ft, primary)
}

func (c *StorageMinerStruct) StorageDropSector(ctx context.Context, storageId stores.ID, s abi.SectorID, ft storiface.SectorFileType) error {
	return c.Internal.StorageDropSector(ctx, storageId, s, ft)
}

func (c *StorageMinerStruct) StorageFindSector(ctx context.Context, si abi.SectorID, types storiface.SectorFileType, ssize abi.SectorSize, allowFetch bool) ([]stores.SectorStorageInfo, error) {
	return c.Internal.StorageFindSector(ctx, si, types, ssize, allowFetch)
}

func (c *StorageMinerStruct) StorageList(ctx context.Context) (map[stores.ID][]stores.Decl, error) {
	return c.Internal.StorageList(ctx)
}

func (c *StorageMinerStruct) StorageLocal(ctx context.Context) (map[stores.ID]string, error) {
	return c.Internal.StorageLocal(ctx)
}

func (c *StorageMinerStruct) StorageStat(ctx context.Context, id stores.ID) (fsutil.FsStat, error) {
	return c.Internal.StorageStat(ctx, id)
}

func (c *StorageMinerStruct) StorageInfo(ctx context.Context, id stores.ID) (stores.StorageInfo, error) {
	return c.Internal.StorageInfo(ctx, id)
}

func (c *StorageMinerStruct) StorageBestAlloc(ctx context.Context, allocate storiface.SectorFileType, ssize abi.SectorSize, pt storiface.PathType) ([]stores.StorageInfo, error) {
	return c.Internal.StorageBestAlloc(ctx, allocate, ssize, pt)
}

func (c *StorageMinerStruct) StorageReportHealth(ctx context.Context, id stores.ID, report stores.HealthReport) error {
	return c.Internal.StorageReportHealth(ctx, id, report)
}

func (c *StorageMinerStruct) StorageLock(ctx context.Context, sector abi.SectorID, read storiface.SectorFileType, write storiface.SectorFileType) error {
	return c.Internal.StorageLock(ctx, sector, read, write)
}

func (c *StorageMinerStruct) StorageTryLock(ctx context.Context, sector abi.SectorID, read storiface.SectorFileType, write storiface.SectorFileType) (bool, error) {
	return c.Internal.StorageTryLock(ctx, sector, read, write)
}

func (c *StorageMinerStruct) MarketImportDealData(ctx context.Context, propcid cid.Cid, path string) error {
	return c.Internal.MarketImportDealData(ctx, propcid, path)
}

func (c *StorageMinerStruct) MarketListDeals(ctx context.Context) ([]MarketDeal, error) {
	return c.Internal.MarketListDeals(ctx)
}

func (c *StorageMinerStruct) MarketListRetrievalDeals(ctx context.Context) ([]retrievalmarket.ProviderDealState, error) {
	return c.Internal.MarketListRetrievalDeals(ctx)
}

func (c *StorageMinerStruct) MarketGetDealUpdates(ctx context.Context) (<-chan storagemarket.MinerDeal, error) {
	return c.Internal.MarketGetDealUpdates(ctx)
}

func (c *StorageMinerStruct) MarketListIncompleteDeals(ctx context.Context) ([]storagemarket.MinerDeal, error) {
	return c.Internal.MarketListIncompleteDeals(ctx)
}

func (c *StorageMinerStruct) MarketSetAsk(ctx context.Context, price types.BigInt, verifiedPrice types.BigInt, duration abi.ChainEpoch, minPieceSize abi.PaddedPieceSize, maxPieceSize abi.PaddedPieceSize) error {
	return c.Internal.MarketSetAsk(ctx, price, verifiedPrice, duration, minPieceSize, maxPieceSize)
}

func (c *StorageMinerStruct) MarketGetAsk(ctx context.Context) (*storagemarket.SignedStorageAsk, error) {
	return c.Internal.MarketGetAsk(ctx)
}

func (c *StorageMinerStruct) MarketSetRetrievalAsk(ctx context.Context, rask *retrievalmarket.Ask) error {
	return c.Internal.MarketSetRetrievalAsk(ctx, rask)
}

func (c *StorageMinerStruct) MarketGetRetrievalAsk(ctx context.Context) (*retrievalmarket.Ask, error) {
	return c.Internal.MarketGetRetrievalAsk(ctx)
}

func (c *StorageMinerStruct) MarketListDataTransfers(ctx context.Context) ([]DataTransferChannel, error) {
	return c.Internal.MarketListDataTransfers(ctx)
}

func (c *StorageMinerStruct) MarketDataTransferUpdates(ctx context.Context) (<-chan DataTransferChannel, error) {
	return c.Internal.MarketDataTransferUpdates(ctx)
}

func (c *StorageMinerStruct) MarketRestartDataTransfer(ctx context.Context, transferID datatransfer.TransferID, otherPeer peer.ID, isInitiator bool) error {
	return c.Internal.MarketRestartDataTransfer(ctx, transferID, otherPeer, isInitiator)
}

func (c *StorageMinerStruct) MarketCancelDataTransfer(ctx context.Context, transferID datatransfer.TransferID, otherPeer peer.ID, isInitiator bool) error {
	return c.Internal.MarketCancelDataTransfer(ctx, transferID, otherPeer, isInitiator)
}

func (c *StorageMinerStruct) DealsImportData(ctx context.Context, dealPropCid cid.Cid, file string) error {
	return c.Internal.DealsImportData(ctx, dealPropCid, file)
}

func (c *StorageMinerStruct) DealsList(ctx context.Context) ([]MarketDeal, error) {
	return c.Internal.DealsList(ctx)
}

func (c *StorageMinerStruct) DealsConsiderOnlineStorageDeals(ctx context.Context) (bool, error) {
	return c.Internal.DealsConsiderOnlineStorageDeals(ctx)
}

func (c *StorageMinerStruct) DealsSetConsiderOnlineStorageDeals(ctx context.Context, b bool) error {
	return c.Internal.DealsSetConsiderOnlineStorageDeals(ctx, b)
}

func (c *StorageMinerStruct) DealsConsiderOnlineRetrievalDeals(ctx context.Context) (bool, error) {
	return c.Internal.DealsConsiderOnlineRetrievalDeals(ctx)
}

func (c *StorageMinerStruct) DealsSetConsiderOnlineRetrievalDeals(ctx context.Context, b bool) error {
	return c.Internal.DealsSetConsiderOnlineRetrievalDeals(ctx, b)
}

func (c *StorageMinerStruct) DealsPieceCidBlocklist(ctx context.Context) ([]cid.Cid, error) {
	return c.Internal.DealsPieceCidBlocklist(ctx)
}

func (c *StorageMinerStruct) DealsSetPieceCidBlocklist(ctx context.Context, cids []cid.Cid) error {
	return c.Internal.DealsSetPieceCidBlocklist(ctx, cids)
}

func (c *StorageMinerStruct) DealsConsiderOfflineStorageDeals(ctx context.Context) (bool, error) {
	return c.Internal.DealsConsiderOfflineStorageDeals(ctx)
}

func (c *StorageMinerStruct) DealsSetConsiderOfflineStorageDeals(ctx context.Context, b bool) error {
	return c.Internal.DealsSetConsiderOfflineStorageDeals(ctx, b)
}

func (c *StorageMinerStruct) DealsConsiderOfflineRetrievalDeals(ctx context.Context) (bool, error) {
	return c.Internal.DealsConsiderOfflineRetrievalDeals(ctx)
}

func (c *StorageMinerStruct) DealsSetConsiderOfflineRetrievalDeals(ctx context.Context, b bool) error {
	return c.Internal.DealsSetConsiderOfflineRetrievalDeals(ctx, b)
}

func (c *StorageMinerStruct) DealsConsiderVerifiedStorageDeals(ctx context.Context) (bool, error) {
	return c.Internal.DealsConsiderVerifiedStorageDeals(ctx)
}

func (c *StorageMinerStruct) DealsSetConsiderVerifiedStorageDeals(ctx context.Context, b bool) error {
	return c.Internal.DealsSetConsiderVerifiedStorageDeals(ctx, b)
}

func (c *StorageMinerStruct) DealsConsiderUnverifiedStorageDeals(ctx context.Context) (bool, error) {
	return c.Internal.DealsConsiderUnverifiedStorageDeals(ctx)
}

func (c *StorageMinerStruct) DealsSetConsiderUnverifiedStorageDeals(ctx context.Context, b bool) error {
	return c.Internal.DealsSetConsiderUnverifiedStorageDeals(ctx, b)
}

func (c *StorageMinerStruct) StorageAddLocal(ctx context.Context, path string) error {
	return c.Internal.StorageAddLocal(ctx, path)
}

func (c *StorageMinerStruct) PiecesListPieces(ctx context.Context) ([]cid.Cid, error) {
	return c.Internal.PiecesListPieces(ctx)
}

func (c *StorageMinerStruct) PiecesListCidInfos(ctx context.Context) ([]cid.Cid, error) {
	return c.Internal.PiecesListCidInfos(ctx)
}

func (c *StorageMinerStruct) PiecesGetPieceInfo(ctx context.Context, pieceCid cid.Cid) (*piecestore.PieceInfo, error) {
	return c.Internal.PiecesGetPieceInfo(ctx, pieceCid)
}

func (c *StorageMinerStruct) PiecesGetCIDInfo(ctx context.Context, payloadCid cid.Cid) (*piecestore.CIDInfo, error) {
	return c.Internal.PiecesGetCIDInfo(ctx, payloadCid)
}

func (c *StorageMinerStruct) CreateBackup(ctx context.Context, fpath string) error {
	return c.Internal.CreateBackup(ctx, fpath)
}

func (c *StorageMinerStruct) CheckProvable(ctx context.Context, pp abi.RegisteredPoStProof, sectors []storage.SectorRef, expensive bool) (map[abi.SectorNumber]string, error) {
	return c.Internal.CheckProvable(ctx, pp, sectors, expensive)
}
