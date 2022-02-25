package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/filecoin-project/go-state-types/network"

	rlepluslazy "github.com/filecoin-project/go-bitfield/rle"
	cbor "github.com/ipfs/go-ipld-cbor"

	"github.com/fatih/color"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-bitfield"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"

	miner2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/miner"

	"github.com/filecoin-project/venus/venus-shared/actors"
	"github.com/filecoin-project/venus/venus-shared/actors/adt"
	"github.com/filecoin-project/venus/venus-shared/actors/builtin/miner"
	"github.com/filecoin-project/venus/venus-shared/types"

	"github.com/filecoin-project/venus-sealer/api"
	"github.com/filecoin-project/venus-sealer/constants"
	"github.com/filecoin-project/venus-sealer/lib/tablewriter"
)

var actorCmd = &cli.Command{
	Name:  "actor",
	Usage: "manipulate the miner actor",
	Subcommands: []*cli.Command{
		actorSetAddrsCmd,
		actorWithdrawCmd,
		actorRepayDebtCmd,
		actorSetPeeridCmd,
		actorSetOwnerCmd,
		actorControl,
		actorProposeChangeWorker,
		actorConfirmChangeWorker,
		actorCompactAllocatedCmd,
	},
}

var actorSetAddrsCmd = &cli.Command{
	Name:  "set-addrs",
	Usage: "set addresses that your miner can be publicly dialed on",
	Flags: []cli.Flag{
		&cli.Int64Flag{
			Name:  "gas-limit",
			Usage: "set gas limit",
			Value: 0,
		},
		&cli.BoolFlag{
			Name:  "unset",
			Usage: "unset address",
			Value: false,
		},
	},
	Action: func(cctx *cli.Context) error {
		args := cctx.Args().Slice()
		unset := cctx.Bool("unset")
		if len(args) == 0 && !unset {
			return cli.ShowSubcommandHelp(cctx)
		}
		if len(args) > 0 && unset {
			return fmt.Errorf("unset can only be used with no arguments")
		}
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

		var addrs []abi.Multiaddrs
		for _, a := range cctx.Args().Slice() {
			maddr, err := ma.NewMultiaddr(a)
			if err != nil {
				return fmt.Errorf("failed to parse %q as a multiaddr: %w", a, err)
			}

			maddrNop2p, strip := ma.SplitFunc(maddr, func(c ma.Component) bool {
				return c.Protocol().Code == ma.P_P2P
			})

			if strip != nil {
				fmt.Println("Stripping peerid ", strip, " from ", maddr)
			}
			addrs = append(addrs, maddrNop2p.Bytes())
		}

		maddr, err := storageAPI.ActorAddress(ctx)
		if err != nil {
			return err
		}

		minfo, err := nodeAPI.StateMinerInfo(ctx, maddr, types.EmptyTSK)
		if err != nil {
			return err
		}

		params, err := actors.SerializeParams(&miner2.ChangeMultiaddrsParams{NewMultiaddrs: addrs})
		if err != nil {
			return err
		}

		gasLimit := cctx.Int64("gas-limit")

		uid, err := storageAPI.MessagerPushMessage(ctx, &types.Message{
			To:       maddr,
			From:     minfo.Worker,
			Value:    types.NewInt(0),
			GasLimit: gasLimit,
			Method:   miner.Methods.ChangeMultiaddrs,
			Params:   params,
		}, nil)
		if err != nil {
			return err
		}

		fmt.Printf("Requested multiaddrs change in message %s\n", uid)
		return nil

	},
}

var actorSetPeeridCmd = &cli.Command{
	Name:  "set-peer-id",
	Usage: "set the peer id of your miner",
	Flags: []cli.Flag{
		&cli.Int64Flag{
			Name:  "gas-limit",
			Usage: "set gas limit",
			Value: 0,
		},
	},
	Action: func(cctx *cli.Context) error {
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

		pid, err := peer.Decode(cctx.Args().Get(0))
		if err != nil {
			return fmt.Errorf("failed to parse input as a peerId: %w", err)
		}

		maddr, err := storageAPI.ActorAddress(ctx)
		if err != nil {
			return err
		}

		minfo, err := nodeAPI.StateMinerInfo(ctx, maddr, types.EmptyTSK)
		if err != nil {
			return err
		}

		params, err := actors.SerializeParams(&miner2.ChangePeerIDParams{NewID: abi.PeerID(pid)})
		if err != nil {
			return err
		}

		gasLimit := cctx.Int64("gas-limit")

		uid, err := storageAPI.MessagerPushMessage(ctx, &types.Message{
			To:       maddr,
			From:     minfo.Worker,
			Value:    types.NewInt(0),
			GasLimit: gasLimit,
			Method:   miner.Methods.ChangePeerID,
			Params:   params,
		}, nil)
		if err != nil {
			return err
		}

		fmt.Printf("Requested peerid change in message %s\n", uid)
		return nil

	},
}

