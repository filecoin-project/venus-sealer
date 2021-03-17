package main

import (
	"fmt"
	"github.com/filecoin-project/venus-sealer/api"

	"github.com/urfave/cli/v2"
)

var versionCmd = &cli.Command{
	Name:  "version",
	Usage: "Print version",
	Action: func(cctx *cli.Context) error {
		nodeAPI, closer, err := api.GetAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		ctx := api.ReqContext(cctx)
		// TODO: print more useful things

		v, err := nodeAPI.Version(ctx)
		if err != nil {
			return err
		}
		fmt.Println("Daemon: ", v)

		fmt.Print("Local: ")
		cli.VersionPrinter(cctx)
		return nil
	},
}
