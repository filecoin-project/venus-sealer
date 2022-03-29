package sealing

import (
	"time"

	"github.com/ipfs/go-cid"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/specs-storage/storage"

	"github.com/filecoin-project/venus-sealer/types"

	"github.com/filecoin-project/venus/venus-shared/actors/builtin/miner"
)

type mutator interface {
	apply(state *types.SectorInfo)
}

// globalMutator is an event which can apply in every state
type globalMutator interface {
	// applyGlobal applies the event to the state. If if returns true,
	//  event processing should be interrupted
	applyGlobal(state *types.SectorInfo) bool
}

type Ignorable interface {
	Ignore()
}

// Global events

type SectorRestart struct{}

func (evt SectorRestart) applyGlobal(*types.SectorInfo) bool { return false }

type SectorFatalError struct{ error }

func (evt SectorFatalError) FormatError(xerrors.Printer) (next error) { return evt.error }

func (evt SectorFatalError) applyGlobal(state *types.SectorInfo) bool {
	log.Errorf("Fatal error on sector %d: %+v", state.SectorNumber, evt.error)
	// TODO: Do we want to mark the state as unrecoverable?
	//  I feel like this should be a softer error, where the user would
	//  be able to send a retry event of some kind
	return true
}

type SectorForceState struct {
	State types.SectorState
}

func (evt SectorForceState) applyGlobal(state *types.SectorInfo) bool {
	state.State = evt.State
	return true
}

// Normal path

type SectorStart struct {
	ID         abi.SectorNumber
	SectorType abi.RegisteredSealProof
}

func (evt SectorStart) apply(state *types.SectorInfo) {
	state.SectorNumber = evt.ID
	state.SectorType = evt.SectorType
}

type SectorStartCC struct {
	ID         abi.SectorNumber
	SectorType abi.RegisteredSealProof
}

func (evt SectorStartCC) apply(state *types.SectorInfo) {
	state.SectorNumber = evt.ID
	state.SectorType = evt.SectorType
}

type SectorAddPiece struct{}

func (evt SectorAddPiece) apply(state *types.SectorInfo) {
	if state.CreationTime == 0 {
		state.CreationTime = time.Now().Unix()
	}
}

type SectorPieceAdded struct {
	NewPieces []types.Piece
}

func (evt SectorPieceAdded) apply(state *types.SectorInfo) {
	state.Pieces = append(state.Pieces, evt.NewPieces...)
}

type SectorAddPieceFailed struct{ error }

func (evt SectorAddPieceFailed) FormatError(xerrors.Printer) (next error) { return evt.error }
func (evt SectorAddPieceFailed) apply(si *types.SectorInfo)               {}

type SectorRetryWaitDeals struct{}

func (evt SectorRetryWaitDeals) apply(si *types.SectorInfo) {}

type SectorStartPacking struct{}

func (evt SectorStartPacking) apply(*types.SectorInfo) {}

func (evt SectorStartPacking) Ignore() {}

type SectorPacked struct{ FillerPieces []abi.PieceInfo }

func (evt SectorPacked) apply(state *types.SectorInfo) {
	for idx := range evt.FillerPieces {
		state.Pieces = append(state.Pieces, types.Piece{
			Piece:    evt.FillerPieces[idx],
			DealInfo: nil, // filler pieces don't have deals associated with them
		})
	}
}

type SectorTicket struct {
	TicketValue abi.SealRandomness
	TicketEpoch abi.ChainEpoch
}

func (evt SectorTicket) apply(state *types.SectorInfo) {
	state.TicketEpoch = evt.TicketEpoch
	state.TicketValue = evt.TicketValue
}

type SectorOldTicket struct{}

func (evt SectorOldTicket) apply(*types.SectorInfo) {}

type SectorPreCommit1 struct {
	PreCommit1Out storage.PreCommit1Out
}

func (evt SectorPreCommit1) apply(state *types.SectorInfo) {
	state.PreCommit1Out = evt.PreCommit1Out
	state.PreCommit2Fails = 0
}

type SectorPreCommit2 struct {
	Sealed   cid.Cid
	Unsealed cid.Cid
}