var actorWithdrawCmd = &cli.Command{
	Name:      "withdraw",
	Usage:     "withdraw available balance",
	ArgsUsage: "[amount (FIL)]",
	Flags: []cli.Flag{
		&cli.IntFlag{
			Name:  "confidence",
			Usage: "number of block confirmations to wait for",
			Value: int(constants.MessageConfidence),
		},
	},
	Action: func(cctx *cli.Context) error {
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

		maddr, err := storageAPI.ActorAddress(ctx)
		if err != nil {
			return err
		}

		mi, err := nodeAPI.StateMinerInfo(ctx, maddr, types.EmptyTSK)
		if err != nil {
			return err
		}

		available, err := nodeAPI.StateMinerAvailableBalance(ctx, maddr, types.EmptyTSK)
		if err != nil {
			return err
		}

		amount := available
		if cctx.Args().Present() {
			f, err := types.ParseFIL(cctx.Args().First())
			if err != nil {
				return xerrors.Errorf("parsing 'amount' argument: %w", err)
			}

			amount = abi.TokenAmount(f)

			if amount.GreaterThan(available) {
				return xerrors.Errorf("can't withdraw more funds than available; requested: %s; available: %s", amount, available)
			}
		}

		params, err := actors.SerializeParams(&miner2.WithdrawBalanceParams{
			AmountRequested: amount, // Default to attempting to withdraw all the extra funds in the miner actor
		})
		if err != nil {
			return err
		}

		uid, err := storageAPI.MessagerPushMessage(ctx, &types.Message{
			To:     maddr,
			From:   mi.Owner,
			Value:  types.NewInt(0),
			Method: miner.Methods.WithdrawBalance,
			Params: params,
		}, nil)
		if err != nil {
			return err
		}

		fmt.Printf("Requested rewards withdrawal in message %s\n", uid)
		// wait for it to get mined into a block
		fmt.Printf("waiting for %d epochs for confirmation..\n", uint64(cctx.Int("confidence")))
		wait, err := storageAPI.MessagerWaitMessage(ctx, uid, uint64(cctx.Int("confidence")))
		if err != nil {
			return err
		}

		// check it executed successfully
		if wait.Receipt.ExitCode != 0 {
			fmt.Println(cctx.App.Writer, "withdrawal failed!")
			return err
		}

		nv, err := nodeAPI.StateNetworkVersion(ctx, wait.TipSet)
		if err != nil {
			return err
		}

		if nv >= network.Version14 {
			var withdrawn abi.TokenAmount
			if err := withdrawn.UnmarshalCBOR(bytes.NewReader(wait.Receipt.Return)); err != nil {
				return err
			}

			fmt.Printf("Successfully withdrew %s FIL\n", withdrawn)
			if withdrawn.LessThan(amount) {
				fmt.Printf("Note that this is less than the requested amount of %s FIL\n", amount)
			}
		}

		return nil
	},
}

