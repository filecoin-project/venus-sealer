package sealing

import (
	"bytes"
	"context"
	"sort"
	"sync"
	"time"

	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-bitfield"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-state-types/network"
	miner5 "github.com/filecoin-project/specs-actors/v5/actors/builtin/miner"
	proof5 "github.com/filecoin-project/specs-actors/v5/actors/runtime/proof"

	actors "github.com/filecoin-project/venus/pkg/specactors"
	"github.com/filecoin-project/venus/pkg/specactors/builtin/miner"
	"github.com/filecoin-project/venus/pkg/specactors/policy"

	"github.com/filecoin-project/venus-sealer/api"
	"github.com/filecoin-project/venus-sealer/config"
	"github.com/filecoin-project/venus-sealer/sector-storage/ffiwrapper"
	"github.com/filecoin-project/venus-sealer/storage-sealing/sealiface"
	"github.com/filecoin-project/venus-sealer/types"
)

const arp = abi.RegisteredAggregationProof_SnarkPackV1

var aggFeeNum = big.NewInt(110)
var aggFeeDen = big.NewInt(100)

//go:generate go run github.com/golang/mock/mockgen -destination=mocks/mock_commit_batcher.go -package=mocks . CommitBatcherApi

type CommitBatcherApi interface {
	//for messager
	MessagerSendMsg(ctx context.Context, from, to address.Address, method abi.MethodNum, value, maxFee abi.TokenAmount, params []byte) (string, error)

	StateMinerInfo(context.Context, address.Address, types.TipSetToken) (miner.MinerInfo, error)
	ChainHead(ctx context.Context) (types.TipSetToken, abi.ChainEpoch, error)
	ChainBaseFee(context.Context, types.TipSetToken) (abi.TokenAmount, error)

	StateSectorPreCommitInfo(ctx context.Context, maddr address.Address, sectorNumber abi.SectorNumber, tok types.TipSetToken) (*miner.SectorPreCommitOnChainInfo, error)
	StateMinerInitialPledgeCollateral(context.Context, address.Address, miner.SectorPreCommitInfo, types.TipSetToken) (big.Int, error)
	StateNetworkVersion(ctx context.Context, tok types.TipSetToken) (network.Version, error)
}

type AggregateInput struct {
	Spt   abi.RegisteredSealProof
	Info  proof5.AggregateSealVerifyInfo
	Proof []byte
}

type CommitBatcher struct {
	api       CommitBatcherApi
	maddr     address.Address
	mctx      context.Context
	addrSel   AddrSel
	feeCfg    config.MinerFeeConfig
	getConfig types.GetSealingConfigFunc
	prover    ffiwrapper.Prover

	cutoffs map[abi.SectorNumber]time.Time
	todo    map[abi.SectorNumber]AggregateInput
	waiting map[abi.SectorNumber][]chan sealiface.CommitBatchRes

	notify, stop, stopped chan struct{}
	force                 chan chan []sealiface.CommitBatchRes
	lk                    sync.Mutex

	networkParams *config.NetParamsConfig
}

func NewCommitBatcher(mctx context.Context, networkParams *config.NetParamsConfig, maddr address.Address, api CommitBatcherApi, addrSel AddrSel, feeCfg config.MinerFeeConfig, getConfig types.GetSealingConfigFunc, prov ffiwrapper.Prover) *CommitBatcher {
	b := &CommitBatcher{
		api:       api,
		maddr:     maddr,
		mctx:      mctx,
		addrSel:   addrSel,
		feeCfg:    feeCfg,
		getConfig: getConfig,
		prover:    prov,

		cutoffs: map[abi.SectorNumber]time.Time{},
		todo:    map[abi.SectorNumber]AggregateInput{},
		waiting: map[abi.SectorNumber][]chan sealiface.CommitBatchRes{},

		notify:  make(chan struct{}, 1),
		force:   make(chan chan []sealiface.CommitBatchRes),
		stop:    make(chan struct{}),
		stopped: make(chan struct{}),

		networkParams: networkParams,
	}

	go b.run()

	return b
}

