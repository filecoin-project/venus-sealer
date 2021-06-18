package main

import (
	"fmt"
	"os"

	gen "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/venus-sealer/types"
)

func main() {
	err := gen.WriteMapEncodersToFile("./cbor_gen.go", "sealing",
		types.Piece{},
		types.DealInfo{},
		types.DealSchedule{},
		types.SectorInfo{},
		types.Log{},
	)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
