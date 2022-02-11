package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/filecoin-project/venus-market/piecestorage"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"github.com/filecoin-project/go-address"
	paramfetch "github.com/filecoin-project/go-paramfetch"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	market2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/market"
	power2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/power"
	"github.com/filecoin-project/venus/fixtures/asset"
	"github.com/filecoin-project/venus/pkg/gen/genesis"

	power6 "github.com/filecoin-project/specs-actors/v6/actors/builtin/power"

	"github.com/docker/go-units"
	"github.com/google/uuid"
	"github.com/mitchellh/go-homedir"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"

	"github.com/ipfs-force-community/venus-common-utils/apiinfo"

	actors "github.com/filecoin-project/venus/venus-shared/actors"
	"github.com/filecoin-project/venus/venus-shared/actors/builtin/miner"
	"github.com/filecoin-project/venus/venus-shared/actors/builtin/power"
	"github.com/filecoin-project/venus/venus-shared/actors/policy"
	"github.com/filecoin-project/venus/venus-shared/types"

	types3 "github.com/filecoin-project/venus/venus-shared/types/messager"

	"github.com/filecoin-project/venus-sealer/api"
	"github.com/filecoin-project/venus-sealer/config"
	"github.com/filecoin-project/venus-sealer/constants"
	"github.com/filecoin-project/venus-sealer/models"
	"github.com/filecoin-project/venus-sealer/sector-storage/stores"
	"github.com/filecoin-project/venus-sealer/service"
	types2 "github.com/filecoin-project/venus-sealer/types"
)