func (b *CommitBatcher) run() {
	var forceRes chan []sealiface.CommitBatchRes
	var lastMsg []sealiface.CommitBatchRes

	cfg, err := b.getConfig()
	if err != nil {
		panic(err)
	}

	timer := time.NewTimer(b.batchWait(cfg.CommitBatchWait, cfg.CommitBatchSlack))
	for {
		if forceRes != nil {
			forceRes <- lastMsg
			forceRes = nil
		}
		lastMsg = nil

		// indicates whether we should only start a batch if we have reached or exceeded cfg.MaxCommitBatch
		var sendAboveMax bool
		select {
		case <-b.stop:
			close(b.stopped)
			return
		case <-b.notify:
			sendAboveMax = true
		case <-timer.C:
			// do nothing
		case fr := <-b.force: // user triggered
			forceRes = fr
		}

		var err error
		lastMsg, err = b.maybeStartBatch(sendAboveMax)
		if err != nil {
			log.Warnw("CommitBatcher processBatch error", "error", err)
		}

		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}

		timer.Reset(b.batchWait(cfg.CommitBatchWait, cfg.CommitBatchSlack))
	}
}

func (b *CommitBatcher) batchWait(maxWait, slack time.Duration) time.Duration {
	now := time.Now()

	b.lk.Lock()
	defer b.lk.Unlock()

	if len(b.todo) == 0 {
		return maxWait
	}

	var cutoff time.Time
	for sn := range b.todo {
		sectorCutoff := b.cutoffs[sn]
		if cutoff.IsZero() || (!sectorCutoff.IsZero() && sectorCutoff.Before(cutoff)) {
			cutoff = sectorCutoff
		}
	}
	for sn := range b.waiting {
		sectorCutoff := b.cutoffs[sn]
		if cutoff.IsZero() || (!sectorCutoff.IsZero() && sectorCutoff.Before(cutoff)) {
			cutoff = sectorCutoff
		}
	}

	if cutoff.IsZero() {
		return maxWait
	}

	cutoff = cutoff.Add(-slack)
	if cutoff.Before(now) {
		return time.Nanosecond // can't return 0
	}

	wait := cutoff.Sub(now)
	if wait > maxWait {
		wait = maxWait
	}

	return wait
}

func (b *CommitBatcher) maybeStartBatch(notif bool) ([]sealiface.CommitBatchRes, error) {
	b.lk.Lock()
	defer b.lk.Unlock()

	total := len(b.todo)
	if total == 0 {
		return nil, nil // nothing to do
	}

	cfg, err := b.getConfig()
	if err != nil {
		return nil, xerrors.Errorf("getting config: %w", err)
	}

	if notif && total < cfg.MaxCommitBatch {
		return nil, nil
	}

	var res []sealiface.CommitBatchRes

	individual := (total < cfg.MinCommitBatch) || (total < miner5.MinAggregatedSectors)

	if !individual && !cfg.AggregateAboveBaseFee.Equals(big.Zero()) {
		tok, _, err := b.api.ChainHead(b.mctx)
		if err != nil {
			return nil, err
		}

		bf, err := b.api.ChainBaseFee(b.mctx, tok)
		if err != nil {
			return nil, xerrors.Errorf("couldn't get base fee: %w", err)
		}

		if bf.LessThan(cfg.AggregateAboveBaseFee) {
			individual = true
		}
	}

	if individual {
		res, err = b.processIndividually()
	} else {
		res, err = b.processBatch(cfg)
	}
	if err != nil && len(res) == 0 {
		return nil, err
	}

	for _, r := range res {
		if err != nil {
			r.Error = err.Error()
		}

		for _, sn := range r.Sectors {
			for _, ch := range b.waiting[sn] {
				ch <- r // buffered
			}

			delete(b.waiting, sn)
			delete(b.todo, sn)
			delete(b.cutoffs, sn)
		}
	}

	return res, nil
}

