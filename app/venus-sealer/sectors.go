package main

import (
	"fmt"
	miner5 "github.com/filecoin-project/specs-actors/v5/actors/builtin/miner"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/docker/go-units"
	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-bitfield"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"

	"github.com/filecoin-project/venus/pkg/types"
	actors "github.com/filecoin-project/venus/pkg/types/specactors"
	"github.com/filecoin-project/venus/pkg/types/specactors/builtin/miner"
	"github.com/filecoin-project/venus/pkg/types/specactors/policy"

	"github.com/filecoin-project/venus-sealer/api"
	"github.com/filecoin-project/venus-sealer/lib/tablewriter"
	"github.com/filecoin-project/venus-sealer/sector-storage/storiface"
	types2 "github.com/filecoin-project/venus-sealer/types"
)

var sectorsCmd = &cli.Command{
	Name:  "sectors",
	Usage: "interact with sector store",
	Subcommands: []*cli.Command{
		sectorsStatusCmd,
		sectorsListCmd,
		sectorsRefsCmd,
		sectorsUpdateCmd,
		sectorsPledgeCmd,
		sectorsDealCmd,
		sectorsExtendCmd,
		sectorsTerminateCmd,
		sectorsRemoveCmd,
		sectorsMarkForUpgradeCmd,
		sectorsStartSealCmd,
		sectorsSealDelayCmd,
		sectorsCapacityCollateralCmd,
		sectorsBatching,
		sectorsRedoCmd,
	},
}

var sectorsRedoCmd = &cli.Command{
	Name:  "redo",
	Usage: "redo the specified sector and support sealer locally",
	ArgsUsage: "<sectorNum>",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "sealPath",
			Usage: "",
		},
		&cli.StringFlag{
			Name:  "storePath",
			Usage: "",
		},
	},
	Action: func(cctx *cli.Context) error {
		nodeApi, closer, err := api.GetStorageMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()
		ctx := api.ReqContext(cctx)

		if !cctx.Args().Present() {
			return fmt.Errorf("must specify sector number to get status of")
		}

		id, err := strconv.ParseUint(cctx.Args().First(), 10, 64)
		if err != nil {
			return err
		}

		err = nodeApi.RedoSector(ctx, storiface.SectorRedoParams{
			SectorNumber: abi.SectorNumber(id),
			SealPath: cctx.String("sealPath"),
			StorePath: cctx.String("storePath"),
		})
		if err != nil {
			return err
		}

		fmt.Println("Redo CC sector: ", id)

		return nil
	},
}

var sectorsPledgeCmd = &cli.Command{
	Name:  "pledge",
	Usage: "store random data in a sector",
	Action: func(cctx *cli.Context) error {
		nodeApi, closer, err := api.GetStorageMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()
		ctx := api.ReqContext(cctx)

		id, err := nodeApi.PledgeSector(ctx)
		if err != nil {
			return err
		}

		fmt.Println("Created CC sector: ", id.Number)

		return nil
	},
}

var sectorsDealCmd = &cli.Command{
	Name:  "deal",
	Usage: "store deal data in a sector",
	Action: func(cctx *cli.Context) error {
		nodeApi, closer, err := api.GetStorageMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()
		ctx := api.ReqContext(cctx)

		assignedDeals, err := nodeApi.DealSector(ctx)
		if err != nil {
			return err
		}

		for _, assignedDeal := range assignedDeals {
			fmt.Println("Assign Deals %d sector %d piece %s offset %d length ", assignedDeal.DealId, assignedDeal.SectorId, assignedDeal.PieceCid, assignedDeal.Offset, assignedDeal.Size)
		}
		return nil
	},
}

