package service

import "github.com/filecoin-project/venus-sealer/models/repo"

type LogService struct {
	repo.LogRepo
}

func NewLogService(repo repo.Repo) *LogService {
	return &LogService{LogRepo: repo.LogRepo()}
}