func (evt SectorPreCommit2) apply(state *types.SectorInfo) {
	commd := evt.Unsealed
	state.CommD = &commd
	commr := evt.Sealed
	state.CommR = &commr
}

type SectorPreCommitBatch struct{}

func (evt SectorPreCommitBatch) apply(*types.SectorInfo) {}

type SectorPreCommitBatchSent struct {
	Message string
}

func (evt SectorPreCommitBatchSent) apply(state *types.SectorInfo) {
	state.PreCommitMessage = evt.Message
}

type SectorPreCommitLanded struct {
	TipSet types.TipSetToken
}

func (evt SectorPreCommitLanded) apply(si *types.SectorInfo) {
	si.PreCommitTipSet = evt.TipSet
}

type SectorSealPreCommit1Failed struct{ error }

func (evt SectorSealPreCommit1Failed) FormatError(xerrors.Printer) (next error) { return evt.error }
func (evt SectorSealPreCommit1Failed) apply(si *types.SectorInfo) {
	si.InvalidProofs = 0 // reset counter
	si.PreCommit2Fails = 0
}

type SectorSealPreCommit2Failed struct{ error }

func (evt SectorSealPreCommit2Failed) FormatError(xerrors.Printer) (next error) { return evt.error }
func (evt SectorSealPreCommit2Failed) apply(si *types.SectorInfo) {
	si.InvalidProofs = 0 // reset counter
	si.PreCommit2Fails++
}

type SectorChainPreCommitFailed struct{ error }

func (evt SectorChainPreCommitFailed) FormatError(xerrors.Printer) (next error) { return evt.error }
func (evt SectorChainPreCommitFailed) apply(*types.SectorInfo)                  {}

type SectorPreCommitted struct {
	Message          string
	PreCommitDeposit big.Int
	PreCommitInfo    miner.SectorPreCommitInfo
}

func (evt SectorPreCommitted) apply(state *types.SectorInfo) {
	state.PreCommitMessage = evt.Message
	state.PreCommitDeposit = evt.PreCommitDeposit
	state.PreCommitInfo = &evt.PreCommitInfo
}

type SectorSeedReady struct {
	SeedValue abi.InteractiveSealRandomness
	SeedEpoch abi.ChainEpoch
}

func (evt SectorSeedReady) apply(state *types.SectorInfo) {
	state.SeedEpoch = evt.SeedEpoch
	state.SeedValue = evt.SeedValue
}

type SectorComputeProofFailed struct{ error }

func (evt SectorComputeProofFailed) FormatError(xerrors.Printer) (next error) { return evt.error }
func (evt SectorComputeProofFailed) apply(*types.SectorInfo)                  {}

type SectorCommitFailed struct{ error }

func (evt SectorCommitFailed) FormatError(xerrors.Printer) (next error) { return evt.error }
func (evt SectorCommitFailed) apply(*types.SectorInfo)                  {}

type SectorRetrySubmitCommit struct{}

func (evt SectorRetrySubmitCommit) apply(*types.SectorInfo) {}

type SectorDealsExpired struct{ error }

func (evt SectorDealsExpired) FormatError(xerrors.Printer) (next error) { return evt.error }
func (evt SectorDealsExpired) apply(*types.SectorInfo)                  {}

type SectorTicketExpired struct{ error }

func (evt SectorTicketExpired) FormatError(xerrors.Printer) (next error) { return evt.error }
func (evt SectorTicketExpired) apply(*types.SectorInfo)                  {}

type SectorCommitted struct {
	Proof []byte
}

func (evt SectorCommitted) apply(state *types.SectorInfo) {
	state.Proof = evt.Proof
}

// like SectorCommitted, but finalizes before sending the proof to the chain
type SectorProofReady struct {
	Proof []byte
}

func (evt SectorProofReady) apply(state *types.SectorInfo) {
	state.Proof = evt.Proof
}

type SectorSubmitCommitAggregate struct{}

func (evt SectorSubmitCommitAggregate) apply(*types.SectorInfo) {}

type SectorCommitSubmitted struct {
	Message string
}