var sectorsStatusCmd = &cli.Command{
	Name:      "status",
	Usage:     "Get the seal status of a sector by its number",
	ArgsUsage: "<sectorNum>",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "log",
			Usage: "display event log",
		},
		&cli.BoolFlag{
			Name:  "on-chain-info",
			Usage: "show sector on chain info",
		},
	},
	Action: func(cctx *cli.Context) error {
		nodeApi, closer, err := api.GetStorageMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()
		ctx := api.ReqContext(cctx)

		if !cctx.Args().Present() {
			return fmt.Errorf("must specify sector number to get status of")
		}

		id, err := strconv.ParseUint(cctx.Args().First(), 10, 64)
		if err != nil {
			return err
		}

		onChainInfo := cctx.Bool("on-chain-info")
		status, err := nodeApi.SectorsStatus(ctx, abi.SectorNumber(id), onChainInfo)
		if err != nil {
			return err
		}

		fmt.Printf("SectorID:\t%d\n", status.SectorID)
		fmt.Printf("Status:\t\t%s\n", status.State)
		fmt.Printf("CIDcommD:\t%s\n", status.CommD)
		fmt.Printf("CIDcommR:\t%s\n", status.CommR)
		fmt.Printf("Ticket:\t\t%x\n", status.Ticket.Value)
		fmt.Printf("TicketH:\t%d\n", status.Ticket.Epoch)
		fmt.Printf("Seed:\t\t%x\n", status.Seed.Value)
		fmt.Printf("SeedH:\t\t%d\n", status.Seed.Epoch)
		fmt.Printf("Precommit:\t%s\n", status.PreCommitMsg)
		fmt.Printf("Commit:\t\t%s\n", status.CommitMsg)
		fmt.Printf("Proof:\t\t%x\n", status.Proof)
		fmt.Printf("Deals:\t\t%v\n", status.Deals)
		fmt.Printf("Retries:\t%d\n", status.Retries)
		if status.LastErr != "" {
			fmt.Printf("Last Error:\t\t%s\n", status.LastErr)
		}

		if onChainInfo {
			fmt.Printf("\nSector On Chain Info\n")
			fmt.Printf("SealProof:\t\t%x\n", status.SealProof)
			fmt.Printf("Activation:\t\t%v\n", status.Activation)
			fmt.Printf("Expiration:\t\t%v\n", status.Expiration)
			fmt.Printf("DealWeight:\t\t%v\n", status.DealWeight)
			fmt.Printf("VerifiedDealWeight:\t\t%v\n", status.VerifiedDealWeight)
			fmt.Printf("InitialPledge:\t\t%v\n", status.InitialPledge)
			fmt.Printf("\nExpiration Info\n")
			fmt.Printf("OnTime:\t\t%v\n", status.OnTime)
			fmt.Printf("Early:\t\t%v\n", status.Early)
		}

		if cctx.Bool("log") {
			fmt.Printf("--------\nEvent Log:\n")

			for i, l := range status.Log {
				fmt.Printf("%d.\t%s:\t[%s]\t%s\n", i, time.Unix(int64(l.Timestamp), 0), l.Kind, l.Message)
				if l.Trace != "" {
					fmt.Printf("\t%s\n", l.Trace)
				}
			}
		}
		return nil
	},
}

