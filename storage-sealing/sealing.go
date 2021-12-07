package sealing

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/go-state-types/dline"
	"github.com/filecoin-project/go-state-types/network"
	statemachine "github.com/filecoin-project/go-statemachine"
	"github.com/filecoin-project/specs-storage/storage"

	"github.com/filecoin-project/venus/app/submodule/apitypes"
	"github.com/filecoin-project/venus/pkg/types"
	"github.com/filecoin-project/venus/pkg/types/specactors/builtin/market"
	"github.com/filecoin-project/venus/pkg/types/specactors/builtin/miner"

	"github.com/filecoin-project/venus-sealer/api"
	"github.com/filecoin-project/venus-sealer/config"
	sectorstorage "github.com/filecoin-project/venus-sealer/sector-storage"
	"github.com/filecoin-project/venus-sealer/sector-storage/ffiwrapper"
	"github.com/filecoin-project/venus-sealer/service"
	"github.com/filecoin-project/venus-sealer/storage-sealing/sealiface"
	types2 "github.com/filecoin-project/venus-sealer/types"

	types3 "github.com/filecoin-project/venus-market/types"
	"github.com/filecoin-project/venus-market/piecestorage"
)

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
	StateMinerAvailableBalance(context.Context, address.Address, types2.TipSetToken) (big.Int, error)
	StateMinerSectorAllocated(context.Context, address.Address, abi.SectorNumber, types2.TipSetToken) (bool, error)
	StateMarketStorageDeal(context.Context, abi.DealID, types2.TipSetToken) (*apitypes.MarketDeal, error)
	StateMarketStorageDealProposal(context.Context, abi.DealID, types2.TipSetToken) (market.DealProposal, error)
	StateNetworkVersion(ctx context.Context, tok types2.TipSetToken) (network.Version, error)
	StateMinerProvingDeadline(context.Context, address.Address, types2.TipSetToken) (*dline.Info, error)
	StateMinerPartitions(ctx context.Context, m address.Address, dlIdx uint64, tok types2.TipSetToken) ([]apitypes.Partition, error)
	//	SendMsg(ctx context.Context, from, to address.Address, method abi.MethodNum, value, maxFee abi.TokenAmount, params []byte) (cid.Cid, error)
	ChainHead(ctx context.Context) (types2.TipSetToken, abi.ChainEpoch, error)
	ChainBaseFee(context.Context, types2.TipSetToken) (abi.TokenAmount, error)
	ChainGetMessage(ctx context.Context, mc cid.Cid) (*types.Message, error)
	StateGetRandomnessFromBeacon(ctx context.Context, personalization crypto.DomainSeparationTag, randEpoch abi.ChainEpoch, entropy []byte, tok types2.TipSetToken) (abi.Randomness, error)
	StateGetRandomnessFromTickets(ctx context.Context, personalization crypto.DomainSeparationTag, randEpoch abi.ChainEpoch, entropy []byte, tok types2.TipSetToken) (abi.Randomness, error)
	ChainReadObj(context.Context, cid.Cid) ([]byte, error)

	//for messager
	MessagerWaitMsg(context.Context, string) (types2.MsgLookup, error)
	MessagerSearchMsg(context.Context, string) (*types2.MsgLookup, error)
	MessagerSendMsg(ctx context.Context, from, to address.Address, method abi.MethodNum, value, maxFee abi.TokenAmount, params []byte) (string, error)

	//for market
	GetUnPackedDeals(ctx context.Context, miner address.Address, spec *types3.GetDealSpec) ([]*types3.DealInfoIncludePath, error)                   //perm:read
	MarkDealsAsPacking(ctx context.Context, miner address.Address, deals []abi.DealID) error                                                        //perm:write
	UpdateDealOnPacking(ctx context.Context, miner address.Address, dealId abi.DealID, sectorid abi.SectorNumber, offset abi.PaddedPieceSize) error //perm:write
}

type SectorStateNotifee func(before, after types2.SectorInfo)

type AddrSel func(ctx context.Context, mi miner.MinerInfo, use api.AddrUse, goodFunds, minFunds abi.TokenAmount) (address.Address, abi.TokenAmount, error)

