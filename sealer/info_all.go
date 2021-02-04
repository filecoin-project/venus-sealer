package main

import (
	"flag"
	"fmt"
	lcli "github.com/filecoin-project/lotus/cli"
	"github.com/filecoin-project/venus-sealer/api"
	"sort"

	"github.com/urfave/cli/v2"
)

var _test = false

var infoAllCmd = &cli.Command{
	Name:  "all",
	Usage: "dump all related miner info",
	Action: func(cctx *cli.Context) error {
		storageAPI, closer, err := api.GetStorageMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		nodeAPI, acloser, err := api.GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer acloser()
		_ = nodeAPI

		ctx := api.ReqContext(cctx)

		// Top-level info

		fmt.Println("#: Version")
		if err := VersionCmd.Action(cctx); err != nil {
			return err
		}

		fmt.Println("\n#: Miner Info")
		if err := infoCmdAct(cctx); err != nil {
			return err
		}

		// Verbose info

		fmt.Println("\n#: Storage List")
		if err := storageListCmd.Action(cctx); err != nil {
			return err
		}

		fmt.Println("\n#: Worker List")
		if err := sealingWorkersCmd.Action(cctx); err != nil {
			return err
		}

		fmt.Println("\n#: Sealing Jobs")
		if err := sealingJobsCmd.Action(cctx); err != nil {
			return err
		}

		fmt.Println("\n#: Sched Diag")
		if err := sealingSchedDiagCmd.Action(cctx); err != nil {
			return err
		}

		fmt.Println("\n#: Sector List")
		if err := sectorsListCmd.Action(cctx); err != nil {
			return err
		}

		fmt.Println("\n#: Sector Refs")
		if err := sectorsRefsCmd.Action(cctx); err != nil {
			return err
		}

		// Very Very Verbose info
		fmt.Println("\n#: Per Sector Info")

		list, err := storageAPI.SectorsList(ctx)
		if err != nil {
			return err
		}

		sort.Slice(list, func(i, j int) bool {
			return list[i] < list[j]
		})

		for _, s := range list {
			fmt.Printf("\n##: Sector %d Status\n", s)

			fs := &flag.FlagSet{}
			for _, f := range sectorsStatusCmd.Flags {
				if err := f.Apply(fs); err != nil {
					return err
				}
			}
			if err := fs.Parse([]string{"--log", "--on-chain-info", fmt.Sprint(s)}); err != nil {
				return err
			}

			if err := sectorsStatusCmd.Action(cli.NewContext(cctx.App, fs, cctx)); err != nil {
				fmt.Println("ERROR: ", err)
			}

			fmt.Printf("\n##: Sector %d Storage Location\n", s)

			fs = &flag.FlagSet{}
			if err := fs.Parse([]string{fmt.Sprint(s)}); err != nil {
				return err
			}

			if err := storageFindCmd.Action(cli.NewContext(cctx.App, fs, cctx)); err != nil {
				fmt.Println("ERROR: ", err)
			}
		}

		if !_test {
			fmt.Println("\n#: Goroutines")
			if err := lcli.PprofGoroutines.Action(cctx); err != nil {
				return err
			}
		}

		return nil
	},
}
