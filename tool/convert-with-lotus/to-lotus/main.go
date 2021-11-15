package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"path/filepath"

	cborutil "github.com/filecoin-project/go-cbor-util"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/specs-actors/actors/builtin/miner"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	levelds "github.com/ipfs/go-ds-leveldb"
	u "github.com/ipfs/go-ipfs-util"
	"github.com/mitchellh/go-homedir"
	ldbopts "github.com/syndtr/goleveldb/leveldb/opt"
	"golang.org/x/xerrors"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/filecoin-project/venus-sealer/api"
	"github.com/filecoin-project/venus-sealer/config"
	"github.com/filecoin-project/venus-sealer/tool/convert-with-lotus/types"
)

type SectorPreCommitInfo struct {
	SealProof     int64  `gorm:"column:seal_proof;type:bigint;" json:"seal_proof"`
	SealedCID     string `gorm:"column:sealed_cid;type:varchar(256);" json:"sealed_cid"`
	SealRandEpoch int64  `gorm:"column:seal_rand_epoch;type:bigint;" json:"seal_rand_epoch"`
	// []uint64
	DealIDs    string `gorm:"column:deal_ids;type:text;" json:"deal_ids"`
	Expiration int64  `gorm:"column:expiration;type:bigint;" json:"expiration"`
	//-1 false 1 true
	ReplaceCapacity        int    `gorm:"column:replace_capacity;type:int;" json:"replace_capacity"`
	ReplaceSectorDeadline  uint64 `gorm:"column:replace_sector_deadline;type:unsigned bigint;" json:"replace_sector_deadline"`
	ReplaceSectorPartition uint64 `gorm:"column:replace_sector_partition;type:unsigned bigint;" json:"replace_sector_partition"`
	ReplaceSectorNumber    uint64 `gorm:"column:replace_sector_number;type:unsigned bigint;" json:"replace_sector_number"`
}

type sectorInfo struct {
	Id           string `gorm:"column:id;type:varchar(36);primary_key;" json:"id"` // 主键
	SectorNumber uint64 `gorm:"uniqueIndex;column:sector_number;type:unsigned bigint;" json:"sector_number"`
	State        string `gorm:"column:state;type:varchar(256);" json:"state"`
	SectorType   int64  `gorm:"column:sector_type;type:bigint;" json:"sector_type"`

	// Packing  []Piece
	Pieces []byte `gorm:"column:pieces;type:blob;" json:"pieces"`

	// PreCommit1
	TicketValue   []byte `gorm:"column:ticket_value;type:blob;" json:"ticket_value"`
	TicketEpoch   int64  `gorm:"column:ticket_epoch;type:bigint;" json:"ticket_epoch"`
	PreCommit1Out []byte `gorm:"column:pre_commit1_out;type:blob;" json:"pre_commit1_out"`

	// PreCommit2
	CommD string `gorm:"column:commd;type:varchar(256);" json:"commd"`
	CommR string `gorm:"column:commr;type:varchar(256);" json:"commr"`
	Proof []byte `gorm:"column:proof;type:blob;" json:"proof"`

	//*miner.SectorPreCommitInfo
	PreCommitInfo    SectorPreCommitInfo `gorm:"embedded;embeddedPrefix:precommit_"`
	PreCommitDeposit string              `gorm:"column:pre_commit_deposit;type:varchar(256);" json:"pre_commit_deposit"`
	PreCommitMessage string              `gorm:"column:pre_commit_message;type:varchar(256);" json:"pre_commit_message"`
	PreCommitTipSet  []byte              `gorm:"column:pre_commit_tipset;type:blob;" json:"pre_commit_tipset"`

	PreCommit2Fails uint64 `gorm:"column:pre_commit2_fails;type:unsigned bigint;" json:"pre_commit2_fails"`

	// WaitSeed
	SeedValue []byte `gorm:"column:seed_value;type:blob;" json:"seed_value"`
	SeedEpoch int64  `gorm:"column:seed_epoch;type:bigint;" json:"seed_epoch"`

	// Committing
	CommitMessage string `gorm:"column:commit_message;type:text;" json:"commit_message"`
	InvalidProofs uint64 `gorm:"column:invalid_proofs;type:unsigned bigint;" json:"invalid_proofs"`

	// Faults
	FaultReportMsg string `gorm:"column:fault_report_msg;type:text;" json:"fault_report_msg"`

	// Recovery
	Return string `gorm:"column:return;type:text;" json:"return"`

	// Termination
	TerminateMessage string `gorm:"column:terminate_message;type:text;" json:"terminate_message"`
	TerminatedAt     int64  `gorm:"column:terminated_at;type:bigint;" json:"terminated_at"`

	// Debug
	LastErr string `gorm:"column:last_err;type:text;" json:"last_err"`
}

