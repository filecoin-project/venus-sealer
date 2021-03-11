//go:generate go run ./gen

package sealing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/filecoin-project/venus-sealer/types"
	"reflect"
	"time"

	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-state-types/abi"
	statemachine "github.com/filecoin-project/go-statemachine"
)

func (m *Sealing) Plan(events []statemachine.Event, user interface{}) (interface{}, uint64, error) {
	next, processed, err := m.plan(events, user.(*types.SectorInfo))
	if err != nil || next == nil {
		return nil, processed, err
	}

	return func(ctx statemachine.Context, si types.SectorInfo) error {
		err := next(ctx, si)
		if err != nil {
			log.Errorf("unhandled sector error (%d): %+v", si.SectorNumber, err)
			return nil
		}

		return nil
	}, processed, nil // TODO: This processed event count is not very correct
}

var fsmPlanners = map[types.SectorState]func(events []statemachine.Event, state *types.SectorInfo) (uint64, error){
	// Sealing

	types.UndefinedSectorState: planOne(
		on(SectorStart{}, types.Empty),
		on(SectorStartCC{}, types.Packing),
	),
	types.Empty: planOne(on(SectorAddPiece{}, types.WaitDeals)),
	types.WaitDeals: planOne(
		on(SectorAddPiece{}, types.WaitDeals),
		on(SectorStartPacking{}, types.Packing),
	),
	types.Packing: planOne(on(SectorPacked{}, types.GetTicket)),
	types.GetTicket: planOne(
		on(SectorTicket{}, types.PreCommit1),
		on(SectorCommitFailed{}, types.CommitFailed),
	),
	types.PreCommit1: planOne(
		on(SectorPreCommit1{}, types.PreCommit2),
		on(SectorSealPreCommit1Failed{}, types.SealPreCommit1Failed),
		on(SectorDealsExpired{}, types.DealsExpired),
		on(SectorInvalidDealIDs{}, types.RecoverDealIDs),
		on(SectorOldTicket{}, types.GetTicket),
	),
	types.PreCommit2: planOne(
		on(SectorPreCommit2{}, types.PreCommitting),
		on(SectorSealPreCommit2Failed{}, types.SealPreCommit2Failed),
		on(SectorSealPreCommit1Failed{}, types.SealPreCommit1Failed),
	),
	types.PreCommitting: planOne(
		on(SectorSealPreCommit1Failed{}, types.SealPreCommit1Failed),
		on(SectorPreCommitted{}, types.PreCommitWait),
		on(SectorChainPreCommitFailed{}, types.PreCommitFailed),
		on(SectorPreCommitLanded{}, types.WaitSeed),
		on(SectorDealsExpired{}, types.DealsExpired),
		on(SectorInvalidDealIDs{}, types.RecoverDealIDs),
	),
	types.PreCommitWait: planOne(
		on(SectorChainPreCommitFailed{}, types.PreCommitFailed),
		on(SectorPreCommitLanded{}, types.WaitSeed),
		on(SectorRetryPreCommit{}, types.PreCommitting),
	),
	types.WaitSeed: planOne(
		on(SectorSeedReady{}, types.Committing),
		on(SectorChainPreCommitFailed{}, types.PreCommitFailed),
	),
	types.Committing: planCommitting,
	types.SubmitCommit: planOne(
		on(SectorCommitSubmitted{}, types.CommitWait),
		on(SectorCommitFailed{}, types.CommitFailed),
	),
	types.CommitWait: planOne(
		on(SectorProving{}, types.FinalizeSector),
		on(SectorCommitFailed{}, types.CommitFailed),
		on(SectorRetrySubmitCommit{}, types.SubmitCommit),
	),

	types.FinalizeSector: planOne(
		on(SectorFinalized{}, types.Proving),
		on(SectorFinalizeFailed{}, types.FinalizeFailed),
	),

	// Sealing errors

	types.SealPreCommit1Failed: planOne(
		on(SectorRetrySealPreCommit1{}, types.PreCommit1),
	),
	types.SealPreCommit2Failed: planOne(
		on(SectorRetrySealPreCommit1{}, types.PreCommit1),
		on(SectorRetrySealPreCommit2{}, types.PreCommit2),
	),
	types.PreCommitFailed: planOne(
		on(SectorRetryPreCommit{}, types.PreCommitting),
		on(SectorRetryPreCommitWait{}, types.PreCommitWait),
		on(SectorRetryWaitSeed{}, types.WaitSeed),
		on(SectorSealPreCommit1Failed{}, types.SealPreCommit1Failed),
		on(SectorPreCommitLanded{}, types.WaitSeed),
		on(SectorDealsExpired{}, types.DealsExpired),
		on(SectorInvalidDealIDs{}, types.RecoverDealIDs),
	),
	types.ComputeProofFailed: planOne(
		on(SectorRetryComputeProof{}, types.Committing),
		on(SectorSealPreCommit1Failed{}, types.SealPreCommit1Failed),
	),
	types.CommitFailed: planOne(
		on(SectorSealPreCommit1Failed{}, types.SealPreCommit1Failed),
		on(SectorRetryWaitSeed{}, types.WaitSeed),
		on(SectorRetryComputeProof{}, types.Committing),
		on(SectorRetryInvalidProof{}, types.Committing),
		on(SectorRetryPreCommitWait{}, types.PreCommitWait),
		on(SectorChainPreCommitFailed{}, types.PreCommitFailed),
		on(SectorRetryPreCommit{}, types.PreCommitting),
		on(SectorRetryCommitWait{}, types.CommitWait),
		on(SectorRetrySubmitCommit{}, types.SubmitCommit),
		on(SectorDealsExpired{}, types.DealsExpired),
		on(SectorInvalidDealIDs{}, types.RecoverDealIDs),
		on(SectorTicketExpired{}, types.Removing),
	),
	types.FinalizeFailed: planOne(
		on(SectorRetryFinalize{}, types.FinalizeSector),
	),
	types.PackingFailed: planOne(), // TODO: Deprecated, remove
	types.DealsExpired:  planOne(
	// SectorRemove (global)
	),
	types.RecoverDealIDs: planOne(
		onReturning(SectorUpdateDealIDs{}),
	),

	// Post-seal

	types.Proving: planOne(
		on(SectorFaultReported{}, types.FaultReported),
		on(SectorFaulty{}, types.Faulty),
	),
	types.Terminating: planOne(
		on(SectorTerminating{}, types.TerminateWait),
		on(SectorTerminateFailed{}, types.TerminateFailed),
	),
	types.TerminateWait: planOne(
		on(SectorTerminated{}, types.TerminateFinality),
		on(SectorTerminateFailed{}, types.TerminateFailed),
	),
	types.TerminateFinality: planOne(
		on(SectorTerminateFailed{}, types.TerminateFailed),
		// SectorRemove (global)
	),
	types.TerminateFailed: planOne(
	// SectorTerminating (global)
	),
	types.Removing: planOne(
		on(SectorRemoved{}, types.Removed),
		on(SectorRemoveFailed{}, types.RemoveFailed),
	),
	types.RemoveFailed: planOne(
	// SectorRemove (global)
	),
	types.Faulty: planOne(
		on(SectorFaultReported{}, types.FaultReported),
	),

	types.FaultReported: final, // not really supported right now

	types.FaultedFinal: final,
	types.Removed:      final,

	types.FailedUnrecoverable: final,
}