var sectorsListCmd = &cli.Command{
	Name:  "list",
	Usage: "List sectors",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "show-removed",
			Usage: "show removed sectors",
		},
		&cli.BoolFlag{
			Name:    "color",
			Aliases: []string{"c"},
			Value:   true,
		},
		&cli.BoolFlag{
			Name:  "fast",
			Usage: "don't show on-chain info for better performance",
		},
		&cli.BoolFlag{
			Name:  "events",
			Usage: "display number of events the sector has received",
		},
		&cli.BoolFlag{
			Name:  "seal-time",
			Usage: "display how long it took for the sector to be sealed",
		},
		&cli.StringFlag{
			Name:  "states",
			Usage: "filter sectors by a comma-separated list of states",
		},
	},
	Action: func(cctx *cli.Context) error {
		color.NoColor = !cctx.Bool("color")

		storageAPI, closer, err := api.GetStorageMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		fullApi, closer2, err := api.GetFullNodeAPIV2(cctx) // TODO: consider storing full node address in config
		if err != nil {
			return err
		}
		defer closer2()

		ctx := api.ReqContext(cctx)

		networkParams, err := storageAPI.NetParamsConfig(ctx)
		if err != nil {
			return err
		}

		var (
			list []api.SectorInfo
			ss   []api.SectorState
		)

		showRemoved := cctx.Bool("show-removed")
		states := cctx.String("states")
		if len(states) != 0 {
			sList := strings.Split(states, ",")
			ss = make([]api.SectorState, len(sList))
			for i := range sList {
				ss[i] = api.SectorState(sList[i])
			}
		}

		// reduce query time
		skipLog := true
		if cctx.Bool("events") || cctx.Bool("seal-time") {
			skipLog = false
		}
		fast := cctx.Bool("fast")
		list, err = storageAPI.SectorsInfoListInStates(ctx, ss, !fast, skipLog)
		if err != nil {
			return err
		}

		maddr, err := storageAPI.ActorAddress(ctx)
		if err != nil {
			return err
		}

		head, err := fullApi.ChainHead(ctx)
		if err != nil {
			return err
		}

		activeSet, err := fullApi.StateMinerActiveSectors(ctx, maddr, head.Key())
		if err != nil {
			return err
		}
		activeIDs := make(map[abi.SectorNumber]struct{}, len(activeSet))
		for _, info := range activeSet {
			activeIDs[info.SectorNumber] = struct{}{}
		}

		sset, err := fullApi.StateMinerSectors(ctx, maddr, nil, head.Key())
		if err != nil {
			return err
		}
		commitedIDs := make(map[abi.SectorNumber]struct{}, len(sset))
		for _, info := range sset {
			commitedIDs[info.SectorNumber] = struct{}{}
		}

		sort.Slice(list, func(i, j int) bool {
			return list[i].SectorID < list[j].SectorID
		})

		tw := tablewriter.New(
			tablewriter.Col("ID"),
			tablewriter.Col("State"),
			tablewriter.Col("OnChain"),
			tablewriter.Col("Active"),
			tablewriter.Col("Expiration"),
			tablewriter.Col("SealTime"),
			tablewriter.Col("Events"),
			tablewriter.Col("Deals"),
			tablewriter.Col("DealWeight"),
			tablewriter.NewLineCol("Error"),
			tablewriter.NewLineCol("RecoveryTimeout"))

		for _, st := range list {
			//st, err := storageAPI.SectorsStatus(ctx, s, !fast)
			//if err != nil {
			//	tw.Write(map[string]interface{}{
			//		"ID":    s,
			//		"Error": err,
			//	})
			//	continue
			//}

			if showRemoved || st.State != api.SectorState(types2.Removed) {
				_, inSSet := commitedIDs[st.SectorID]
				_, inASet := activeIDs[st.SectorID]

				dw := .0
				if st.Expiration-st.Activation > 0 {
					dw = float64(big.Div(st.DealWeight, big.NewInt(int64(st.Expiration-st.Activation))).Uint64())
				}

				var deals int
				for _, deal := range st.Deals {
					if deal != 0 {
						deals++
					}
				}

				exp := st.Expiration
				if st.OnTime > 0 && st.OnTime < exp {
					exp = st.OnTime // Can be different when the sector was CC upgraded
				}

				m := map[string]interface{}{
					"ID":      st.SectorID,
					"State":   color.New(stateOrder[types2.SectorState(st.State)].col).Sprint(st.State),
					"OnChain": yesno(inSSet),
					"Active":  yesno(inASet),
				}

				if deals > 0 {
					m["Deals"] = color.GreenString("%d", deals)
				} else {
					m["Deals"] = color.BlueString("CC")
					if st.ToUpgrade {
						m["Deals"] = color.CyanString("CC(upgrade)")
					}
				}

				if !fast {
					if !inSSet {
						m["Expiration"] = "n/a"
					} else {
						m["Expiration"] = EpochTime(head.Height(), exp, networkParams.BlockDelaySecs)

						if !fast && deals > 0 {
							m["DealWeight"] = units.BytesSize(dw)
						}

						if st.Early > 0 {
							m["RecoveryTimeout"] = color.YellowString(EpochTime(head.Height(), st.Early, networkParams.BlockDelaySecs))
						}
					}
				}

				if cctx.Bool("events") {
					var events int
					for _, sectorLog := range st.Log {
						if !strings.HasPrefix(sectorLog.Kind, "event") {
							continue
						}
						if sectorLog.Kind == "event;sealing.SectorRestart" {
							continue
						}
						events++
					}

					pieces := len(st.Deals)

					switch {
					case events < 12+pieces:
						m["Events"] = color.GreenString("%d", events)
					case events < 20+pieces:
						m["Events"] = color.YellowString("%d", events)
					default:
						m["Events"] = color.RedString("%d", events)
					}
				}

				if cctx.Bool("seal-time") && len(st.Log) > 1 {
					start := time.Unix(int64(st.Log[0].Timestamp), 0)

					for _, sectorLog := range st.Log {
						if sectorLog.Kind == "event;sealing.SectorProving" {
							end := time.Unix(int64(sectorLog.Timestamp), 0)
							dur := end.Sub(start)

							switch {
							case dur < 12*time.Hour:
								m["SealTime"] = color.GreenString("%s", dur)
							case dur < 24*time.Hour:
								m["SealTime"] = color.YellowString("%s", dur)
							default:
								m["SealTime"] = color.RedString("%s", dur)
							}

							break
						}
					}
				}

				tw.Write(m)
			}
		}

		return tw.Flush(os.Stdout)
	},
}

