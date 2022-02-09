package main

import (
	"github.com/docker/go-units"
	paramfetch "github.com/filecoin-project/go-paramfetch"
	"github.com/filecoin-project/venus-sealer/api"
	"github.com/filecoin-project/venus/fixtures/asset"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"
)

var fetchParamCmd = &cli.Command{
	Name:      "fetch-params",
	Usage:     "Fetch proving parameters",
	ArgsUsage: "[sectorSize]",
	Action: func(cctx *cli.Context) error {
		if !cctx.Args().Present() {
			return xerrors.Errorf("must pass sector size to fetch params for (specify as \"32GiB\", for instance)")
		}
		sectorSizeInt, err := units.RAMInBytes(cctx.Args().First())
		if err != nil {
			return xerrors.Errorf("error parsing sector size (specify as \"32GiB\", for instance): %w", err)
		}
		sectorSize := uint64(sectorSizeInt)

		ps, err := asset.Asset("fixtures/_assets/proof-params/parameters.json")
		if err != nil {
			return err
		}
		srs, err := asset.Asset("fixtures/_assets/proof-params/srs-inner-product.json")
		if err != nil {
			return err
		}

		err = paramfetch.GetParams(api.ReqContext(cctx), ps, srs, sectorSize)
		if err != nil {
			return xerrors.Errorf("fetching proof parameters: %w", err)
		}

		return nil
	},
}