func (m *Sealing) plan(events []statemachine.Event, state *types.SectorInfo) (func(statemachine.Context, types.SectorInfo) error, uint64, error) {
	/////
	// First process all events

	for _, event := range events {
		e, err := json.Marshal(event)
		if err != nil {
			log.Errorf("marshaling event for logging: %+v", err)
			continue
		}

		l := types.Log{
			SectorNumber: state.SectorNumber,
			Timestamp:    uint64(time.Now().Unix()),
			Message:      string(e),
			Kind:         fmt.Sprintf("event;%T", event.User),
		}

		if err, iserr := event.User.(xerrors.Formatter); iserr {
			l.Trace = fmt.Sprintf("%+v", err)
		}
		count, err := m.logService.Count(state.SectorNumber)
		if err != nil {
			return nil, 0, xerrors.Errorf("get log count error %s, check db connection", err)
		}

		if count > 8000 {
			log.Warnw("truncating sector log", "sector", state.SectorNumber)
			/*	state.Log[2000] = types.Log{
					Timestamp: uint64(time.Now().Unix()),
					Message:   "truncating log (above 8000 entries)",
					Kind:      "truncate",
				}

				state.Log = append(state.Log[:2000], state.Log[6000:]...)*/
			err := m.logService.Truncate(state.SectorNumber)
			if err != nil {
				return nil, 0, xerrors.Errorf("fail to truncate log %w", err)
			}
		}
		err = m.logService.Append(&l)
		if err != nil {
			return nil, 0, xerrors.Errorf("fail to append log %w", err)
		}
	}

	if m.notifee != nil {
		defer func(before types.SectorInfo) {
			m.notifee(before, *state)
		}(*state) // take safe-ish copy of the before state (except for nested pointers)
	}

	p := fsmPlanners[state.State]
	if p == nil {
		return nil, 0, xerrors.Errorf("planner for state %s not found", state.State)
	}

	processed, err := p(events, state)
	if err != nil {
		return nil, 0, xerrors.Errorf("running planner for state %s failed: %w", state.State, err)
	}

	/////
	// Now decide what to do next

	/*

				*   Empty <- incoming deals
				|   |
				|   v
			    *<- WaitDeals <- incoming deals
				|   |
				|   v
				*<- Packing <- incoming committed capacity
				|   |
				|   v
				|   GetTicket
				|   |   ^
				|   v   |
				*<- PreCommit1 <--> SealPreCommit1Failed
				|   |       ^          ^^
				|   |       *----------++----\
				|   v       v          ||    |
				*<- PreCommit2 --------++--> SealPreCommit2Failed
				|   |                  ||
				|   v          /-------/|
				*   PreCommitting <-----+---> PreCommitFailed
				|   |                   |     ^
				|   v                   |     |
				*<- WaitSeed -----------+-----/
				|   |||  ^              |
				|   |||  \--------*-----/
				|   |||           |
				|   vvv      v----+----> ComputeProofFailed
				*<- Committing    |
				|   |        ^--> CommitFailed
				|   v             ^
		        |   SubmitCommit  |
		        |   |             |
		        |   v             |
				*<- CommitWait ---/
				|   |
				|   v
				|   FinalizeSector <--> FinalizeFailed
				|   |
				|   v
				*<- Proving
				|
				v
				FailedUnrecoverable

				UndefinedSectorState <- ¯\_(ツ)_/¯
					|                     ^
					*---------------------/

	*/

	m.stats.UpdateSector(m.minerSectorID(state.SectorNumber), state.State)

	switch state.State {
	// Happy path
	case types.Empty:
		fallthrough
	case types.WaitDeals:
		log.Infof("Waiting for deals %d", state.SectorNumber)
	case types.Packing:
		return m.handlePacking, processed, nil
	case types.GetTicket:
		return m.handleGetTicket, processed, nil
	case types.PreCommit1:
		return m.handlePreCommit1, processed, nil
	case types.PreCommit2:
		return m.handlePreCommit2, processed, nil
	case types.PreCommitting:
		return m.handlePreCommitting, processed, nil
	case types.PreCommitWait:
		return m.handlePreCommitWait, processed, nil
	case types.WaitSeed:
		return m.handleWaitSeed, processed, nil
	case types.Committing:
		return m.handleCommitting, processed, nil
	case types.SubmitCommit:
		return m.handleSubmitCommit, processed, nil
	case types.CommitWait:
		return m.handleCommitWait, processed, nil
	case types.FinalizeSector:
		return m.handleFinalizeSector, processed, nil

	// Handled failure modes
	case types.SealPreCommit1Failed:
		return m.handleSealPrecommit1Failed, processed, nil
	case types.SealPreCommit2Failed:
		return m.handleSealPrecommit2Failed, processed, nil
	case types.PreCommitFailed:
		return m.handlePreCommitFailed, processed, nil
	case types.ComputeProofFailed:
		return m.handleComputeProofFailed, processed, nil
	case types.CommitFailed:
		return m.handleCommitFailed, processed, nil
	case types.FinalizeFailed:
		return m.handleFinalizeFailed, processed, nil
	case types.PackingFailed: // DEPRECATED: remove this for the next reset
		state.State = types.DealsExpired
		fallthrough
	case types.DealsExpired:
		return m.handleDealsExpired, processed, nil
	case types.RecoverDealIDs:
		return m.handleRecoverDealIDs, processed, nil

	// Post-seal
	case types.Proving:
		return m.handleProvingSector, processed, nil
	case types.Terminating:
		return m.handleTerminating, processed, nil
	case types.TerminateWait:
		return m.handleTerminateWait, processed, nil
	case types.TerminateFinality:
		return m.handleTerminateFinality, processed, nil
	case types.TerminateFailed:
		return m.handleTerminateFailed, processed, nil
	case types.Removing:
		return m.handleRemoving, processed, nil
	case types.Removed:
		return nil, processed, nil

	case types.RemoveFailed:
		return m.handleRemoveFailed, processed, nil

		// Faults
	case types.Faulty:
		return m.handleFaulty, processed, nil
	case types.FaultReported:
		return m.handleFaultReported, processed, nil

	// Fatal errors
	case types.UndefinedSectorState:
		log.Error("sector update with undefined state!")
	case types.FailedUnrecoverable:
		log.Errorf("sector %d failed unrecoverably", state.SectorNumber)
	default:
		log.Errorf("unexpected sector update state: %s", state.State)
	}

	return nil, processed, nil
}

