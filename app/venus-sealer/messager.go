package main

import (
	"fmt"
	"github.com/filecoin-project/venus-sealer/api"
	"github.com/filecoin-project/venus-sealer/constants"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"
)

var messagerCmds = &cli.Command{
	Name:  "messager",
	Usage: "message cmds",
	Subcommands: []*cli.Command{
		waitMessagerCmds,
		searchMessagerCmds,
	},
}

var waitMessagerCmds = &cli.Command{
	Name:  "wait",
	Usage: "wait a messager msg uid for result",
	Action: func(cctx *cli.Context) error {
		storageAPI, closer, err := api.GetStorageMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		if cctx.NArg() == 0 {
			return xerrors.New("must has uuid argument")
		}

		uid := cctx.Args().Get(0)

		msg, err := storageAPI.MessagerWaitMessage(cctx.Context, uid, constants.MessageConfidence)
		if err != nil {
			return err
		}

		fmt.Println("message cid ", msg.Message)
		fmt.Println("Height:", msg.Height)
		fmt.Println("Tipset:", msg.TipSet.String())
		fmt.Println("exitcode:", msg.Receipt.ExitCode)
		fmt.Println("gas_used:", msg.Receipt.GasUsed)
		fmt.Println("return_value:", msg.Receipt.Return)
		return nil
	},
}

var searchMessagerCmds = &cli.Command{
	Name:  "search",
	Usage: "search a messager msg uid for result",
	Action: func(cctx *cli.Context) error {
		storageAPI, closer, err := api.GetStorageMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		if cctx.NArg() == 0 {
			return xerrors.New("must has uuid argument")
		}

		uid := cctx.Args().Get(0)

		msg, err := storageAPI.MessagerGetMessage(cctx.Context, uid)
		if err != nil {
			return err
		}

		fmt.Println("message cid ", msg.SignedCid)
		fmt.Println("Height:", msg.Height)
		fmt.Println("Confidence:", msg.Confidence)
		fmt.Println("Tipset:", msg.TipSetKey.String())
		fmt.Println("exitcode:", msg.Receipt.ExitCode)
		fmt.Println("gas_used:", msg.Receipt.GasUsed)
		fmt.Println("return_value:", msg.Receipt.Return)
		return nil
	},
}
