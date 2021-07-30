package main

import (
	"fmt"
	"github.com/filecoin-project/venus-sealer/api"
	_ "net/http/pprof"

	"github.com/urfave/cli/v2"
)

var tokenCmd = &cli.Command{
	Name:  "token",
	Usage: "Print venus sealer token",
	Action: func(cctx *cli.Context) error {
		storageAPI, closer, err := api.GetStorageMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		token, err := storageAPI.Token(api.ReqContext(cctx))
		if err != nil {
			return err
		}
		fmt.Println("token: ", string(token))
		return nil
	},
}