func (b *CommitBatcher) processBatch(cfg sealiface.Config) ([]sealiface.CommitBatchRes, error) {
	tok, _, err := b.api.ChainHead(b.mctx)
	if err != nil {
		return nil, err
	}

	total := len(b.todo)

	res := sealiface.CommitBatchRes{
		FailedSectors: map[abi.SectorNumber]string{},
	}

	params := miner5.ProveCommitAggregateParams{
		SectorNumbers: bitfield.New(),
	}

	proofs := make([][]byte, 0, total)
	infos := make([]proof5.AggregateSealVerifyInfo, 0, total)
	collateral := big.Zero()

	for id, p := range b.todo {
		if len(infos) >= cfg.MaxCommitBatch {
			log.Infow("commit batch full")
			break
		}

		res.Sectors = append(res.Sectors, id)

		sc, err := b.getSectorCollateral(id, tok)
		if err != nil {
			res.FailedSectors[id] = err.Error()
			continue
		}

		collateral = big.Add(collateral, sc)

		params.SectorNumbers.Set(uint64(id))
		infos = append(infos, p.Info)
	}

	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Number < infos[j].Number
	})

	for _, info := range infos {
		proofs = append(proofs, b.todo[info.Number].Proof)
	}

	mid, err := address.IDFromAddress(b.maddr)
	if err != nil {
		return []sealiface.CommitBatchRes{res}, xerrors.Errorf("getting miner id: %w", err)
	}

	params.AggregateProof, err = b.prover.AggregateSealProofs(proof5.AggregateSealVerifyProofAndInfos{
		Miner:          abi.ActorID(mid),
		SealProof:      b.todo[infos[0].Number].Spt,
		AggregateProof: arp,
		Infos:          infos,
	}, proofs)
	if err != nil {
		return []sealiface.CommitBatchRes{res}, xerrors.Errorf("aggregating proofs: %w", err)
	}

	enc := new(bytes.Buffer)
	if err := params.MarshalCBOR(enc); err != nil {
		return []sealiface.CommitBatchRes{res}, xerrors.Errorf("couldn't serialize ProveCommitAggregateParams: %w", err)
	}

	mi, err := b.api.StateMinerInfo(b.mctx, b.maddr, nil)
	if err != nil {
		return []sealiface.CommitBatchRes{res}, xerrors.Errorf("couldn't get miner info: %w", err)
	}

	maxFee := b.feeCfg.MaxCommitBatchGasFee.FeeForSectors(len(infos))

	bf, err := b.api.ChainBaseFee(b.mctx, tok)
	if err != nil {
		return []sealiface.CommitBatchRes{res}, xerrors.Errorf("couldn't get base fee: %w", err)
	}

	nv, err := b.api.StateNetworkVersion(b.mctx, tok)
	if err != nil {
		log.Errorf("getting network version: %s", err)
		return []sealiface.CommitBatchRes{res}, xerrors.Errorf("getting network version: %s", err)
	}

	aggFee := big.Div(big.Mul(policy.AggregateNetworkFee(nv, len(infos), bf), aggFeeNum), aggFeeDen)

	needFunds := big.Add(collateral, aggFee)

	goodFunds := big.Add(maxFee, needFunds)

	from, _, err := b.addrSel(b.mctx, mi, api.CommitAddr, goodFunds, needFunds)
	if err != nil {
		return []sealiface.CommitBatchRes{res}, xerrors.Errorf("no good address found: %w", err)
	}

	uid, err := b.api.MessagerSendMsg(b.mctx, from, b.maddr, miner.Methods.ProveCommitAggregate, needFunds, maxFee, enc.Bytes())
	if err != nil {
		return []sealiface.CommitBatchRes{res}, xerrors.Errorf("sending message failed: %w", err)
	}

	res.Msg = uid

	log.Infow("Sent ProveCommitAggregate message", "uid", uid, "from", from, "todo", total, "sectors", len(infos))

	return []sealiface.CommitBatchRes{res}, nil
}

func (b *CommitBatcher) processIndividually() ([]sealiface.CommitBatchRes, error) {
	mi, err := b.api.StateMinerInfo(b.mctx, b.maddr, nil)
	if err != nil {
		return nil, xerrors.Errorf("couldn't get miner info: %w", err)
	}

	tok, _, err := b.api.ChainHead(b.mctx)
	if err != nil {
		return nil, err
	}

	var res []sealiface.CommitBatchRes

	for sn, info := range b.todo {
		r := sealiface.CommitBatchRes{
			Sectors:       []abi.SectorNumber{sn},
			FailedSectors: map[abi.SectorNumber]string{},
		}

		uid, err := b.processSingle(mi, sn, info, tok)
		if err != nil {
			log.Errorf("process single error: %+v", err) // todo: return to user
			r.FailedSectors[sn] = err.Error()
		} else {
			r.Msg = uid
		}

		res = append(res, r)
	}

	return res, nil
}

func (b *CommitBatcher) processSingle(mi miner.MinerInfo, sn abi.SectorNumber, info AggregateInput, tok types.TipSetToken) (string, error) {
	enc := new(bytes.Buffer)
	params := &miner.ProveCommitSectorParams{
		SectorNumber: sn,
		Proof:        info.Proof,
	}

	if err := params.MarshalCBOR(enc); err != nil {
		return "", xerrors.Errorf("marshaling commit params: %w", err)
	}

	collateral, err := b.getSectorCollateral(sn, tok)
	if err != nil {
		return "", err
	}

	goodFunds := big.Add(collateral, big.Int(b.feeCfg.MaxCommitGasFee))

	from, _, err := b.addrSel(b.mctx, mi, api.CommitAddr, goodFunds, collateral)
	if err != nil {
		return "", xerrors.Errorf("no good address to send commit message from: %w", err)
	}

	uid, err := b.api.MessagerSendMsg(b.mctx, from, b.maddr, miner.Methods.ProveCommitSector, collateral, big.Int(b.feeCfg.MaxCommitGasFee), enc.Bytes())
	if err != nil {
		return "", xerrors.Errorf("pushing message to mpool: %w", err)
	}

	return uid, nil
}