type Sealing struct {
	api      SealingAPI
	DealInfo *CurrentDealInfoManager

	feeCfg config.MinerFeeConfig
	events Events

	startupWait sync.WaitGroup

	maddr address.Address

	sealer  sectorstorage.SectorManager
	sectors *statemachine.StateGroup
	sc      types2.SectorIDCounter
	verif   ffiwrapper.Verifier
	pcp     PreCommitPolicy

	inputLk        sync.Mutex
	openSectors    map[abi.SectorID]*openSector
	sectorTimers   map[abi.SectorID]*time.Timer
	pendingPieces  map[cid.Cid]*pendingPiece
	assignedPieces map[abi.SectorID][]cid.Cid
	creating       *abi.SectorNumber // used to prevent a race where we could create a new sector more than once

	upgradeLk sync.Mutex
	toUpgrade map[abi.SectorNumber]struct{}

	networkParams *config.NetParamsConfig
	notifee       SectorStateNotifee
	addrSel       AddrSel

	stats types2.SectorStats

	terminator  *TerminateBatcher
	precommiter *PreCommitBatcher
	commiter    *CommitBatcher

	getConfig    types2.GetSealingConfigFunc
	pieceStorage piecestorage.IPieceStorage
	//service
	logService *service.LogService
}

type openSector struct {
	used abi.UnpaddedPieceSize // change to bitfield/rle when AddPiece gains offset support to better fill sectors

	maybeAccept func(cid.Cid) error // called with inputLk
}

type pendingPiece struct {
	size abi.UnpaddedPieceSize
	deal types2.PieceDealInfo

	data storage.Data

	assigned bool // assigned to a sector?
	accepted func(abi.SectorNumber, abi.UnpaddedPieceSize, error)
}

func New(mctx context.Context, api SealingAPI, fc config.MinerFeeConfig, events Events, maddr address.Address, metaDataService *service.MetadataService, sectorInfoService *service.SectorInfoService, logService *service.LogService, sealer sectorstorage.SectorManager, sc types2.SectorIDCounter, verif ffiwrapper.Verifier, prov ffiwrapper.Prover, pcp PreCommitPolicy, gc types2.GetSealingConfigFunc, notifee SectorStateNotifee, as AddrSel, networkParams *config.NetParamsConfig, pieceStorage piecestorage.IPieceStorage) *Sealing {
	s := &Sealing{
		api:      api,
		DealInfo: &CurrentDealInfoManager{api},
		feeCfg:   fc,
		events:   events,

		pieceStorage:  pieceStorage,
		networkParams: networkParams,
		maddr:         maddr,
		sealer:        sealer,
		sc:            sc,
		verif:         verif,
		pcp:           pcp,
		logService:    logService,

		openSectors:    map[abi.SectorID]*openSector{},
		sectorTimers:   map[abi.SectorID]*time.Timer{},
		pendingPieces:  map[cid.Cid]*pendingPiece{},
		assignedPieces: map[abi.SectorID][]cid.Cid{},
		toUpgrade:      map[abi.SectorNumber]struct{}{},

		notifee: notifee,
		addrSel: as,

		terminator:  NewTerminationBatcher(mctx, maddr, api, as, fc),
		precommiter: NewPreCommitBatcher(mctx, networkParams, maddr, api, as, fc, gc),
		commiter:    NewCommitBatcher(mctx, networkParams, maddr, api, as, fc, gc, prov),

		getConfig: gc,

		stats: types2.SectorStats{
			BySector: map[abi.SectorID]types2.StatSectorState{},
		},
	}
	s.startupWait.Add(1)

	s.sectors = statemachine.New(sectorInfoService, s, types2.SectorInfo{})

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

func (m *Sealing) Remove(ctx context.Context, sid abi.SectorNumber) error {
	m.startupWait.Wait()

	return m.sectors.Send(uint64(sid), SectorRemove{})
}

func (m *Sealing) Terminate(ctx context.Context, sid abi.SectorNumber) error {
	m.startupWait.Wait()

	return m.sectors.Send(uint64(sid), SectorTerminate{})
}

func (m *Sealing) TerminateFlush(ctx context.Context) (string, error) {
	return m.terminator.Flush(ctx)
}

func (m *Sealing) TerminatePending(ctx context.Context) ([]abi.SectorID, error) {
	return m.terminator.Pending(ctx)
}

func (m *Sealing) SectorPreCommitFlush(ctx context.Context) ([]sealiface.PreCommitBatchRes, error) {
	return m.precommiter.Flush(ctx)
}

func (m *Sealing) SectorPreCommitPending(ctx context.Context) ([]abi.SectorID, error) {
	return m.precommiter.Pending(ctx)
}

func (m *Sealing) CommitFlush(ctx context.Context) ([]sealiface.CommitBatchRes, error) {
	return m.commiter.Flush(ctx)
}

func (m *Sealing) CommitPending(ctx context.Context) ([]abi.SectorID, error) {
	return m.commiter.Pending(ctx)
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

func getDealPerSectorLimit(size abi.SectorSize) (int, error) {
	if size < 64<<30 {
		return 256, nil
	}
	return 512, nil
}
