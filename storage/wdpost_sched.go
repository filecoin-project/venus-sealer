package storage

import (
	"context"
	"github.com/filecoin-project/venus-sealer/constants"
	"github.com/filecoin-project/venus/pkg/chain"
	"time"

	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/dline"
	"github.com/filecoin-project/specs-storage/storage"

	"github.com/filecoin-project/venus-sealer/config"
	sectorstorage "github.com/filecoin-project/venus-sealer/extern/sector-storage"
	"github.com/filecoin-project/venus-sealer/extern/sector-storage/ffiwrapper"
	"github.com/filecoin-project/venus-sealer/journal"
	"github.com/filecoin-project/venus/pkg/types"

	"go.opencensus.io/trace"
)

type WindowPoStScheduler struct {
	api              storageMinerApi
	feeCfg           config.MinerFeeConfig
	addrSel          *AddressSelector
	prover           storage.Prover
	verifier         ffiwrapper.Verifier
	faultTracker     sectorstorage.FaultTracker
	proofType        abi.RegisteredPoStProof
	partitionSectors uint64
	ch               *changeHandler
	networkParams    *config.NetParamsConfig
	actor            address.Address

	evtTypes [4]journal.EventType
	journal  journal.Journal

	// failed abi.ChainEpoch // eps
	// failLk sync.Mutex
}

func NewWindowedPoStScheduler(api storageMinerApi, fc config.MinerFeeConfig, as *AddressSelector, sb storage.Prover, verif ffiwrapper.Verifier, ft sectorstorage.FaultTracker, j journal.Journal, actor address.Address, networkParams *config.NetParamsConfig) (*WindowPoStScheduler, error) {
	mi, err := api.StateMinerInfo(context.TODO(), actor, types.EmptyTSK)
	if err != nil {
		return nil, xerrors.Errorf("getting sector size: %w", err)
	}

	return &WindowPoStScheduler{
		api:              api,
		feeCfg:           fc,
		addrSel:          as,
		prover:           sb,
		verifier:         verif,
		faultTracker:     ft,
		proofType:        mi.WindowPoStProofType,
		partitionSectors: mi.WindowPoStPartitionSectors,
		networkParams:    networkParams,
		actor:            actor,
		evtTypes: [...]journal.EventType{
			evtTypeWdPoStScheduler:  j.RegisterEventType("wdpost", "scheduler"),
			evtTypeWdPoStProofs:     j.RegisterEventType("wdpost", "proofs_processed"),
			evtTypeWdPoStRecoveries: j.RegisterEventType("wdpost", "recoveries_processed"),
			evtTypeWdPoStFaults:     j.RegisterEventType("wdpost", "faults_processed"),
		},
		journal: j,
	}, nil
}

type changeHandlerAPIImpl struct {
	storageMinerApi
	*WindowPoStScheduler
}

func (s *WindowPoStScheduler) Run(ctx context.Context) {
	// Initialize change handler
	chImpl := &changeHandlerAPIImpl{storageMinerApi: s.api, WindowPoStScheduler: s}
	s.ch = newChangeHandler(chImpl, s.actor)
	defer s.ch.shutdown()
	s.ch.start()

	var notifs <-chan []*chain.HeadChange
	var err error
	var gotCur bool

	// not fine to panic after this point
	for {
		if notifs == nil {
			notifs, err = s.api.ChainNotify(ctx)
			if err != nil {
				log.Errorf("ChainNotify error: %+v", err)

				constants.Clock.Sleep(10 * time.Second)
				continue
			}

			gotCur = false
		}

		select {
		case changes, ok := <-notifs:
			if !ok {
				log.Warn("window post scheduler notifs channel closed")
				notifs = nil
				continue
			}

			if !gotCur {
				if len(changes) != 1 {
					log.Errorf("expected first notif to have len = 1")
					continue
				}
				chg := changes[0]
				if chg.Type != chain.HCCurrent {
					log.Errorf("expected first notif to tell current ts")
					continue
				}

				ctx, span := trace.StartSpan(ctx, "WindowPoStScheduler.headChange")

				s.update(ctx, nil, chg.Val)

				span.End()
				gotCur = true
				continue
			}

			ctx, span := trace.StartSpan(ctx, "WindowPoStScheduler.headChange")

			var lowest, highest *types.TipSet = nil, nil

			for _, change := range changes {
				if change.Val == nil {
					log.Errorf("change.Val was nil")
				}
				switch change.Type {
				case chain.HCRevert:
					lowest = change.Val
				case chain.HCApply:
					highest = change.Val
				}
			}

			s.update(ctx, lowest, highest)

			span.End()
		case <-ctx.Done():
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
