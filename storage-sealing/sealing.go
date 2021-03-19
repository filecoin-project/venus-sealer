package sealing

import (
	"context"
	"errors"
	"github.com/filecoin-project/venus-sealer/config"
	"github.com/filecoin-project/venus-sealer/lib/reader"
	"github.com/filecoin-project/venus-sealer/service"
	types2 "github.com/filecoin-project/venus-sealer/types"
	"github.com/filecoin-project/venus/app/submodule/chain"
	types3 "github.com/ipfs-force-community/venus-messager/types"
	"io"
	"math"
	"sync"
	"time"

	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	padreader "github.com/filecoin-project/go-padreader"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/go-state-types/dline"
	"github.com/filecoin-project/go-state-types/network"
	statemachine "github.com/filecoin-project/go-statemachine"
	"github.com/filecoin-project/specs-storage/storage"

	"github.com/filecoin-project/venus-sealer/api"
	sectorstorage "github.com/filecoin-project/venus-sealer/sector-storage"
	"github.com/filecoin-project/venus-sealer/sector-storage/ffiwrapper"
	"github.com/filecoin-project/venus/pkg/specactors/builtin/market"
	"github.com/filecoin-project/venus/pkg/specactors/builtin/miner"
	"github.com/filecoin-project/venus/pkg/types"
)

const SectorStorePrefix = "/sectors"

var ErrTooManySectorsSealing = xerrors.New("too many sectors sealing")

var log = logging.Logger("sectors")

type SectorLocation struct {
	Deadline  uint64
	Partition uint64
}

var ErrSectorAllocated = errors.New("sectorNumber is allocated, but PreCommit info wasn't found on chain")

type SealingAPI interface {
	StateWaitMsg(context.Context, cid.Cid) (types2.MsgLookup, error)
	StateSearchMsg(context.Context, cid.Cid) (*types2.MsgLookup, error)
	StateComputeDataCommitment(ctx context.Context, maddr address.Address, sectorType abi.RegisteredSealProof, deals []abi.DealID, tok types2.TipSetToken) (cid.Cid, error)

	// Can return ErrSectorAllocated in case precommit info wasn't found, but the sector number is marked as allocated
	StateSectorPreCommitInfo(ctx context.Context, maddr address.Address, sectorNumber abi.SectorNumber, tok types2.TipSetToken) (*miner.SectorPreCommitOnChainInfo, error)
	StateSectorGetInfo(ctx context.Context, maddr address.Address, sectorNumber abi.SectorNumber, tok types2.TipSetToken) (*miner.SectorOnChainInfo, error)
	StateSectorPartition(ctx context.Context, maddr address.Address, sectorNumber abi.SectorNumber, tok types2.TipSetToken) (*SectorLocation, error)
	StateLookupID(context.Context, address.Address, types2.TipSetToken) (address.Address, error)
	StateMinerSectorSize(context.Context, address.Address, types2.TipSetToken) (abi.SectorSize, error)
	StateMinerWorkerAddress(ctx context.Context, maddr address.Address, tok types2.TipSetToken) (address.Address, error)
	StateMinerPreCommitDepositForPower(context.Context, address.Address, miner.SectorPreCommitInfo, types2.TipSetToken) (big.Int, error)
	StateMinerInitialPledgeCollateral(context.Context, address.Address, miner.SectorPreCommitInfo, types2.TipSetToken) (big.Int, error)
	StateMinerInfo(context.Context, address.Address, types2.TipSetToken) (miner.MinerInfo, error)
	StateMinerSectorAllocated(context.Context, address.Address, abi.SectorNumber, types2.TipSetToken) (bool, error)
	StateMarketStorageDeal(context.Context, abi.DealID, types2.TipSetToken) (*chain.MarketDeal, error)
	StateMarketStorageDealProposal(context.Context, abi.DealID, types2.TipSetToken) (market.DealProposal, error)
	StateNetworkVersion(ctx context.Context, tok types2.TipSetToken) (network.Version, error)
	StateMinerProvingDeadline(context.Context, address.Address, types2.TipSetToken) (*dline.Info, error)
	StateMinerPartitions(ctx context.Context, m address.Address, dlIdx uint64, tok types2.TipSetToken) ([]chain.Partition, error)
	//	SendMsg(ctx context.Context, from, to address.Address, method abi.MethodNum, value, maxFee abi.TokenAmount, params []byte) (cid.Cid, error)
	ChainHead(ctx context.Context) (types2.TipSetToken, abi.ChainEpoch, error)
	ChainGetMessage(ctx context.Context, mc cid.Cid) (*types.Message, error)
	ChainGetRandomnessFromBeacon(ctx context.Context, tok types2.TipSetToken, personalization crypto.DomainSeparationTag, randEpoch abi.ChainEpoch, entropy []byte) (abi.Randomness, error)
	ChainGetRandomnessFromTickets(ctx context.Context, tok types2.TipSetToken, personalization crypto.DomainSeparationTag, randEpoch abi.ChainEpoch, entropy []byte) (abi.Randomness, error)
	ChainReadObj(context.Context, cid.Cid) ([]byte, error)

	//for messager
	MessagerWaitMsg(context.Context, types3.UUID) (types2.MsgLookup, error)
	MessagerSearchMsg(context.Context, types3.UUID) (*types2.MsgLookup, error)
	MessagerSendMsg(ctx context.Context, from, to address.Address, method abi.MethodNum, value, maxFee abi.TokenAmount, params []byte) (types3.UUID, error)
}

