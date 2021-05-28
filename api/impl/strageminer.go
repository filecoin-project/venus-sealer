package impl

import (
	"context"
	"encoding/json"
	"github.com/filecoin-project/go-fil-markets/piecestore"
	"github.com/filecoin-project/go-fil-markets/retrievalmarket"
	proof2 "github.com/filecoin-project/specs-actors/v2/actors/runtime/proof"
	types3 "github.com/filecoin-project/venus-messager/types"
	"github.com/filecoin-project/venus-sealer/config"
	"github.com/filecoin-project/venus-sealer/service"
	types2 "github.com/filecoin-project/venus-sealer/types"
	chain2 "github.com/filecoin-project/venus/app/submodule/chain"
	"github.com/filecoin-project/venus/pkg/chain"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/ipfs/go-cid"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc/auth"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"

	sto "github.com/filecoin-project/specs-storage/storage"
	"github.com/filecoin-project/venus-sealer/api"
	sectorstorage "github.com/filecoin-project/venus-sealer/sector-storage"
	"github.com/filecoin-project/venus-sealer/sector-storage/fsutil"
	"github.com/filecoin-project/venus-sealer/sector-storage/stores"
	"github.com/filecoin-project/venus-sealer/sector-storage/storiface"
	"github.com/filecoin-project/venus-sealer/storage"
	"github.com/filecoin-project/venus-sealer/storage/sectorblocks"
	"github.com/filecoin-project/venus/pkg/types"
	logging "github.com/ipfs/go-log/v2"
)

var log = logging.Logger("sealer")

