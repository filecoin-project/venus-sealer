package main

import (
	"context"
	"github.com/filecoin-project/venus-sealer/types"
	"strings"

	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/venus-sealer/api"
)

var tasksCmd = &cli.Command{
	Name:  "tasks",
	Usage: "Manage task processing",
	Subcommands: []*cli.Command{
		tasksEnableCmd,
		tasksDisableCmd,
	},
}

var allowSetting = map[types.TaskType]struct{}{
	types.TTAddPiece:   {},
	types.TTPreCommit1: {},
	types.TTPreCommit2: {},
	types.TTCommit2:    {},
	types.TTUnseal:     {},
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

func taskAction(tf func(a api.WorkerAPI, ctx context.Context, tt types.TaskType) error) func(cctx *cli.Context) error {
	return func(cctx *cli.Context) error {
		if cctx.NArg() != 1 {
			return xerrors.Errorf("expected 1 argument")
		}

		var tt types.TaskType
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