var initCmd = &cli.Command{
	Name:  "init",
	Usage: "Initialize a venus sealer repo",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "actor",
			Usage: "specify the address of an already created miner actor",
		},
		&cli.BoolFlag{
			Name:   "genesis-miner",
			Usage:  "enable genesis mining (DON'T USE ON BOOTSTRAPPED NETWORK)",
			Hidden: true,
		},
		&cli.StringFlag{
			Name:    "worker",
			Aliases: []string{"w"},
			Usage:   "worker key to use",
		},
		&cli.StringFlag{
			Name:    "owner",
			Aliases: []string{"o"},
			Usage:   "owner key to use",
		},
		&cli.StringFlag{
			Name:  "sector-size",
			Usage: "specify sector size to use",
			Value: units.BytesSize(float64(policy.GetDefaultSectorSize())),
		},
		&cli.StringSliceFlag{
			Name:  "pre-sealed-sectors",
			Usage: "specify set of presealed sectors for starting as a genesis miner",
		},
		&cli.StringFlag{
			Name:  "pre-sealed-metadata",
			Usage: "specify the metadata file for the presealed sectors",
		},
		&cli.BoolFlag{
			Name:  "nosync",
			Usage: "don't check full-node sync status",
		},
		&cli.BoolFlag{
			Name:  "symlink-imported-sectors",
			Usage: "attempt to symlink to presealed sectors instead of copying them into place",
		},
		&cli.BoolFlag{
			Name:  "no-local-storage",
			Usage: "don't use storageminer repo for sector storage",
		},
		&cli.StringFlag{
			Name:  "gas-premium",
			Usage: "set gas premium for initialization messages in AttoFIL",
			Value: "0",
		},
		&cli.StringFlag{
			Name:  "from",
			Usage: "select which address to send actor creation message from",
		},
		&cli.StringFlag{
			Name:  "messager-url",
			Usage: "messager url",
		},
		&cli.StringFlag{
			Name:  "messager-token",
			Usage: "messager token",
		},
		&cli.StringFlag{
			Name:  "node-url",
			Usage: "node url",
		},
		&cli.StringFlag{
			Name:  "node-token",
			Usage: "node token",
		},

		&cli.StringSliceFlag{
			Name:  "gateway-url",
			Usage: "gateway url",
		},
		&cli.StringFlag{
			Name:  "gateway-token",
			Usage: "gateway token",
		},

		&cli.StringFlag{
			Name:  "market-mode",
			Value: "solo",
			Usage: "indicate the deployment method of the venus-market, one of `solo`, `pool`, Default: `solo`",
		},
		&cli.StringFlag{
			Name:  "market-url",
			Usage: "market url",
		},
		&cli.StringFlag{
			Name:  "market-token",
			Usage: "market token",
		},

		&cli.StringFlag{
			Name:  "auth-token",
			Usage: "auth token",
		},

		&cli.StringFlag{
			Name:  "piecestorage",
			Usage: "config storage for piece  (eg  fs:/mnt/piece   s3:{access key}:{secret key}:{option token}@{region}host/{bucket}",
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := api.ReqContext(cctx)

		log.Info("Initializing venus sealer")

		symlink := cctx.Bool("symlink-imported-sectors")
		if symlink {
			log.Info("will attempt to symlink to imported sectors")
		}

		network := cctx.String("network")
		defaultCfg, err := config.GetDefaultStorageConfig(network)
		if err != nil {
			return err
		}

		err = parseServiceFlag(defaultCfg, cctx)
		if err != nil {
			return err
		}

		if err := checkServiceConfig(defaultCfg); err != nil {
			return err
		}

		log.Info("Checking if repo exists")

		defaultCfg.ConfigPath = config.FsConfig(defaultCfg.DataDir)
		exit, err := config.ConfigExist(defaultCfg.DataDir)
		if err != nil {
			return err
		}
		if exit {
			return xerrors.Errorf("repo is already initialized at %s", defaultCfg.DataDir)
		}

		log.Info("Checking proof parameters")

		ps, err := asset.Asset("fixtures/_assets/proof-params/parameters.json")
		if err != nil {
			return err
		}
		srs, err := asset.Asset("fixtures/_assets/proof-params/srs-inner-product.json")
		if err != nil {
			return err
		}

		log.Info("Trying to connect to full node RPC")

		fullNode, closer, err := api.GetFullNodeFromNodeConfig(ctx, &defaultCfg.Node) // TODO: consider storing full node address in config
		if err != nil {
			return err
		}
		defer closer()

		log.Info("Checking full node sync status")

		if !cctx.Bool("genesis-miner") && !cctx.Bool("nosync") {
			if err := api.SyncWait(ctx, fullNode, defaultCfg.NetParams.BlockDelaySecs, false); err != nil {
				return xerrors.Errorf("sync wait: %w", err)
			}
		}

		log.Info("Checking full node version")

		v, err := fullNode.Version(ctx)
		if err != nil {
			return err
		}

		if !v.APIVersion.EqMajorMinor(constants.FullAPIVersion1) {
			return xerrors.Errorf("Remote API version didn't match (expected %s, remote %s)", constants.FullAPIVersion1, v.APIVersion)
		}

		log.Info("Initializing repo")
		{
			// write config
			err = config.SaveConfig(defaultCfg.ConfigPath, defaultCfg)
			if err != nil {
				return err
			}

			var localPaths []stores.LocalPath

			if pssb := cctx.StringSlice("pre-sealed-sectors"); len(pssb) != 0 {
				log.Infof("Setting up storage config with presealed sectors: %v", pssb)

				for _, psp := range pssb {
					psp, err := homedir.Expand(psp)
					if err != nil {
						return err
					}
					localPaths = append(localPaths, stores.LocalPath{
						Path: psp,
					})
				}
			}

			if !cctx.Bool("no-local-storage") {
				b, err := json.MarshalIndent(&stores.LocalStorageMeta{
					ID:       stores.ID(uuid.New().String()),
					Weight:   10,
					CanSeal:  true,
					CanStore: true,
				}, "", "  ")
				if err != nil {
					return xerrors.Errorf("marshaling storage config: %w", err)
				}
				dataDir, err := homedir.Expand(defaultCfg.DataDir)
				if err != nil {
					return err
				}
				if err := ioutil.WriteFile(filepath.Join(dataDir, "sectorstore.json"), b, 0644); err != nil {
					return xerrors.Errorf("persisting storage metadata (%s): %w", filepath.Join(dataDir, "sectorstore.json"), err)
				}

				localPaths = append(localPaths, stores.LocalPath{
					Path: dataDir,
				})
			}

			localStorage := defaultCfg.LocalStorage()
			if err := localStorage.SetStorage(func(sc *stores.StorageConfig) {
				sc.StoragePaths = append(sc.StoragePaths, localPaths...)
			}); err != nil {
				return xerrors.Errorf("set storage config: %w", err)
			}
		}

		messagerClient, closer, err := api.NewMessageRPC(&defaultCfg.Messager)
		if err != nil {
			return err
		}
		defer closer()

		ssize, err := units.RAMInBytes(cctx.String("sector-size"))
		if err != nil {
			return fmt.Errorf("failed to parse sector size: %w", err)
		}

		gasPrice, err := types.BigFromString(cctx.String("gas-premium"))
		if err != nil {
			return xerrors.Errorf("failed to parse gas-price flag: %s", err)
		}

		minerAddr, err := storageMinerInit(ctx, cctx, fullNode, messagerClient, defaultCfg, abi.SectorSize(ssize), gasPrice)
		if err != nil {
			log.Errorf("Failed to initialize venus-miner: %+v", err)
			path, err := homedir.Expand(defaultCfg.DataDir)
			if err != nil {
				return err
			}
			log.Infof("Cleaning up %s after attempt...", path)
			if err := os.RemoveAll(path); err != nil {
				log.Errorf("Failed to clean up failed storage repo: %s", err)
			}
			return xerrors.Errorf("Storage-miner init failed")
		}

		minerInfo, err := fullNode.StateMinerInfo(ctx, minerAddr, types.EmptyTSK)
		if err != nil {
			return err
		}
		// TODO: Point to setting storage price, maybe do it interactively or something
		log.Info("Sealer successfully created, you can now start it with 'venus-sealer run'")
		if err := paramfetch.GetParams(ctx, ps, srs, uint64(minerInfo.SectorSize)); err != nil {
			return xerrors.Errorf("fetching proof parameters: %w", err)
		}
		return nil
	},
}

