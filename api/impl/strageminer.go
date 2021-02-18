package impl

import (
	"context"
	"encoding/json"
	datatransfer "github.com/filecoin-project/go-data-transfer"
	"github.com/filecoin-project/go-fil-markets/piecestore"
	"github.com/filecoin-project/go-fil-markets/retrievalmarket"
	"github.com/filecoin-project/go-fil-markets/storagemarket"
	proof2 "github.com/filecoin-project/specs-actors/v2/actors/runtime/proof"
	"github.com/filecoin-project/venus-sealer/config"
	chain2 "github.com/filecoin-project/venus/app/submodule/chain"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/peer"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc/auth"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"

	sectorstorage "github.com/filecoin-project/venus-sealer/extern/sector-storage"
	"github.com/filecoin-project/venus-sealer/extern/sector-storage/fsutil"
	"github.com/filecoin-project/venus-sealer/extern/sector-storage/stores"
	"github.com/filecoin-project/venus-sealer/extern/sector-storage/storiface"
	sealing "github.com/filecoin-project/venus-sealer/extern/storage-sealing"

	sto "github.com/filecoin-project/specs-storage/storage"
	"github.com/filecoin-project/venus-sealer/api"
	"github.com/filecoin-project/venus-sealer/dtypes"
	"github.com/filecoin-project/venus-sealer/storage"
	"github.com/filecoin-project/venus-sealer/storage/sectorblocks"
	"github.com/filecoin-project/venus/pkg/types"
)

type StorageMinerAPI struct {
	CommonAPI
	prover       storage.WinningPoStProver
	SectorBlocks *sectorblocks.SectorBlocks
	Miner        *storage.Miner
	Full         api.FullNode
	StorageMgr   *sectorstorage.Manager `optional:"true"`
	IStorageMgr  sectorstorage.SectorManager
	*stores.Index
	storiface.WorkerReturn

	AddrSel *storage.AddressSelector

	DS dtypes.MetadataDS

	NetParams            *config.NetParamsConfig
	SetSealingConfigFunc dtypes.SetSealingConfigFunc
	GetSealingConfigFunc dtypes.GetSealingConfigFunc
}

func (sm *StorageMinerAPI) ServeRemote(w http.ResponseWriter, r *http.Request) {
	if !auth.HasPerm(r.Context(), nil, api.PermAdmin) {
		w.WriteHeader(401)
		_ = json.NewEncoder(w).Encode(struct{ Error string }{"unauthorized: missing write permission"})
		return
	}

	sm.StorageMgr.ServeHTTP(w, r)
}

func (sm *StorageMinerAPI) WorkerStats(context.Context) (map[uuid.UUID]storiface.WorkerStats, error) {
	return sm.StorageMgr.WorkerStats(), nil
}

func (sm *StorageMinerAPI) WorkerJobs(ctx context.Context) (map[uuid.UUID][]storiface.WorkerJob, error) {
	return sm.StorageMgr.WorkerJobs(), nil
}

func (sm *StorageMinerAPI) ActorAddress(context.Context) (address.Address, error) {
	return sm.Miner.Address(), nil
}

func (sm *StorageMinerAPI) ActorSectorSize(ctx context.Context, addr address.Address) (abi.SectorSize, error) {
	mi, err := sm.Full.StateMinerInfo(ctx, addr, types.EmptyTSK)
	if err != nil {
		return 0, err
	}
	return mi.SectorSize, nil
}

func (sm *StorageMinerAPI) PledgeSector(ctx context.Context) error {
	return sm.Miner.PledgeSector()
}

