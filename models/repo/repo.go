package repo

import (
	"gorm.io/gorm"
)

type Repo interface {
	GetDb() *gorm.DB
	MetaDataRepo() MetaDataRepo
	WorkerStateRepo() WorkerStateRepo
	WorkerCallRepo() WorkerCallRepo
	SectorInfoRepo() SectorInfoRepo
	DealRefRepo() DealRefRepo
	LogRepo() LogRepo
	DbClose() error
	AutoMigrate() error
}
