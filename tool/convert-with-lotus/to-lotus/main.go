package main

import (
	"context"
	"encoding/binary"
	"flag"
	"fmt"
	sqlite "github.com/filecoin-project/venus-sealer/models/sqlite"
	"github.com/filecoin-project/venus-sealer/tool/convert-with-lotus/modules"
	"path/filepath"

	cborutil "github.com/filecoin-project/go-cbor-util"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/venus-sealer/api"
	"github.com/filecoin-project/venus-sealer/config"
	"github.com/filecoin-project/venus-sealer/tool/convert-with-lotus/types"
	"github.com/ipfs/go-datastore"
	"github.com/mitchellh/go-homedir"
	"golang.org/x/xerrors"
)

func ImportToLotusMiner(lmRepo, vsRepo string, sid abi.SectorNumber, taskType int) error {
	// db for venus
	path, err := homedir.Expand(filepath.Join(vsRepo, "sealer.db"))
	if err != nil {
		return xerrors.Errorf("expand path error %v", err)
	}

	db, err := sqlite.OpenSqlite(&config.SqliteConfig{Path: path})
	if err != nil {
		return xerrors.Errorf("fail to connect sqlite: %v", err)
	}
	defer db.DbClose()

	// db for lotus
	ds, err := modules.OpenLotusDatastore(lmRepo, false)
	if err != nil {
		return err
	}
	defer ds.Close()

	if taskType > 0 {
		sectorInfos, err := db.SectorInfoRepo().GetAllSectorInfos()
		if err != nil {
			return err
		}

		// var mc api.IMessager
		// read message config
		cfgPath := config.FsConfig(vsRepo)
		cfg, err := config.MinerFromFile(cfgPath)
		if err != nil {
			return err
		}

		mc, closer, err := api.NewMessageRPC(&cfg.Messager)
		if err != nil {
			return err
		}
		defer closer()

		result := make([]types.SectorInfo, len(sectorInfos))
		for index, st := range sectorInfos {
			result[index].FromVenus(st, mc)
		}

		for _, sector := range result {
			b, err := cborutil.Dump(&sector)
			if err != nil {
				return xerrors.Errorf("serializing refs: %w", err)
			}
			sectorKey := modules.SectorPrefixKey.ChildString(fmt.Sprintf("%d", sector.SectorNumber))
			if err = ds.Put(context.TODO(), sectorKey, b); err != nil {
				fmt.Printf("put sector info [%s] err: %s \n", sectorKey, err.Error())
			}
		}
	}

	if taskType == 1 {
		return nil
	}

	// update sector_count
	var maxSectorID abi.SectorNumber = 0
	if sid > 0 {
		maxSectorID = sid
	} else {
		if maxSectorID, err = db.MetaDataRepo().GetStorageCounter(); err != nil {
			return xerrors.Errorf("fail to query latest sector id: %v", err)
		}
	}

	buf := make([]byte, binary.MaxVarintLen64)
	size := binary.PutUvarint(buf, uint64(sid))
	err = ds.Put(context.TODO(), datastore.NewKey("/storage/nextid"), buf[:size])
	if err != nil {
		return xerrors.Errorf("fail to update latest sector id: %v", err)
	}
	fmt.Printf("latest sector id: %d\n", maxSectorID)

	return nil
}

func main() {
	var (
		lmRepo, vsRepo string
		sid            uint64
		taskType       int
	)

	flag.StringVar(&lmRepo, "lotus-miner-repo", "", "repo path for lotus-miner")
	flag.StringVar(&vsRepo, "venus-sealer-repo", "", "repo path for venus-sealer")
	flag.Uint64Var(&sid, "sid", 0, "last sector id, default max sector id")
	flag.IntVar(&taskType, "taskType", 0, "0-only set sid,1-only import sectors,2-all")

	flag.Parse()

	if err := ImportToLotusMiner(lmRepo, vsRepo, abi.SectorNumber(sid), taskType); err != nil {
		fmt.Printf("import sectors err: %s\n", err.Error())
		return
	}

	fmt.Println("import success.")
}
