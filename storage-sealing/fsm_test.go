package sealing

import (

	"testing"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	logging "github.com/ipfs/go-log/v2"
	"github.com/stretchr/testify/require"

	"github.com/filecoin-project/go-statemachine"

	"github.com/filecoin-project/venus-sealer/types"
)

func init() {
	_ = logging.SetLogLevel("*", "INFO")
}

func (t *test) planSingle(evt interface{}) {
	_, _, err := t.s.plan([]statemachine.Event{{User: evt}}, t.state)
	require.NoError(t.t, err)
}

type test struct {
	s     *Sealing
	t     *testing.T
	state *types.SectorInfo
}

func TestHappyPath(t *testing.T) {
	var notif []struct{ before, after types.SectorInfo }
	ma, _ := address.NewIDAddress(55151)
	m := test{
		s: &Sealing{
			maddr: ma,
			stats: types.SectorStats{
				BySector: map[abi.SectorID]types.StatSectorState{},
			},
			notifee: func(before, after types.SectorInfo) {
				notif = append(notif, struct{ before, after types.SectorInfo }{before, after})
			},
		},
		t:     t,
		state: &types.SectorInfo{State: types.Packing},
	}

	m.planSingle(SectorPacked{})
	require.Equal(m.t, m.state.State, types.GetTicket)

	m.planSingle(SectorTicket{})
	require.Equal(m.t, m.state.State, types.PreCommit1)

	m.planSingle(SectorPreCommit1{})
	require.Equal(m.t, m.state.State, types.PreCommit2)

	m.planSingle(SectorPreCommit2{})
	require.Equal(m.t, m.state.State, types.PreCommitting)

	m.planSingle(SectorPreCommitted{})
	require.Equal(m.t, m.state.State, types.PreCommitWait)

	m.planSingle(SectorPreCommitLanded{})
	require.Equal(m.t, m.state.State, types.WaitSeed)

	m.planSingle(SectorSeedReady{})
	require.Equal(m.t, m.state.State, types.Committing)

	m.planSingle(SectorCommitted{})
	require.Equal(m.t, m.state.State, types.SubmitCommit)

	m.planSingle(SectorCommitSubmitted{})
	require.Equal(m.t, m.state.State, types.CommitWait)

	m.planSingle(SectorProving{})
	require.Equal(m.t, m.state.State, types.FinalizeSector)

	m.planSingle(SectorFinalized{})
	require.Equal(m.t, m.state.State, types.Proving)

	expected := []types.SectorState{types.Packing, types.GetTicket, types.PreCommit1, types.PreCommit2, types.PreCommitting, types.PreCommitWait, types.WaitSeed, types.Committing, types.SubmitCommit, types.CommitWait, types.FinalizeSector, types.Proving}
	for i, n := range notif {
		if n.before.State != expected[i] {
			t.Fatalf("expected before state: %s, got: %s", expected[i], n.before.State)
		}
		if n.after.State != expected[i+1] {
			t.Fatalf("expected after state: %s, got: %s", expected[i+1], n.after.State)
		}
	}
}