var sectorsRefsCmd = &cli.Command{
	Name:  "refs",
	Usage: "List References to sectors",
	Action: func(cctx *cli.Context) error {
		nodeApi, closer, err := api.GetStorageMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()
		ctx := api.ReqContext(cctx)

		refs, err := nodeApi.SectorsRefs(ctx)
		if err != nil {
			return err
		}

		for name, refs := range refs {
			fmt.Printf("Block %s:\n", name)
			for _, ref := range refs {
				fmt.Printf("\t%d+%d %d bytes\n", ref.SectorID, ref.Offset, ref.Size)
			}
		}
		return nil
	},
}

var sectorsExtendCmd = &cli.Command{
	Name:      "extend",
	Usage:     "Extend sector expiration",
	ArgsUsage: "<sectorNumbers...>",
	Flags: []cli.Flag{
		&cli.Int64Flag{
			Name:     "new-expiration",
			Usage:    "new expiration epoch",
			Required: false,
		},
		&cli.BoolFlag{
			Name:     "v1-sectors",
			Usage:    "renews all v1 sectors up to the maximum possible lifetime",
			Required: false,
		},
		&cli.Int64Flag{
			Name:     "tolerance",
			Value:    20160,
			Usage:    "when extending v1 sectors, don't try to extend sectors by fewer than this number of epochs",
			Required: false,
		},
		&cli.Int64Flag{
			Name:     "expiration-cutoff",
			Usage:    "when extending v1 sectors, skip sectors whose current expiration is more than <cutoff> epochs from now (infinity if unspecified)",
			Required: false,
		},
		&cli.StringFlag{},
	},
	Action: func(cctx *cli.Context) error {

		fullApi, closer2, err := api.GetFullNodeAPIV2(cctx)
		if err != nil {
			return err
		}
		defer closer2()

		ctx := api.ReqContext(cctx)

		nodeApi, closer, err := api.GetStorageMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		maddr, err := getActorAddress(ctx, nodeApi, cctx.String("actor"))
		if err != nil {
			return err
		}

		var params []miner5.ExtendSectorExpirationParams

		if cctx.Bool("v1-sectors") {

			head, err := fullApi.ChainHead(ctx)
			if err != nil {
				return err
			}

			nv, err := fullApi.StateNetworkVersion(ctx, types.EmptyTSK)
			if err != nil {
				return err
			}

			extensions := map[miner.SectorLocation]map[abi.ChainEpoch][]uint64{}

			// are given durations within tolerance epochs
			withinTolerance := func(a, b abi.ChainEpoch) bool {
				diff := a - b
				if diff < 0 {
					diff = b - a
				}

				return diff <= abi.ChainEpoch(cctx.Int64("tolerance"))
			}

			sis, err := fullApi.StateMinerActiveSectors(ctx, maddr, types.EmptyTSK)
			if err != nil {
				return xerrors.Errorf("getting miner sector infos: %w", err)
			}

			for _, si := range sis {
				if si.SealProof >= abi.RegisteredSealProof_StackedDrg2KiBV1_1 {
					continue
				}

				if si.Expiration < (head.Height() + abi.ChainEpoch(cctx.Int64("expiration-ignore"))) {
					continue
				}

				if cctx.IsSet("expiration-cutoff") {
					if si.Expiration > (head.Height() + abi.ChainEpoch(cctx.Int64("expiration-cutoff"))) {
						continue
					}
				}

				ml := policy.GetSectorMaxLifetime(si.SealProof, nv)
				// if the sector's missing less than "tolerance" of its maximum possible lifetime, don't bother extending it
				if withinTolerance(si.Expiration-si.Activation, ml) {
					continue
				}

				// Set the new expiration to 48 hours less than the theoretical maximum lifetime
				newExp := ml - (miner5.WPoStProvingPeriod * 2) + si.Activation
				if withinTolerance(si.Expiration, newExp) || si.Expiration >= newExp {
					continue
				}

				p, err := fullApi.StateSectorPartition(ctx, maddr, si.SectorNumber, types.EmptyTSK)
				if err != nil {
					return xerrors.Errorf("getting sector location for sector %d: %w", si.SectorNumber, err)
				}

				if p == nil {
					return xerrors.Errorf("sector %d not found in any partition", si.SectorNumber)
				}

				es, found := extensions[*p]
				if !found {
					ne := make(map[abi.ChainEpoch][]uint64)
					ne[newExp] = []uint64{uint64(si.SectorNumber)}
					extensions[*p] = ne
				} else {
					added := false
					for exp := range es {
						if withinTolerance(exp, newExp) && newExp >= exp && exp > si.Expiration {
							es[exp] = append(es[exp], uint64(si.SectorNumber))
							added = true
							break
						}
					}

					if !added {
						es[newExp] = []uint64{uint64(si.SectorNumber)}
					}
				}
			}

			p := miner5.ExtendSectorExpirationParams{}
			scount := 0

			for l, exts := range extensions {
				for newExp, numbers := range exts {
					scount += len(numbers)
					addressedMax, err := policy.GetAddressedSectorsMax(nv)
					if err != nil {
						return xerrors.Errorf("failed to get addressed sectors max")
					}
					declMax, err := policy.GetDeclarationsMax(nv)
					if err != nil {
						return xerrors.Errorf("failed to get declarations max")
					}
					if scount > addressedMax || len(p.Extensions) == declMax {
						params = append(params, p)
						p = miner5.ExtendSectorExpirationParams{}
						scount = len(numbers)
					}

					p.Extensions = append(p.Extensions, miner5.ExpirationExtension{
						Deadline:      l.Deadline,
						Partition:     l.Partition,
						Sectors:       bitfield.NewFromSet(numbers),
						NewExpiration: newExp,
					})
				}
			}

			// if we have any sectors, then one last append is needed here
			if scount != 0 {
				params = append(params, p)
			}

		} else {
			if !cctx.Args().Present() || !cctx.IsSet("new-expiration") {
				return xerrors.Errorf("must pass at least one sector number and new expiration")
			}
			sectors := map[miner.SectorLocation][]uint64{}

			for i, s := range cctx.Args().Slice() {
				id, err := strconv.ParseUint(s, 10, 64)
				if err != nil {
					return xerrors.Errorf("could not parse sector %d: %w", i, err)
				}

				p, err := fullApi.StateSectorPartition(ctx, maddr, abi.SectorNumber(id), types.EmptyTSK)
				if err != nil {
					return xerrors.Errorf("getting sector location for sector %d: %w", id, err)
				}

				if p == nil {
					return xerrors.Errorf("sector %d not found in any partition", id)
				}

				sectors[*p] = append(sectors[*p], id)
			}

			p := miner5.ExtendSectorExpirationParams{}
			for l, numbers := range sectors {

				// TODO: Dedup with above loop
				p.Extensions = append(p.Extensions, miner5.ExpirationExtension{
					Deadline:      l.Deadline,
					Partition:     l.Partition,
					Sectors:       bitfield.NewFromSet(numbers),
					NewExpiration: abi.ChainEpoch(cctx.Int64("new-expiration")),
				})
			}

			params = append(params, p)
		}

		if len(params) == 0 {
			fmt.Println("nothing to extend")
			return nil
		}

		mi, err := fullApi.StateMinerInfo(ctx, maddr, types.EmptyTSK)
		if err != nil {
			return xerrors.Errorf("getting miner info: %w", err)
		}

		for i := range params {
			sp, aerr := actors.SerializeParams(&params[i])
			if aerr != nil {
				return xerrors.Errorf("serializing params: %w", err)
			}

			cid, err := nodeApi.MessagerPushMessage(ctx, &types.Message{
				From:   mi.Worker,
				To:     maddr,
				Method: miner.Methods.ExtendSectorExpiration,

				Value:  big.Zero(),
				Params: sp,
			}, nil)
			if err != nil {
				return xerrors.Errorf("mpool push message: %w", err)
			}

			fmt.Println(cid)
		}

		return nil
	},
}

