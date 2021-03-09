package sectorstorage

import (
	"github.com/filecoin-project/go-statestore"
	"github.com/filecoin-project/venus-sealer/types"
)

type workerCallTracker struct {
	st statestore.StateStore // by CallID
}

func (wt *workerCallTracker) onStart(ci types.CallID, rt types.ReturnType) error {
	return wt.st.Begin(ci, &types.Call{
		ID:      ci,
		RetType: rt,
		State:   types.CallStarted,
	})
}

func (wt *workerCallTracker) onDone(ci types.CallID, ret []byte) error {
	st := wt.st.Get(ci)
	return st.Mutate(func(cs *types.Call) error {
		cs.State = types.CallDone
		cs.Result = types.NewManyBytes(ret)
		return nil
	})
}

func (wt *workerCallTracker) onReturned(ci types.CallID) error {
	st := wt.st.Get(ci)
	return st.End()
}

func (wt *workerCallTracker) unfinished() ([]types.Call, error) {
	var out []types.Call
	return out, wt.st.List(&out)
}