func TestHappyPathFinalizeEarly(t *testing.T) {
	var notif []struct{ before, after types.SectorInfo }
	ma, _ := address.NewIDAddress(55151)
	m := test{
		s: &Sealing{
			maddr: ma,
			stats: types.SectorStats{
				BySector: map[abi.SectorID]types.StatSectorState{},
			},
			notifee: func(before, after types.SectorInfo) {
				notif = append(notif, struct{ before, after types.SectorInfo }{before, after})
			},
		},
		t:     t,
		state: &types.SectorInfo{State: types.Packing},
	}

	m.planSingle(SectorPacked{})
	require.Equal(m.t, m.state.State, types.GetTicket)

	m.planSingle(SectorTicket{})
	require.Equal(m.t, m.state.State, types.PreCommit1)

	m.planSingle(SectorPreCommit1{})
	require.Equal(m.t, m.state.State, types.PreCommit2)

	m.planSingle(SectorPreCommit2{})
	require.Equal(m.t, m.state.State, types.PreCommitting)

	m.planSingle(SectorPreCommitted{})
	require.Equal(m.t, m.state.State, types.PreCommitWait)

	m.planSingle(SectorPreCommitLanded{})
	require.Equal(m.t, m.state.State, types.WaitSeed)

	m.planSingle(SectorSeedReady{})
	require.Equal(m.t, m.state.State, types.Committing)

	m.planSingle(SectorProofReady{})
	require.Equal(m.t, m.state.State, types.CommitFinalize)

	m.planSingle(SectorFinalized{})
	require.Equal(m.t, m.state.State, types.SubmitCommit)

	m.planSingle(SectorSubmitCommitAggregate{})
	require.Equal(m.t, m.state.State, types.SubmitCommitAggregate)

	m.planSingle(SectorCommitAggregateSent{})
	require.Equal(m.t, m.state.State, types.CommitWait)

	m.planSingle(SectorProving{})
	require.Equal(m.t, m.state.State, types.FinalizeSector)

	m.planSingle(SectorFinalized{})
	require.Equal(m.t, m.state.State, types.Proving)

	expected := []types.SectorState{types.Packing, types.GetTicket, types.PreCommit1, types.PreCommit2, types.PreCommitting, types.PreCommitWait, types.WaitSeed, types.Committing, types.CommitFinalize, types.SubmitCommit, types.SubmitCommitAggregate, types.CommitWait, types.FinalizeSector, types.Proving}
	for i, n := range notif {
		if n.before.State != expected[i] {
			t.Fatalf("expected before state: %s, got: %s", expected[i], n.before.State)
		}
		if n.after.State != expected[i+1] {
			t.Fatalf("expected after state: %s, got: %s", expected[i+1], n.after.State)
		}
	}
}

func TestCommitFinalizeFailed(t *testing.T) {
	var notif []struct{ before, after types.SectorInfo }
	ma, _ := address.NewIDAddress(55151)
	m := test{
		s: &Sealing{
			maddr: ma,
			stats: types.SectorStats{
				BySector: map[abi.SectorID]types.StatSectorState{},
			},
			notifee: func(before, after types.SectorInfo) {
				notif = append(notif, struct{ before, after types.SectorInfo }{before, after})
			},
		},
		t:     t,
		state: &types.SectorInfo{State: types.Committing},
	}

	m.planSingle(SectorProofReady{})
	require.Equal(m.t, m.state.State, types.CommitFinalize)

	m.planSingle(SectorFinalizeFailed{})
	require.Equal(m.t, m.state.State, types.CommitFinalizeFailed)

	m.planSingle(SectorRetryFinalize{})
	require.Equal(m.t, m.state.State, types.CommitFinalize)

	m.planSingle(SectorFinalized{})
	require.Equal(m.t, m.state.State, types.SubmitCommit)

	expected := []types.SectorState{types.Committing, types.CommitFinalize, types.CommitFinalizeFailed, types.CommitFinalize, types.SubmitCommit}
	for i, n := range notif {
		if n.before.State != expected[i] {
			t.Fatalf("expected before state: %s, got: %s", expected[i], n.before.State)
		}
		if n.after.State != expected[i+1] {
			t.Fatalf("expected after state: %s, got: %s", expected[i+1], n.after.State)
		}
	}
}
func TestSeedRevert(t *testing.T) {
	ma, _ := address.NewIDAddress(55151)
	m := test{
		s: &Sealing{
			maddr: ma,
			stats: types.SectorStats{
				BySector: map[abi.SectorID]types.StatSectorState{},
			},
		},
		t:     t,
		state: &types.SectorInfo{State: types.Packing},
	}

	m.planSingle(SectorPacked{})
	require.Equal(m.t, m.state.State, types.GetTicket)

	m.planSingle(SectorTicket{})
	require.Equal(m.t, m.state.State, types.PreCommit1)

	m.planSingle(SectorPreCommit1{})
	require.Equal(m.t, m.state.State, types.PreCommit2)

	m.planSingle(SectorPreCommit2{})
	require.Equal(m.t, m.state.State, types.PreCommitting)

	m.planSingle(SectorPreCommitted{})
	require.Equal(m.t, m.state.State, types.PreCommitWait)

	m.planSingle(SectorPreCommitLanded{})
	require.Equal(m.t, m.state.State, types.WaitSeed)

	m.planSingle(SectorSeedReady{})
	require.Equal(m.t, m.state.State, types.Committing)

	_, _, err := m.s.plan([]statemachine.Event{{User: SectorSeedReady{SeedValue: nil, SeedEpoch: 5}}, {User: SectorCommitted{}}}, m.state)
	require.NoError(t, err)
	require.Equal(m.t, m.state.State, types.Committing)

	// not changing the seed this time
	_, _, err = m.s.plan([]statemachine.Event{{User: SectorSeedReady{SeedValue: nil, SeedEpoch: 5}}, {User: SectorCommitted{}}}, m.state)
	require.NoError(t, err)
	require.Equal(m.t, m.state.State, types.SubmitCommit)

	m.planSingle(SectorCommitSubmitted{})
	require.Equal(m.t, m.state.State, types.CommitWait)

	m.planSingle(SectorProving{})
	require.Equal(m.t, m.state.State, types.FinalizeSector)

	m.planSingle(SectorFinalized{})
	require.Equal(m.t, m.state.State, types.Proving)
}