func (sm *StorageMinerAPI) SectorsStatus(ctx context.Context, sid abi.SectorNumber, showOnChainInfo bool) (api.SectorInfo, error) {
	info, err := sm.Miner.GetSectorInfo(sid)
	if err != nil {
		return api.SectorInfo{}, err
	}

	deals := make([]abi.DealID, len(info.Pieces))
	for i, piece := range info.Pieces {
		if piece.DealInfo == nil {
			continue
		}
		deals[i] = piece.DealInfo.DealID
	}

	log := make([]api.SectorLog, len(info.Log))
	for i, l := range info.Log {
		log[i] = api.SectorLog{
			Kind:      l.Kind,
			Timestamp: l.Timestamp,
			Trace:     l.Trace,
			Message:   l.Message,
		}
	}

	sInfo := api.SectorInfo{
		SectorID: sid,
		State:    api.SectorState(info.State),
		CommD:    info.CommD,
		CommR:    info.CommR,
		Proof:    info.Proof,
		Deals:    deals,
		Ticket: api.SealTicket{
			Value: info.TicketValue,
			Epoch: info.TicketEpoch,
		},
		Seed: api.SealSeed{
			Value: info.SeedValue,
			Epoch: info.SeedEpoch,
		},
		PreCommitMsg: info.PreCommitMessage,
		CommitMsg:    info.CommitMessage,
		Retries:      info.InvalidProofs,
		ToUpgrade:    sm.Miner.IsMarkedForUpgrade(sid),

		LastErr: info.LastErr,
		Log:     log,
		// on chain info
		SealProof:          0,
		Activation:         0,
		Expiration:         0,
		DealWeight:         big.Zero(),
		VerifiedDealWeight: big.Zero(),
		InitialPledge:      big.Zero(),
		OnTime:             0,
		Early:              0,
	}

	if !showOnChainInfo {
		return sInfo, nil
	}

	onChainInfo, err := sm.Full.StateSectorGetInfo(ctx, sm.Miner.Address(), sid, types.EmptyTSK)
	if err != nil {
		return sInfo, err
	}
	if onChainInfo == nil {
		return sInfo, nil
	}
	sInfo.SealProof = onChainInfo.SealProof
	sInfo.Activation = onChainInfo.Activation
	sInfo.Expiration = onChainInfo.Expiration
	sInfo.DealWeight = onChainInfo.DealWeight
	sInfo.VerifiedDealWeight = onChainInfo.VerifiedDealWeight
	sInfo.InitialPledge = onChainInfo.InitialPledge

	ex, err := sm.Full.StateSectorExpiration(ctx, sm.Miner.Address(), sid, types.EmptyTSK)
	if err != nil {
		return sInfo, nil
	}
	sInfo.OnTime = ex.OnTime
	sInfo.Early = ex.Early

	return sInfo, nil
}

// List all staged sectors
func (sm *StorageMinerAPI) SectorsList(context.Context) ([]abi.SectorNumber, error) {
	sectors, err := sm.Miner.ListSectors()
	if err != nil {
		return nil, err
	}

	out := make([]abi.SectorNumber, len(sectors))
	for i, sector := range sectors {
		out[i] = sector.SectorNumber
	}
	return out, nil
}

func (sm *StorageMinerAPI) SectorsListInStates(ctx context.Context, states []api.SectorState) ([]abi.SectorNumber, error) {
	filterStates := make(map[sealing.SectorState]struct{})
	for _, state := range states {
		st := sealing.SectorState(state)
		if _, ok := sealing.ExistSectorStateList[st]; !ok {
			continue
		}
		filterStates[st] = struct{}{}
	}

	var sns []abi.SectorNumber
	if len(filterStates) == 0 {
		return sns, nil
	}

	sectors, err := sm.Miner.ListSectors()
	if err != nil {
		return nil, err
	}

	for i := range sectors {
		if _, ok := filterStates[sectors[i].State]; ok {
			sns = append(sns, sectors[i].SectorNumber)
		}
	}
	return sns, nil
}

func (sm *StorageMinerAPI) SectorsSummary(ctx context.Context) (map[api.SectorState]int, error) {
	sectors, err := sm.Miner.ListSectors()
	if err != nil {
		return nil, err
	}

	out := make(map[api.SectorState]int)
	for i := range sectors {
		state := api.SectorState(sectors[i].State)
		out[state]++
	}

	return out, nil
}

func (sm *StorageMinerAPI) StorageLocal(ctx context.Context) (map[stores.ID]string, error) {
	return sm.StorageMgr.StorageLocal(ctx)
}

func (sm *StorageMinerAPI) SectorsRefs(context.Context) (map[string][]api.SealedRef, error) {
	// json can't handle cids as map keys
	out := map[string][]api.SealedRef{}

	refs, err := sm.SectorBlocks.List()
	if err != nil {
		return nil, err
	}

	for k, v := range refs {
		out[strconv.FormatUint(k, 10)] = v
	}

	return out, nil
}