func planCommitting(events []statemachine.Event, state *types.SectorInfo) (uint64, error) {
	for i, event := range events {
		switch e := event.User.(type) {
		case globalMutator:
			if e.applyGlobal(state) {
				return uint64(i + 1), nil
			}
		case SectorCommitted: // the normal case
			e.apply(state)
			state.State = types.SubmitCommit
		case SectorSeedReady: // seed changed :/
			if e.SeedEpoch == state.SeedEpoch && bytes.Equal(e.SeedValue, state.SeedValue) {
				log.Warnf("planCommitting: got SectorSeedReady, but the seed didn't change")
				continue // or it didn't!
			}

			log.Warnf("planCommitting: commit Seed changed")
			e.apply(state)
			state.State = types.Committing
			return uint64(i + 1), nil
		case SectorComputeProofFailed:
			state.State = types.ComputeProofFailed
		case SectorSealPreCommit1Failed:
			state.State = types.SealPreCommit1Failed
		case SectorCommitFailed:
			state.State = types.CommitFailed
		case SectorRetryCommitWait:
			state.State = types.CommitWait
		default:
			return uint64(i), xerrors.Errorf("planCommitting got event of unknown type %T, events: %+v", event.User, events)
		}
	}
	return uint64(len(events)), nil
}

func (m *Sealing) restartSectors(ctx context.Context) error {
	trackedSectors, err := m.ListSectors()
	if err != nil {
		log.Errorf("loading sector list: %+v", err)
	}

	cfg, err := m.getConfig()
	if err != nil {
		return xerrors.Errorf("getting the sealing delay: %w", err)
	}

	spt, err := m.currentSealProof(ctx)
	if err != nil {
		return xerrors.Errorf("getting current seal proof: %w", err)
	}
	ssize, err := spt.SectorSize()
	if err != nil {
		return err
	}

	// m.unsealedInfoMap.lk.Lock() taken early in .New to prevent races
	defer m.unsealedInfoMap.lk.Unlock()

	for _, sector := range trackedSectors {
		if err := m.sectors.Send(uint64(sector.SectorNumber), SectorRestart{}); err != nil {
			log.Errorf("restarting sector %d: %+v", sector.SectorNumber, err)
		}

		if sector.State == types.WaitDeals {

			// put the sector in the unsealedInfoMap
			if _, ok := m.unsealedInfoMap.infos[sector.SectorNumber]; ok {
				// something's funky here, but probably safe to move on
				log.Warnf("sector %v was already in the unsealedInfoMap when restarting", sector.SectorNumber)
			} else {
				ui := UnsealedSectorInfo{
					ssize: ssize,
				}
				for _, p := range sector.Pieces {
					if p.DealInfo != nil {
						ui.numDeals++
					}
					ui.stored += p.Piece.Size
					ui.pieceSizes = append(ui.pieceSizes, p.Piece.Size.Unpadded())
				}

				m.unsealedInfoMap.infos[sector.SectorNumber] = ui
			}

			// start a fresh timer for the sector
			if cfg.WaitDealsDelay > 0 {
				timer := time.NewTimer(cfg.WaitDealsDelay)
				go func() {
					<-timer.C
					if err := m.StartPacking(sector.SectorNumber); err != nil {
						log.Errorf("starting sector %d: %+v", sector.SectorNumber, err)
					}
				}()
			}
		}
	}

	// TODO: Grab on-chain sector set and diff with trackedSectors

	return nil
}