var sectorsTerminateCmd = &cli.Command{
	Name:      "terminate",
	Usage:     "Terminate sector on-chain then remove (WARNING: This means losing power and collateral for the removed sector)",
	ArgsUsage: "<sectorNum>",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "really-do-it",
			Usage: "pass this flag if you know what you are doing",
		},
	},
	Subcommands: []*cli.Command{
		sectorsTerminateFlushCmd,
		sectorsTerminatePendingCmd,
	},
	Action: func(cctx *cli.Context) error {
		if !cctx.Bool("really-do-it") {
			return xerrors.Errorf("pass --really-do-it to confirm this action")
		}
		nodeApi, closer, err := api.GetStorageMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()
		ctx := api.ReqContext(cctx)
		if cctx.Args().Len() != 1 {
			return xerrors.Errorf("must pass sector number")
		}

		id, err := strconv.ParseUint(cctx.Args().Get(0), 10, 64)
		if err != nil {
			return xerrors.Errorf("could not parse sector number: %w", err)
		}

		return nodeApi.SectorTerminate(ctx, abi.SectorNumber(id))
	},
}

var sectorsTerminateFlushCmd = &cli.Command{
	Name:  "flush",
	Usage: "Send a terminate message if there are sectors queued for termination",
	Action: func(cctx *cli.Context) error {
		nodeApi, closer, err := api.GetStorageMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()
		ctx := api.ReqContext(cctx)

		mcid, err := nodeApi.SectorTerminateFlush(ctx)
		if err != nil {
			return err
		}

		if mcid == "" {
			return xerrors.New("no sectors were queued for termination")
		}

		fmt.Println(mcid)

		return nil
	},
}

