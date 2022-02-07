package main

import (
	"fmt"
	"github.com/filecoin-project/venus-sealer/tool/convert-with-lotus/types"
	gen "github.com/whyrusleeping/cbor-gen"
	"os"
)

func main() {
	err := gen.WriteMapEncodersToFile("./types/cbor_gen.go", "types",
		types.Piece{},
		types.DealSchedule{},
		types.PieceDealInfo{},
		types.SectorInfo{},
		types.Log{},
	)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