func (sm *StorageMinerAPI) StorageStat(ctx context.Context, id stores.ID) (fsutil.FsStat, error) {
	return sm.StorageMgr.FsStat(ctx, id)
}

func (sm *StorageMinerAPI) SectorStartSealing(ctx context.Context, number abi.SectorNumber) error {
	return sm.Miner.StartPackingSector(number)
}

func (sm *StorageMinerAPI) SectorSetSealDelay(ctx context.Context, delay time.Duration) error {
	cfg, err := sm.GetSealingConfigFunc()
	if err != nil {
		return xerrors.Errorf("get config: %w", err)
	}

	cfg.WaitDealsDelay = delay

	return sm.SetSealingConfigFunc(cfg)
}

func (sm *StorageMinerAPI) SectorGetSealDelay(ctx context.Context) (time.Duration, error) {
	cfg, err := sm.GetSealingConfigFunc()
	if err != nil {
		return 0, err
	}
	return cfg.WaitDealsDelay, nil
}

func (sm *StorageMinerAPI) SectorSetExpectedSealDuration(ctx context.Context, delay time.Duration) error {
	//return sm.SetExpectedSealDurationFunc(delay)
	panic("not impl")
}

func (sm *StorageMinerAPI) SectorGetExpectedSealDuration(ctx context.Context) (time.Duration, error) {
	//return sm.GetExpectedSealDurationFunc()
	panic("not impl")
}

func (sm *StorageMinerAPI) SectorsUpdate(ctx context.Context, id abi.SectorNumber, state api.SectorState) error {
	return sm.Miner.ForceSectorState(ctx, id, sealing.SectorState(state))
}

func (sm *StorageMinerAPI) SectorRemove(ctx context.Context, id abi.SectorNumber) error {
	return sm.Miner.RemoveSector(ctx, id)
}

func (sm *StorageMinerAPI) SectorTerminate(ctx context.Context, id abi.SectorNumber) error {
	return sm.Miner.TerminateSector(ctx, id)
}

func (sm *StorageMinerAPI) SectorTerminateFlush(ctx context.Context) (*cid.Cid, error) {
	return sm.Miner.TerminateFlush(ctx)
}

func (sm *StorageMinerAPI) SectorTerminatePending(ctx context.Context) ([]abi.SectorID, error) {
	return sm.Miner.TerminatePending(ctx)
}

func (sm *StorageMinerAPI) SectorMarkForUpgrade(ctx context.Context, id abi.SectorNumber) error {
	return sm.Miner.MarkForUpgrade(id)
}

func (sm *StorageMinerAPI) WorkerConnect(ctx context.Context, url string) error {
	/*w, err := connectRemoteWorker(ctx, sm, url)
	if err != nil {
		return xerrors.Errorf("connecting remote storage failed: %w", err)
	}

	log.Infof("Connected to a remote worker at %s", url)

	return sm.StorageMgr.AddWorker(ctx, w)*/
	panic("not impl")
}

func (sm *StorageMinerAPI) SealingSchedDiag(ctx context.Context, doSched bool) (interface{}, error) {
	return sm.StorageMgr.SchedDiag(ctx, doSched)
}

func (sm *StorageMinerAPI) SealingAbort(ctx context.Context, call storiface.CallID) error {
	return sm.StorageMgr.Abort(ctx, call)
}

func (sm *StorageMinerAPI) MarketImportDealData(ctx context.Context, propCid cid.Cid, path string) error {
	/*	fi, err := os.Open(path)
		if err != nil {
			return xerrors.Errorf("failed to open file: %w", err)
		}
		defer fi.Close() //nolint:errcheck

		return sm.StorageProvider.ImportDataForDeal(ctx, propCid, fi)*/
	panic("not impl")
}

func (sm *StorageMinerAPI) listDeals(ctx context.Context) ([]chain2.MarketDeal, error) {
	ts, err := sm.Full.ChainHead(ctx)
	if err != nil {
		return nil, err
	}
	tsk := ts.Key()
	allDeals, err := sm.Full.StateMarketDeals(ctx, tsk)
	if err != nil {
		return nil, err
	}

	var out []chain2.MarketDeal

	for _, deal := range allDeals {
		if deal.Proposal.Provider == sm.Miner.Address() {
			out = append(out, deal)
		}
	}

	return out, nil
}