func (sectorInfo *sectorInfo) SectorInfo(api api.IMessager) (*types.SectorInfo, error) {
	tc := cid.NewCidV0(u.Hash([]byte("undef")))

	pcCid := &tc
	cCid := &tc
	frCid := &tc
	tCid := &tc
	if msg, err := api.GetMessageByUid(context.TODO(), sectorInfo.PreCommitMessage); err == nil && msg.SignedCid != nil {
		pcCid = msg.SignedCid
	}

	if msg, err := api.GetMessageByUid(context.TODO(), sectorInfo.CommitMessage); err == nil && msg.SignedCid != nil {
		pcCid = msg.SignedCid
	}

	if msg, err := api.GetMessageByUid(context.TODO(), sectorInfo.FaultReportMsg); err == nil && msg.SignedCid != nil {
		frCid = msg.SignedCid
	}

	if msg, err := api.GetMessageByUid(context.TODO(), sectorInfo.TerminateMessage); err == nil && msg.SignedCid != nil {
		tCid = msg.SignedCid
	}

	sinfo := &types.SectorInfo{
		State:        types.SectorState(sectorInfo.State),
		SectorNumber: abi.SectorNumber(sectorInfo.SectorNumber),
		SectorType:   abi.RegisteredSealProof(sectorInfo.SectorType),
		//	Pieces:           pieces,
		TicketValue:   sectorInfo.TicketValue,
		TicketEpoch:   abi.ChainEpoch(sectorInfo.TicketEpoch),
		PreCommit1Out: sectorInfo.PreCommit1Out,
		//	CommD:            &commD,
		//CommR:            &commR,
		Proof: sectorInfo.Proof,
		//PreCommitInfo:    sectorInfo.PreCommitInfo,
		//	PreCommitDeposit: deposit,
		PreCommitMessage: pcCid,
		PreCommitTipSet:  sectorInfo.PreCommitTipSet,
		PreCommit2Fails:  sectorInfo.PreCommit2Fails,
		SeedValue:        sectorInfo.SeedValue,
		SeedEpoch:        abi.ChainEpoch(sectorInfo.SeedEpoch),
		CommitMessage:    cCid,
		InvalidProofs:    sectorInfo.InvalidProofs,
		FaultReportMsg:   frCid,
		Return:           types.ReturnState(sectorInfo.Return),
		TerminateMessage: tCid,
		TerminatedAt:     abi.ChainEpoch(sectorInfo.TerminatedAt),
		LastErr:          sectorInfo.LastErr,
	}
	if len(sectorInfo.Pieces) > 0 {
		err := json.Unmarshal(sectorInfo.Pieces, &sinfo.Pieces)
		if err != nil {
			return nil, err
		}
	}

	if len(sectorInfo.CommD) > 0 {
		commD, err := cid.Decode(sectorInfo.CommD)
		if err != nil {
			return nil, err
		}
		sinfo.CommD = &commD
	}
	if len(sectorInfo.CommR) > 0 {
		commR, err := cid.Decode(sectorInfo.CommR)
		if err != nil {
			return nil, err
		}
		sinfo.CommR = &commR
	}

	if len(sectorInfo.PreCommitDeposit) > 0 {
		deposit, err := big.FromString(sectorInfo.PreCommitDeposit)
		if err != nil {
			return nil, err
		}
		sinfo.PreCommitDeposit = deposit
	}

	if len(sectorInfo.PreCommitInfo.SealedCID) > 0 {
		sealedCid, err := cid.Decode(sectorInfo.PreCommitInfo.SealedCID)
		if err != nil {
			return nil, err
		}

		sinfo.PreCommitInfo = &miner.SectorPreCommitInfo{
			SealProof:              abi.RegisteredSealProof(sectorInfo.PreCommitInfo.SealProof),
			SectorNumber:           abi.SectorNumber(sectorInfo.SectorNumber),
			SealedCID:              sealedCid,
			SealRandEpoch:          abi.ChainEpoch(sectorInfo.PreCommitInfo.SealRandEpoch),
			DealIDs:                nil,
			Expiration:             abi.ChainEpoch(sectorInfo.PreCommitInfo.Expiration),
			ReplaceCapacity:        sectorInfo.PreCommitInfo.ReplaceCapacity != -1,
			ReplaceSectorDeadline:  sectorInfo.PreCommitInfo.ReplaceSectorDeadline,
			ReplaceSectorPartition: sectorInfo.PreCommitInfo.ReplaceSectorPartition,
			ReplaceSectorNumber:    abi.SectorNumber(sectorInfo.PreCommitInfo.ReplaceSectorNumber),
		}
		if len(sectorInfo.PreCommitInfo.DealIDs) > 0 {
			err := json.Unmarshal([]byte(sectorInfo.PreCommitInfo.DealIDs), &sinfo.PreCommitInfo.DealIDs)
			if err != nil {
				return nil, err
			}
		}
	}

	return sinfo, nil
}