type SectorStateNotifee func(before, after types2.SectorInfo)

type AddrSel func(ctx context.Context, mi miner.MinerInfo, use api.AddrUse, goodFunds, minFunds abi.TokenAmount) (address.Address, abi.TokenAmount, error)

type Sealing struct {
	api    SealingAPI
	feeCfg FeeConfig
	events Events

	maddr address.Address

	sealer  sectorstorage.SectorManager
	sectors *statemachine.StateGroup
	sc      types2.SectorIDCounter
	verif   ffiwrapper.Verifier

	pcp             PreCommitPolicy
	unsealedInfoMap UnsealedSectorMap

	upgradeLk     sync.Mutex
	toUpgrade     map[abi.SectorNumber]struct{}
	networkParams *config.NetParamsConfig
	notifee       SectorStateNotifee
	addrSel       AddrSel

	stats types2.SectorStats

	terminator *TerminateBatcher

	getConfig types2.GetSealingConfigFunc
	dealInfo  *CurrentDealInfoManager

	//service
	logService *service.LogService
}

type FeeConfig struct {
	MaxPreCommitGasFee abi.TokenAmount
	MaxCommitGasFee    abi.TokenAmount
	MaxTerminateGasFee abi.TokenAmount
}

type UnsealedSectorMap struct {
	infos map[abi.SectorNumber]UnsealedSectorInfo
	lk    sync.Mutex
}

type UnsealedSectorInfo struct {
	numDeals uint64
	// stored should always equal sum of pieceSizes.Padded()
	stored     abi.PaddedPieceSize
	pieceSizes []abi.UnpaddedPieceSize
	ssize      abi.SectorSize
}

func New(api SealingAPI, fc FeeConfig, events Events, maddr address.Address, metaDataService *service.MetadataService, sectorInfoService *service.SectorInfoService, logService *service.LogService, sealer sectorstorage.SectorManager, sc types2.SectorIDCounter, verif ffiwrapper.Verifier, pcp PreCommitPolicy, gc types2.GetSealingConfigFunc, notifee SectorStateNotifee, as AddrSel, networkParams *config.NetParamsConfig) *Sealing {
	s := &Sealing{
		api:           api,
		feeCfg:        fc,
		events:        events,
		networkParams: networkParams,
		maddr:         maddr,
		sealer:        sealer,
		sc:            sc,
		verif:         verif,
		pcp:           pcp,
		logService:    logService,
		unsealedInfoMap: UnsealedSectorMap{
			infos: make(map[abi.SectorNumber]UnsealedSectorInfo),
			lk:    sync.Mutex{},
		},

		toUpgrade: map[abi.SectorNumber]struct{}{},

		notifee: notifee,
		addrSel: as,

		terminator: NewTerminationBatcher(context.TODO(), maddr, api, as, fc),

		getConfig: gc,
		dealInfo:  &CurrentDealInfoManager{api},

		stats: types2.SectorStats{
			BySector: map[abi.SectorID]types2.StatSectorState{},
		},
	}

	s.sectors = statemachine.New(sectorInfoService, s, types2.SectorInfo{})

	s.unsealedInfoMap.lk.Lock() // released after initialized in .Run()

	return s
}