func parseServiceFlag(cfg *config.StorageMiner, cctx *cli.Context) error {
	cfg.DataDir = cctx.String("repo")

	if cctx.IsSet("auth-token") {
		authToken := cctx.String("auth-token")

		cfg.Node.Token = authToken

		cfg.Messager.Token = authToken

		cfg.RegisterProof.Token = authToken
	}

	if cctx.IsSet("node-url") {
		cfg.Node.Url = cctx.String("node-url")
	}
	if cctx.IsSet("node-token") {
		cfg.Node.Token = cctx.String("node-token")
	}

	if cctx.IsSet("messager-url") {
		cfg.Messager.Url = cctx.String("messager-url")
	}
	if cctx.IsSet("messager-token") {
		cfg.Messager.Token = cctx.String("messager-token")
	}

	if cctx.IsSet("gateway-url") {
		cfg.RegisterProof.Urls = cctx.StringSlice("gateway-url")
	}
	if cctx.IsSet("gateway-token") {
		cfg.RegisterProof.Token = cctx.String("gateway-token")
	}

	if cctx.IsSet("market-mode") {
		mode := cctx.String("market-mode")
		if cctx.IsSet("market-url") {
			cfg.MarketNode.Url = cctx.String("market-url")
		} else {
			return xerrors.Errorf("must set market url when set market mode")
		}

		if cctx.IsSet("auth-token") {
			authToken := cctx.String("auth-token")
			cfg.MarketNode.Token = authToken
		}

		if cctx.IsSet("market-token") {
			cfg.MarketNode.Token = cctx.String("market-token")
		}

		if mode == "solo" { // when venus-market is deployed independently, it not only provides services but also handles events
			cfg.RegisterMarket.Urls = []string{cfg.MarketNode.Url}
			cfg.RegisterMarket.Token = cfg.MarketNode.Token
		} else {
			cfg.RegisterMarket.Urls = cfg.RegisterProof.Urls
			cfg.RegisterMarket.Token = cfg.RegisterProof.Token
		}
	}

	if cctx.IsSet("piecestorage") {
		if err := piecestorage.ParserProtocol(cctx.String("piecestorage"), &cfg.PieceStorage); err != nil {
			return err
		}
	}
	return nil
}

func parseMultiAddr(url string) error {
	ai := apiinfo.APIInfo{Addr: url}

	str, err := ai.DialArgs("v0")

	log.Infof("parse url: %s, multi_url: %s", url, str)

	return err
}