func TestPlanCommittingHandlesSectorCommitFailed(t *testing.T) {
	ma, _ := address.NewIDAddress(55151)
	m := test{
		s: &Sealing{
			maddr: ma,
			stats: types.SectorStats{
				BySector: map[abi.SectorID]types.StatSectorState{},
			},
		},
		t:     t,
		state: &types.SectorInfo{State: types.Committing},
	}

	events := []statemachine.Event{{User: SectorCommitFailed{}}}

	_, err := planCommitting(events, m.state)
	require.NoError(t, err)

	require.Equal(t, types.CommitFailed, m.state.State)
}

func TestPlannerList(t *testing.T) {
	for state := range types.ExistSectorStateList {
		_, ok := fsmPlanners[state]
		require.True(t, ok, "state %s", state)
	}

	for state := range fsmPlanners {
		if state == types.UndefinedSectorState {
			continue
		}
		_, ok := types.ExistSectorStateList[state]
		require.True(t, ok, "state %s", state)
	}
}

func TestBrokenState(t *testing.T) {
	var notif []struct{ before, after types.SectorInfo }
	ma, _ := address.NewIDAddress(55151)
	m := test{
		s: &Sealing{
			maddr: ma,
			stats: types.SectorStats{
				BySector: map[abi.SectorID]types.StatSectorState{},
			},
			notifee: func(before, after types.SectorInfo) {
				notif = append(notif, struct{ before, after types.SectorInfo }{before, after})
			},
		},
		t:     t,
		state: &types.SectorInfo{State: "not a state"},
	}

	_, _, err := m.s.plan([]statemachine.Event{{User: SectorPacked{}}}, m.state)
	require.Error(t, err)
	require.Equal(m.t, m.state.State, types.SectorState("not a state"))

	m.planSingle(SectorRemove{})
	require.Equal(m.t, m.state.State, types.Removing)

	expected := []types.SectorState{"not a state", "not a state", types.Removing}
	for i, n := range notif {
		if n.before.State != expected[i] {
			t.Fatalf("expected before state: %s, got: %s", expected[i], n.before.State)
		}
		if n.after.State != expected[i+1] {
			t.Fatalf("expected after state: %s, got: %s", expected[i+1], n.after.State)
		}
	}
}

func TestTicketExpired(t *testing.T) {
	var notif []struct{ before, after types.SectorInfo }
	ma, _ := address.NewIDAddress(55151)
	m := test{
		s: &Sealing{
			maddr: ma,
			stats: types.SectorStats{
				BySector: map[abi.SectorID]types.StatSectorState{},
			},
			notifee: func(before, after types.SectorInfo) {
				notif = append(notif, struct{ before, after types.SectorInfo }{before, after})
			},
		},
		t:     t,
		state: &types.SectorInfo{State: types.Packing},
	}

	m.planSingle(SectorPacked{})
	require.Equal(m.t, m.state.State, types.GetTicket)

	m.planSingle(SectorTicket{})
	require.Equal(m.t, m.state.State, types.PreCommit1)

	expired := checkTicketExpired(0, types.MaxTicketAge+1)
	require.True(t, expired)

	m.planSingle(SectorOldTicket{})
	require.Equal(m.t, m.state.State, types.GetTicket)

	expected := []types.SectorState{types.Packing, types.GetTicket, types.PreCommit1, types.GetTicket}
	for i, n := range notif {
		if n.before.State != expected[i] {
			t.Fatalf("expected before state: %s, got: %s", expected[i], n.before.State)
		}
		if n.after.State != expected[i+1] {
			t.Fatalf("expected after state: %s, got: %s", expected[i+1], n.after.State)
		}
	}
}
