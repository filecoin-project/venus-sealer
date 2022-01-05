package main

import (
	"context"
	"fmt"
	"math"
	corebig "math/big"
	"sort"
	"time"

	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"

	cbor "github.com/ipfs/go-ipld-cbor"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"

	"github.com/filecoin-project/venus-sealer/api"
	"github.com/filecoin-project/venus-sealer/constants"
	"github.com/filecoin-project/venus-sealer/lib/blockstore"
	types2 "github.com/filecoin-project/venus-sealer/types"

	"github.com/filecoin-project/venus/venus-shared/actors/adt"
	"github.com/filecoin-project/venus/venus-shared/actors/builtin"
	"github.com/filecoin-project/venus/venus-shared/actors/builtin/miner"
	"github.com/filecoin-project/venus/venus-shared/types"
)

var infoCmd = &cli.Command{
	Name:  "info",
	Usage: "Print miner info",
	Subcommands: []*cli.Command{
		infoAllCmd,
	},
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "hide-sectors-info",
			Usage: "hide sectors info",
		},
	},
	Action: infoCmdAct,
}

func infoCmdAct(cctx *cli.Context) error {
	color.NoColor = !cctx.Bool("color")

	storageAPI, closer, err := api.GetStorageMinerAPI(cctx)
	if err != nil {
		return err
	}
	defer closer()

	nodeAPI, acloser, err := api.GetFullNodeAPIV2(cctx)
	if err != nil {
		return err
	}
	defer acloser()

	ctx := api.ReqContext(cctx)

	params, err := storageAPI.NetParamsConfig(ctx)
	if err != nil {
		return err
	}

	fmt.Print("Chain: ")

	head, err := nodeAPI.ChainHead(ctx)
	if err != nil {
		return err
	}

	switch {
	case time.Now().Unix()-int64(head.MinTimestamp()) < int64(params.BlockDelaySecs*3/2): // within 1.5 epochs
		fmt.Printf("[%s]", color.GreenString("sync ok"))
	case time.Now().Unix()-int64(head.MinTimestamp()) < int64(params.BlockDelaySecs*5): // within 5 epochs
		fmt.Printf("[%s]", color.YellowString("sync slow (%s behind)", time.Since(time.Unix(int64(head.MinTimestamp()), 0)).Truncate(time.Second)))
	default:
		fmt.Printf("[%s]", color.RedString("sync behind! (%s behind)", time.Since(time.Unix(int64(head.MinTimestamp()), 0)).Truncate(time.Second)))
	}

	basefee := head.MinTicketBlock().ParentBaseFee
	gasCol := []color.Attribute{color.FgBlue}
	switch {
	case basefee.GreaterThan(big.NewInt(7000_000_000)): // 7 nFIL
		gasCol = []color.Attribute{color.BgRed, color.FgBlack}
	case basefee.GreaterThan(big.NewInt(3000_000_000)): // 3 nFIL
		gasCol = []color.Attribute{color.FgRed}
	case basefee.GreaterThan(big.NewInt(750_000_000)): // 750 uFIL
		gasCol = []color.Attribute{color.FgYellow}
	case basefee.GreaterThan(big.NewInt(100_000_000)): // 100 uFIL
		gasCol = []color.Attribute{color.FgGreen}
	}
	fmt.Printf(" [basefee %s]", color.New(gasCol...).Sprint(types.FIL(basefee).Short()))

	fmt.Println()

	maddr, err := getActorAddress(ctx, storageAPI, cctx.String("actor"))
	if err != nil {
		return err
	}

	mact, err := nodeAPI.StateGetActor(ctx, maddr, types.EmptyTSK)
	if err != nil {
		return err
	}

	tbs := blockstore.NewTieredBstore(api.NewAPIBlockstore(nodeAPI), blockstore.NewMemory())
	mas, err := miner.Load(adt.WrapStore(ctx, cbor.NewCborStore(tbs)), mact)
	if err != nil {
		return err
	}

	// Sector size
	mi, err := nodeAPI.StateMinerInfo(ctx, maddr, types.EmptyTSK)
	if err != nil {
		return err
	}

	ssize := types.SizeStr(types.NewInt(uint64(mi.SectorSize)))
	fmt.Printf("Sealer: %s (%s sectors)\n", color.BlueString("%s", maddr), ssize)

	pow, err := nodeAPI.StateMinerPower(ctx, maddr, types.EmptyTSK)
	if err != nil {
		return err
	}

	fmt.Printf("Power: %s / %s (%0.4f%%)\n",
		color.GreenString(types.DeciStr(pow.MinerPower.QualityAdjPower)),
		types.DeciStr(pow.TotalPower.QualityAdjPower),
		types.BigDivFloat(
			types.BigMul(pow.MinerPower.QualityAdjPower, big.NewInt(100)),
			pow.TotalPower.QualityAdjPower,
		),
	)

	fmt.Printf("\tRaw: %s / %s (%0.4f%%)\n",
		color.BlueString(types.SizeStr(pow.MinerPower.RawBytePower)),
		types.SizeStr(pow.TotalPower.RawBytePower),
		types.BigDivFloat(
			types.BigMul(pow.MinerPower.RawBytePower, big.NewInt(100)),
			pow.TotalPower.RawBytePower,
		),
	)

	secCounts, err := nodeAPI.StateMinerSectorCount(ctx, maddr, types.EmptyTSK)
	if err != nil {
		return err
	}

	proving := secCounts.Active + secCounts.Faulty
	nfaults := secCounts.Faulty
	fmt.Printf("\tCommitted: %s\n", types.SizeStr(types.BigMul(types.NewInt(secCounts.Live), types.NewInt(uint64(mi.SectorSize)))))
	if nfaults == 0 {
		fmt.Printf("\tProving: %s\n", types.SizeStr(types.BigMul(types.NewInt(proving), types.NewInt(uint64(mi.SectorSize)))))
	} else {
		var faultyPercentage float64
		if secCounts.Live != 0 {
			faultyPercentage = float64(100*nfaults) / float64(secCounts.Live)
		}
		fmt.Printf("\tProving: %s (%s Faulty, %.2f%%)\n",
			types.SizeStr(types.BigMul(types.NewInt(proving), types.NewInt(uint64(mi.SectorSize)))),
			types.SizeStr(types.BigMul(types.NewInt(nfaults), types.NewInt(uint64(mi.SectorSize)))),
			faultyPercentage)
	}

	if !pow.HasMinPower {
		fmt.Print("Below minimum power threshold, no blocks will be won")
	} else {

		winRatio := new(corebig.Rat).SetFrac(
			types.BigMul(pow.MinerPower.QualityAdjPower, types.NewInt(constants.BlocksPerEpoch)).Int,
			pow.TotalPower.QualityAdjPower.Int,
		)

		if winRatioFloat, _ := winRatio.Float64(); winRatioFloat > 0 {

			// if the corresponding poisson distribution isn't infinitely small then
			// throw it into the mix as well, accounting for multi-wins
			winRationWithPoissonFloat := -math.Expm1(-winRatioFloat)
			winRationWithPoisson := new(corebig.Rat).SetFloat64(winRationWithPoissonFloat)
			if winRationWithPoisson != nil {
				winRatio = winRationWithPoisson
				winRatioFloat = winRationWithPoissonFloat
			}
			weekly, _ := new(corebig.Rat).Mul(
				winRatio,
				new(corebig.Rat).SetInt64(7*builtin.EpochsInDay),
			).Float64()

			avgDuration, _ := new(corebig.Rat).Mul(
				new(corebig.Rat).SetInt64(builtin.EpochDurationSeconds),
				new(corebig.Rat).Inv(winRatio),
			).Float64()

			fmt.Print("Projected average block win rate: ")
			color.Blue(
				"%.02f/week (every %s)",
				weekly,
				(time.Second * time.Duration(avgDuration)).Truncate(time.Second).String(),
			)

			// Geometric distribution of P(Y < k) calculated as described in https://en.wikipedia.org/wiki/Geometric_distribution#Probability_Outcomes_Examples
			// https://www.wolframalpha.com/input/?i=t+%3E+0%3B+p+%3E+0%3B+p+%3C+1%3B+c+%3E+0%3B+c+%3C1%3B+1-%281-p%29%5E%28t%29%3Dc%3B+solve+t
			// t == how many dice-rolls (epochs) before win
			// p == winRate == ( minerPower / netPower )
			// c == target probability of win ( 99.9% in this case )
			fmt.Print("Projected block win with ")
			color.Green(
				"99.9%% probability every %s",
				(time.Second * time.Duration(
					builtin.EpochDurationSeconds*math.Log(1-0.999)/
						math.Log(1-winRatioFloat),
				)).Truncate(time.Second).String(),
			)
			fmt.Println("(projections DO NOT account for future network and miner growth)")
		}
	}

	fmt.Println()
	spendable := big.Zero()

	// NOTE: there's no need to unlock anything here. Funds only
	// vest on deadline boundaries, and they're unlocked by cron.
	lockedFunds, err := mas.LockedFunds()
	if err != nil {
		return xerrors.Errorf("getting locked funds: %w", err)
	}
	availBalance, err := mas.AvailableBalance(mact.Balance)
	if err != nil {
		return xerrors.Errorf("getting available balance: %w", err)
	}
	spendable = big.Add(spendable, availBalance)

	fmt.Printf("Sealer Balance:    %s\n", color.YellowString("%s", types.FIL(mact.Balance).Short()))
	fmt.Printf("      PreCommit:  %s\n", types.FIL(lockedFunds.PreCommitDeposits).Short())
	fmt.Printf("      Pledge:     %s\n", types.FIL(lockedFunds.InitialPledgeRequirement).Short())
	fmt.Printf("      Vesting:    %s\n", types.FIL(lockedFunds.VestingFunds).Short())
	colorTokenAmount("      Available:  %s\n", availBalance)

	mb, err := nodeAPI.StateMarketBalance(ctx, maddr, types.EmptyTSK)
	if err != nil {
		return xerrors.Errorf("getting market balance: %w", err)
	}
	spendable = big.Add(spendable, big.Sub(mb.Escrow, mb.Locked))

	fmt.Printf("Market Balance:   %s\n", types.FIL(mb.Escrow).Short())
	fmt.Printf("       Locked:    %s\n", types.FIL(mb.Locked).Short())
	colorTokenAmount("       Available: %s\n", big.Sub(mb.Escrow, mb.Locked))

	wb, err := nodeAPI.WalletBalance(ctx, mi.Worker)
	if err != nil {
		return xerrors.Errorf("getting worker balance: %w", err)
	}
	spendable = big.Add(spendable, wb)
	color.Cyan("Worker Balance:   %s", types.FIL(wb).Short())
	if len(mi.ControlAddresses) > 0 {
		cbsum := big.Zero()
		for _, ca := range mi.ControlAddresses {
			b, err := nodeAPI.WalletBalance(ctx, ca)
			if err != nil {
				return xerrors.Errorf("getting control address balance: %w", err)
			}
			cbsum = big.Add(cbsum, b)
		}
		spendable = big.Add(spendable, cbsum)

		fmt.Printf("       Control:   %s\n", types.FIL(cbsum).Short())
	}
	colorTokenAmount("Total Spendable:  %s\n", spendable)

	fmt.Println()

	if !cctx.Bool("hide-sectors-info") {
		fmt.Println("Sectors:")
		err = sectorsInfo(ctx, storageAPI)
		if err != nil {
			return err
		}
	}

	// TODO: grab actr state / info
	//  * Sealed sectors (count / bytes)
	//  * Power
	return nil
}

func sectorsInfo(ctx context.Context, napi api.StorageMiner) error {
	summary, err := napi.SectorsSummary(ctx)
	if err != nil {
		return err
	}

	buckets := make(map[types2.SectorState]int)
	var total int
	for s, c := range summary {
		buckets[types2.SectorState(s)] = c
		total += c
	}
	buckets["Total"] = total

	var sorted []stateMeta
	for state, i := range buckets {
		sorted = append(sorted, stateMeta{i: i, state: state})
	}

	sort.Slice(sorted, func(i, j int) bool {
		return stateOrder[sorted[i].state].i < stateOrder[sorted[j].state].i
	})

	for _, s := range sorted {
		_, _ = color.New(stateOrder[s.state].col).Printf("\t%s: %d\n", s.state, s.i)
	}

	return nil
}

func colorTokenAmount(format string, amount abi.TokenAmount) {
	if amount.GreaterThan(big.Zero()) {
		color.Green(format, types.FIL(amount).Short())
	} else if amount.Equals(big.Zero()) {
		color.Yellow(format, types.FIL(amount).Short())
	} else {
		color.Red(format, types.FIL(amount).Short())
	}
}
