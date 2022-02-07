package main

import (
	"flag"
	"fmt"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/venus-sealer/config"
	"github.com/filecoin-project/venus-sealer/models/sqlite"
	"github.com/filecoin-project/venus-sealer/tool/convert-with-lotus/modules"
	"github.com/ipfs/go-datastore/namespace"
	"path/filepath"

	"github.com/filecoin-project/venus-sealer/tool/convert-with-lotus/types"
	"github.com/mitchellh/go-homedir"
	"golang.org/x/xerrors"
)

func ImportFromLotus(lmRepo, vsRepo string, sid abi.SectorNumber, taskType int) error {
	//  db for lotus
	ds, err := modules.OpenLotusDatastore(lmRepo, true)
	if err != nil {
		return err
	}
	ds = namespace.Wrap(ds, modules.SectorPrefixKey)

	defer ds.Close()

	// db for venus
	path, err := homedir.Expand(filepath.Join(vsRepo, "sealer.db"))
	if err != nil {
		return xerrors.Errorf("expand path error %v", err)
	}

	db, err := sqlite.OpenSqlite(&config.SqliteConfig{Path: path})
	if err != nil {
		return xerrors.Errorf("fail to connect sqlite: %v", err)
	}

	var maxSectorID abi.SectorNumber = 0

	if err = modules.ForEachSector(ds, func(sector *types.SectorInfo) error {
		if taskType > 0 {
			vSector := sector.ToVenus()
			if err = db.SectorInfoRepo().Save(vSector); err != nil {
				fmt.Printf("put [%d] err: %s \n", vSector.SectorNumber, err.Error())
			}
		}
		if maxSectorID < sector.SectorNumber {
			maxSectorID = sector.SectorNumber
		}
		return nil
	}); err != nil {
		return err
	}

	if taskType == 1 {
		return nil
	}

	// update latest sector id
	if sid > 0 {
		maxSectorID = sid
	}

	if err = db.MetaDataRepo().SetStorageCounter(uint64(maxSectorID)); err != nil {
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

	if err := ImportFromLotus(lmRepo, vsRepo, abi.SectorNumber(sid), taskType); err != nil {
		fmt.Printf("import err: %s\n", err.Error())
		return
	}

	fmt.Println("import success.")
}