func checkServiceConfig(cfg *config.StorageMiner) error {
	if err := parseMultiAddr(cfg.Messager.Url); err != nil {
		return xerrors.Errorf("messager node url: %w", err)
	}

	if err := parseMultiAddr(cfg.Node.Url); err != nil {
		return xerrors.Errorf("node url: %w", err)
	}

	for _, url := range cfg.RegisterProof.Urls {
		if err := parseMultiAddr(url); err != nil {
			return xerrors.Errorf("gateway node url:[%s]: %w", url, err)
		}
	}

	if err := parseMultiAddr(cfg.MarketNode.Url); err != nil {
		return xerrors.Errorf("market node url: %w", err)
	}

	for _, url := range cfg.RegisterMarket.Urls {
		if err := parseMultiAddr(url); err != nil {
			return xerrors.Errorf("register market node url:[%s]: %w", url, err)
		}
	}

	return nil
}

func storageMinerInit(ctx context.Context, cctx *cli.Context, api api.FullNode, messagerClient api.IMessager, cfg *config.StorageMiner, ssize abi.SectorSize, gasPrice types.BigInt) (address.Address, error) {
	log.Info("Initializing libp2p identity")

	repo, err := models.SetDataBase(config.HomeDir(cfg.DataDir), &cfg.DB)
	if err != nil {
		return address.Undef, err
	}
	err = repo.AutoMigrate()
	if err != nil {
		return address.Undef, err
	}

	metaDataService := service.NewMetadataService(repo)
	sectorInfoService := service.NewSectorInfoService(repo)

	var addr address.Address
	if act := cctx.String("actor"); act != "" {
		a, err := address.NewFromString(act)
		if err != nil {
			return address.Undef, xerrors.Errorf("failed parsing actor flag value (%q): %w", act, err)
		}

		if cctx.Bool("genesis-miner") {
			if err := metaDataService.SaveMinerAddress(a); err != nil {
				return address.Undef, err
			}

			if pssb := cctx.String("pre-sealed-metadata"); pssb != "" {
				pssb, err := homedir.Expand(pssb)
				if err != nil {
					return address.Undef, err
				}

				log.Infof("Importing pre-sealed sector metadata for %s", a)

				if err := migratePreSealMeta(ctx, api, pssb, a, metaDataService, sectorInfoService); err != nil {
					return address.Undef, xerrors.Errorf("migrating presealed sector metadata: %w", err)
				}
			}

			return a, nil
		}

		if pssb := cctx.String("pre-sealed-metadata"); pssb != "" {
			pssb, err := homedir.Expand(pssb)
			if err != nil {
				return address.Undef, err
			}

			log.Infof("Importing pre-sealed sector metadata for %s", a)

			if err := migratePreSealMeta(ctx, api, pssb, a, metaDataService, sectorInfoService); err != nil {
				return address.Undef, xerrors.Errorf("migrating presealed sector metadata: %w", err)
			}
		}

		addr = a
	} else {
		a, err := createStorageMiner(ctx, api, messagerClient, gasPrice, cctx)
		if err != nil {
			return address.Undef, xerrors.Errorf("creating miner failed: %w", err)
		}

		addr = a
	}

	log.Infof("Created new miner: %s", addr)

	if err := metaDataService.SaveMinerAddress(addr); err != nil {
		return address.Undef, err
	}

	return addr, nil
}

