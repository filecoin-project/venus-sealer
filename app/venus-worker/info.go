package main

import (
	"fmt"
	"github.com/filecoin-project/venus-sealer/api"
	types2 "github.com/filecoin-project/venus-sealer/types"
	"sort"

	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/venus/pkg/types"
)

var infoCmd = &cli.Command{
	Name:  "info",
	Usage: "Print worker info",
	Action: func(cctx *cli.Context) error {
		workerApi, closer, err := api.GetWorkerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		ctx := api.ReqContext(cctx)

		ver, err := workerApi.Version(ctx)
		if err != nil {
			return xerrors.Errorf("getting version: %w", err)
		}

		fmt.Println("Worker version: ", ver)
		fmt.Print("CLI version: ")
		cli.VersionPrinter(cctx)
		fmt.Println()

		sess, err := workerApi.ProcessSession(ctx)
		if err != nil {
			return xerrors.Errorf("getting session: %w", err)
		}
		fmt.Printf("Session: %s\n", sess)

		enabled, err := workerApi.Enabled(ctx)
		if err != nil {
			return xerrors.Errorf("checking worker status: %w", err)
		}
		fmt.Printf("Enabled: %t\n", enabled)

		info, err := workerApi.Info(ctx)
		if err != nil {
			return xerrors.Errorf("getting info: %w", err)
		}

		tt, err := workerApi.TaskTypes(ctx)
		if err != nil {
			return xerrors.Errorf("getting task types: %w", err)
		}

		fmt.Printf("Hostname: %s\n", info.Hostname)
		fmt.Printf("CPUs: %d; GPUs: %v\n", info.Resources.CPUs, info.Resources.GPUs)
		fmt.Printf("RAM: %s; Swap: %s\n", types.SizeStr(types.NewInt(info.Resources.MemPhysical)), types.SizeStr(types.NewInt(info.Resources.MemSwap)))
		fmt.Printf("Reserved memory: %s\n", types.SizeStr(types.NewInt(info.Resources.MemReserved)))

		fmt.Printf("Task types: ")
		for _, t := range ttList(tt) {
			fmt.Printf("%s ", t.Short())
		}
		fmt.Println()

		fmt.Println()

		paths, err := workerApi.Paths(ctx)
		if err != nil {
			return xerrors.Errorf("getting path info: %w", err)
		}

		for _, path := range paths {
			fmt.Printf("%s:\n", path.ID)
			fmt.Printf("\tWeight: %d; Use: ", path.Weight)
			if path.CanSeal || path.CanStore {
				if path.CanSeal {
					fmt.Print("Seal ")
				}
				if path.CanStore {
					fmt.Print("Store")
				}
				fmt.Println("")
			} else {
				fmt.Print("Use: ReadOnly")
			}
			fmt.Printf("\tLocal: %s\n", path.LocalPath)
		}

		return nil
	},
}

func ttList(tt map[types2.TaskType]struct{}) []types2.TaskType {
	tasks := make([]types2.TaskType, 0, len(tt))
	for taskType := range tt {
		tasks = append(tasks, taskType)
	}
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].Less(tasks[j])
	})
	return tasks
}