var actorRepayDebtCmd = &cli.Command{
	Name:      "repay-debt",
	Usage:     "pay down a miner's debt",
	ArgsUsage: "[amount (FIL)]",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "from",
			Usage: "optionally specify the account to send funds from",
		},
	},
	Action: func(cctx *cli.Context) error {
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

		maddr, err := storageAPI.ActorAddress(ctx)
		if err != nil {
			return err
		}

		mi, err := nodeAPI.StateMinerInfo(ctx, maddr, types.EmptyTSK)
		if err != nil {
			return err
		}

		var amount abi.TokenAmount
		if cctx.Args().Present() {
			f, err := types.ParseFIL(cctx.Args().First())
			if err != nil {
				return xerrors.Errorf("parsing 'amount' argument: %w", err)
			}

			amount = abi.TokenAmount(f)
		} else {
			mact, err := nodeAPI.StateGetActor(ctx, maddr, types.EmptyTSK)
			if err != nil {
				return err
			}

			store := adt.WrapStore(ctx, cbor.NewCborStore(api.NewAPIBlockstore(nodeAPI)))

			mst, err := miner.Load(store, mact)
			if err != nil {
				return err
			}

			amount, err = mst.FeeDebt()
			if err != nil {
				return err
			}

		}

		fromAddr := mi.Worker
		if from := cctx.String("from"); from != "" {
			addr, err := address.NewFromString(from)
			if err != nil {
				return err
			}

			fromAddr = addr
		}

		fromId, err := nodeAPI.StateLookupID(ctx, fromAddr, types.EmptyTSK)
		if err != nil {
			return err
		}

		if !mi.IsController(fromId) {
			return xerrors.Errorf("sender isn't a controller of miner: %s", fromId)
		}

		uid, err := storageAPI.MessagerPushMessage(ctx, &types.Message{
			To:     maddr,
			From:   fromId,
			Value:  amount,
			Method: miner.Methods.RepayDebt,
			Params: nil,
		}, nil)
		if err != nil {
			return err
		}

		fmt.Printf("Sent repay debt message %s\n", uid)

		return nil
	},
}

var actorControl = &cli.Command{
	Name:  "control",
	Usage: "Manage control addresses",
	Subcommands: []*cli.Command{
		actorControlList,
		actorControlSet,
	},
}

var actorControlList = &cli.Command{
	Name:  "list",
	Usage: "Get currently set control addresses",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name: "verbose",
		},
		&cli.BoolFlag{
			Name:        "color",
			Usage:       "use color in display output",
			DefaultText: "depends on output being a TTY",
		},
	},
	Action: func(cctx *cli.Context) error {
		if cctx.IsSet("color") {
			color.NoColor = !cctx.Bool("color")
		}

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

		maddr, err := getActorAddress(ctx, storageAPI, cctx.String("actor"))
		if err != nil {
			return err
		}

		mi, err := nodeAPI.StateMinerInfo(ctx, maddr, types.EmptyTSK)
		if err != nil {
			return err
		}

		tw := tablewriter.New(
			tablewriter.Col("name"),
			tablewriter.Col("ID"),
			tablewriter.Col("key"),
			tablewriter.Col("use"),
			tablewriter.Col("balance"),
		)

		ac, err := storageAPI.ActorAddressConfig(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("miner address config: %v\n", ac)

		commit := map[address.Address]struct{}{}
		precommit := map[address.Address]struct{}{}
		terminate := map[address.Address]struct{}{}
		dealPublish := map[address.Address]struct{}{}
		post := map[address.Address]struct{}{}

		for _, ca := range mi.ControlAddresses {
			post[ca] = struct{}{}
		}

		for _, ca := range ac.PreCommitControl {
			ca, err := nodeAPI.StateLookupID(ctx, ca, types.EmptyTSK)
			if err != nil {
				return err
			}

			delete(post, ca)
			precommit[ca] = struct{}{}
		}

		for _, ca := range ac.CommitControl {
			ca, err := nodeAPI.StateLookupID(ctx, ca, types.EmptyTSK)
			if err != nil {
				return err
			}

			delete(post, ca)
			commit[ca] = struct{}{}
		}

		for _, ca := range ac.TerminateControl {
			ca, err := nodeAPI.StateLookupID(ctx, ca, types.EmptyTSK)
			if err != nil {
				return err
			}

			delete(post, ca)
			terminate[ca] = struct{}{}
		}

		for _, ca := range ac.DealPublishControl {
			ca, err := nodeAPI.StateLookupID(ctx, ca, types.EmptyTSK)
			if err != nil {
				return err
			}

			delete(post, ca)
			dealPublish[ca] = struct{}{}
		}

		printKey := func(name string, a address.Address) {
			b, err := nodeAPI.WalletBalance(ctx, a)
			if err != nil {
				fmt.Printf("%s\t%s: error getting balance: %s\n", name, a, err)
				return
			}

			k, err := nodeAPI.StateAccountKey(ctx, a, types.EmptyTSK)
			if err != nil {
				fmt.Printf("%s\t%s: error getting account key: %s\n", name, a, err)
				return
			}

			kstr := k.String()
			if !cctx.Bool("verbose") {
				kstr = kstr[:9] + "..."
			}

			bstr := types.FIL(b).String()
			switch {
			case b.LessThan(types.FromFil(10)):
				bstr = color.RedString(bstr)
			case b.LessThan(types.FromFil(50)):
				bstr = color.YellowString(bstr)
			default:
				bstr = color.GreenString(bstr)
			}

			var uses []string
			if a == mi.Worker {
				uses = append(uses, color.YellowString("other"))
			}
			if _, ok := post[a]; ok {
				uses = append(uses, color.GreenString("post"))
			}
			if _, ok := precommit[a]; ok {
				uses = append(uses, color.CyanString("precommit"))
			}
			if _, ok := commit[a]; ok {
				uses = append(uses, color.BlueString("commit"))
			}
			if _, ok := terminate[a]; ok {
				uses = append(uses, color.YellowString("terminate"))
			}
			if _, ok := dealPublish[a]; ok {
				uses = append(uses, color.MagentaString("deals"))
			}

			tw.Write(map[string]interface{}{
				"name":    name,
				"ID":      a,
				"key":     kstr,
				"use":     strings.Join(uses, " "),
				"balance": bstr,
			})
		}

		printKey("owner", mi.Owner)
		printKey("worker", mi.Worker)
		for i, ca := range mi.ControlAddresses {
			printKey(fmt.Sprintf("control-%d", i), ca)
		}

		return tw.Flush(os.Stdout)
	},
}