func (m *Sealing) ForceSectorState(ctx context.Context, id abi.SectorNumber, state types.SectorState) error {
	return m.sectors.Send(id, SectorForceState{state})
}

func final(events []statemachine.Event, state *types.SectorInfo) (uint64, error) {
	if len(events) > 0 {
		if gm, ok := events[0].User.(globalMutator); ok {
			gm.applyGlobal(state)
			return 1, nil
		}
	}

	return 0, xerrors.Errorf("didn't expect any events in state %s, got %+v", state.State, events)
}

func on(mut mutator, next types.SectorState) func() (mutator, func(*types.SectorInfo) error) {
	return func() (mutator, func(*types.SectorInfo) error) {
		return mut, func(state *types.SectorInfo) error {
			state.State = next
			return nil
		}
	}
}

func onReturning(mut mutator) func() (mutator, func(*types.SectorInfo) error) {
	return func() (mutator, func(*types.SectorInfo) error) {
		return mut, func(state *types.SectorInfo) error {
			if state.Return == "" {
				return xerrors.Errorf("return state not set")
			}

			state.State = types.SectorState(state.Return)
			state.Return = ""
			return nil
		}
	}
}

func planOne(ts ...func() (mut mutator, next func(*types.SectorInfo) error)) func(events []statemachine.Event, state *types.SectorInfo) (uint64, error) {
	return func(events []statemachine.Event, state *types.SectorInfo) (uint64, error) {
		if gm, ok := events[0].User.(globalMutator); ok {
			gm.applyGlobal(state)
			return 1, nil
		}

		for _, t := range ts {
			mut, next := t()

			if reflect.TypeOf(events[0].User) != reflect.TypeOf(mut) {
				continue
			}

			if err, iserr := events[0].User.(error); iserr {
				log.Warnf("sector %d got error event %T: %+v", state.SectorNumber, events[0].User, err)
			}

			events[0].User.(mutator).apply(state)
			return 1, next(state)
		}

		_, ok := events[0].User.(Ignorable)
		if ok {
			return 1, nil
		}

		return 0, xerrors.Errorf("planner for state %s received unexpected event %T (%+v)", state.State, events[0].User, events[0])
	}
}