func createStorageMiner(ctx context.Context, nodeAPI api.FullNode, messagerClient api.IMessager, gasPrice types.BigInt, cctx *cli.Context) (address.Address, error) {
	var err error
	var owner address.Address
	if cctx.String("owner") != "" {
		owner, err = address.NewFromString(cctx.String("owner"))
	} else {
		owner, err = nodeAPI.WalletDefaultAddress(ctx)
	}
	if err != nil {
		return address.Undef, err
	}

	ssize, err := units.RAMInBytes(cctx.String("sector-size"))
	if err != nil {
		return address.Undef, fmt.Errorf("failed to parse sector size: %w", err)
	}

	worker := owner
	if cctx.String("worker") != "" {
		worker, err = address.NewFromString(cctx.String("worker"))
		if err != nil {
			return address.Address{}, err
		}
	}

	sender := owner
	if fromstr := cctx.String("from"); fromstr != "" {
		faddr, err := address.NewFromString(fromstr)
		if err != nil {
			return address.Undef, fmt.Errorf("could not parse from address: %w", err)
		}
		sender = faddr
	}

	// make sure the sender account exists on chain
	_, err = nodeAPI.StateLookupID(ctx, owner, types.EmptyTSK)
	if err != nil {
		return address.Undef, xerrors.Errorf("sender must exist on chain: %w", err)
	}

	// make sure the worker account exists on chain
	_, err = nodeAPI.StateLookupID(ctx, worker, types.EmptyTSK)
	if err != nil {
		msgUid, err := messagerClient.PushMessage(ctx, &types.Message{
			From:  sender,
			To:    worker,
			Value: types.NewInt(0),
		}, nil)
		if err != nil {
			return address.Undef, xerrors.Errorf("push worker init: %w", err)
		}

		log.Infof("Initializing worker account %s, message uid: %s", worker, msgUid)
		log.Infof("Waiting for confirmation")

		mw, err := messagerClient.WaitMessage(ctx, msgUid, constants.MessageConfidence)
		if err != nil {
			return address.Undef, xerrors.Errorf("waiting for worker init: %w", err)
		}
		if mw.Receipt.ExitCode != 0 {
			return address.Undef, xerrors.Errorf("initializing worker account failed: exit code %d", mw.Receipt.ExitCode)
		}
	}

	// make sure the owner account exists on chain
	_, err = nodeAPI.StateLookupID(ctx, owner, types.EmptyTSK)
	if err != nil {
		msgUid, err := messagerClient.PushMessage(ctx, &types.Message{
			From:  sender,
			To:    owner,
			Value: types.NewInt(0),
		}, nil)
		if err != nil {
			return address.Undef, xerrors.Errorf("push owner init: %w", err)
		}

		log.Infof("Initializing owner account %s, message: %s", worker, msgUid)
		log.Infof("Waiting for confirmation")

		mw, err := messagerClient.WaitMessage(ctx, msgUid, constants.MessageConfidence)
		if err != nil {
			return address.Undef, xerrors.Errorf("waiting for owner init: %w", err)
		}
		if mw.Receipt.ExitCode != 0 {
			return address.Undef, xerrors.Errorf("initializing owner account failed: exit code %d", mw.Receipt.ExitCode)
		}
	}

	// Note: the correct thing to do would be to call SealProofTypeFromSectorSize if actors version is v3 or later, but this still works
	spt, err := miner.WindowPoStProofTypeFromSectorSize(abi.SectorSize(ssize))
	if err != nil {
		return address.Undef, xerrors.Errorf("getting post proof type: %w", err)
	}

	params, err := actors.SerializeParams(&power6.CreateMinerParams{
		Owner:               owner,
		Worker:              worker,
		WindowPoStProofType: spt,
	})
	if err != nil {
		return address.Undef, err
	}

	createStorageMinerMsg := &types.Message{
		To:    power.Address,
		From:  sender,
		Value: big.Zero(),

		Method: power.Methods.CreateMiner,
		Params: params,

		GasLimit:   0,
		GasPremium: gasPrice,
	}

	msgUid, err := messagerClient.PushMessage(ctx, createStorageMinerMsg, &types3.SendSpec{MaxFee: types.FromFil(1)})
	if err != nil {
		return address.Undef, xerrors.Errorf("pushing createMiner message: %w", err)
	}

	log.Infof("Pushed CreateMiner message: %s", msgUid)
	log.Infof("Waiting for confirmation")

	mw, err := messagerClient.WaitMessage(ctx, msgUid, constants.MessageConfidence)
	if err != nil {
		return address.Undef, xerrors.Errorf("waiting for createMiner message: %w", err)
	}

	if mw.Receipt.ExitCode != 0 {
		return address.Undef, xerrors.Errorf("create miner failed: exit code %d", mw.Receipt.ExitCode)
	}

	var retval power2.CreateMinerReturn
	if err := retval.UnmarshalCBOR(bytes.NewReader(mw.Receipt.Return)); err != nil {
		return address.Undef, err
	}

	log.Infof("New miners address is: %s (%s)", retval.IDAddress, retval.RobustAddress)
	return retval.IDAddress, nil
}

