package sqlite

import (
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/venus-sealer/models/repo"
	"github.com/filecoin-project/venus-sealer/types"
	"gorm.io/gorm"
	"sort"
	"time"
)

type Log struct {
	Id           int64  `gorm:"column:id;types:integer;primary_key;autoIncrement;" json:"id"` // 主键、
	SectorNumber uint64 `gorm:"column:sector_number;type:unsigned bigint;index:log_sector_number" json:"sector_number"`
	Timestamp    uint64 `gorm:"column:timestamp;type:unsigned bigint;" json:"timestamp"`
	// for errors
	Trace   string `gorm:"column:trace;type:text;" json:"trace"`
	Message string `gorm:"column:message;type:text;" json:"message"`
	// additional data (Event info)
	Kind string `gorm:"column:kind;type:varchar(256);" json:"kind"`
}

func (log *Log) TableName() string {
	return "logs"
}

func (log *Log) Log() *types.Log {
	return &types.Log{
		SectorNumber: abi.SectorNumber(log.SectorNumber),
		Timestamp:    log.Timestamp,
		Trace:        log.Trace,
		Message:      log.Message,
		Kind:         log.Kind,
	}
}

var _ repo.LogRepo = (*logRepo)(nil)

type logRepo struct {
	*gorm.DB
}

func newLogRepo(db *gorm.DB) *logRepo {
	return &logRepo{DB: db}
}

func (s *logRepo) LatestLog(sectorNumber uint64) (*types.Log, error) {
	var log Log
	err := s.DB.Table("logs").Where("sector_number=?", sectorNumber).Order("id desc").Scan(&log).Error
	if err != nil {
		return nil, err
	}
	return log.Log(), nil
}

func (s *logRepo) Count(sectorNumber abi.SectorNumber) (int64, error) {
	var count int64
	err := s.DB.Table("logs").Where("sector_number=?", sectorNumber).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (s *logRepo) Truncate(sectorNumber abi.SectorNumber) error {
	var ids []int64
	err := s.DB.Exec("SELECT id FROM logs WHERE id =?", sectorNumber).Scan(ids).Error
	if err != nil {
		return err
	}
	sort.Slice(ids, func(i, j int) bool {
		return ids[i] < ids[j]
	})
	if len(ids) > 8000 {
		removeIds := ids[2000:6000]
		leftIds := append(ids[:2000], ids[6000:]...)
		modifyId := leftIds[2000]
		err = s.DB.Delete(&Log{}, "id in ?", removeIds).Error
		if err != nil {
			return err
		}
		err = s.DB.Save(&Log{
			Id:           modifyId,
			SectorNumber: uint64(sectorNumber),
			Timestamp:    uint64(time.Now().Unix()),
			Message:      "truncating log (above 8000 entries)",
			Kind:         "truncate",
		}).Error
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *logRepo) Append(log *types.Log) error {
	return s.DB.Save(&Log{
		SectorNumber: uint64(log.SectorNumber),
		Timestamp:    uint64(time.Now().Unix()),
		Trace:        log.Trace,
		Message:      log.Message,
		Kind:         log.Kind,
	}).Error
}

func (s *logRepo) List(sectorNumber abi.SectorNumber) ([]*types.Log, error) {
	var logs []Log
	err := s.DB.Find(&logs, "sector_number=?", sectorNumber).Error
	if err != nil {
		return nil, err
	}

	tLogs := make([]*types.Log, len(logs))
	for index, log := range logs {
		tLogs[index] = log.Log()
	}
	return tLogs, nil
}

func (s *logRepo) DelLogs(sectorNumber uint64) error {
	return s.DB.Table("logs").Delete(&Log{}, "sector_number=?", sectorNumber).Error
}

func (s *logRepo) GetLogs(sectorNumber uint64) ([]types.Log, error) {
	var logs []Log
	err := s.DB.Table("logs").Find(&logs, "sector_number=?", sectorNumber).Error
	if err != nil {
		return nil, err
	}

	typesLogs := make([]types.Log, len(logs))
	for index, log := range logs {
		typesLogs[index] = types.Log{
			Timestamp: log.Timestamp,
			Trace:     log.Trace,
			Message:   log.Message,
			Kind:      log.Kind,
		}
	}
	return typesLogs, nil
}
