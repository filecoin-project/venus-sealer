package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
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

	"github.com/docker/go-units"
	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/mitchellh/go-homedir"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"

	"github.com/ipfs-force-community/venus-common-utils/apiinfo"

	actors "github.com/filecoin-project/venus/pkg/specactors"
	"github.com/filecoin-project/venus/pkg/specactors/builtin/miner"
	"github.com/filecoin-project/venus/pkg/specactors/builtin/power"
	"github.com/filecoin-project/venus/pkg/specactors/policy"
	"github.com/filecoin-project/venus/pkg/types"

	types3 "github.com/filecoin-project/venus-messager/types"

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
			Name:        "network",
			Usage:       "network type: one of mainnet,calibration,2k&nerpa",
			Value:       "mainnet",
			DefaultText: "mainnet",
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
			Name:  "auth-token",
			Usage: "auth token",
		},
	},
	Action: func(cctx *cli.Context) error {
		log.Info("Initializing venus sealer")

		sectorSizeInt, err := units.RAMInBytes(cctx.String("sector-size"))
		if err != nil {
			return err
		}
		ssize := abi.SectorSize(sectorSizeInt)

		gasPrice, err := types.BigFromString(cctx.String("gas-premium"))
		if err != nil {
			return xerrors.Errorf("failed to parse gas-price flag: %s", err)
		}

		symlink := cctx.Bool("symlink-imported-sectors")
		if symlink {
			log.Info("will attempt to symlink to imported sectors")
		}

		network := cctx.String("network")
		defaultCfg, err := config.GetDefaultStorageConfig(network)
		if err != nil {
			return err
		}

		setAuthToken(cctx)
		parseFlag(defaultCfg, cctx)
		if err := checkURL(defaultCfg); err != nil {
			return err
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

		ctx := api.ReqContext(cctx)
		if err := paramfetch.GetParams(ctx, ps, srs, uint64(ssize)); err != nil {
			return xerrors.Errorf("fetching proof parameters: %w", err)
		}

		log.Info("Trying to connect to full node RPC")

		fullNode, closer, err := api.GetFullNodeAPIV2(cctx) // TODO: consider storing full node address in config
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

		log.Info("Checking if repo exists")

		cfgPath := cctx.String("config")
		defaultCfg.ConfigPath = cfgPath

		exit, err := config.ConfigExist(defaultCfg.DataDir)
		if err != nil {
			return err
		}
		if exit {
			return xerrors.Errorf("repo is already initialized at %s", defaultCfg.DataDir)
		}

		log.Info("Checking full node version")

		v, err := fullNode.Version(ctx)
		if err != nil {
			return err
		}

		if !v.APIVersion.EqMajorMinor(constants.FullAPIVersion1) {
			return xerrors.Errorf("Remote API version didn't match (expected %s, remote %s)", constants.FullAPIVersion0, v.APIVersion)
		}

		log.Info("Initializing repo")
		messagerClient, closer, err := api.NewMessageRPC(&defaultCfg.Messager)
		if err != nil {
			return err
		}
		defer closer()

		log.Info("Initializing repo")

		{
			//write config
			err = config.SaveConfig(cfgPath, defaultCfg)
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

		if err := storageMinerInit(ctx, cctx, fullNode, messagerClient, defaultCfg, ssize, gasPrice); err != nil {
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

		// TODO: Point to setting storage price, maybe do it interactively or something
		log.Info("Sealer successfully created, you can now start it with 'venus-sealer run'")

		return nil
	},
}

func setAuthToken(cctx *cli.Context) {
	if cctx.IsSet("auth-token") {
		authToken := cctx.String("auth-token")
		_ = cctx.Set("node-token", authToken)
		_ = cctx.Set("messager-token", authToken)
		_ = cctx.Set("gateway-token", authToken)
	}
}

func parseFlag(cfg *config.StorageMiner, cctx *cli.Context) {
	if cctx.IsSet("repo") {
		cfg.DataDir = cctx.String("repo")
	}

	if cctx.IsSet("messager-url") {
		cfg.Messager.Url = cctx.String("messager-url")
	}

	if cctx.IsSet("node-url") {
		cfg.Node.Url = cctx.String("node-url")
	}

	if cctx.IsSet("gateway-url") {
		cfg.RegisterProof.Urls = cctx.StringSlice("gateway-url")
	}

	if cctx.IsSet("node-token") {
		cfg.Node.Token = cctx.String("node-token")
	}

	if cctx.IsSet("messager-token") {
		cfg.Messager.Token = cctx.String("messager-token")
	}

	if cctx.IsSet("gateway-token") {
		cfg.RegisterProof.Token = cctx.String("gateway-token")
	}
}

func parseMultiAddr(url string) error {
	ai := apiinfo.APIInfo{Addr: url}

	str, err := ai.DialArgs("v0")

	log.Infof("parse url: %s, multi_url: %s", url, str)

	return err
}

func checkURL(cfg *config.StorageMiner) error {
	if err := parseMultiAddr(cfg.Messager.Url); err != nil {
		return xerrors.Errorf("message url: %w", err)
	}

	if err := parseMultiAddr(cfg.Node.Url); err != nil {
		return xerrors.Errorf("node url: %w", err)
	}

	for _, url := range cfg.RegisterProof.Urls {
		if err := parseMultiAddr(url); err != nil {
			return xerrors.Errorf("gateway url:[%s]: %w", url, err)
		}
	}

	return nil
}

func storageMinerInit(ctx context.Context, cctx *cli.Context, api api.FullNode, messagerClient api.IMessager, cfg *config.StorageMiner, ssize abi.SectorSize, gasPrice types.BigInt) error {
	log.Info("Initializing libp2p identity")

	repo, err := models.SetDataBase(config.HomeDir(cfg.DataDir), &cfg.DB)
	if err != nil {
		return err
	}
	err = repo.AutoMigrate()
	if err != nil {
		return err
	}

	metaDataService := service.NewMetadataService(repo)
	sectorInfoService := service.NewSectorInfoService(repo)
	p2pSk, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		return xerrors.Errorf("make host key: %w", err)
	}

	peerid, err := peer.IDFromPrivateKey(p2pSk)
	if err != nil {
		return xerrors.Errorf("peer ID from private key: %w", err)
	}

	var addr address.Address
	if act := cctx.String("actor"); act != "" {
		a, err := address.NewFromString(act)
		if err != nil {
			return xerrors.Errorf("failed parsing actor flag value (%q): %w", act, err)
		}

		if cctx.Bool("genesis-miner") {
			if err := metaDataService.SaveMinerAddress(a); err != nil {
				return err
			}

			if pssb := cctx.String("pre-sealed-metadata"); pssb != "" {
				pssb, err := homedir.Expand(pssb)
				if err != nil {
					return err
				}

				log.Infof("Importing pre-sealed sector metadata for %s", a)

				if err := migratePreSealMeta(ctx, api, pssb, a, metaDataService, sectorInfoService); err != nil {
					return xerrors.Errorf("migrating presealed sector metadata: %w", err)
				}
			}

			return nil
		}

		if pssb := cctx.String("pre-sealed-metadata"); pssb != "" {
			pssb, err := homedir.Expand(pssb)
			if err != nil {
				return err
			}

			log.Infof("Importing pre-sealed sector metadata for %s", a)

			if err := migratePreSealMeta(ctx, api, pssb, a, metaDataService, sectorInfoService); err != nil {
				return xerrors.Errorf("migrating presealed sector metadata: %w", err)
			}
		}

		addr = a
	} else {
		a, err := createStorageMiner(ctx, api, messagerClient, peerid, gasPrice, cctx)
		if err != nil {
			return xerrors.Errorf("creating miner failed: %w", err)
		}

		addr = a
	}

	log.Infof("Created new miner: %s", addr)

	if err := metaDataService.SaveMinerAddress(addr); err != nil {
		return err
	}

	return nil
}

func createStorageMiner(ctx context.Context, nodeAPI api.FullNode, messagerClient api.IMessager, peerid peer.ID, gasPrice types.BigInt, cctx *cli.Context) (address.Address, error) {
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

	// make sure the worker account exists on chain
	_, err = nodeAPI.StateLookupID(ctx, worker, types.EmptyTSK)
	if err != nil {
		msgUid, err := messagerClient.PushMessage(ctx, &types.Message{
			From:  owner,
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

	nv, err := nodeAPI.StateNetworkVersion(ctx, types.EmptyTSK)
	if err != nil {
		return address.Undef, xerrors.Errorf("getting network version: %w", err)
	}

	spt, err := miner.SealProofTypeFromSectorSize(abi.SectorSize(ssize), nv)
	if err != nil {
		return address.Undef, xerrors.Errorf("getting seal proof type: %w", err)
	}

	params, err := actors.SerializeParams(&power2.CreateMinerParams{
		Owner:         owner,
		Worker:        worker,
		SealProofType: spt,
		Peer:          abi.PeerID(peerid),
	})
	if err != nil {
		return address.Undef, err
	}

	sender := owner
	if fromstr := cctx.String("from"); fromstr != "" {
		faddr, err := address.NewFromString(fromstr)
		if err != nil {
			return address.Undef, fmt.Errorf("could not parse from address: %w", err)
		}
		sender = faddr
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

	msgUid, err := messagerClient.PushMessage(ctx, createStorageMinerMsg, &types3.MsgMeta{MaxFee: types.FromFil(1)})
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
	if err := retval.UnmarshalCBOR(bytes.NewReader(mw.Receipt.ReturnValue)); err != nil {
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
					DealInfo: &types2.DealInfo{
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
