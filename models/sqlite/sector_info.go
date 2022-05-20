package sqlite

import (
	"encoding/json"
	"sync"

	"github.com/filecoin-project/go-state-types/abi"
	fbig "github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-state-types/builtin/v8/miner"
	"github.com/filecoin-project/venus-sealer/models/repo"
	"github.com/filecoin-project/venus-sealer/types"
	"github.com/google/uuid"
	"github.com/ipfs/go-cid"
	"gorm.io/gorm"
)

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

	// snap-deal related members
	CCUpdate             bool   `gorm:"column:cc_update;type:bool;" json:"cc_update"`
	CCPieces             []byte `grom:"column:cc_pieces;type:blob;" json:"cc_pieces"`
	UpdateSealed         string `gorm:"column:update_sealed;type:varchar(256);" json:"update_sealed"`
	UpdateUnsealed       string `gorm:"column:update_unsealed;type:varchar(256);" json:"update_unsealed"`
	ReplicaUpdateProof   []byte `gorm:"column:replica_update_proof;type:blob;" json:"replica_update_proof"`
	ReplicaUpdateMessage string `gorm:"column:replica_update_message;type:varchar(256);" json:"replica_update_message"`

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

func (sectorInfo *sectorInfo) SectorInfo() (*types.SectorInfo, error) {
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
		Proof:        sectorInfo.Proof,
		CreationTime: sectorInfo.CreationTime,
		//PreCommitInfo:    sectorInfo.PreCommitInfo,
		//	PreCommitDeposit: deposit,
		PreCommitMessage: sectorInfo.PreCommitMessage,
		PreCommitTipSet:  sectorInfo.PreCommitTipSet,
		PreCommit2Fails:  sectorInfo.PreCommit2Fails,
		SeedValue:        sectorInfo.SeedValue,
		SeedEpoch:        abi.ChainEpoch(sectorInfo.SeedEpoch),
		CommitMessage:    sectorInfo.CommitMessage,
		InvalidProofs:    sectorInfo.InvalidProofs,

		CCUpdate:             sectorInfo.CCUpdate,
		ReplicaUpdateMessage: sectorInfo.ReplicaUpdateMessage,

		FaultReportMsg:   sectorInfo.FaultReportMsg,
		Return:           types.ReturnState(sectorInfo.Return),
		TerminateMessage: sectorInfo.TerminateMessage,
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
		deposit, err := fbig.FromString(sectorInfo.PreCommitDeposit)
		if err != nil {
			return nil, err
		}
		sinfo.PreCommitDeposit = deposit
	}

	if len(sectorInfo.CCPieces) > 0 {
		if err := json.Unmarshal(sectorInfo.CCPieces, &sinfo.CCPieces); err != nil {
			return nil, err
		}
	}

	if len(sectorInfo.UpdateSealed) > 0 {
		updateSealed, err := cid.Decode(sectorInfo.UpdateSealed)
		if err != nil {
			return nil, err
		}
		sinfo.UpdateSealed = &updateSealed
	}

	if len(sectorInfo.UpdateUnsealed) > 0 {
		updateUnSealed, err := cid.Decode(sectorInfo.UpdateUnsealed)
		if err != nil {
			return nil, err
		}
		sinfo.UpdateUnsealed = &updateUnSealed
	}

	if len(sectorInfo.ReplicaUpdateProof) > 0 {
		if err := json.Unmarshal(sectorInfo.ReplicaUpdateProof, &sinfo.ReplicaUpdateProof); err != nil {
			return nil, err
		}
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

func FromSectorInfo(sector *types.SectorInfo) (*sectorInfo, error) {
	sectorInfo := &sectorInfo{
		Id:           uuid.New().String(),
		SectorNumber: uint64(sector.SectorNumber),
		State:        string(sector.State),
		SectorType:   int64(sector.SectorType),
		CreationTime: sector.CreationTime,
		//	Pieces:           nil,
		TicketValue:   sector.TicketValue,
		TicketEpoch:   int64(sector.TicketEpoch),
		PreCommit1Out: sector.PreCommit1Out,
		//CommD:            sector.CommD,
		//CommR:            sector.CommR,
		Proof: sector.Proof,
		/*		PreCommitInfo:    SectorPreCommitInfo{
				SealProof:              0,
				SealedCID:              "",
				SealRandEpoch:          0,
				DealIDs:                "",
				Expiration:             0,
				ReplaceCapacity:        0,
				ReplaceSectorDeadline:  0,
				ReplaceSectorPartition: 0,
				ReplaceSectorNumber:    0,
			},*/
		//PreCommitDeposit: sector.PreCommitDeposit,
		PreCommitMessage: sector.PreCommitMessage,
		PreCommitTipSet:  sector.PreCommitTipSet,
		PreCommit2Fails:  sector.PreCommit2Fails,
		SeedValue:        sector.SeedValue,
		SeedEpoch:        int64(sector.SeedEpoch),
		CommitMessage:    sector.CommitMessage,
		InvalidProofs:    sector.InvalidProofs,
		FaultReportMsg:   sector.FaultReportMsg,
		Return:           string(sector.Return),
		TerminateMessage: sector.TerminateMessage,
		TerminatedAt:     int64(sector.TerminatedAt),
		LastErr:          sector.LastErr,

		CCUpdate:             sector.CCUpdate,
		ReplicaUpdateMessage: sector.ReplicaUpdateMessage,
	}

	if len(sector.CCPieces) != 0 {
		ccPieces, err := json.Marshal(sector.CCPieces)
		if err != nil {
			return nil, err
		}
		sectorInfo.CCPieces = ccPieces
	}

	if sector.UpdateSealed != nil {
		sectorInfo.UpdateSealed = sector.UpdateSealed.String()
	}

	if sector.UpdateUnsealed != nil {
		sectorInfo.UpdateUnsealed = sector.UpdateUnsealed.String()
	}

	if sector.ReplicaUpdateProof != nil {
		replicaUpdateProof, err := json.Marshal(sector.ReplicaUpdateProof)
		if err != nil {
			return nil, err
		}
		sectorInfo.ReplicaUpdateProof = replicaUpdateProof
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

var _ repo.SectorInfoRepo = (*sectorInfoRepo)(nil)

type sectorInfoRepo struct {
	*gorm.DB
	lk sync.Mutex
}

func newSectorInfoRepo(db *gorm.DB) *sectorInfoRepo {
	return &sectorInfoRepo{DB: db, lk: sync.Mutex{}}
}

func (s *sectorInfoRepo) GetSectorInfoByID(sectorNumber uint64) (*types.SectorInfo, error) {
	var sectorInfo sectorInfo
	err := s.DB.Debug().Table("sectors_infos").
		Limit(1).
		Where("sector_number=?", sectorNumber).
		Take(&sectorInfo).Error
	if err != nil {
		return nil, err
	}
	//read log
	sinfo, err := sectorInfo.SectorInfo()
	if err != nil {
		return nil, err
	}
	return sinfo, nil
}

func (s *sectorInfoRepo) HasSectorInfo(sectorNumber uint64) (bool, error) {
	var count int64
	err := s.DB.Table("sectors_infos").
		Where("sector_number=?", sectorNumber).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *sectorInfoRepo) Save(sector *types.SectorInfo) error {
	sSector, err := FromSectorInfo(sector)
	if err != nil {
		return err
	}
	sSector.Id = uuid.New().String()
	return s.DB.Create(&sSector).Error
}

func (s *sectorInfoRepo) GetAllSectorInfos() ([]*types.SectorInfo, error) {
	var sectorInfos []*sectorInfo
	err := s.DB.Table("sectors_infos").Find(&sectorInfos).Error
	if err != nil {
		return nil, err
	}
	result := make([]*types.SectorInfo, len(sectorInfos))
	for index, st := range sectorInfos {
		newSt, err := st.SectorInfo()
		if err != nil {
			return nil, err
		}
		result[index] = newSt
	}
	return result, nil
}

func (s *sectorInfoRepo) DeleteBySectorId(sectorNumber uint64) error {
	return s.DB.Delete(&sectorInfo{},
		"sector_number=?", sectorNumber).Error
}

func (s *sectorInfoRepo) UpdateSectorInfoBySectorId(inSectorInfo *types.SectorInfo, sectorNumber uint64) error {
	var sInfo sectorInfo
	err := s.DB.Table("sectors_infos").Find(&sInfo, "sector_number=?", sectorNumber).Error
	if err != nil {
		return err
	}

	sSector, err := FromSectorInfo(inSectorInfo)
	if err != nil {
		return err
	}
	sSector.Id = sInfo.Id
	return s.DB.Save(sSector).Error
}
