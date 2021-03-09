package service

import (
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/venus-sealer/models/repo"
	"github.com/filecoin-project/venus-sealer/types"
)

var _ types.SectorIDCounter = (*MetadataService)(nil)

type MetadataService struct {
	repo.MetaDataRepo
}

func NewMetadataService(repo repo.Repo) *MetadataService {
	return &MetadataService{MetaDataRepo: repo.MetaDataRepo()}
}

func (metadataService *MetadataService) GetMinerAddress() (address.Address, error) {
	return metadataService.MetaDataRepo.GetMinerAddress()
}

func (metadataService *MetadataService) SetStorageCounter(count uint64) error {
	return metadataService.MetaDataRepo.SetStorageCounter(count)
}

func (metadataService *MetadataService) SaveMinerAddress(mAddr address.Address) error {
	return metadataService.MetaDataRepo.SaveMinerAddress(mAddr)
}

func (metadataService *MetadataService) Next() (abi.SectorNumber, error) {
	return metadataService.MetaDataRepo.IncreaseStorageCounter()
}
