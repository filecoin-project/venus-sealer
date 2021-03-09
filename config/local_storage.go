package config

import (
	"bytes"
	"github.com/BurntSushi/toml"
	"github.com/filecoin-project/venus-sealer/sector-storage/fsutil"
	"github.com/filecoin-project/venus-sealer/sector-storage/stores"
	"github.com/mitchellh/go-homedir"
	"github.com/multiformats/go-multiaddr"
	"golang.org/x/xerrors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	fsAPI           = "api"
	fsAPIToken      = "token"
	fsStorageConfig = "storage.json"
	fsLock          = "repo.lock"
)

var (
	ErrNoAPIEndpoint = xerrors.New("API not running (no endpoint)")
	ErrNoAPIToken    = xerrors.New("API token not set")
)

type LocalStorage struct {
	dataDir    string
	configPath string
	storageLk  sync.Mutex
	configLk   sync.Mutex
}

func NewLocalStorage(path string, cfgPath string) *LocalStorage {
	path, _ = homedir.Expand(path)
	cfgPath, _ = homedir.Expand(cfgPath)
	return &LocalStorage{
		dataDir:    path,
		configPath: cfgPath,
		storageLk:  sync.Mutex{},
		configLk:   sync.Mutex{},
	}
}

func (fsr *LocalStorage) GetStorage() (stores.StorageConfig, error) {
	fsr.storageLk.Lock()
	defer fsr.storageLk.Unlock()

	return fsr.getStorage(nil)
}

func (fsr *LocalStorage) getStorage(def *stores.StorageConfig) (stores.StorageConfig, error) {
	c, err := StorageFromFile(fsr.join(fsStorageConfig), def)
	if err != nil {
		return stores.StorageConfig{}, err
	}
	return *c, nil
}

func (fsr *LocalStorage) SetStorage(c func(*stores.StorageConfig)) error {
	fsr.storageLk.Lock()
	defer fsr.storageLk.Unlock()

	sc, err := fsr.getStorage(&stores.StorageConfig{})
	if err != nil {
		return xerrors.Errorf("get storage: %w", err)
	}

	c(&sc)

	return WriteStorageFile(fsr.join(fsStorageConfig), &sc)
}

func (fsr *LocalStorage) Stat(path string) (fsutil.FsStat, error) {
	return fsutil.Statfs(path)
}

func (fsr *LocalStorage) DiskUsage(path string) (int64, error) {
	si, err := fsutil.FileSize(path)
	if err != nil {
		return 0, err
	}
	return si.OnDisk, nil
}

// join joins dataDir elements with fsr.dataDir
func (fsr *LocalStorage) join(paths ...string) string {
	return filepath.Join(append([]string{fsr.dataDir}, paths...)...)
}

func (fsr *LocalStorage) SetConfig(c func(interface{})) error {
	fsr.configLk.Lock()
	defer fsr.configLk.Unlock()

	cfg, err := MinerFromFile(fsr.configPath)
	if err != nil {
		return err
	}

	// mutate in-memory representation of config
	c(cfg)

	// buffer into which we write TOML bytes
	buf := new(bytes.Buffer)

	// encode now-mutated config as TOML and write to buffer
	err = toml.NewEncoder(buf).Encode(cfg)
	if err != nil {
		return err
	}

	// write buffer of TOML bytes to config file
	err = ioutil.WriteFile(fsr.configPath, buf.Bytes(), 0644)
	if err != nil {
		return err
	}

	return nil
}

// APIEndpoint returns endpoint of API in this repo
func (fsr *LocalStorage) APIEndpoint() (multiaddr.Multiaddr, error) {
	p := filepath.Join(fsr.dataDir, fsAPI)

	f, err := os.Open(p)
	if os.IsNotExist(err) {
		return nil, ErrNoAPIEndpoint
	} else if err != nil {
		return nil, err
	}
	defer f.Close() //nolint: errcheck // Read only op

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, xerrors.Errorf("failed to read %q: %w", p, err)
	}
	strma := string(data)
	strma = strings.TrimSpace(strma)

	apima, err := multiaddr.NewMultiaddr(strma)
	if err != nil {
		return nil, err
	}
	return apima, nil
}

func (fsr *LocalStorage) APIToken() ([]byte, error) {
	p := filepath.Join(fsr.dataDir, fsAPIToken)
	f, err := os.Open(p)

	if os.IsNotExist(err) {
		return nil, ErrNoAPIEndpoint
	} else if err != nil {
		return nil, err
	}
	defer f.Close() //nolint: errcheck // Read only op

	tb, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	return bytes.TrimSpace(tb), nil
}

func (fsr *LocalStorage) SetAPIEndpoint(ma multiaddr.Multiaddr) error {
	return ioutil.WriteFile(fsr.join(fsAPI), []byte(ma.String()), 0644)
}

func (fsr *LocalStorage) SetAPIToken(token []byte) error {
	return ioutil.WriteFile(fsr.join(fsAPIToken), token, 0600)
}