func (m *Sealing) Run(ctx context.Context) error {
	if err := m.restartSectors(ctx); err != nil {
		log.Errorf("%+v", err)
		return xerrors.Errorf("failed load sector states: %w", err)
	}

	return nil
}

func (m *Sealing) Stop(ctx context.Context) error {
	if err := m.terminator.Stop(ctx); err != nil {
		return err
	}

	if err := m.sectors.Stop(ctx); err != nil {
		return err
	}
	return nil
}

func (m *Sealing) AddPieceToAnySector(ctx context.Context, size abi.UnpaddedPieceSize, r io.Reader, d types2.DealInfo) (abi.SectorNumber, abi.PaddedPieceSize, error) {
	log.Infof("Adding piece for deal %d (publish msg: %s)", d.DealID, d.PublishCid)
	if (padreader.PaddedSize(uint64(size))) != size {
		return 0, 0, xerrors.Errorf("cannot allocate unpadded piece")
	}

	sp, err := m.currentSealProof(ctx)
	if err != nil {
		return 0, 0, xerrors.Errorf("getting current seal proof type: %w", err)
	}

	ssize, err := sp.SectorSize()
	if err != nil {
		return 0, 0, err
	}

	if size > abi.PaddedPieceSize(ssize).Unpadded() {
		return 0, 0, xerrors.Errorf("piece cannot fit into a sector")
	}

	m.unsealedInfoMap.lk.Lock()

	sid, pads, err := m.getSectorAndPadding(ctx, size)
	if err != nil {
		m.unsealedInfoMap.lk.Unlock()
		return 0, 0, xerrors.Errorf("getting available sector: %w", err)
	}

	for _, p := range pads {
		err = m.addPiece(ctx, sid, p.Unpadded(), reader.NewNullReader(p.Unpadded()), nil)
		if err != nil {
			m.unsealedInfoMap.lk.Unlock()
			return 0, 0, xerrors.Errorf("writing pads: %w", err)
		}
	}

	offset := m.unsealedInfoMap.infos[sid].stored
	err = m.addPiece(ctx, sid, size, r, &d)

	if err != nil {
		m.unsealedInfoMap.lk.Unlock()
		return 0, 0, xerrors.Errorf("adding piece to sector: %w", err)
	}

	startPacking := m.unsealedInfoMap.infos[sid].numDeals >= getDealPerSectorLimit(ssize)

	m.unsealedInfoMap.lk.Unlock()

	if startPacking {
		if err := m.StartPacking(sid); err != nil {
			return 0, 0, xerrors.Errorf("start packing: %w", err)
		}
	}

	return sid, offset, nil
}

// Caller should hold m.unsealedInfoMap.lk
func (m *Sealing) addPiece(ctx context.Context, sectorID abi.SectorNumber, size abi.UnpaddedPieceSize, r io.Reader, di *types2.DealInfo) error {
	log.Infof("Adding piece to sector %d", sectorID)
	sp, err := m.currentSealProof(ctx)
	if err != nil {
		return xerrors.Errorf("getting current seal proof type: %w", err)
	}
	ssize, err := sp.SectorSize()
	if err != nil {
		return err
	}

	ppi, err := m.sealer.AddPiece(types2.WithPriority(ctx, types2.DealSectorPriority), m.minerSector(sp, sectorID), m.unsealedInfoMap.infos[sectorID].pieceSizes, size, r)
	if err != nil {
		return xerrors.Errorf("writing piece: %w", err)
	}
	piece := types2.Piece{
		Piece:    ppi,
		DealInfo: di,
	}

	err = m.sectors.Send(uint64(sectorID), SectorAddPiece{NewPiece: piece})
	if err != nil {
		return err
	}

	ui := m.unsealedInfoMap.infos[sectorID]
	num := m.unsealedInfoMap.infos[sectorID].numDeals
	if di != nil {
		num = num + 1
	}
	m.unsealedInfoMap.infos[sectorID] = UnsealedSectorInfo{
		numDeals:   num,
		stored:     ui.stored + piece.Piece.Size,
		pieceSizes: append(ui.pieceSizes, piece.Piece.Size.Unpadded()),
		ssize:      ssize,
	}

	return nil
}