var actorControlSet = &cli.Command{
	Name:      "set",
	Usage:     "Set control address(-es)",
	ArgsUsage: "[...address]",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "really-do-it",
			Usage: "Actually send transaction performing the action",
			Value: false,
		},
	},
	Action: func(cctx *cli.Context) error {
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

		maddr, err := storageAPI.ActorAddress(ctx)
		if err != nil {
			return err
		}

		mi, err := nodeAPI.StateMinerInfo(ctx, maddr, types.EmptyTSK)
		if err != nil {
			return err
		}

		del := map[address.Address]struct{}{}
		existing := map[address.Address]struct{}{}
		for _, controlAddress := range mi.ControlAddresses {
			ka, err := nodeAPI.StateAccountKey(ctx, controlAddress, types.EmptyTSK)
			if err != nil {
				return err
			}

			del[ka] = struct{}{}
			existing[ka] = struct{}{}
		}

		var toSet []address.Address

		for i, as := range cctx.Args().Slice() {
			a, err := address.NewFromString(as)
			if err != nil {
				return xerrors.Errorf("parsing address %d: %w", i, err)
			}

			ka, err := nodeAPI.StateAccountKey(ctx, a, types.EmptyTSK)
			if err != nil {
				return err
			}

			// make sure the address exists on chain
			_, err = nodeAPI.StateLookupID(ctx, ka, types.EmptyTSK)
			if err != nil {
				return xerrors.Errorf("looking up %s: %w", ka, err)
			}

			delete(del, ka)
			toSet = append(toSet, ka)
		}

		for a := range del {
			fmt.Println("Remove", a)
		}
		for _, a := range toSet {
			if _, exists := existing[a]; !exists {
				fmt.Println("Add", a)
			}
		}

		if !cctx.Bool("really-do-it") {
			fmt.Println("Pass --really-do-it to actually execute this action")
			return nil
		}

		cwp := &miner2.ChangeWorkerAddressParams{
			NewWorker:       mi.Worker,
			NewControlAddrs: toSet,
		}

		sp, err := actors.SerializeParams(cwp)
		if err != nil {
			return xerrors.Errorf("serializing params: %w", err)
		}

		uid, err := storageAPI.MessagerPushMessage(ctx, &types.Message{
			From:   mi.Owner,
			To:     maddr,
			Method: miner.Methods.ChangeWorkerAddress,

			Value:  big.Zero(),
			Params: sp,
		}, nil)
		if err != nil {
			return xerrors.Errorf("mpool push: %w", err)
		}

		fmt.Println("Message CID:", uid)

		return nil
	},
}