var sectorsTerminatePendingCmd = &cli.Command{
	Name:  "pending",
	Usage: "List sector numbers of sectors pending termination",
	Action: func(cctx *cli.Context) error {
		nodeApi, closer, err := api.GetStorageMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()
		nodeAPI, nCloser, err := api.GetFullNodeAPIV2(cctx)
		if err != nil {
			return err
		}
		defer nCloser()
		ctx := api.ReqContext(cctx)

		pending, err := nodeApi.SectorTerminatePending(ctx)
		if err != nil {
			return err
		}

		maddr, err := nodeApi.ActorAddress(ctx)
		if err != nil {
			return err
		}

		dl, err := nodeAPI.StateMinerProvingDeadline(ctx, maddr, types.EmptyTSK)
		if err != nil {
			return xerrors.Errorf("getting proving deadline info failed: %w", err)
		}
		for _, id := range pending {
			loc, err := nodeAPI.StateSectorPartition(ctx, maddr, id.Number, types.EmptyTSK)
			if err != nil {
				return xerrors.Errorf("finding sector partition: %w", err)
			}

			fmt.Print(id.Number)

			if loc.Deadline == (dl.Index+1)%miner.WPoStPeriodDeadlines || // not in next (in case the terminate message takes a while to get on chain)
				loc.Deadline == dl.Index || // not in current
				(loc.Deadline+1)%miner.WPoStPeriodDeadlines == dl.Index { // not in previous
				fmt.Print(" (in proving window)")
			}
			fmt.Println()
		}

		return nil
	},
}

var sectorsRemoveCmd = &cli.Command{
	Name:      "remove",
	Usage:     "Forcefully remove a sector (WARNING: This means losing power and collateral for the removed sector (use 'terminate' for lower penalty))",
	ArgsUsage: "<sectorNum>",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "really-do-it",
			Usage: "pass this flag if you know what you are doing",
		},
	},
	Action: func(cctx *cli.Context) error {
		if !cctx.Bool("really-do-it") {
			return xerrors.Errorf("this is a command for advanced users, only use it if you are sure of what you are doing")
		}
		nodeApi, closer, err := api.GetStorageMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()
		ctx := api.ReqContext(cctx)
		if cctx.Args().Len() != 1 {
			return xerrors.Errorf("must pass sector number")
		}

		id, err := strconv.ParseUint(cctx.Args().Get(0), 10, 64)
		if err != nil {
			return xerrors.Errorf("could not parse sector number: %w", err)
		}

		return nodeApi.SectorRemove(ctx, abi.SectorNumber(id))
	},
}

