package main

import (
	"context"
	"fmt"
	"time"

	"github.com/fatih/color"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/venus-sealer/api"
	"github.com/filecoin-project/venus-sealer/types"
	"github.com/hako/durafmt"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"

	venustypes "github.com/filecoin-project/venus/pkg/types"
)

type PrintHelpErr struct {
	Err error
	Ctx *cli.Context
}

func (e *PrintHelpErr) Error() string {
	return e.Err.Error()
}

func (e *PrintHelpErr) Unwrap() error {
	return e.Err
}

func (e *PrintHelpErr) Is(o error) bool {
	_, ok := o.(*PrintHelpErr)
	return ok
}

func ShowHelp(cctx *cli.Context, err error) error {
	return &PrintHelpErr{Err: err, Ctx: cctx}
}
func EpochTime(curr, e abi.ChainEpoch, blockDelay uint64) string {
	switch {
	case curr > e:
		return fmt.Sprintf("%d (%s ago)", e, durafmt.Parse(time.Second*time.Duration(int64(blockDelay)*int64(curr-e))).LimitFirstN(2))
	case curr == e:
		return fmt.Sprintf("%d (now)", e)
	case curr < e:
		return fmt.Sprintf("%d (in %s)", e, durafmt.Parse(time.Second*time.Duration(int64(blockDelay)*int64(e-curr))).LimitFirstN(2))
	}

	panic("math broke")
}

func HeightToTime(ts *venustypes.TipSet, openHeight abi.ChainEpoch, blockDelay uint64) string {
	if ts.Len() == 0 {
		return ""
	}
	firstBlkTime := ts.Blocks()[0].Timestamp - uint64(ts.Height())*blockDelay
	return time.Unix(int64(firstBlkTime+blockDelay*uint64(openHeight)), 0).Format("15:04:05")
}

type stateMeta struct {
	i     int
	col   color.Attribute
	state types.SectorState
}

var stateOrder = map[types.SectorState]stateMeta{}
var stateList = []stateMeta{
	{col: 39, state: "Total"},
	{col: color.FgGreen, state: types.Proving},

	{col: color.FgBlue, state: types.Empty},
	{col: color.FgBlue, state: types.WaitDeals},

	{col: color.FgRed, state: types.UndefinedSectorState},
	{col: color.FgYellow, state: types.Packing},
	{col: color.FgYellow, state: types.GetTicket},
	{col: color.FgYellow, state: types.PreCommit1},
	{col: color.FgYellow, state: types.PreCommit2},
	{col: color.FgYellow, state: types.PreCommitting},
	{col: color.FgYellow, state: types.PreCommitWait},
	{col: color.FgYellow, state: types.WaitSeed},
	{col: color.FgYellow, state: types.Committing},
	{col: color.FgYellow, state: types.SubmitCommit},
	{col: color.FgYellow, state: types.CommitWait},
	{col: color.FgYellow, state: types.FinalizeSector},

	{col: color.FgCyan, state: types.Terminating},
	{col: color.FgCyan, state: types.TerminateWait},
	{col: color.FgCyan, state: types.TerminateFinality},
	{col: color.FgCyan, state: types.TerminateFailed},
	{col: color.FgCyan, state: types.Removing},
	{col: color.FgCyan, state: types.Removed},

	{col: color.FgRed, state: types.FailedUnrecoverable},
	{col: color.FgRed, state: types.SealPreCommit1Failed},
	{col: color.FgRed, state: types.SealPreCommit2Failed},
	{col: color.FgRed, state: types.PreCommitFailed},
	{col: color.FgRed, state: types.ComputeProofFailed},
	{col: color.FgRed, state: types.CommitFailed},
	{col: color.FgRed, state: types.PackingFailed},
	{col: color.FgRed, state: types.FinalizeFailed},
	{col: color.FgRed, state: types.Faulty},
	{col: color.FgRed, state: types.FaultReported},
	{col: color.FgRed, state: types.FaultedFinal},
	{col: color.FgRed, state: types.RemoveFailed},
	{col: color.FgRed, state: types.DealsExpired},
	{col: color.FgRed, state: types.RecoverDealIDs},
}

func init() {
	for i, state := range stateList {
		stateOrder[state.state] = stateMeta{
			i:   i,
			col: state.col,
		}
	}
}

func getActorAddress(ctx context.Context, nodeAPI api.StorageMiner, overrideMaddr string) (maddr address.Address, err error) {
	if overrideMaddr != "" {
		maddr, err = address.NewFromString(overrideMaddr)
		if err != nil {
			return maddr, err
		}
		return
	}

	maddr, err = nodeAPI.ActorAddress(ctx)
	if err != nil {
		return maddr, xerrors.Errorf("getting actor address: %w", err)
	}

	return maddr, nil
}