func (m *Sealing) Remove(ctx context.Context, sid abi.SectorNumber) error {
	return m.sectors.Send(uint64(sid), SectorRemove{})
}

func (m *Sealing) Terminate(ctx context.Context, sid abi.SectorNumber) error {
	return m.sectors.Send(uint64(sid), SectorTerminate{})
}

func (m *Sealing) TerminateFlush(ctx context.Context) (*types3.UUID, error) {
	return m.terminator.Flush(ctx)
}

func (m *Sealing) TerminatePending(ctx context.Context) ([]abi.SectorID, error) {
	return m.terminator.Pending(ctx)
}

// Caller should NOT hold m.unsealedInfoMap.lk
func (m *Sealing) StartPacking(sectorID abi.SectorNumber) error {
	// locking here ensures that when the SectorStartPacking event is sent, the sector won't be picked up anywhere else
	m.unsealedInfoMap.lk.Lock()
	defer m.unsealedInfoMap.lk.Unlock()

	// cannot send SectorStartPacking to sectors that have already been packed, otherwise it will cause the state machine to exit
	if _, ok := m.unsealedInfoMap.infos[sectorID]; !ok {
		log.Warnf("call start packing, but sector %v not in unsealedInfoMap.infos, maybe have called", sectorID)
		return nil
	}
	log.Infof("Starting packing sector %d", sectorID)
	err := m.sectors.Send(uint64(sectorID), SectorStartPacking{})
	if err != nil {
		return err
	}
	log.Infof("send Starting packing event success sector %d", sectorID)

	delete(m.unsealedInfoMap.infos, sectorID)

	return nil
}

// Caller should hold m.unsealedInfoMap.lk
func (m *Sealing) getSectorAndPadding(ctx context.Context, size abi.UnpaddedPieceSize) (abi.SectorNumber, []abi.PaddedPieceSize, error) {
	for tries := 0; tries < 100; tries++ {
		for k, v := range m.unsealedInfoMap.infos {
			pads, padLength := ffiwrapper.GetRequiredPadding(v.stored, size.Padded())

			if v.stored+size.Padded()+padLength <= abi.PaddedPieceSize(v.ssize) {
				return k, pads, nil
			}
		}

		if len(m.unsealedInfoMap.infos) > 0 {
			log.Infow("tried to put a piece into an open sector, found none with enough space", "open", len(m.unsealedInfoMap.infos), "size", size, "tries", tries)
		}

		ns, ssize, err := m.newDealSector(ctx)
		switch err {
		case nil:
			m.unsealedInfoMap.infos[ns] = UnsealedSectorInfo{
				numDeals:   0,
				stored:     0,
				pieceSizes: nil,
				ssize:      ssize,
			}
		case errTooManySealing:
			m.unsealedInfoMap.lk.Unlock()

			select {
			case <-time.After(2 * time.Second):
			case <-ctx.Done():
				m.unsealedInfoMap.lk.Lock()
				return 0, nil, xerrors.Errorf("getting sector for piece: %w", ctx.Err())
			}

			m.unsealedInfoMap.lk.Lock()
			continue
		default:
			return 0, nil, xerrors.Errorf("creating new sector: %w", err)
		}

		return ns, nil, nil
	}

	return 0, nil, xerrors.Errorf("failed to allocate piece to a sector")
}

var errTooManySealing = errors.New("too many sectors sealing")

