package main

import (
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/venus-sealer/api"
	"github.com/filecoin-project/venus-sealer/lib/tablewriter"
	"github.com/urfave/cli/v2"
	"math"
	"os"
)

var dealsCmd = &cli.Command{
	Name:  "deals",
	Usage: "interact with sector store",
	Subcommands: []*cli.Command{
		dealListCmd,
		updateDealStatusListCmd,
	},
}

var dealListCmd = &cli.Command{
	Name:  "list",
	Usage: "show deal list",
	Flags: []cli.Flag{},
	Action: func(cctx *cli.Context) error {
		nodeApi, closer, err := api.GetStorageMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()
		ctx := api.ReqContext(cctx)

		deals, err := nodeApi.GetDeals(ctx, 0, math.MaxInt32)
		if err != nil {
			return err
		}

		tw := tablewriter.New(
			tablewriter.Col("DealId"),
			tablewriter.Col("PieceCID"),
			tablewriter.Col("PieceSize"),
			tablewriter.Col("Client"),
			tablewriter.Col("Provider"),
			tablewriter.Col("StartEpoch"),
			tablewriter.Col("EndEpoch"),
			tablewriter.Col("Price"),
			tablewriter.Col("Verified"),
			tablewriter.Col("Packed"),
			tablewriter.Col("FastRetrieval"),
			tablewriter.Col("Status"),
		)

		for _, deal := range deals {
			tw.Write(map[string]interface{}{
				"DealId":        deal.DealId,
				"PieceCID":      deal.Proposal.PieceCID,
				"PieceSize":     deal.Proposal.PieceSize,
				"Client":        deal.Proposal.Client,
				"Provider":      deal.Proposal.Provider,
				"StartEpoch":    deal.Proposal.StartEpoch,
				"EndEpoch":      deal.Proposal.EndEpoch,
				"Price":         deal.Proposal.StoragePricePerEpoch,
				"Verified":      deal.Proposal.VerifiedDeal,
				"Packed":        deal.Offset > 0,
				"FastRetrieval": deal.FastRetrieval,
				"Status":        deal.Status,
			})
		}
		return tw.Flush(os.Stdout)
	},
}

var updateDealStatusListCmd = &cli.Command{
	Name:  "update-status",
	Usage: "update deal status",
	Flags: []cli.Flag{
		&cli.Uint64Flag{
			Name:     "id",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "status",
			Required: true,
		},
	},
	Action: func(cctx *cli.Context) error {
		nodeApi, closer, err := api.GetStorageMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()
		ctx := api.ReqContext(cctx)

		uid := cctx.Uint64("id")
		status := cctx.String("status")
		return nodeApi.UpdateDealStatus(ctx, abi.DealID(uid), status)
	},
}