func migratePreSealMeta(ctx context.Context, api api.FullNode, metadata string, maddr address.Address, metadataService *service.MetadataService, sectorInfoService *service.SectorInfoService) error {
	metadata, err := homedir.Expand(metadata)
	if err != nil {
		return xerrors.Errorf("expanding preseal dir: %w", err)
	}

	b, err := ioutil.ReadFile(metadata)
	if err != nil {
		return xerrors.Errorf("reading preseal metadata: %w", err)
	}

	apsm := map[string]genesis.Miner{}
	if err := json.Unmarshal(b, &apsm); err != nil {
		return xerrors.Errorf("unmarshaling preseal metadata: %w", err)
	}

	psm := map[address.Address]genesis.Miner{}
	for addrStr, miner := range apsm {
		addr, err := address.NewFromString(addrStr)
		if err != nil {
			return xerrors.Errorf("unable to decode address : %w", err)
		}
		psm[addr] = miner
	}
	meta, ok := psm[maddr]
	if !ok {
		return xerrors.Errorf("preseal file didn't contain metadata for miner %s", maddr)
	}

	maxSectorID := abi.SectorNumber(0)
	for _, sector := range meta.Sectors {
		//	sectorKey := datastore.NewKey(sealing.SectorStorePrefix).ChildString(fmt.Sprint(sector.SectorID))

		dealID, err := findMarketDealID(ctx, api, sector.Deal)
		if err != nil {
			return xerrors.Errorf("finding storage deal for pre-sealed sector %d: %w", sector.SectorID, err)
		}
		commD := sector.CommD
		commR := sector.CommR

		info := &types2.SectorInfo{
			State:        types2.Proving,
			SectorNumber: sector.SectorID,
			Pieces: []types2.Piece{
				{
					Piece: abi.PieceInfo{
						Size:     abi.PaddedPieceSize(meta.SectorSize),
						PieceCID: commD,
					},
					DealInfo: &types2.PieceDealInfo{
						DealID: dealID,
						DealSchedule: types2.DealSchedule{
							StartEpoch: sector.Deal.StartEpoch,
							EndEpoch:   sector.Deal.EndEpoch,
						},
					},
				},
			},
			CommD:            &commD,
			CommR:            &commR,
			Proof:            nil,
			TicketValue:      abi.SealRandomness{},
			TicketEpoch:      0,
			PreCommitMessage: "",
			SeedValue:        abi.InteractiveSealRandomness{},
			SeedEpoch:        0,
			CommitMessage:    "",
		}

		if err := sectorInfoService.Save(info); err != nil {
			return err
		}

		if sector.SectorID > maxSectorID {
			maxSectorID = sector.SectorID
		}

		/* // TODO: Import deals into market
		pnd, err := cborutil.AsIpld(sector.Deal)
		if err != nil {
			return err
		}

		dealKey := datastore.NewKey(deals.ProviderDsPrefix).ChildString(pnd.Cid().String())

		deal := &deals.MinerDeal{
			MinerDeal: storagemarket.MinerDeal{
				ClientDealProposal: sector.Deal,
				ProposalCid: pnd.Cid(),
				State:       storagemarket.StorageDealActive,
				Ref:         &storagemarket.DataRef{Root: proposalCid}, // TODO: This is super wrong, but there
				// are no params for CommP CIDs, we can't recover unixfs cid easily,
				// and this isn't even used after the deal enters Complete state
				DealID: dealID,
			},
		}

		b, err = cborutil.Dump(deal)
		if err != nil {
			return err
		}

		if err := metadataService.Put(dealKey, b); err != nil {
			return err
		}*/
	}
	return metadataService.SetStorageCounter(uint64(maxSectorID))
}

func findMarketDealID(ctx context.Context, api api.FullNode, deal market2.DealProposal) (abi.DealID, error) {
	// TODO: find a better way
	//  (this is only used by genesis miners)

	deals, err := api.StateMarketDeals(ctx, types.EmptyTSK)
	if err != nil {
		return 0, xerrors.Errorf("getting market deals: %w", err)
	}

	for k, v := range deals {
		if v.Proposal.PieceCID.Equals(deal.PieceCID) {
			id, err := strconv.ParseUint(k, 10, 64)
			return abi.DealID(id), err
		}
	}

	return 0, xerrors.New("deal not found")
}