func (sm *StorageMinerAPI) MarketListDeals(ctx context.Context) ([]chain2.MarketDeal, error) {
	return sm.listDeals(ctx)
}

func (sm *StorageMinerAPI) MarketListRetrievalDeals(ctx context.Context) ([]retrievalmarket.ProviderDealState, error) {
	/*var out []retrievalmarket.ProviderDealState
	deals := sm.RetrievalProvider.ListDeals()

	for _, deal := range deals {
		out = append(out, deal)
	}

	return out, nil*/
	panic("not impl")
}

func (sm *StorageMinerAPI) MarketGetDealUpdates(ctx context.Context) (<-chan storagemarket.MinerDeal, error) {
	/*results := make(chan storagemarket.MinerDeal)
	unsub := sm.StorageProvider.SubscribeToEvents(func(evt storagemarket.ProviderEvent, deal storagemarket.MinerDeal) {
		select {
		case results <- deal:
		case <-ctx.Done():
		}
	})
	go func() {
		<-ctx.Done()
		unsub()
		close(results)
	}()
	return results, nil*/
	panic("not impl")
}

func (sm *StorageMinerAPI) MarketListIncompleteDeals(ctx context.Context) ([]storagemarket.MinerDeal, error) {
	//return sm.StorageProvider.ListLocalDeals()
	panic("not impl")
}

func (sm *StorageMinerAPI) MarketSetAsk(ctx context.Context, price types.BigInt, verifiedPrice types.BigInt, duration abi.ChainEpoch, minPieceSize abi.PaddedPieceSize, maxPieceSize abi.PaddedPieceSize) error {
	/*options := []storagemarket.StorageAskOption{
		storagemarket.MinPieceSize(minPieceSize),
		storagemarket.MaxPieceSize(maxPieceSize),
	}

	return sm.StorageProvider.SetAsk(price, verifiedPrice, duration, options...)*/
	panic("not impl")
}

func (sm *StorageMinerAPI) MarketGetAsk(ctx context.Context) (*storagemarket.SignedStorageAsk, error) {
	//return sm.StorageProvider.GetAsk(), nil
	panic("not impl")
}

func (sm *StorageMinerAPI) MarketSetRetrievalAsk(ctx context.Context, rask *retrievalmarket.Ask) error {
	//sm.RetrievalProvider.SetAsk(rask)
	panic("not impl")
	return nil
}

func (sm *StorageMinerAPI) MarketGetRetrievalAsk(ctx context.Context) (*retrievalmarket.Ask, error) {
	//return sm.RetrievalProvider.GetAsk(), nil
	panic("not impl")
}

func (sm *StorageMinerAPI) MarketListDataTransfers(ctx context.Context) ([]api.DataTransferChannel, error) {
	/*inProgressChannels, err := sm.DataTransfer.InProgressChannels(ctx)
	if err != nil {
		return nil, err
	}

	apiChannels := make([]api.DataTransferChannel, 0, len(inProgressChannels))
	for _, channelState := range inProgressChannels {
		apiChannels = append(apiChannels, api.NewDataTransferChannel(sm.Host.ID(), channelState))
	}

	return apiChannels, nil*/
	panic("not impl")
}

func (sm *StorageMinerAPI) MarketRestartDataTransfer(ctx context.Context, transferID datatransfer.TransferID, otherPeer peer.ID, isInitiator bool) error {
	/*selfPeer := sm.Host.ID()
	if isInitiator {
		return sm.DataTransfer.RestartDataTransferChannel(ctx, datatransfer.ChannelID{Initiator: selfPeer, Responder: otherPeer, ID: transferID})
	}
	return sm.DataTransfer.RestartDataTransferChannel(ctx, datatransfer.ChannelID{Initiator: otherPeer, Responder: selfPeer, ID: transferID})*/
	panic("not impl")
}

func (sm *StorageMinerAPI) MarketCancelDataTransfer(ctx context.Context, transferID datatransfer.TransferID, otherPeer peer.ID, isInitiator bool) error {
	/*selfPeer := sm.Host.ID()
	if isInitiator {
		return sm.DataTransfer.CloseDataTransferChannel(ctx, datatransfer.ChannelID{Initiator: selfPeer, Responder: otherPeer, ID: transferID})
	}
	return sm.DataTransfer.CloseDataTransferChannel(ctx, datatransfer.ChannelID{Initiator: otherPeer, Responder: selfPeer, ID: transferID})*/
	panic("not impl")
}

