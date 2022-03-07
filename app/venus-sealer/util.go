package main

import (
	"context"
	"fmt"
	"time"

	"github.com/fatih/color"
	"github.com/hako/durafmt"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"

	"github.com/filecoin-project/venus-sealer/api"
	"github.com/filecoin-project/venus-sealer/constants"
	types2 "github.com/filecoin-project/venus-sealer/types"

	"github.com/filecoin-project/venus/venus-shared/types"
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

func EpochGap(curr, e abi.ChainEpoch) string {
	switch {
	case curr > e:
		return fmt.Sprintf("%d (%s ago)", e, durafmt.Parse(time.Second*time.Duration(int64(constants.BlockDelaySecs)*int64(curr-e))).LimitFirstN(2))
	case curr == e:
		return fmt.Sprintf("%d (now)", e)
	case curr < e:
		return fmt.Sprintf("%d (in %s)", e, durafmt.Parse(time.Second*time.Duration(int64(constants.BlockDelaySecs)*int64(e-curr))).LimitFirstN(2))
	}

	panic("math broke")
}

func HeightToTime(ts *types.TipSet, openHeight abi.ChainEpoch, blockDelay uint64) string {
	if ts.Len() == 0 {
		return ""
	}
	firstBlkTime := ts.Blocks()[0].Timestamp - uint64(ts.Height())*blockDelay
	return time.Unix(int64(firstBlkTime+blockDelay*uint64(openHeight)), 0).Format("15:04:05")
}

type stateMeta struct {
	i     int
	col   color.Attribute
	state types2.SectorState
}

var stateOrder = map[types2.SectorState]stateMeta{}
var stateList = []stateMeta{
	{col: 39, state: "Total"},
	{col: color.FgGreen, state: types2.Proving},
	{col: color.FgGreen, state: types2.UpdateActivating},

	{col: color.FgBlue, state: types2.Empty},
	{col: color.FgBlue, state: types2.WaitDeals},
	{col: color.FgBlue, state: types2.AddPiece},
	{col: color.FgBlue, state: types2.SnapDealsWaitDeals},
	{col: color.FgBlue, state: types2.SnapDealsAddPiece},

	{col: color.FgRed, state: types2.UndefinedSectorState},
	{col: color.FgYellow, state: types2.Packing},
	{col: color.FgYellow, state: types2.GetTicket},
	{col: color.FgYellow, state: types2.PreCommit1},
	{col: color.FgYellow, state: types2.PreCommit2},
	{col: color.FgYellow, state: types2.PreCommitting},
	{col: color.FgYellow, state: types2.PreCommitWait},
	{col: color.FgYellow, state: types2.SubmitPreCommitBatch},
	{col: color.FgYellow, state: types2.PreCommitBatchWait},
	{col: color.FgYellow, state: types2.WaitSeed},
	{col: color.FgYellow, state: types2.Committing},
	{col: color.FgYellow, state: types2.CommitFinalize},
	{col: color.FgYellow, state: types2.SubmitCommit},
	{col: color.FgYellow, state: types2.CommitWait},
	{col: color.FgYellow, state: types2.SubmitCommitAggregate},
	{col: color.FgYellow, state: types2.CommitAggregateWait},
	{col: color.FgYellow, state: types2.FinalizeSector},
	{col: color.FgYellow, state: types2.SnapDealsPacking},
	{col: color.FgYellow, state: types2.UpdateReplica},
	{col: color.FgYellow, state: types2.ProveReplicaUpdate},
	{col: color.FgYellow, state: types2.SubmitReplicaUpdate},
	{col: color.FgYellow, state: types2.ReplicaUpdateWait},
	{col: color.FgYellow, state: types2.FinalizeReplicaUpdate},
	{col: color.FgYellow, state: types2.ReleaseSectorKey},

	{col: color.FgCyan, state: types2.Terminating},
	{col: color.FgCyan, state: types2.TerminateWait},
	{col: color.FgCyan, state: types2.TerminateFinality},
	{col: color.FgCyan, state: types2.TerminateFailed},
	{col: color.FgCyan, state: types2.Removing},
	{col: color.FgCyan, state: types2.Removed},
	{col: color.FgCyan, state: types2.AbortUpgrade},

	{col: color.FgRed, state: types2.FailedUnrecoverable},
	{col: color.FgRed, state: types2.AddPieceFailed},
	{col: color.FgRed, state: types2.SealPreCommit1Failed},
	{col: color.FgRed, state: types2.SealPreCommit2Failed},
	{col: color.FgRed, state: types2.PreCommitFailed},
	{col: color.FgRed, state: types2.ComputeProofFailed},
	{col: color.FgRed, state: types2.CommitFailed},
	{col: color.FgRed, state: types2.CommitFinalizeFailed},
	{col: color.FgRed, state: types2.PackingFailed},
	{col: color.FgRed, state: types2.FinalizeFailed},
	{col: color.FgRed, state: types2.Faulty},
	{col: color.FgRed, state: types2.FaultReported},
	{col: color.FgRed, state: types2.FaultedFinal},
	{col: color.FgRed, state: types2.RemoveFailed},
	{col: color.FgRed, state: types2.DealsExpired},
	{col: color.FgRed, state: types2.RecoverDealIDs},
	{col: color.FgRed, state: types2.SnapDealsAddPieceFailed},
	{col: color.FgRed, state: types2.SnapDealsDealsExpired},
	{col: color.FgRed, state: types2.ReplicaUpdateFailed},
	{col: color.FgRed, state: types2.ReleaseSectorKeyFailed},
	{col: color.FgRed, state: types2.FinalizeReplicaUpdateFailed},
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