func (evt SectorCommitSubmitted) apply(state *types.SectorInfo) {
	state.CommitMessage = evt.Message
}

type SectorCommitAggregateSent struct {
	Message string
}

func (evt SectorCommitAggregateSent) apply(state *types.SectorInfo) {
	state.CommitMessage = evt.Message
}

type SectorProving struct{}

func (evt SectorProving) apply(*types.SectorInfo) {}

type SectorFinalized struct{}

func (evt SectorFinalized) apply(*types.SectorInfo) {}

type SectorFinalizedAvailable struct{}

func (evt SectorFinalizedAvailable) apply(*types.SectorInfo) {}

type SectorRetryFinalize struct{}

func (evt SectorRetryFinalize) apply(*types.SectorInfo) {}

type SectorFinalizeFailed struct{ error }

func (evt SectorFinalizeFailed) FormatError(xerrors.Printer) (next error) { return evt.error }
func (evt SectorFinalizeFailed) apply(*types.SectorInfo)                  {}

// Snap deals // CC update path

type SectorMarkForUpdate struct{}

func (evt SectorMarkForUpdate) apply(state *types.SectorInfo) {}

type SectorStartCCUpdate struct{}

func (evt SectorStartCCUpdate) apply(state *types.SectorInfo) {
	state.CCUpdate = true
	// Clear filler piece but remember in case of abort
	state.CCPieces = state.Pieces
	state.Pieces = nil
}

type SectorReplicaUpdate struct {
	Out storage.ReplicaUpdateOut
}

func (evt SectorReplicaUpdate) apply(state *types.SectorInfo) {
	state.UpdateSealed = &evt.Out.NewSealed
	state.UpdateUnsealed = &evt.Out.NewUnsealed
}

type SectorProveReplicaUpdate struct {
	Proof storage.ReplicaUpdateProof
}

func (evt SectorProveReplicaUpdate) apply(state *types.SectorInfo) {
	state.ReplicaUpdateProof = evt.Proof
}

type SectorReplicaUpdateSubmitted struct {
	Message string
}

func (evt SectorReplicaUpdateSubmitted) apply(state *types.SectorInfo) {
	state.ReplicaUpdateMessage = evt.Message
}

type SectorReplicaUpdateLanded struct{}

func (evt SectorReplicaUpdateLanded) apply(state *types.SectorInfo) {}

type SectorUpdateActive struct{}

func (evt SectorUpdateActive) apply(state *types.SectorInfo) {}

type SectorKeyReleased struct{}

func (evt SectorKeyReleased) apply(state *types.SectorInfo) {}

// Failed state recovery

type SectorRetrySealPreCommit1 struct{}

func (evt SectorRetrySealPreCommit1) apply(state *types.SectorInfo) {}

type SectorRetrySealPreCommit2 struct{}

func (evt SectorRetrySealPreCommit2) apply(state *types.SectorInfo) {}

type SectorRetryPreCommit struct{}

func (evt SectorRetryPreCommit) apply(state *types.SectorInfo) {}

type SectorRetryWaitSeed struct{}

func (evt SectorRetryWaitSeed) apply(state *types.SectorInfo) {}

type SectorRetryPreCommitWait struct{}

func (evt SectorRetryPreCommitWait) apply(state *types.SectorInfo) {}

type SectorRetryComputeProof struct{}

func (evt SectorRetryComputeProof) apply(state *types.SectorInfo) {
	state.InvalidProofs++
}

type SectorRetryInvalidProof struct{}

func (evt SectorRetryInvalidProof) apply(state *types.SectorInfo) {
	state.InvalidProofs++
}

type SectorRetryCommitWait struct{}

func (evt SectorRetryCommitWait) apply(state *types.SectorInfo) {}

type SectorInvalidDealIDs struct {
	Return types.ReturnState
}

func (evt SectorInvalidDealIDs) apply(state *types.SectorInfo) {
	state.Return = evt.Return
}

type SectorUpdateDealIDs struct {
	Updates map[int]abi.DealID
}