// newDealSector creates a new sector for deal storage
func (m *Sealing) newDealSector(ctx context.Context) (abi.SectorNumber, abi.SectorSize, error) {
	// First make sure we don't have too many 'open' sectors

	cfg, err := m.getConfig()
	if err != nil {
		return 0, 0, xerrors.Errorf("getting config: %w", err)
	}

	if cfg.MaxSealingSectorsForDeals > 0 {
		if m.stats.CurSealing() > cfg.MaxSealingSectorsForDeals {
			return 0, 0, ErrTooManySectorsSealing
		}
	}

	if cfg.MaxWaitDealsSectors > 0 && uint64(len(m.unsealedInfoMap.infos)) >= cfg.MaxWaitDealsSectors {
		// Too many sectors are sealing in parallel. Start sealing one, and retry
		// allocating the piece to a sector (we're dropping the lock here, so in
		// case other goroutines are also trying to create a sector, we retry in
		// getSectorAndPadding instead of here - otherwise if we have lots of
		// parallel deals in progress, we can start creating a ton of sectors
		// with just a single deal in them)
		var mostStored abi.PaddedPieceSize = math.MaxUint64
		var best abi.SectorNumber = math.MaxUint64

		for sn, info := range m.unsealedInfoMap.infos {
			if info.stored+1 > mostStored+1 { // 18446744073709551615 + 1 = 0
				best = sn
			}
		}

		if best != math.MaxUint64 {
			m.unsealedInfoMap.lk.Unlock()
			err := m.StartPacking(best)
			m.unsealedInfoMap.lk.Lock()

			if err != nil {
				log.Errorf("newDealSector StartPacking error: %+v", err)
				// let's pretend this is fine
			}
		}

		return 0, 0, errTooManySealing // will wait a bit and retry
	}

	spt, err := m.currentSealProof(ctx)
	if err != nil {
		return 0, 0, xerrors.Errorf("getting current seal proof type: %w", err)
	}

	// Now actually create a new sector

	sid, err := m.sc.Next()
	if err != nil {
		return 0, 0, xerrors.Errorf("getting sector number: %w", err)
	}

	err = m.sealer.NewSector(context.TODO(), m.minerSector(spt, sid))
	if err != nil {
		return 0, 0, xerrors.Errorf("initializing sector: %w", err)
	}

	log.Infof("Creating sector %d", sid)
	err = m.sectors.Send(uint64(sid), SectorStart{
		ID:         sid,
		SectorType: spt,
	})

	if err != nil {
		return 0, 0, xerrors.Errorf("starting the sector fsm: %w", err)
	}

	cf, err := m.getConfig()
	if err != nil {
		return 0, 0, xerrors.Errorf("getting the sealing delay: %w", err)
	}

	if cf.WaitDealsDelay > 0 {
		timer := time.NewTimer(cf.WaitDealsDelay)
		go func() {
			<-timer.C
			if err := m.StartPacking(sid); err != nil {
				log.Errorf("starting sector %d: %+v", sid, err)
			}
		}()
	}

	ssize, err := spt.SectorSize()
	return sid, ssize, err
}

// newSectorCC accepts a slice of pieces with no deal (junk data)
func (m *Sealing) newSectorCC(ctx context.Context, sid abi.SectorNumber, pieces []types2.Piece) error {
	spt, err := m.currentSealProof(ctx)
	if err != nil {
		return xerrors.Errorf("getting current seal proof type: %w", err)
	}

	log.Infof("Creating CC sector %d", sid)
	return m.sectors.Send(uint64(sid), SectorStartCC{
		ID:         sid,
		Pieces:     pieces,
		SectorType: spt,
	})
}

func (m *Sealing) currentSealProof(ctx context.Context) (abi.RegisteredSealProof, error) {
	mi, err := m.api.StateMinerInfo(ctx, m.maddr, nil)
	if err != nil {
		return 0, err
	}

	ver, err := m.api.StateNetworkVersion(ctx, nil)
	if err != nil {
		return 0, err
	}

	return miner.PreferredSealProofTypeFromWindowPoStType(ver, mi.WindowPoStProofType)
}

func (m *Sealing) minerSector(spt abi.RegisteredSealProof, num abi.SectorNumber) storage.SectorRef {
	return storage.SectorRef{
		ID:        m.minerSectorID(num),
		ProofType: spt,
	}
}

func (m *Sealing) minerSectorID(num abi.SectorNumber) abi.SectorID {
	mid, err := address.IDFromAddress(m.maddr)
	if err != nil {
		panic(err)
	}

	return abi.SectorID{
		Number: num,
		Miner:  abi.ActorID(mid),
	}
}

func (m *Sealing) Address() address.Address {
	return m.maddr
}

func getDealPerSectorLimit(size abi.SectorSize) uint64 {
	if size < 64<<30 {
		return 256
	}
	return 512
}