var sectorsMarkForUpgradeCmd = &cli.Command{
	Name:      "mark-for-upgrade",
	Usage:     "Mark a committed capacity sector for replacement by a sector with deals",
	ArgsUsage: "<sectorNum>",
	Action: func(cctx *cli.Context) error {
		if cctx.Args().Len() != 1 {
			return ShowHelp(cctx, xerrors.Errorf("must pass sector number"))
		}

		nodeApi, closer, err := api.GetStorageMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()
		ctx := api.ReqContext(cctx)

		id, err := strconv.ParseUint(cctx.Args().Get(0), 10, 64)
		if err != nil {
			return xerrors.Errorf("could not parse sector number: %w", err)
		}

		return nodeApi.SectorMarkForUpgrade(ctx, abi.SectorNumber(id))
	},
}

var sectorsStartSealCmd = &cli.Command{
	Name:      "seal",
	Usage:     "Manually start sealing a sector (filling any unused space with junk)",
	ArgsUsage: "<sectorNum>",
	Action: func(cctx *cli.Context) error {
		nodeApi, closer, err := api.GetStorageMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()
		ctx := api.ReqContext(cctx)
		if cctx.Args().Len() != 1 {
			return xerrors.Errorf("must pass sector number")
		}

		id, err := strconv.ParseUint(cctx.Args().Get(0), 10, 64)
		if err != nil {
			return xerrors.Errorf("could not parse sector number: %w", err)
		}

		return nodeApi.SectorStartSealing(ctx, abi.SectorNumber(id))
	},
}

var sectorsSealDelayCmd = &cli.Command{
	Name:      "set-seal-delay",
	Usage:     "Set the time, in minutes, that a new sector waits for deals before sealing starts",
	ArgsUsage: "<minutes>",
	Action: func(cctx *cli.Context) error {
		nodeApi, closer, err := api.GetStorageMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()
		ctx := api.ReqContext(cctx)
		if cctx.Args().Len() != 1 {
			return xerrors.Errorf("must pass duration in minutes")
		}

		hs, err := strconv.ParseUint(cctx.Args().Get(0), 10, 64)
		if err != nil {
			return xerrors.Errorf("could not parse sector number: %w", err)
		}

		delay := hs * uint64(time.Minute)

		return nodeApi.SectorSetSealDelay(ctx, time.Duration(delay))
	},
}

var sectorsCapacityCollateralCmd = &cli.Command{
	Name:  "get-cc-collateral",
	Usage: "Get the collateral required to pledge a committed capacity sector",
	Flags: []cli.Flag{
		&cli.Uint64Flag{
			Name:  "expiration",
			Usage: "the epoch when the sector will expire",
		},
	},
	Action: func(cctx *cli.Context) error {

		mApi, mCloser, err := api.GetStorageMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer mCloser()

		nApi, nCloser, err := api.GetFullNodeAPIV2(cctx)
		if err != nil {
			return err
		}
		defer nCloser()

		ctx := api.ReqContext(cctx)

		maddr, err := mApi.ActorAddress(ctx)
		if err != nil {
			return err
		}

		pci := miner.SectorPreCommitInfo{
			Expiration: abi.ChainEpoch(cctx.Uint64("expiration")),
		}
		if pci.Expiration == 0 {
			pci.Expiration = policy.GetMaxSectorExpirationExtension()
		}
		pc, err := nApi.StateMinerInitialPledgeCollateral(ctx, maddr, pci, types.EmptyTSK)
		if err != nil {
			return err
		}

		fmt.Printf("Estimated collateral: %s\n", types.FIL(pc))

		return nil
	},
}