func (evt SectorUpdateDealIDs) apply(state *types.SectorInfo) {
	for i, id := range evt.Updates {
		state.Pieces[i].DealInfo.DealID = id
	}
}

// Snap Deals failure and recovery

type SectorRetryReplicaUpdate struct{}

func (evt SectorRetryReplicaUpdate) apply(state *types.SectorInfo) {}

type SectorRetryProveReplicaUpdate struct{}

func (evt SectorRetryProveReplicaUpdate) apply(state *types.SectorInfo) {}

type SectorUpdateReplicaFailed struct{ error }

func (evt SectorUpdateReplicaFailed) FormatError(xerrors.Printer) (next error) { return evt.error }
func (evt SectorUpdateReplicaFailed) apply(state *types.SectorInfo)            {}

type SectorProveReplicaUpdateFailed struct{ error }

func (evt SectorProveReplicaUpdateFailed) FormatError(xerrors.Printer) (next error) {
	return evt.error
}
func (evt SectorProveReplicaUpdateFailed) apply(state *types.SectorInfo) {}

type SectorAbortUpgrade struct{ error }

func (evt SectorAbortUpgrade) apply(state *types.SectorInfo) {}
func (evt SectorAbortUpgrade) FormatError(xerrors.Printer) (next error) {
	return evt.error
}

type SectorRevertUpgradeToProving struct{}

func (evt SectorRevertUpgradeToProving) apply(state *types.SectorInfo) {
	// cleanup sector state so that it is back in proving
	state.CCUpdate = false
	state.UpdateSealed = nil
	state.UpdateUnsealed = nil
	state.ReplicaUpdateProof = nil
	state.ReplicaUpdateMessage = ""
	state.Pieces = state.CCPieces
	state.CCPieces = nil
}

type SectorRetrySubmitReplicaUpdateWait struct{}

func (evt SectorRetrySubmitReplicaUpdateWait) apply(state *types.SectorInfo) {}

type SectorRetrySubmitReplicaUpdate struct{}

func (evt SectorRetrySubmitReplicaUpdate) apply(state *types.SectorInfo) {}

type SectorSubmitReplicaUpdateFailed struct{}

func (evt SectorSubmitReplicaUpdateFailed) apply(state *types.SectorInfo) {}

type SectorReleaseKeyFailed struct{ error }

func (evt SectorReleaseKeyFailed) FormatError(xerrors.Printer) (next error) {
	return evt.error
}

func (evt SectorReleaseKeyFailed) apply(state *types.SectorInfo) {}

// Faults

type SectorFaulty struct{}

func (evt SectorFaulty) apply(state *types.SectorInfo) {}

type SectorFaultReported struct{ reportMsg string }

func (evt SectorFaultReported) apply(state *types.SectorInfo) {
	state.FaultReportMsg = evt.reportMsg
}

type SectorFaultedFinal struct{}

// Terminating

type SectorTerminate struct{}

func (evt SectorTerminate) applyGlobal(state *types.SectorInfo) bool {
	state.State = types.Terminating
	return true
}

type SectorTerminating struct{ Message string }

func (evt SectorTerminating) apply(state *types.SectorInfo) {
	state.TerminateMessage = evt.Message
}

type SectorTerminated struct{ TerminatedAt abi.ChainEpoch }

func (evt SectorTerminated) apply(state *types.SectorInfo) {
	state.TerminatedAt = evt.TerminatedAt
}

type SectorTerminateFailed struct{ error }

func (evt SectorTerminateFailed) FormatError(xerrors.Printer) (next error) { return evt.error }
func (evt SectorTerminateFailed) apply(*types.SectorInfo)                  {}

// External events

type SectorRemove struct{}

func (evt SectorRemove) applyGlobal(state *types.SectorInfo) bool {
	state.State = types.Removing
	return true
}

type SectorRemoved struct{}

func (evt SectorRemoved) apply(state *types.SectorInfo) {}

type SectorRemoveFailed struct{ error }

func (evt SectorRemoveFailed) FormatError(xerrors.Printer) (next error) { return evt.error }
func (evt SectorRemoveFailed) apply(*types.SectorInfo)                  {}
