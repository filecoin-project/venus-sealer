package sqlite

import (
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/venus-sealer/models/repo"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"sync"
	"time"
)

type metadata struct {
	Id           string    `gorm:"column:id;type:varchar(36);primary_key;"json:"id"` // 主键
	MinerAddress string    `gorm:"column:miner_address;type:varchar(256);NOT NULL"json:"miner_address"`
	SectorCount  uint64    `gorm:"column:sector_count;type:bigint(20);NOT NULL"json:"sector_count"`
	IsDeleted    int       `gorm:"column:is_deleted;default:-1;NOT NULL"`                // 是否删除 1:是  -1:否
	CreatedAt    time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP;NOT NULL"` // 创建时间
	UpdatedAt    time.Time `gorm:"column:updated_at;default:CURRENT_TIMESTAMP;NOT NULL"` // 更新时间
}

func (m *metadata) TableName() string {
	return "metadata"
}

var _ repo.MetaDataRepo = (*metadataRepo)(nil)

type metadataRepo struct {
	*gorm.DB
	lk sync.Mutex
}

func newMetadataRepo(db *gorm.DB) *metadataRepo {
	return &metadataRepo{DB: db, lk: sync.Mutex{}}
}

func (m metadataRepo) SaveMinerAddress(mAddr address.Address) error {
	return m.DB.Create(&metadata{
		Id:           uuid.New().String(),
		MinerAddress: mAddr.String(),
		SectorCount:  0,
		IsDeleted:    -1,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}).Error
}

func (m metadataRepo) GetMinerAddress() (address.Address, error) {
	var meta metadata
	if err := m.DB.First(&meta).Error; err != nil {
		return address.Undef, err
	}
	addr, err := address.NewFromString(meta.MinerAddress)
	if err != nil {
		return address.Undef, err
	}
	return addr, nil
}

func (m metadataRepo) IncreaseStorageCounter() (abi.SectorNumber, error) {
	m.lk.Lock()
	defer m.lk.Unlock()
	var meta metadata
	if err := m.DB.First(&meta).Error; err != nil {
		return 0, err
	}
	//use transaction to protect atomic
	sql := `
UPDATE metadata SET sector_count = sector_count + 1;
SELECT * FROM metadata WHERE id = ?;
`
	var metaAfterUpdate metadata
	if err := m.DB.Table("metadata").Transaction(func(tx *gorm.DB) error {
		return tx.Exec(sql, meta.Id).Scan(&metaAfterUpdate).Error
	}); err != nil {
		return 0, err
	}
	return abi.SectorNumber(metaAfterUpdate.SectorCount), nil
}

func (m metadataRepo) SetStorageCounter(counter uint64) error {
	m.lk.Lock()
	defer m.lk.Unlock()
	var meta metadata
	if err := m.DB.First(&meta).Error; err != nil {
		return err
	}
	meta.SectorCount = counter
	return m.DB.Save(&meta).Error
}