var actorSetOwnerCmd = &cli.Command{
	Name:      "set-owner",
	Usage:     "Set owner address (this command should be invoked twice, first with the old owner as the senderAddress, and then with the new owner)",
	ArgsUsage: "[newOwnerAddress senderAddress]",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "really-do-it",
			Usage: "Actually send transaction performing the action",
			Value: false,
		},
	},
	Action: func(cctx *cli.Context) error {
		if !cctx.Bool("really-do-it") {
			fmt.Println("Pass --really-do-it to actually execute this action")
			return nil
		}

		if cctx.NArg() != 2 {
			return fmt.Errorf("must pass new owner address and sender address")
		}

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

		na, err := address.NewFromString(cctx.Args().First())
		if err != nil {
			return err
		}

		newAddrId, err := nodeAPI.StateLookupID(ctx, na, types.EmptyTSK)
		if err != nil {
			return err
		}

		fa, err := address.NewFromString(cctx.Args().Get(1))
		if err != nil {
			return err
		}

		fromAddrId, err := nodeAPI.StateLookupID(ctx, fa, types.EmptyTSK)
		if err != nil {
			return err
		}

		maddr, err := storageAPI.ActorAddress(ctx)
		if err != nil {
			return err
		}

		mi, err := nodeAPI.StateMinerInfo(ctx, maddr, types.EmptyTSK)
		if err != nil {
			return err
		}

		if fromAddrId != mi.Owner && fromAddrId != newAddrId {
			return xerrors.New("from address must either be the old owner or the new owner")
		}

		sp, err := actors.SerializeParams(&newAddrId)
		if err != nil {
			return xerrors.Errorf("serializing params: %w", err)
		}

		uid, err := storageAPI.MessagerPushMessage(ctx, &types.Message{
			From:   fromAddrId,
			To:     maddr,
			Method: miner.Methods.ChangeOwnerAddress,
			Value:  big.Zero(),
			Params: sp,
		}, nil)
		if err != nil {
			return xerrors.Errorf("mpool push: %w", err)
		}

		fmt.Println("Message CID:", uid)

		// wait for it to get mined into a block
		wait, err := storageAPI.MessagerWaitMessage(ctx, uid, constants.MessageConfidence)
		if err != nil {
			return err
		}

		// check it executed successfully
		if wait.Receipt.ExitCode != 0 {
			fmt.Println("owner change failed!")
			return err
		}

		fmt.Println("message succeeded!")

		return nil
	},
}

var actorProposeChangeWorker = &cli.Command{
	Name:      "propose-change-worker",
	Usage:     "Propose a worker address change",
	ArgsUsage: "[address]",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "really-do-it",
			Usage: "Actually send transaction performing the action",
			Value: false,
		},
	},
	Action: func(cctx *cli.Context) error {
		if !cctx.Args().Present() {
			return fmt.Errorf("must pass address of new worker address")
		}

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

		na, err := address.NewFromString(cctx.Args().First())
		if err != nil {
			return err
		}

		newAddr, err := nodeAPI.StateLookupID(ctx, na, types.EmptyTSK)
		if err != nil {
			return err
		}

		maddr, err := storageAPI.ActorAddress(ctx)
		if err != nil {
			return err
		}

		mi, err := nodeAPI.StateMinerInfo(ctx, maddr, types.EmptyTSK)
		if err != nil {
			return err
		}

		if mi.NewWorker.Empty() {
			if mi.Worker == newAddr {
				return fmt.Errorf("worker address already set to %s", na)
			}
		} else {
			if mi.NewWorker == newAddr {
				return fmt.Errorf("change to worker address %s already pending", na)
			}
		}

		if !cctx.Bool("really-do-it") {
			fmt.Fprintln(cctx.App.Writer, "Pass --really-do-it to actually execute this action")
			return nil
		}

		cwp := &miner2.ChangeWorkerAddressParams{
			NewWorker:       newAddr,
			NewControlAddrs: mi.ControlAddresses,
		}

		sp, err := actors.SerializeParams(cwp)
		if err != nil {
			return xerrors.Errorf("serializing params: %w", err)
		}

		uid, err := storageAPI.MessagerPushMessage(ctx, &types.Message{
			From:   mi.Owner,
			To:     maddr,
			Method: miner.Methods.ChangeWorkerAddress,
			Value:  big.Zero(),
			Params: sp,
		}, nil)
		if err != nil {
			return xerrors.Errorf("mpool push: %w", err)
		}

		fmt.Fprintln(cctx.App.Writer, "Propose Message Uid:", uid)

		// wait for it to get mined into a block
		wait, err := storageAPI.MessagerWaitMessage(ctx, uid, constants.MessageConfidence)
		if err != nil {
			return err
		}

		// check it executed successfully
		if wait.Receipt.ExitCode != 0 {
			fmt.Fprintln(cctx.App.Writer, "Propose worker change failed!")
			return err
		}

		mi, err = nodeAPI.StateMinerInfo(ctx, maddr, wait.TipSet)
		if err != nil {
			return err
		}
		if mi.NewWorker != newAddr {
			return fmt.Errorf("Proposed worker address change not reflected on chain: expected '%s', found '%s'", na, mi.NewWorker)
		}

		fmt.Fprintf(cctx.App.Writer, "Worker key change to %s successfully proposed.\n", na)
		fmt.Fprintf(cctx.App.Writer, "Call 'confirm-change-worker' at or after height %d to complete.\n", mi.WorkerChangeEpoch)

		return nil
	},
}

