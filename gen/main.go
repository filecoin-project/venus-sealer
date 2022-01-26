package main

import (
	"fmt"
	"os"

	"github.com/filecoin-project/venus-sealer/types"

	gen "github.com/whyrusleeping/cbor-gen"
)

func main() {
	err := gen.WriteMapEncodersToFile("./types/cbor_gen.go", "types",
		types.Call{},
		types.CallID{},
		types.WorkState{},
		types.WorkID{},
		types.SectorInfo{},
		types.PieceDealInfo{},
		types.Piece{},
		types.DealSchedule{},
		types.Log{},
	)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