// register commit, wait for batch message, return message CID
func (b *CommitBatcher) AddCommit(ctx context.Context, s types.SectorInfo, in AggregateInput) (res sealiface.CommitBatchRes, err error) {
	sn := s.SectorNumber

	cu, err := b.getCommitCutoff(s)
	if err != nil {
		return sealiface.CommitBatchRes{}, err
	}

	b.lk.Lock()
	b.cutoffs[sn] = cu
	b.todo[sn] = in

	sent := make(chan sealiface.CommitBatchRes, 1)
	b.waiting[sn] = append(b.waiting[sn], sent)

	select {
	case b.notify <- struct{}{}:
	default: // already have a pending notification, don't need more
	}
	b.lk.Unlock()

	select {
	case r := <-sent:
		return r, nil
	case <-ctx.Done():
		return sealiface.CommitBatchRes{}, ctx.Err()
	}
}

func (b *CommitBatcher) Flush(ctx context.Context) ([]sealiface.CommitBatchRes, error) {
	resCh := make(chan []sealiface.CommitBatchRes, 1)
	select {
	case b.force <- resCh:
		select {
		case res := <-resCh:
			return res, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (b *CommitBatcher) Pending(ctx context.Context) ([]abi.SectorID, error) {
	b.lk.Lock()
	defer b.lk.Unlock()

	mid, err := address.IDFromAddress(b.maddr)
	if err != nil {
		return nil, err
	}

	res := make([]abi.SectorID, 0)
	for _, s := range b.todo {
		res = append(res, abi.SectorID{
			Miner:  abi.ActorID(mid),
			Number: s.Info.Number,
		})
	}

	sort.Slice(res, func(i, j int) bool {
		if res[i].Miner != res[j].Miner {
			return res[i].Miner < res[j].Miner
		}

		return res[i].Number < res[j].Number
	})

	return res, nil
}

func (b *CommitBatcher) Stop(ctx context.Context) error {
	close(b.stop)

	select {
	case <-b.stopped:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// TODO: If this returned epochs, it would make testing much easier
func (b *CommitBatcher) getCommitCutoff(si types.SectorInfo) (time.Time, error) {
	tok, curEpoch, err := b.api.ChainHead(b.mctx)
	if err != nil {
		return time.Now(), xerrors.Errorf("getting chain head: %s", err)
	}

	nv, err := b.api.StateNetworkVersion(b.mctx, tok)
	if err != nil {
		log.Errorf("getting network version: %s", err)
		return time.Now(), xerrors.Errorf("getting network version: %s", err)
	}

	pci, err := b.api.StateSectorPreCommitInfo(b.mctx, b.maddr, si.SectorNumber, tok)
	if err != nil {
		log.Errorf("getting precommit info: %s", err)
		return time.Now(), err
	}

	cutoffEpoch := pci.PreCommitEpoch + policy.GetMaxProveCommitDuration(actors.VersionForNetwork(nv), si.SectorType)

	for _, p := range si.Pieces {
		if p.DealInfo == nil {
			continue
		}

		startEpoch := p.DealInfo.DealSchedule.StartEpoch
		if startEpoch < cutoffEpoch {
			cutoffEpoch = startEpoch
		}
	}

	if cutoffEpoch <= curEpoch {
		return time.Now(), nil
	}

	return time.Now().Add(time.Duration(cutoffEpoch-curEpoch) * time.Duration(b.networkParams.BlockDelaySecs) * time.Second), nil
}

func (b *CommitBatcher) getSectorCollateral(sn abi.SectorNumber, tok types.TipSetToken) (abi.TokenAmount, error) {
	pci, err := b.api.StateSectorPreCommitInfo(b.mctx, b.maddr, sn, tok)
	if err != nil {
		return big.Zero(), xerrors.Errorf("getting precommit info: %w", err)
	}
	if pci == nil {
		return big.Zero(), xerrors.Errorf("precommit info not found on chain")
	}

	collateral, err := b.api.StateMinerInitialPledgeCollateral(b.mctx, b.maddr, pci.Info, tok)
	if err != nil {
		return big.Zero(), xerrors.Errorf("getting initial pledge collateral: %w", err)
	}

	collateral = big.Sub(collateral, pci.PreCommitDeposit)
	if collateral.LessThan(big.Zero()) {
		collateral = big.Zero()
	}

	return collateral, nil
}