var actorConfirmChangeWorker = &cli.Command{
	Name:      "confirm-change-worker",
	Usage:     "Confirm a worker address change",
	ArgsUsage: "[address]",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "really-do-it",
			Usage: "Actually send transaction performing the action",
			Value: false,
		},
	},
	Action: func(cctx *cli.Context) error {
		if !cctx.Args().Present() {
			return fmt.Errorf("must pass address of new worker address")
		}

		storageApi, closer, err := api.GetStorageMinerAPI(cctx)
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

		na, err := address.NewFromString(cctx.Args().First())
		if err != nil {
			return err
		}

		newAddr, err := nodeAPI.StateLookupID(ctx, na, types.EmptyTSK)
		if err != nil {
			return err
		}

		maddr, err := storageApi.ActorAddress(ctx)
		if err != nil {
			return err
		}

		mi, err := nodeAPI.StateMinerInfo(ctx, maddr, types.EmptyTSK)
		if err != nil {
			return err
		}

		if mi.NewWorker.Empty() {
			return xerrors.Errorf("no worker key change proposed")
		} else if mi.NewWorker != newAddr {
			return xerrors.Errorf("worker key %s does not match current worker key proposal %s", newAddr, mi.NewWorker)
		}

		if head, err := nodeAPI.ChainHead(ctx); err != nil {
			return xerrors.Errorf("failed to get the chain head: %w", err)
		} else if head.Height() < mi.WorkerChangeEpoch {
			return xerrors.Errorf("worker key change cannot be confirmed until %d, current height is %d", mi.WorkerChangeEpoch, head.Height())
		}

		if !cctx.Bool("really-do-it") {
			fmt.Fprintln(cctx.App.Writer, "Pass --really-do-it to actually execute this action")
			return nil
		}

		uid, err := storageApi.MessagerPushMessage(ctx, &types.Message{
			From:   mi.Owner,
			To:     maddr,
			Method: miner.Methods.ConfirmUpdateWorkerKey,
			Value:  big.Zero(),
		}, nil)
		if err != nil {
			return xerrors.Errorf("mpool push: %w", err)
		}

		fmt.Fprintln(cctx.App.Writer, "Confirm Message CID:", uid)

		// wait for it to get mined into a block
		wait, err := storageApi.MessagerWaitMessage(ctx, uid, constants.MessageConfidence)
		if err != nil {
			return err
		}

		// check it executed successfully
		if wait.Receipt.ExitCode != 0 {
			fmt.Fprintln(cctx.App.Writer, "Worker change failed!")
			return err
		}

		mi, err = nodeAPI.StateMinerInfo(ctx, maddr, wait.TipSet)
		if err != nil {
			return err
		}
		if mi.Worker != newAddr {
			return fmt.Errorf("Confirmed worker address change not reflected on chain: expected '%s', found '%s'", newAddr, mi.Worker)
		}

		return nil
	},
}

