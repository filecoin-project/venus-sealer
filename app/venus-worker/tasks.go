package main

import (
	"context"
	"strings"

	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/venus-sealer/api"
	"github.com/filecoin-project/venus-sealer/extern/sector-storage/sealtasks"
)

var tasksCmd = &cli.Command{
	Name:  "tasks",
	Usage: "Manage task processing",
	Subcommands: []*cli.Command{
		tasksEnableCmd,
		tasksDisableCmd,
	},
}

var allowSetting = map[sealtasks.TaskType]struct{}{
	sealtasks.TTAddPiece:   {},
	sealtasks.TTPreCommit1: {},
	sealtasks.TTPreCommit2: {},
	sealtasks.TTCommit2:    {},
	sealtasks.TTUnseal:     {},
}

var settableStr = func() string {
	var s []string
	for _, tt := range ttList(allowSetting) {
		s = append(s, tt.Short())
	}
	return strings.Join(s, "|")
}()

var tasksEnableCmd = &cli.Command{
	Name:      "enable",
	Usage:     "Enable a task type",
	ArgsUsage: "[" + settableStr + "]",
	Action:    taskAction(api.WorkerAPI.TaskEnable),
}

var tasksDisableCmd = &cli.Command{
	Name:      "disable",
	Usage:     "Disable a task type",
	ArgsUsage: "[" + settableStr + "]",
	Action:    taskAction(api.WorkerAPI.TaskDisable),
}

func taskAction(tf func(a api.WorkerAPI, ctx context.Context, tt sealtasks.TaskType) error) func(cctx *cli.Context) error {
	return func(cctx *cli.Context) error {
		if cctx.NArg() != 1 {
			return xerrors.Errorf("expected 1 argument")
		}

		var tt sealtasks.TaskType
		for taskType := range allowSetting {
			if taskType.Short() == cctx.Args().First() {
				tt = taskType
				break
			}
		}

		if tt == "" {
			return xerrors.Errorf("unknown task type '%s'", cctx.Args().First())
		}

		workerApi, closer, err := api.GetWorkerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		ctx := api.ReqContext(cctx)

		return tf(workerApi, ctx, tt)
	}
}
