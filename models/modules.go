package models

import (
	"github.com/filecoin-project/venus-sealer/config"
	"github.com/filecoin-project/venus-sealer/models/mysql"
	"github.com/filecoin-project/venus-sealer/models/repo"
	"github.com/filecoin-project/venus-sealer/models/sqlite"
	"golang.org/x/xerrors"
	"path"
)

func SetDataBase(homeDir config.HomeDir, cfg *config.DbConfig) (repo.Repo, error) {
	switch cfg.Type {
	case "sqlite":
		cfg.Sqlite.Path = path.Join(string(homeDir), cfg.Sqlite.Path)
		return sqlite.OpenSqlite(&cfg.Sqlite)
	case "mysql":
		return mysql.OpenMysql(&cfg.MySql)
	default:
		return nil, xerrors.Errorf("unsupport db type,(%s, %s)", "sqlite", "mysql")
	}
}

func AutoMigrate(repo repo.Repo) error {
	return repo.AutoMigrate()
}
