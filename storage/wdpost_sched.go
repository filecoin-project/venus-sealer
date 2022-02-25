package storage

import (
	"context"
	"time"

	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/dline"
	"github.com/filecoin-project/specs-storage/storage"

	"github.com/filecoin-project/venus-sealer/api"
	"github.com/filecoin-project/venus-sealer/config"
	"github.com/filecoin-project/venus-sealer/journal"
	sectorstorage "github.com/filecoin-project/venus-sealer/sector-storage"
	"github.com/filecoin-project/venus-sealer/sector-storage/ffiwrapper"

	"github.com/filecoin-project/venus/venus-shared/types"
)

// WindowPoStScheduler is the coordinator for WindowPoSt submissions, fault
// declaration, and recovery declarations. It watches the chain for reverts and
// applies, and schedules/run those processes as partition deadlines arrive.
//
// WindowPoStScheduler watches the chain though the changeHandler, which in turn
// turn calls the scheduler when the time arrives to do work.
type WindowPoStScheduler struct {
	Messager      api.IMessager
	networkParams *config.NetParamsConfig

	api              fullNodeFilteredAPI
	feeCfg           config.MinerFeeConfig
	addrSel          *AddressSelector
	prover           storage.Prover
	verifier         ffiwrapper.Verifier
	faultTracker     sectorstorage.FaultTracker
	proofType        abi.RegisteredPoStProof
	partitionSectors uint64
	ch               *changeHandler

	actor address.Address

	evtTypes [4]journal.EventType
	journal  journal.Journal

	// failed abi.ChainEpoch // eps
	// failLk sync.Mutex
}

// NewWindowedPoStScheduler creates a new WindowPoStScheduler scheduler.
func NewWindowedPoStScheduler(api fullNodeFilteredAPI,
	messager api.IMessager,
	fc config.MinerFeeConfig,
	as *AddressSelector,
	sp storage.Prover,
	verif ffiwrapper.Verifier,
	ft sectorstorage.FaultTracker,
	j journal.Journal,
	actor address.Address,
	networkParams *config.NetParamsConfig) (*WindowPoStScheduler, error) {
	mi, err := api.StateMinerInfo(context.TODO(), actor, types.EmptyTSK)
	if err != nil {
		return nil, xerrors.Errorf("getting sector size: %w", err)
	}

	return &WindowPoStScheduler{
		Messager:         messager,
		networkParams:    networkParams,
		api:              api,
		feeCfg:           fc,
		addrSel:          as,
		prover:           sp,
		verifier:         verif,
		faultTracker:     ft,
		proofType:        mi.WindowPoStProofType,
		partitionSectors: mi.WindowPoStPartitionSectors,

		actor: actor,
		evtTypes: [...]journal.EventType{
			evtTypeWdPoStScheduler:  j.RegisterEventType("wdpost", "scheduler"),
			evtTypeWdPoStProofs:     j.RegisterEventType("wdpost", "proofs_processed"),
			evtTypeWdPoStRecoveries: j.RegisterEventType("wdpost", "recoveries_processed"),
			evtTypeWdPoStFaults:     j.RegisterEventType("wdpost", "faults_processed"),
		},
		journal: j,
	}, nil
}

func (s *WindowPoStScheduler) Run(ctx context.Context) {
	// Initialize change handler.

	// callbacks is a union of the fullNodeFilteredAPI and ourselves.
	callbacks := struct {
		fullNodeFilteredAPI
		*WindowPoStScheduler
	}{s.api, s}

	s.ch = newChangeHandler(callbacks, s.actor)
	defer s.ch.shutdown()
	s.ch.start()

	latest, err := s.api.ChainHead(ctx)
	if err != nil {
		log.Errorf("error to get latest chain head %v", err)
	}
	s.update(ctx, nil, latest)

	tm := time.NewTicker(time.Duration(s.networkParams.BlockDelaySecs*2) * time.Second)
	defer tm.Stop()

	for {
		select {
		case <-tm.C:
			current, err := s.api.ChainHead(ctx)
			if err != nil {
				log.Errorf("error to get latest chain head %v", err)
				continue
			}
			changes, err := s.api.ChainGetPath(ctx, latest.Key(), current.Key())
			if err != nil {
				log.Errorf("error to get chain path %v", err)
				continue
			}

			var lowest, highest *types.TipSet = nil, nil

			for _, change := range changes {
				if change.Val == nil {
					log.Errorf("change.Val was nil")
				}
				switch change.Type {
				case types.HCRevert:
					lowest = change.Val
				case types.HCApply:
					highest = change.Val
				}
			}

			s.update(ctx, lowest, highest)
			latest = current
		case <-ctx.Done():

			log.Warnf("cancel windows post scheduler")
			return
		}
	}
}

func (s *WindowPoStScheduler) update(ctx context.Context, revert, apply *types.TipSet) {
	if apply == nil {
		log.Error("no new tipset in window post WindowPoStScheduler.update")
		return
	}
	err := s.ch.update(ctx, revert, apply)
	if err != nil {
		log.Errorf("handling head updates in window post sched: %+v", err)
	}
}

// onAbort is called when generating proofs or submitting proofs is aborted
func (s *WindowPoStScheduler) onAbort(ts *types.TipSet, deadline *dline.Info) {
	s.journal.RecordEvent(s.evtTypes[evtTypeWdPoStScheduler], func() interface{} {
		c := evtCommon{}
		if ts != nil {
			c.Deadline = deadline
			c.Height = ts.Height()
			c.TipSet = ts.Cids()
		}
		return WdPoStSchedulerEvt{
			evtCommon: c,
			State:     SchedulerStateAborted,
		}
	})
}

func (s *WindowPoStScheduler) getEvtCommon(err error) evtCommon {
	c := evtCommon{Error: err}
	currentTS, currentDeadline := s.ch.currentTSDI()
	if currentTS != nil {
		c.Deadline = currentDeadline
		c.Height = currentTS.Height()
		c.TipSet = currentTS.Cids()
	}
	return c
}