func (sm *StorageMinerAPI) MarketDataTransferUpdates(ctx context.Context) (<-chan api.DataTransferChannel, error) {
	/*channels := make(chan api.DataTransferChannel)

	unsub := sm.DataTransfer.SubscribeToEvents(func(evt datatransfer.Event, channelState datatransfer.ChannelState) {
		channel := api.NewDataTransferChannel(sm.Host.ID(), channelState)
		select {
		case <-ctx.Done():
		case channels <- channel:
		}
	})

	go func() {
		defer unsub()
		<-ctx.Done()
	}()

	return channels, nil*/
	panic("not impl")
}

func (sm *StorageMinerAPI) DealsList(ctx context.Context) ([]chain2.MarketDeal, error) {
	return sm.listDeals(ctx)
}

func (sm *StorageMinerAPI) RetrievalDealsList(ctx context.Context) (map[retrievalmarket.ProviderDealIdentifier]retrievalmarket.ProviderDealState, error) {
	//return sm.RetrievalProvider.ListDeals(), nil
	panic("not impl")
}

func (sm *StorageMinerAPI) DealsConsiderOnlineStorageDeals(ctx context.Context) (bool, error) {
	//return sm.ConsiderOnlineStorageDealsConfigFunc()
	panic("not impl")
}

func (sm *StorageMinerAPI) DealsSetConsiderOnlineStorageDeals(ctx context.Context, b bool) error {
	//return sm.SetConsiderOnlineStorageDealsConfigFunc(b)
	panic("not impl")
}

func (sm *StorageMinerAPI) DealsConsiderOnlineRetrievalDeals(ctx context.Context) (bool, error) {
	//return sm.ConsiderOnlineRetrievalDealsConfigFunc()
	panic("not impl")
}

func (sm *StorageMinerAPI) DealsSetConsiderOnlineRetrievalDeals(ctx context.Context, b bool) error {
	//return sm.SetConsiderOnlineRetrievalDealsConfigFunc(b)
	panic("not impl")
}

func (sm *StorageMinerAPI) DealsConsiderOfflineStorageDeals(ctx context.Context) (bool, error) {
	//return sm.ConsiderOfflineStorageDealsConfigFunc()
	panic("not impl")
}

func (sm *StorageMinerAPI) DealsSetConsiderOfflineStorageDeals(ctx context.Context, b bool) error {
	//return sm.SetConsiderOfflineStorageDealsConfigFunc(b)
	panic("not impl")
}

func (sm *StorageMinerAPI) DealsConsiderOfflineRetrievalDeals(ctx context.Context) (bool, error) {
	//return sm.ConsiderOfflineRetrievalDealsConfigFunc()
	panic("not impl")
}

func (sm *StorageMinerAPI) DealsSetConsiderOfflineRetrievalDeals(ctx context.Context, b bool) error {
	//return sm.SetConsiderOfflineRetrievalDealsConfigFunc(b)
	panic("not impl")
}

func (sm *StorageMinerAPI) DealsConsiderVerifiedStorageDeals(ctx context.Context) (bool, error) {
	//return sm.ConsiderVerifiedStorageDealsConfigFunc()
	panic("not impl")
}

func (sm *StorageMinerAPI) DealsSetConsiderVerifiedStorageDeals(ctx context.Context, b bool) error {
	//return sm.SetConsiderVerifiedStorageDealsConfigFunc(b)
	panic("not impl")
}

func (sm *StorageMinerAPI) DealsConsiderUnverifiedStorageDeals(ctx context.Context) (bool, error) {
	//return sm.ConsiderUnverifiedStorageDealsConfigFunc()
	panic("not impl")
}

func (sm *StorageMinerAPI) DealsSetConsiderUnverifiedStorageDeals(ctx context.Context, b bool) error {
	//return sm.SetConsiderUnverifiedStorageDealsConfigFunc(b)
	panic("not impl")
}

func (sm *StorageMinerAPI) DealsGetExpectedSealDurationFunc(ctx context.Context) (time.Duration, error) {
	//return sm.GetExpectedSealDurationFunc()
	panic("not impl")
}