type StorageMinerAPI struct {
	CommonAPI
	Prover       storage.WinningPoStProver
	SectorBlocks *sectorblocks.SectorBlocks
	Miner        *storage.Miner
	Full         api.FullNode
	Messager     api.IMessager
	StorageMgr   *sectorstorage.Manager `optional:"true"`
	IStorageMgr  sectorstorage.SectorManager
	*stores.Index
	storiface.WorkerReturn

	AddrSel *storage.AddressSelector

	LogService           *service.LogService
	NetParams            *config.NetParamsConfig
	SetSealingConfigFunc types2.SetSealingConfigFunc
	GetSealingConfigFunc types2.GetSealingConfigFunc
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

func (sm *StorageMinerAPI) PledgeSector(ctx context.Context) (abi.SectorID, error) {
	sr, err := sm.Miner.PledgeSector(ctx)
	if err != nil {
		return abi.SectorID{}, err
	}

	// wait for the sector to enter the Packing state
	// TODO: instead of polling implement some pubsub-type thing in storagefsm
	for {
		info, err := sm.Miner.GetSectorInfo(sr.ID.Number)
		if err != nil {
			return abi.SectorID{}, xerrors.Errorf("getting pledged sector info: %w", err)
		}

		if info.State != types2.UndefinedSectorState {
			return sr.ID, nil
		}

		select {
		case <-time.After(10 * time.Millisecond):
		case <-ctx.Done():
			return abi.SectorID{}, ctx.Err()
		}
	}
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

	logs, err := sm.LogService.List(sid)
	if err != nil {
		return api.SectorInfo{}, err
	}
	log := make([]api.SectorLog, len(logs))
	for i, l := range logs {
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

// List all staged sector's info in particular states
func (sm *StorageMinerAPI) SectorsInfoListInStates(ctx context.Context, states []api.SectorState, showOnChainInfo bool) ([]api.SectorInfo, error) {
	sectors, err := sm.Miner.ListSectors()
	if err != nil {
		return nil, err
	}

	var sis []types2.SectorInfo
	if states != nil && len(states) > 0 {
		filterStates := make(map[types2.SectorState]struct{})
		for _, state := range states {
			st := types2.SectorState(state)
			if _, ok := types2.ExistSectorStateList[st]; !ok {
				continue
			}
			filterStates[st] = struct{}{}
		}

		if len(filterStates) > 0 {
			for i := range sectors {
				if _, ok := filterStates[sectors[i].State]; ok {
					sis = append(sis, sectors[i])
				}
			}
		} else {
			sis = append(sis, sectors...)
		}
	} else {
		sis = append(sis, sectors...)
	}

	out := make([]api.SectorInfo, len(sis))
	for i, sector := range sis {
		deals := make([]abi.DealID, len(sector.Pieces))
		for i, piece := range sector.Pieces {
			if piece.DealInfo == nil {
				continue
			}
			deals[i] = piece.DealInfo.DealID
		}

		logs, err := sm.LogService.List(sector.SectorNumber)
		if err != nil {
			return nil, err
		}
		log := make([]api.SectorLog, len(logs))
		for i, l := range logs {
			log[i] = api.SectorLog{
				Kind:      l.Kind,
				Timestamp: l.Timestamp,
				Trace:     l.Trace,
				Message:   l.Message,
			}
		}

		sInfo := api.SectorInfo{
			SectorID: sector.SectorNumber,
			State:    api.SectorState(sector.State),
			CommD:    sector.CommD,
			CommR:    sector.CommR,
			Proof:    sector.Proof,
			Deals:    deals,
			Ticket: api.SealTicket{
				Value: sector.TicketValue,
				Epoch: sector.TicketEpoch,
			},
			Seed: api.SealSeed{
				Value: sector.SeedValue,
				Epoch: sector.SeedEpoch,
			},
			PreCommitMsg: sector.PreCommitMessage,
			CommitMsg:    sector.CommitMessage,
			Retries:      sector.InvalidProofs,
			ToUpgrade:    sm.Miner.IsMarkedForUpgrade(sector.SectorNumber),

			LastErr: sector.LastErr,
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

		if showOnChainInfo {
			onChainInfo, err := sm.Full.StateSectorGetInfo(ctx, sm.Miner.Address(), sector.SectorNumber, types.EmptyTSK)
			if err != nil {
				return nil, err
			}
			if onChainInfo != nil {
				sInfo.SealProof = onChainInfo.SealProof
				sInfo.Activation = onChainInfo.Activation
				sInfo.Expiration = onChainInfo.Expiration
				sInfo.DealWeight = onChainInfo.DealWeight
				sInfo.VerifiedDealWeight = onChainInfo.VerifiedDealWeight
				sInfo.InitialPledge = onChainInfo.InitialPledge

				ex, err := sm.Full.StateSectorExpiration(ctx, sm.Miner.Address(), sector.SectorNumber, types.EmptyTSK)
				if err == nil {
					sInfo.OnTime = ex.OnTime
					sInfo.Early = ex.Early
				} else {
					// TODO The official didn't deal with this
				}
			}
		}

		out[i] = sInfo
	}

	return out, nil
}

func (sm *StorageMinerAPI) SectorsListInStates(ctx context.Context, states []api.SectorState) ([]abi.SectorNumber, error) {
	filterStates := make(map[types2.SectorState]struct{})
	for _, state := range states {
		st := types2.SectorState(state)
		if _, ok := types2.ExistSectorStateList[st]; !ok {
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

func (sm *StorageMinerAPI) SectorsRefs(context.Context) (map[string][]types2.SealedRef, error) {
	// json can't handle cids as map keys
	out := map[string][]types2.SealedRef{}

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
	return sm.Miner.ForceSectorState(ctx, id, types2.SectorState(state))
}

func (sm *StorageMinerAPI) SectorRemove(ctx context.Context, id abi.SectorNumber) error {
	return sm.Miner.RemoveSector(ctx, id)
}

func (sm *StorageMinerAPI) SectorTerminate(ctx context.Context, id abi.SectorNumber) error {
	return sm.Miner.TerminateSector(ctx, id)
}

func (sm *StorageMinerAPI) SectorTerminateFlush(ctx context.Context) (string, error) {
	return sm.Miner.TerminateFlush(ctx)
}

func (sm *StorageMinerAPI) SectorTerminatePending(ctx context.Context) ([]abi.SectorID, error) {
	return sm.Miner.TerminatePending(ctx)
}

func (sm *StorageMinerAPI) SectorMarkForUpgrade(ctx context.Context, id abi.SectorNumber) error {
	return sm.Miner.MarkForUpgrade(id)
}

func (sm *StorageMinerAPI) WorkerConnect(ctx context.Context, url string) error {
	w, err := connectRemoteWorker(ctx, sm, url)
	if err != nil {
		return xerrors.Errorf("connecting remote storage failed: %w", err)
	}

	log.Infof("Connected to a remote worker at %s", url)

	return sm.StorageMgr.AddWorker(ctx, w)
}

func (sm *StorageMinerAPI) SealingSchedDiag(ctx context.Context, doSched bool) (interface{}, error) {
	return sm.StorageMgr.SchedDiag(ctx, doSched)
}

func (sm *StorageMinerAPI) SealingAbort(ctx context.Context, call types2.CallID) error {
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
	return sm.Prover.ComputeProof(ctx, sectorInfo, randoness)
}

func (sm *StorageMinerAPI) MessagerWaitMessage(ctx context.Context, uuid string, confidence uint64) (*chain.MsgLookup, error) {
	msg, err := sm.Messager.WaitMessage(ctx, uuid, confidence)
	if err != nil {
		return nil, err
	}

	return &chain.MsgLookup{
		Message: *msg.SignedCid,
		Receipt: *msg.Receipt,
		//	ReturnDec interface{}
		//TipSet   :msg.
		Height: abi.ChainEpoch(msg.Height),
	}, nil
}

func (sm *StorageMinerAPI) MessagerPushMessage(ctx context.Context, msg *types.Message, meta *types3.MsgMeta) (string, error) {
	return sm.Messager.PushMessage(ctx, msg, meta)
}

func (sm *StorageMinerAPI) MessagerGetMessage(ctx context.Context, uuid string) (*types3.Message, error) {
	msg, err := sm.Messager.GetMessageByUid(ctx, uuid)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

var _ api.StorageMiner = &StorageMinerAPI{}
