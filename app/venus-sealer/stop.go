package main

import (
	"github.com/filecoin-project/venus-sealer/api"
	_ "net/http/pprof"

	"github.com/urfave/cli/v2"
)

var stopCmd = &cli.Command{
	Name:  "stop",
	Usage: "Stop a running venus sealer",
	Flags: []cli.Flag{},
	Action: func(cctx *cli.Context) error {
		storageAPI, closer, err := api.GetStorageMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		err = storageAPI.Shutdown(api.ReqContext(cctx))
		if err != nil {
			return err
		}

		return nil
	},
}
