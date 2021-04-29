package main

import (
	"fmt"
	"github.com/filecoin-project/venus-sealer/types"
	"os"

	gen "github.com/whyrusleeping/cbor-gen"
)

func main() {
	err := gen.WriteMapEncodersToFile("./types/cbor_gen.go", "types",
		types.Call{},
		types.CallID{},
		types.WorkState{},
		types.WorkID{},
		types.SectorInfo{},
		types.Piece{},
		types.DealInfo{},
		types.DealSchedule{},
		types.Log{},
	)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
