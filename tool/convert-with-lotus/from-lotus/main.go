package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"path/filepath"
	"reflect"

	cborutil "github.com/filecoin-project/go-cbor-util"
	"github.com/filecoin-project/go-state-types/abi"

	"github.com/google/uuid"
	"github.com/mitchellh/go-homedir"
	"go.uber.org/multierr"
	"golang.org/x/xerrors"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/namespace"
	"github.com/ipfs/go-datastore/query"
	levelds "github.com/ipfs/go-ds-leveldb"
	ldbopts "github.com/syndtr/goleveldb/leveldb/opt"

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
	CreationTime int64  `gorm:"column:create_time;type:bigint;" json:"create_time"`
	Pieces       []byte `gorm:"column:pieces;type:blob;" json:"pieces"`

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

func (sectorInfo *sectorInfo) TableName() string {
	return "sectors_infos"
}

func fromSectorInfo(sector *types.SectorInfo) (*sectorInfo, error) {
	sectorInfo := &sectorInfo{
		Id:            uuid.New().String(),
		SectorNumber:  uint64(sector.SectorNumber),
		State:         string(sector.State),
		SectorType:    int64(sector.SectorType),
		CreationTime:  sector.CreationTime,
		TicketValue:   sector.TicketValue,
		TicketEpoch:   int64(sector.TicketEpoch),
		PreCommit1Out: sector.PreCommit1Out,
		Proof:         sector.Proof,
		//PreCommitDeposit: sector.PreCommitDeposit,
		PreCommitMessage: "",
		PreCommitTipSet:  sector.PreCommitTipSet,
		PreCommit2Fails:  sector.PreCommit2Fails,
		SeedValue:        sector.SeedValue,
		SeedEpoch:        int64(sector.SeedEpoch),
		CommitMessage:    "",
		InvalidProofs:    sector.InvalidProofs,
		FaultReportMsg:   "",
		Return:           string(sector.Return),
		TerminateMessage: "",
		TerminatedAt:     int64(sector.TerminatedAt),
		LastErr:          sector.LastErr,
	}

	if sector.PreCommitDeposit.Int == nil {
		sectorInfo.PreCommitDeposit = "0"
	}
	if len(sector.Pieces) > 0 {
		pieces, err := json.Marshal(sector.Pieces)
		if err != nil {
			return nil, err
		}
		sectorInfo.Pieces = pieces
	}

	if sector.CommD != nil {
		sectorInfo.CommD = sector.CommD.String()
	}

	if sector.CommR != nil {
		sectorInfo.CommR = sector.CommR.String()
	}

	if sector.PreCommitInfo != nil {
		dealIds, err := json.Marshal(sector.PreCommitInfo.DealIDs)
		if err != nil {
			return nil, err
		}

		replaceCapacity := -1
		if sector.PreCommitInfo.ReplaceCapacity {
			replaceCapacity = 1
		}
		sectorInfo.PreCommitInfo = SectorPreCommitInfo{
			SealProof:              int64(sector.PreCommitInfo.SealProof),
			SealedCID:              sector.PreCommitInfo.SealedCID.String(),
			SealRandEpoch:          int64(sector.PreCommitInfo.SealRandEpoch),
			DealIDs:                string(dealIds),
			Expiration:             int64(sector.PreCommitInfo.Expiration),
			ReplaceCapacity:        replaceCapacity,
			ReplaceSectorDeadline:  sector.PreCommitInfo.ReplaceSectorDeadline,
			ReplaceSectorPartition: sector.PreCommitInfo.ReplaceSectorPartition,
			ReplaceSectorNumber:    uint64(sector.PreCommitInfo.ReplaceSectorNumber),
		}
	}
	return sectorInfo, nil
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

func ImportFromLotus(lmRepo, vsRepo string, sid abi.SectorNumber, taskType int) error {
	//  db for lotus
	lds, err := levelDs(filepath.Join(lmRepo, fsDatastore, "metadata"), true)
	if err != nil {
		return err
	}
	ds := namespace.Wrap(lds, datastore.NewKey(SectorStorePrefix))
	defer ds.Close()

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

	var maxSectorID abi.SectorNumber = 0
	var sectors []types.SectorInfo
	res, err := ds.Query(context.TODO(), query.Query{})
	if err != nil {
		return err
	}
	defer res.Close()

	outT := reflect.TypeOf(&sectors).Elem().Elem()
	rout := reflect.ValueOf(&sectors)

	var errs error

	for {
		res, ok := res.NextSync()
		if !ok {
			break
		}
		if res.Error != nil {
			return res.Error
		}

		elem := reflect.New(outT)
		err := cborutil.ReadCborRPC(bytes.NewReader(res.Value), elem.Interface())
		if err != nil {
			errs = multierr.Append(errs, xerrors.Errorf("decoding state for key '%s': %w", res.Key, err))
			continue
		}

		rout.Elem().Set(reflect.Append(rout.Elem(), elem.Elem()))
	}

	if errs != nil {
		return errs
	}

	for _, sector := range sectors {
		if taskType > 0 {
			sSector, err := fromSectorInfo(&sector)
			if err != nil {
				return err
			}
			sSector.Id = uuid.New().String()
			err = db.Create(&sSector).Error
			if err != nil {
				fmt.Printf("put [%d] err: %s \n", sSector.SectorNumber, err.Error())
			}
		}
		if maxSectorID < sector.SectorNumber {
			maxSectorID = sector.SectorNumber
		}
	}

	if taskType == 1 {
		return nil
	}

	// update latest sector id
	if sid > 0 {
		maxSectorID = sid
	}
	err = db.Exec("update `metadata` set `sector_count`=?", maxSectorID).Error
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

	if err := ImportFromLotus(lmRepo, vsRepo, abi.SectorNumber(sid), taskType); err != nil {
		fmt.Printf("import err: %s\n", err.Error())
		return
	}

	fmt.Println("import success.")
}
