//go:generate go run ./gen

package sealing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-state-types/abi"
	statemachine "github.com/filecoin-project/go-statemachine"

	"github.com/filecoin-project/venus-sealer/types"
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
		on(SectorStart{}, types.WaitDeals),
		on(SectorStartCC{}, types.Packing),
	),
	types.Empty: planOne( // deprecated
		on(SectorAddPiece{}, types.AddPiece),
		on(SectorStartPacking{}, types.Packing),
	),
	types.WaitDeals: planOne(
		on(SectorAddPiece{}, types.AddPiece),
		on(SectorStartPacking{}, types.Packing),
	),
	types.AddPiece: planOne(
		on(SectorPieceAdded{}, types.WaitDeals),
		apply(SectorStartPacking{}),
		apply(SectorAddPiece{}),
		on(SectorAddPieceFailed{}, types.AddPieceFailed),
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
		on(SectorOldTicket{}, types.GetTicket),
	),
	types.PreCommitting: planOne(
		on(SectorPreCommitBatch{}, types.SubmitPreCommitBatch),
		on(SectorPreCommitted{}, types.PreCommitWait),
		on(SectorSealPreCommit1Failed{}, types.SealPreCommit1Failed),
		on(SectorChainPreCommitFailed{}, types.PreCommitFailed),
		on(SectorPreCommitLanded{}, types.WaitSeed),
		on(SectorDealsExpired{}, types.DealsExpired),
		on(SectorInvalidDealIDs{}, types.RecoverDealIDs),
	),
	types.SubmitPreCommitBatch: planOne(
		on(SectorPreCommitBatchSent{}, types.PreCommitBatchWait),
		on(SectorSealPreCommit1Failed{}, types.SealPreCommit1Failed),
		on(SectorChainPreCommitFailed{}, types.PreCommitFailed),
		on(SectorPreCommitLanded{}, types.WaitSeed),
		on(SectorDealsExpired{}, types.DealsExpired),
		on(SectorInvalidDealIDs{}, types.RecoverDealIDs),
	),
	types.PreCommitBatchWait: planOne(
		on(SectorChainPreCommitFailed{}, types.PreCommitFailed),
		on(SectorPreCommitLanded{}, types.WaitSeed),
		on(SectorRetryPreCommit{}, types.PreCommitting),
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
	types.CommitFinalize: planOne(
		on(SectorFinalized{}, types.SubmitCommit),
		on(SectorFinalizeFailed{}, types.CommitFinalizeFailed),
	),
	types.SubmitCommit: planOne(
		on(SectorCommitSubmitted{}, types.CommitWait),
		on(SectorSubmitCommitAggregate{}, types.SubmitCommitAggregate),
		on(SectorCommitFailed{}, types.CommitFailed),
	),
	types.SubmitCommitAggregate: planOne(
		on(SectorCommitAggregateSent{}, types.CommitAggregateWait),
		on(SectorCommitFailed{}, types.CommitFailed),
		on(SectorRetrySubmitCommit{}, types.SubmitCommit),
	),
	types.CommitWait: planOne(
		on(SectorProving{}, types.FinalizeSector),
		on(SectorCommitFailed{}, types.CommitFailed),
		on(SectorRetrySubmitCommit{}, types.SubmitCommit),
	),
	types.CommitAggregateWait: planOne(
		on(SectorProving{}, types.FinalizeSector),
		on(SectorCommitFailed{}, types.CommitFailed),
		on(SectorRetrySubmitCommit{}, types.SubmitCommit),
	),

	types.FinalizeSector: planOne(
		on(SectorFinalized{}, types.Proving),
		on(SectorFinalizeFailed{}, types.FinalizeFailed),
	),

	// Snap deals
	types.SnapDealsWaitDeals: planOne(
		on(SectorAddPiece{}, types.SnapDealsAddPiece),
		on(SectorStartPacking{}, types.SnapDealsPacking),
		on(SectorAbortUpgrade{}, types.AbortUpgrade),
	),
	types.SnapDealsAddPiece: planOne(
		on(SectorPieceAdded{}, types.SnapDealsWaitDeals),
		apply(SectorStartPacking{}),
		apply(SectorAddPiece{}),
		on(SectorAddPieceFailed{}, types.SnapDealsAddPieceFailed),
		on(SectorAbortUpgrade{}, types.AbortUpgrade),
	),
	types.SnapDealsPacking: planOne(
		on(SectorPacked{}, types.UpdateReplica),
		on(SectorAbortUpgrade{}, types.AbortUpgrade),
	),
	types.UpdateReplica: planOne(
		on(SectorReplicaUpdate{}, types.ProveReplicaUpdate),
		on(SectorUpdateReplicaFailed{}, types.ReplicaUpdateFailed),
		on(SectorDealsExpired{}, types.SnapDealsDealsExpired),
		on(SectorInvalidDealIDs{}, types.SnapDealsRecoverDealIDs),
		on(SectorAbortUpgrade{}, types.AbortUpgrade),
	),
	types.ProveReplicaUpdate: planOne(
		on(SectorProveReplicaUpdate{}, types.SubmitReplicaUpdate),
		on(SectorProveReplicaUpdateFailed{}, types.ReplicaUpdateFailed),
		on(SectorDealsExpired{}, types.SnapDealsDealsExpired),
		on(SectorInvalidDealIDs{}, types.SnapDealsRecoverDealIDs),
		on(SectorAbortUpgrade{}, types.AbortUpgrade),
	),
	types.SubmitReplicaUpdate: planOne(
		on(SectorReplicaUpdateSubmitted{}, types.ReplicaUpdateWait),
		on(SectorSubmitReplicaUpdateFailed{}, types.ReplicaUpdateFailed),
	),
	types.ReplicaUpdateWait: planOne(
		on(SectorReplicaUpdateLanded{}, types.FinalizeReplicaUpdate),
		on(SectorSubmitReplicaUpdateFailed{}, types.ReplicaUpdateFailed),
		on(SectorAbortUpgrade{}, types.AbortUpgrade),
	),
	types.FinalizeReplicaUpdate: planOne(
		on(SectorFinalized{}, types.Proving),
	),
	types.UpdateActivating: planOne(
		on(SectorUpdateActive{}, types.ReleaseSectorKey),
	),
	types.ReleaseSectorKey: planOne(
		on(SectorKeyReleased{}, types.Proving),
		on(SectorReleaseKeyFailed{}, types.ReleaseSectorKeyFailed),
	),

	// Sealing errors
	types.AddPieceFailed: planOne(
		on(SectorRetryWaitDeals{}, types.WaitDeals),
		apply(SectorStartPacking{}),
		apply(SectorAddPiece{}),
	),
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
	types.CommitFinalizeFailed: planOne(
		on(SectorRetryFinalize{}, types.CommitFinalize),
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

	// Snap Deals Errors
	types.SnapDealsAddPieceFailed: planOne(
		on(SectorRetryWaitDeals{}, types.SnapDealsWaitDeals),
		apply(SectorStartPacking{}),
		apply(SectorAddPiece{}),
		on(SectorAbortUpgrade{}, types.AbortUpgrade),
	),
	types.SnapDealsDealsExpired: planOne(
		on(SectorAbortUpgrade{}, types.AbortUpgrade),
	),
	types.SnapDealsRecoverDealIDs: planOne(
		on(SectorUpdateDealIDs{}, types.SubmitReplicaUpdate),
		on(SectorAbortUpgrade{}, types.AbortUpgrade),
	),
	types.AbortUpgrade: planOneOrIgnore(
		on(SectorRevertUpgradeToProving{}, types.Proving),
	),
	types.ReplicaUpdateFailed: planOne(
		on(SectorRetrySubmitReplicaUpdateWait{}, types.ReplicaUpdateWait),
		on(SectorRetrySubmitReplicaUpdate{}, types.SubmitReplicaUpdate),
		on(SectorRetryReplicaUpdate{}, types.UpdateReplica),
		on(SectorRetryProveReplicaUpdate{}, types.ProveReplicaUpdate),
		on(SectorInvalidDealIDs{}, types.SnapDealsRecoverDealIDs),
		on(SectorDealsExpired{}, types.SnapDealsDealsExpired),
		on(SectorAbortUpgrade{}, types.AbortUpgrade),
	),
	types.ReleaseSectorKeyFailed: planOne(
		on(SectorUpdateActive{}, types.ReleaseSectorKey),
	),

	// Post-seal

	types.Proving: planOne(
		on(SectorFaultReported{}, types.FaultReported),
		on(SectorFaulty{}, types.Faulty),
		on(SectorStartCCUpdate{}, types.SnapDealsWaitDeals),
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
	types.Removing: planOneOrIgnore(
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

func (m *Sealing) logEvents(events []statemachine.Event, state *types.SectorInfo) error {
	for _, event := range events {
		log.Debugw("sector event", "sector", state.SectorNumber, "type", fmt.Sprintf("%T", event.User), "event", event.User)

		e, err := json.Marshal(event)
		if err != nil {
			log.Errorf("marshaling event for logging: %+v", err)
			continue
		}

		if event.User == (SectorRestart{}) {
			continue // don't log on every fsm restart
		}

		if len(e) > 8000 {
			e = []byte(string(e[:8000]) + "... truncated")
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
			return xerrors.Errorf("get log count error %s, check db connection", err)
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
				return xerrors.Errorf("fail to truncate log %w", err)
			}
		}
		err = m.logService.Append(&l)
		if err != nil {
			return xerrors.Errorf("fail to append log %w", err)
		}
	}

	return nil
}

func (m *Sealing) plan(events []statemachine.Event, state *types.SectorInfo) (func(statemachine.Context, types.SectorInfo) error, uint64, error) {
	/////
	// First process all events

	if err := m.logEvents(events, state); err != nil {
		return nil, 0, err
	}

	if m.notifee != nil {
		defer func(before types.SectorInfo) {
			m.notifee(before, *state)
		}(*state) // take safe-ish copy of the before state (except for nested pointers)
	}

	p := fsmPlanners[state.State]
	if p == nil {
		if len(events) == 1 {
			if _, ok := events[0].User.(globalMutator); ok {
				p = planOne() // in case we're in a really weird state, allow restart / update state / remove
			}
		}

		if p == nil {
			return nil, 0, xerrors.Errorf("planner for state %s not found", state.State)
		}
	}

	processed, err := p(events, state)
	if err != nil {
		return nil, 0, xerrors.Errorf("running planner for state %s failed: %w", state.State, err)
	}

	/////
	// Now decide what to do next

	/*

				      UndefinedSectorState (start)
				       v                     |
				*<- WaitDeals <-> AddPiece   |
				|   |   /--------------------/
				|   v   v
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

	*/

	if err := m.onUpdateSector(context.TODO(), state); err != nil {
		log.Errorw("update sector stats", "error", err)
	}

	switch state.State {
	// Happy path
	case types.Empty:
		fallthrough
	case types.WaitDeals:
		return m.handleWaitDeals, processed, nil
	case types.AddPiece:
		return m.handleAddPiece, processed, nil
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
	case types.SubmitPreCommitBatch:
		return m.handleSubmitPreCommitBatch, processed, nil
	case types.PreCommitBatchWait:
		fallthrough
	case types.PreCommitWait:
		return m.handlePreCommitWait, processed, nil
	case types.WaitSeed:
		return m.handleWaitSeed, processed, nil
	case types.Committing:
		return m.handleCommitting, processed, nil
	case types.SubmitCommit:
		return m.handleSubmitCommit, processed, nil
	case types.SubmitCommitAggregate:
		return m.handleSubmitCommitAggregate, processed, nil
	case types.CommitAggregateWait:
		fallthrough
	case types.CommitWait:
		return m.handleCommitWait, processed, nil
	case types.CommitFinalize:
		fallthrough
	case types.FinalizeSector:
		return m.handleFinalizeSector, processed, nil

	// Snap deals updates
	case types.SnapDealsWaitDeals:
		return m.handleWaitDeals, processed, nil
	case types.SnapDealsAddPiece:
		return m.handleAddPiece, processed, nil
	case types.SnapDealsPacking:
		return m.handlePacking, processed, nil
	case types.UpdateReplica:
		return m.handleReplicaUpdate, processed, nil
	case types.ProveReplicaUpdate:
		return m.handleProveReplicaUpdate, processed, nil
	case types.SubmitReplicaUpdate:
		return m.handleSubmitReplicaUpdate, processed, nil
	case types.ReplicaUpdateWait:
		return m.handleReplicaUpdateWait, processed, nil
	case types.FinalizeReplicaUpdate:
		return m.handleFinalizeReplicaUpdate, processed, nil
	case types.UpdateActivating:
		return m.handleUpdateActivating, processed, nil
	case types.ReleaseSectorKey:
		return m.handleReleaseSectorKey, processed, nil

	// Handled failure modes
	case types.AddPieceFailed:
		return m.handleAddPieceFailed, processed, nil
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
	case types.CommitFinalizeFailed:
		fallthrough
	case types.FinalizeFailed:
		return m.handleFinalizeFailed, processed, nil
	case types.PackingFailed: // DEPRECATED: remove this for the next reset
		state.State = types.DealsExpired
		fallthrough
	case types.DealsExpired:
		return m.handleDealsExpired, processed, nil
	case types.RecoverDealIDs:
		return m.HandleRecoverDealIDs, processed, nil

	// Snap Deals failure modes
	case types.SnapDealsAddPieceFailed:
		return m.handleAddPieceFailed, processed, nil

	case types.SnapDealsDealsExpired:
		return m.handleDealsExpiredSnapDeals, processed, nil
	case types.SnapDealsRecoverDealIDs:
		return m.handleSnapDealsRecoverDealIDs, processed, nil
	case types.ReplicaUpdateFailed:
		return m.handleSubmitReplicaUpdateFailed, processed, nil
	case types.ReleaseSectorKeyFailed:
		return m.handleReleaseSectorKeyFailed, 0, err
	case types.AbortUpgrade:
		return m.handleAbortUpgrade, processed, nil

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

func (m *Sealing) onUpdateSector(ctx context.Context, state *types.SectorInfo) error {
	if m.getConfig == nil {
		return nil // tests
	}

	cfg, err := m.getConfig()
	if err != nil {
		return xerrors.Errorf("getting config: %w", err)
	}

	shouldUpdateInput := m.stats.UpdateSector(cfg, m.minerSectorID(state.SectorNumber), state.State)

	// trigger more input processing when we've dipped below max sealing limits
	if shouldUpdateInput {
		sp, err := m.currentSealProof(ctx)
		if err != nil {
			return xerrors.Errorf("getting seal proof type: %w", err)
		}

		go func() {
			m.inputLk.Lock()
			defer m.inputLk.Unlock()

			if err := m.updateInput(ctx, sp); err != nil {
				log.Errorf("%+v", err)
			}
		}()
	}

	return nil
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
		case SectorProofReady: // early finalize
			e.apply(state)
			state.State = types.CommitFinalize
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
	defer m.startupWait.Done()

	trackedSectors, err := m.ListSectors()
	if err != nil {
		log.Errorf("loading sector list: %+v", err)
	}

	for _, sector := range trackedSectors {
		if err := m.sectors.Send(uint64(sector.SectorNumber), SectorRestart{}); err != nil {
			log.Errorf("restarting sector %d: %+v", sector.SectorNumber, err)
		}
	}

	// TODO: Grab on-chain sector set and diff with trackedSectors

	return nil
}

func (m *Sealing) ForceSectorState(ctx context.Context, id abi.SectorNumber, state types.SectorState) error {
	m.startupWait.Wait()
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

func on(mut mutator, next types.SectorState) func() (mutator, func(*types.SectorInfo) (bool, error)) {
	return func() (mutator, func(*types.SectorInfo) (bool, error)) {
		return mut, func(state *types.SectorInfo) (bool, error) {
			state.State = next
			return false, nil
		}
	}
}

// like `on`, but doesn't change state
func apply(mut mutator) func() (mutator, func(*types.SectorInfo) (bool, error)) {
	return func() (mutator, func(*types.SectorInfo) (bool, error)) {
		return mut, func(state *types.SectorInfo) (bool, error) {
			return true, nil
		}
	}
}

func onReturning(mut mutator) func() (mutator, func(*types.SectorInfo) (bool, error)) {
	return func() (mutator, func(*types.SectorInfo) (bool, error)) {
		return mut, func(state *types.SectorInfo) (bool, error) {
			if state.Return == "" {
				return false, xerrors.Errorf("return state not set")
			}

			state.State = types.SectorState(state.Return)
			state.Return = ""
			return false, nil
		}
	}
}

func planOne(ts ...func() (mut mutator, next func(*types.SectorInfo) (more bool, err error))) func(events []statemachine.Event, state *types.SectorInfo) (uint64, error) {
	return func(events []statemachine.Event, state *types.SectorInfo) (uint64, error) {
	eloop:
		for i, event := range events {
			if gm, ok := event.User.(globalMutator); ok {
				gm.applyGlobal(state)
				return uint64(i + 1), nil
			}

			for _, t := range ts {
				mut, next := t()

				if reflect.TypeOf(event.User) != reflect.TypeOf(mut) {
					continue
				}

				if err, iserr := event.User.(error); iserr {
					log.Warnf("sector %d got error event %T: %+v", state.SectorNumber, event.User, err)
				}

				event.User.(mutator).apply(state)
				more, err := next(state)
				if err != nil || !more {
					return uint64(i + 1), err
				}

				continue eloop
			}

			_, ok := event.User.(Ignorable)
			if ok {
				continue
			}

			return uint64(i + 1), xerrors.Errorf("planner for state %s received unexpected event %T (%+v)", state.State, event.User, event)
		}

		return uint64(len(events)), nil
	}
}

// planOne but ignores unhandled states without erroring, this prevents the need to handle all possible events creating
// error during forced override
func planOneOrIgnore(ts ...func() (mut mutator, next func(*types.SectorInfo) (more bool, err error))) func(events []statemachine.Event, state *types.SectorInfo) (uint64, error) {
	f := planOne(ts...)
	return func(events []statemachine.Event, state *types.SectorInfo) (uint64, error) {
		cnt, err := f(events, state)
		if err != nil {
			log.Warnf("planOneOrIgnore: ignoring error from planOne: %s", err)
		}
		return cnt, nil
	}
}