var sectorsUpdateCmd = &cli.Command{
	Name:      "update-state",
	Usage:     "ADVANCED: manually update the state of a sector, this may aid in error recovery",
	ArgsUsage: "<sectorNum> <newState>",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "really-do-it",
			Usage: "pass this flag if you know what you are doing",
		},
	},
	Action: func(cctx *cli.Context) error {
		if !cctx.Bool("really-do-it") {
			return xerrors.Errorf("this is a command for advanced users, only use it if you are sure of what you are doing")
		}
		nodeApi, closer, err := api.GetStorageMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()
		ctx := api.ReqContext(cctx)
		if cctx.Args().Len() < 2 {
			return xerrors.Errorf("must pass sector number and new state")
		}

		id, err := strconv.ParseUint(cctx.Args().Get(0), 10, 64)
		if err != nil {
			return xerrors.Errorf("could not parse sector number: %w", err)
		}

		newState := cctx.Args().Get(1)
		if _, ok := types2.ExistSectorStateList[types2.SectorState(newState)]; !ok {
			fmt.Printf(" \"%s\" is not a valid state. Possible states for sectors are: \n", newState)
			for state := range types2.ExistSectorStateList {
				fmt.Printf("%s\n", string(state))
			}
			return nil
		}

		return nodeApi.SectorsUpdate(ctx, abi.SectorNumber(id), api.SectorState(cctx.Args().Get(1)))
	},
}

var sectorsBatching = &cli.Command{
	Name:  "batching",
	Usage: "manage batch sector operations",
	Subcommands: []*cli.Command{
		sectorsBatchingPendingCommit,
		sectorsBatchingPendingPreCommit,
	},
}

var sectorsBatchingPendingCommit = &cli.Command{
	Name:  "commit",
	Usage: "list sectors waiting in commit batch queue",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "publish-now",
			Usage: "send a batch now",
		},
	},
	Action: func(cctx *cli.Context) error {
		storageAPI, closer, err := api.GetStorageMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		ctx := api.ReqContext(cctx)

		if cctx.Bool("publish-now") {
			res, err := storageAPI.SectorCommitFlush(ctx)
			if err != nil {
				return xerrors.Errorf("flush: %w", err)
			}
			if res == nil {
				return xerrors.Errorf("no sectors to publish")
			}

			for i, re := range res {
				fmt.Printf("Batch %d:\n", i)
				if re.Error != "" {
					fmt.Printf("\tError: %s\n", re.Error)
				} else {
					fmt.Printf("\tMessage: %s\n", re.Msg)
				}
				fmt.Printf("\tSectors:\n")
				for _, sector := range re.Sectors {
					if e, found := re.FailedSectors[sector]; found {
						fmt.Printf("\t\t%d\tERROR %s\n", sector, e)
					} else {
						fmt.Printf("\t\t%d\tOK\n", sector)
					}
				}
			}
			return nil
		}

		pending, err := storageAPI.SectorCommitPending(ctx)
		if err != nil {
			return xerrors.Errorf("getting pending deals: %w", err)
		}

		if len(pending) > 0 {
			for _, sector := range pending {
				fmt.Println(sector.Number)
			}
			return nil
		}

		fmt.Println("No sectors queued to be committed")
		return nil
	},
}

var sectorsBatchingPendingPreCommit = &cli.Command{
	Name:  "precommit",
	Usage: "list sectors waiting in precommit batch queue",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "publish-now",
			Usage: "send a batch now",
		},
	},
	Action: func(cctx *cli.Context) error {
		storageAPI, closer, err := api.GetStorageMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()
		ctx := api.ReqContext(cctx)

		if cctx.Bool("publish-now") {
			res, err := storageAPI.SectorPreCommitFlush(ctx)
			if err != nil {
				return xerrors.Errorf("flush: %w", err)
			}
			if res == nil {
				return xerrors.Errorf("no sectors to publish")
			}

			for i, re := range res {
				fmt.Printf("Batch %d:\n", i)
				if re.Error != "" {
					fmt.Printf("\tError: %s\n", re.Error)
				} else {
					fmt.Printf("\tMessage: %s\n", re.Msg)
				}
				fmt.Printf("\tSectors:\n")
				for _, sector := range re.Sectors {
					fmt.Printf("\t\t%d\tOK\n", sector)
				}
			}
			return nil
		}

		pending, err := storageAPI.SectorPreCommitPending(ctx)
		if err != nil {
			return xerrors.Errorf("getting pending deals: %w", err)
		}

		if len(pending) > 0 {
			for _, sector := range pending {
				fmt.Println(sector.Number)
			}
			return nil
		}

		fmt.Println("No sectors queued to be committed")
		return nil
	},
}

func yesno(b bool) string {
	if b {
		return color.GreenString("YES")
	}
	return color.RedString("NO")
}