const (
	fsDatastore       = "datastore"
	SectorStorePrefix = "/sectors"
)

func levelDs(path string, readonly bool) (datastore.Batching, error) {
	return levelds.NewDatastore(path, &levelds.Options{
		Compression: ldbopts.NoCompression,
		NoSync:      false,
		Strict:      ldbopts.StrictAll,
		ReadOnly:    readonly,
	})
}

func ImportToLotusMiner(lmRepo, vsRepo string, sid abi.SectorNumber, taskType int) error {
	// db for venus
	path, err := homedir.Expand(filepath.Join(vsRepo, "sealer.db"))
	if err != nil {
		return xerrors.Errorf("expand path error %v", err)
	}

	db, err := gorm.Open(sqlite.Open(path+"?cache=shared&_cache_size=204800&_journal_mode=wal&sync=normal"), &gorm.Config{
		// Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return xerrors.Errorf("fail to connect sqlite: %v", err)
	}
	db.Set("gorm:table_options", "CHARSET=utf8mb4")

	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	defer sqlDB.Close()

	// db for lotus
	ds, err := levelDs(filepath.Join(lmRepo, fsDatastore, "metadata"), false)
	if err != nil {
		return err
	}
	defer ds.Close()

	if taskType > 0 {
		var sectorInfos []*sectorInfo
		err = db.Table("sectors_infos").Find(&sectorInfos).Error
		if err != nil {
			return err
		}

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
			newSt, err := st.SectorInfo(mc)
			if err != nil {
				return err
			}
			result[index] = *newSt
		}

		for _, sector := range result {
			b, err := cborutil.Dump(&sector)
			if err != nil {
				return xerrors.Errorf("serializing refs: %w", err)
			}

			sectorKey := datastore.NewKey(SectorStorePrefix).ChildString(fmt.Sprint(sector.SectorNumber))
			err = ds.Put(sectorKey, b)
			if err != nil {
				fmt.Printf("put [%s] err: %s \n", sectorKey, err.Error())
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
		err = db.Raw("select `sector_count` from `metadata` LIMIT 1").Scan(&maxSectorID).Error
		if err != nil {
			return xerrors.Errorf("fail to query latest sector id: %v", err)
		}
	}

	buf := make([]byte, binary.MaxVarintLen64)
	size := binary.PutUvarint(buf, uint64(sid))
	err = ds.Put(datastore.NewKey("/storage/nextid"), buf[:size])
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
		taskType           int
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
