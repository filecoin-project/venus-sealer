package sectorstorage

import (
	"fmt"
	"testing"

	"github.com/filecoin-project/venus-sealer/types"
)

func TestRequestQueue(t *testing.T) {
	rq := &requestQueue{}

	rq.Push(&workerRequest{taskType: types.TTAddPiece})
	rq.Push(&workerRequest{taskType: types.TTPreCommit1})
	rq.Push(&workerRequest{taskType: types.TTPreCommit2})
	rq.Push(&workerRequest{taskType: types.TTPreCommit1})
	rq.Push(&workerRequest{taskType: types.TTAddPiece})

	dump := func(s string) {
		fmt.Println("---")
		fmt.Println(s)

		for sqi := 0; sqi < rq.Len(); sqi++ {
			task := (*rq)[sqi]
			fmt.Println(sqi, task.taskType)
		}
	}

	dump("start")

	pt := rq.Remove(0)

	dump("pop 1")

	if pt.taskType != types.TTPreCommit2 {
		t.Error("expected precommit2, got", pt.taskType)
	}

	pt = rq.Remove(0)

	dump("pop 2")

	if pt.taskType != types.TTPreCommit1 {
		t.Error("expected precommit1, got", pt.taskType)
	}

	pt = rq.Remove(1)

	dump("pop 3")

	if pt.taskType != types.TTAddPiece {
		t.Error("expected addpiece, got", pt.taskType)
	}

	pt = rq.Remove(0)

	dump("pop 4")

	if pt.taskType != types.TTPreCommit1 {
		t.Error("expected precommit1, got", pt.taskType)
	}
}
