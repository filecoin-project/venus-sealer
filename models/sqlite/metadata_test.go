package sqlite

import (
	"github.com/filecoin-project/go-address"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"os"
	"testing"
	"time"
)

func setupMeta(suffix string, t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("./meta_"+suffix), &gorm.Config{
		//Logger: logger.Default.LogMode(logger.Info), // 日志配置
	})
	if err != nil {
		t.Error(err)
	}
	err = db.AutoMigrate(&metadata{})
	if err != nil {
		t.Error(err)
	}
	return db
}

func cleanDb(suffix string, t *testing.T) {
	os.Remove("./meta_" + suffix)
}

func Test_metadataRepo_GetMinerAddress(t *testing.T) {
	db := setupMeta("get_miner_address", t)
	defer cleanDb("get_miner_address", t)
	mRepo := newMetadataRepo(db)
	id := uuid.New().String()
	err := mRepo.Save(metadata{
		Id:           id,
		MinerAddress: "t01000",
		SectorCount:  0,
		IsDeleted:    -1,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}).Error

	if err != nil {
		t.Error(err)
	}

	ds := metadata{}
	err = db.First(&ds).Error
	if err != nil {
		t.Error(err)
	}

	if ds.Id != id {
		t.Errorf("expect id %s but got %s", id, ds.Id)
	}

	if ds.MinerAddress != "t01000" {
		t.Errorf("expect address %s but got %s", "t01000", ds.MinerAddress)
	}
}

func Test_metadataRepo_IncreaseStorageCounter(t *testing.T) {
	db := setupMeta("increase_counter", t)
	defer cleanDb("increase_counter", t)
	mRepo := newMetadataRepo(db)
	id := uuid.New().String()
	err := mRepo.Save(metadata{
		Id:           id,
		MinerAddress: "t01000",
		SectorCount:  0,
		IsDeleted:    -1,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}).Error

	if err != nil {
		t.Error(err)
	}

	count, err := mRepo.IncreaseStorageCounter()
	if err != nil {
		t.Error(err)
	}

	if count != 1 {
		t.Errorf("expect count 1 but got %d", count)
	}

	ds := metadata{}
	err = db.First(&ds).Error
	if err != nil {
		t.Error(err)
	}

	if ds.SectorCount != 1 {
		t.Errorf("expect count 1 but got %d", ds.SectorCount)
	}
}

func Test_metadataRepo_SaveMinerAddress(t *testing.T) {
	db := setupMeta("save_miner_address", t)
	defer cleanDb("save_miner_address", t)
	mRepo := newMetadataRepo(db)
	addr, _ := address.NewFromString("t01000")
	err := mRepo.SaveMinerAddress(addr)
	if err != nil {
		t.Error(err)
	}

	ds := metadata{}
	err = db.First(&ds).Error
	if err != nil {
		t.Error(err)
	}

	if ds.MinerAddress != "t01000" {
		t.Errorf("expect address t01000 but got %d", ds.SectorCount)
	}
}

func Test_metadataRepo_SetStorageCounter(t *testing.T) {
	db := setupMeta("set_count", t)
	defer cleanDb("set_count", t)
	mRepo := newMetadataRepo(db)
	id := uuid.New().String()
	err := mRepo.Save(metadata{
		Id:           id,
		MinerAddress: "t01000",
		SectorCount:  0,
		IsDeleted:    -1,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}).Error

	if err != nil {
		t.Error(err)
	}

	err = mRepo.SetStorageCounter(10)
	if err != nil {
		t.Error(err)
	}

	ds := metadata{}
	err = db.First(&ds).Error
	if err != nil {
		t.Error(err)
	}

	if ds.SectorCount != 10 {
		t.Errorf("expect sector count 10 but got %d", ds.SectorCount)
	}
}