func (sm *StorageMinerAPI) DealsSetExpectedSealDurationFunc(ctx context.Context, d time.Duration) error {
	//return sm.SetExpectedSealDurationFunc(d)
	panic("not impl")
}

func (sm *StorageMinerAPI) DealsImportData(ctx context.Context, deal cid.Cid, fname string) error {
	/*fi, err := os.Open(fname)
	if err != nil {
		return xerrors.Errorf("failed to open given file: %w", err)
	}
	defer fi.Close() //nolint:errcheck

	return sm.StorageProvider.ImportDataForDeal(ctx, deal, fi)*/
	panic("not impl")
}

func (sm *StorageMinerAPI) DealsPieceCidBlocklist(ctx context.Context) ([]cid.Cid, error) {
	//return sm.StorageDealPieceCidBlocklistConfigFunc()
	panic("not impl")
}

func (sm *StorageMinerAPI) DealsSetPieceCidBlocklist(ctx context.Context, cids []cid.Cid) error {
	//return sm.SetStorageDealPieceCidBlocklistConfigFunc(cids)
	panic("not impl")
}

func (sm *StorageMinerAPI) StorageAddLocal(ctx context.Context, path string) error {
	if sm.StorageMgr == nil {
		return xerrors.Errorf("no storage manager")
	}

	return sm.StorageMgr.AddLocalStorage(ctx, path)
}

func (sm *StorageMinerAPI) PiecesListPieces(ctx context.Context) ([]cid.Cid, error) {
	//return sm.PieceStore.ListPieceInfoKeys()
	panic("not impl")
}

func (sm *StorageMinerAPI) PiecesListCidInfos(ctx context.Context) ([]cid.Cid, error) {
	//return sm.PieceStore.ListCidInfoKeys()
	panic("not impl")
}

func (sm *StorageMinerAPI) PiecesGetPieceInfo(ctx context.Context, pieceCid cid.Cid) (*piecestore.PieceInfo, error) {
	/*pi, err := sm.PieceStore.GetPieceInfo(pieceCid)
	if err != nil {
		return nil, err
	}
	return &pi, nil*/
	panic("not impl")
}

func (sm *StorageMinerAPI) PiecesGetCIDInfo(ctx context.Context, payloadCid cid.Cid) (*piecestore.CIDInfo, error) {
	/*ci, err := sm.PieceStore.GetCIDInfo(payloadCid)
	if err != nil {
		return nil, err
	}

	return &ci, nil*/
	panic("not impl")
}

func (sm *StorageMinerAPI) CreateBackup(ctx context.Context, fpath string) error {
	//return backup(sm.DS, fpath)
	panic("not impl")
}

func (sm *StorageMinerAPI) CheckProvable(ctx context.Context, pp abi.RegisteredPoStProof, sectors []sto.SectorRef, expensive bool) (map[abi.SectorNumber]string, error) {
	var rg storiface.RGetter
	if expensive {
		rg = func(ctx context.Context, id abi.SectorID) (cid.Cid, error) {
			si, err := sm.Miner.GetSectorInfo(id.Number)
			if err != nil {
				return cid.Undef, err
			}
			if si.CommR == nil {
				return cid.Undef, xerrors.Errorf("commr is nil")
			}

			return *si.CommR, nil
		}
	}

	bad, err := sm.StorageMgr.CheckProvable(ctx, pp, sectors, rg)
	if err != nil {
		return nil, err
	}

	var out = make(map[abi.SectorNumber]string)
	for sid, err := range bad {
		out[sid.Number] = err
	}

	return out, nil
}

func (sm *StorageMinerAPI) ActorAddressConfig(ctx context.Context) (api.AddressConfig, error) {
	return sm.AddrSel.AddressConfig, nil
}

func (sm *StorageMinerAPI) NetParamsConfig(ctx context.Context) (*config.NetParamsConfig, error) {
	return sm.NetParams, nil
}

func (sm *StorageMinerAPI) ComputeProof(ctx context.Context, sectorInfo []proof2.SectorInfo, randoness abi.PoStRandomness) ([]proof2.PoStProof, error) {
	return sm.prover.ComputeProof(ctx, sectorInfo, randoness)
}

var _ api.StorageMiner = &StorageMinerAPI{}
