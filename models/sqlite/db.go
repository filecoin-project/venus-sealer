package sqlite

import (
	"github.com/filecoin-project/venus-sealer/config"
	"github.com/filecoin-project/venus-sealer/models/repo"
	"github.com/mitchellh/go-homedir"
	"golang.org/x/xerrors"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type SqlLiteRepo struct {
	*gorm.DB
}

func (d SqlLiteRepo) DealRefRepo() repo.DealRefRepo {
	return newDealRefRepo(d.GetDb())
}

func (d SqlLiteRepo) LogRepo() repo.LogRepo {
	return newLogRepo(d.GetDb())
}

func (d SqlLiteRepo) WorkerCallRepo() repo.WorkerCallRepo {
	return newWorkerCallRepo(d.GetDb())
}

func (d SqlLiteRepo) SectorInfoRepo() repo.SectorInfoRepo {
	return newSectorInfoRepo(d.GetDb())
}

func (d SqlLiteRepo) WorkerStateRepo() repo.WorkerStateRepo {
	return newWorkerStateRepo(d.GetDb())
}

func (d SqlLiteRepo) MetaDataRepo() repo.MetaDataRepo {
	return newMetadataRepo(d.GetDb())
}

func (d SqlLiteRepo) AutoMigrate() error {
	err := d.GetDb().AutoMigrate(&dealRef{})
	if err != nil {
		return err
	}

	err = d.GetDb().AutoMigrate(&metadata{})
	if err != nil {
		return err
	}

	err = d.GetDb().AutoMigrate(&Log{})
	if err != nil {
		return err
	}

	err = d.GetDb().AutoMigrate(&sectorInfo{})
	if err != nil {
		return err
	}

	err = d.GetDb().AutoMigrate(&workerCall{})
	if err != nil {
		return err
	}

	err = d.GetDb().AutoMigrate(&workerState{})
	if err != nil {
		return err
	}

	return nil
}

func (d SqlLiteRepo) GetDb() *gorm.DB {
	return d.DB
}

func (d SqlLiteRepo) DbClose() error {
	return nil
}

func OpenSqlite(cfg *config.SqliteConfig) (repo.Repo, error) {
	//cache=shared&_journal_mode=wal&sync=normal
	//cache=shared&sync=full
	path, err := homedir.Expand(cfg.Path)
	if err != nil {
		return nil, xerrors.Errorf("expand path error %v", err)
	}

	db, err := gorm.Open(sqlite.Open(path+"?cache=shared&_journal_mode=wal&sync=normal"), &gorm.Config{
		// Logger: logger.Default.LogMode(logger.Info), // 日志配置
	})
	if err != nil {
		return nil, xerrors.Errorf("fail to connect sqlite: %s %w", cfg.Path, err)
	}
	db.Set("gorm:table_options", "CHARSET=utf8mb4")

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	return &SqlLiteRepo{
		db,
	}, nil
}