var actorCompactAllocatedCmd = &cli.Command{
	Name:  "compact-allocated",
	Usage: "compact allocated sectors bitfield",
	Flags: []cli.Flag{
		&cli.Uint64Flag{
			Name:  "mask-last-offset",
			Usage: "Mask sector IDs from 0 to 'higest_allocated - offset'",
		},
		&cli.Uint64Flag{
			Name:  "mask-upto-n",
			Usage: "Mask sector IDs from 0 to 'n'",
		},
		&cli.BoolFlag{
			Name:  "really-do-it",
			Usage: "Actually send transaction performing the action",
			Value: false,
		},
	},
	Action: func(cctx *cli.Context) error {
		if !cctx.Bool("really-do-it") {
			fmt.Println("Pass --really-do-it to actually execute this action")
			return nil
		}

		if !cctx.Args().Present() {
			return fmt.Errorf("must pass address of new owner address")
		}

		storageApi, closer, err := api.GetStorageMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		nodeApi, acloser, err := api.GetFullNodeAPIV2(cctx)
		if err != nil {
			return err
		}
		defer acloser()

		ctx := api.ReqContext(cctx)

		maddr, err := storageApi.ActorAddress(ctx)
		if err != nil {
			return err
		}

		mact, err := nodeApi.StateGetActor(ctx, maddr, types.EmptyTSK)
		if err != nil {
			return err
		}

		store := adt.WrapStore(ctx, cbor.NewCborStore(api.NewAPIBlockstore(nodeApi)))

		mst, err := miner.Load(store, mact)
		if err != nil {
			return err
		}

		allocs, err := mst.GetAllocatedSectors()
		if err != nil {
			return err
		}

		var maskBf bitfield.BitField

		{
			exclusiveFlags := []string{"mask-last-offset", "mask-upto-n"}
			hasFlag := false
			for _, f := range exclusiveFlags {
				if hasFlag && cctx.IsSet(f) {
					return xerrors.Errorf("more than one 'mask` flag set")
				}
				hasFlag = hasFlag || cctx.IsSet(f)
			}
		}
		switch {
		case cctx.IsSet("mask-last-offset"):
			last, err := allocs.Last()
			if err != nil {
				return err
			}

			m := cctx.Uint64("mask-last-offset")
			if last <= m+1 {
				return xerrors.Errorf("highest allocated sector lower than mask offset %d: %d", m+1, last)
			}
			// securty to not brick a miner
			if last > 1<<60 {
				return xerrors.Errorf("very high last sector number, refusing to mask: %d", last)
			}

			maskBf, err = bitfield.NewFromIter(&rlepluslazy.RunSliceIterator{
				Runs: []rlepluslazy.Run{{Val: true, Len: last - m}}})
			if err != nil {
				return xerrors.Errorf("forming bitfield: %w", err)
			}
		case cctx.IsSet("mask-upto-n"):
			n := cctx.Uint64("mask-upto-n")
			maskBf, err = bitfield.NewFromIter(&rlepluslazy.RunSliceIterator{
				Runs: []rlepluslazy.Run{{Val: true, Len: n}}})
			if err != nil {
				return xerrors.Errorf("forming bitfield: %w", err)
			}
		default:
			return xerrors.Errorf("no 'mask' flags set")
		}

		mi, err := nodeApi.StateMinerInfo(ctx, maddr, types.EmptyTSK)
		if err != nil {
			return err
		}

		params := &miner2.CompactSectorNumbersParams{
			MaskSectorNumbers: maskBf,
		}

		sp, err := actors.SerializeParams(params)
		if err != nil {
			return xerrors.Errorf("serializing params: %w", err)
		}

		uid, err := storageApi.MessagerPushMessage(ctx, &types.Message{
			From:   mi.Worker,
			To:     maddr,
			Method: miner.Methods.CompactSectorNumbers,
			Value:  big.Zero(),
			Params: sp,
		}, nil)
		if err != nil {
			return xerrors.Errorf("mpool push: %w", err)
		}

		fmt.Println("CompactSectorNumbers Message CID:", uid)

		// wait for it to get mined into a block
		wait, err := storageApi.MessagerWaitMessage(ctx, uid, constants.MessageConfidence)
		if err != nil {
			return err
		}

		// check it executed successfully
		if wait.Receipt.ExitCode != 0 {
			fmt.Println("Propose owner change failed!")
			return err
		}

		return nil
	},
}
