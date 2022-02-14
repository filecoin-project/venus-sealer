package config

import (
	"bytes"
	"fmt"
	"github.com/mitchellh/go-homedir"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/kelseyhightower/envconfig"
	"golang.org/x/xerrors"
)

// MinerFromFile loads config from a specified file overriding defaults specified in
// the def parameter. If file does not exist or is empty defaults are assumed.
func WorkerFromFile(path string) (*StorageWorker, error) {
	path, err := homedir.Expand(path)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer file.Close() // nolint:errcheck
	return WorkerFromReader(file)
}

// MinerFromFile loads config from a specified file overriding defaults specified in
// the def parameter. If file does not exist or is empty defaults are assumed.
func MinerFromFile(path string) (*StorageMiner, error) {
	path, err := homedir.Expand(path)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close() // nolint:errcheck

	return MinerFromReader(file)
}

// ConfigExist check if the configuration file exists
func ConfigExist(path string) (bool, error) {
	path, err := homedir.Expand(path)
	if err != nil {
		return false, err
	}

	file, err := os.Open(path)
	switch {
	case os.IsNotExist(err):
		return false, nil
	case err != nil:
		return false, err
	}
	_ = file.Close()

	return true, nil
}

func SaveConfig(path string, cfg interface{}) error {
	path, err := homedir.Expand(path)
	if err != nil {
		return xerrors.Errorf("homedir expand error %s", path)
	}
	dir := filepath.Dir(path)
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		return xerrors.Errorf("make dir faile: %w", err)
	}
	buf := new(bytes.Buffer)
	_, _ = buf.WriteString("# Default config:\n")
	encoder := toml.NewEncoder(buf)
	if err := encoder.Encode(cfg); err != nil {
		return xerrors.Errorf("encoding config: %w", err)
	}

	return ioutil.WriteFile(path, buf.Bytes(), 0666)
}

func UpdateConfig(path string, cfg interface{}) error {
	path, err := homedir.Expand(path)
	if err != nil {
		return xerrors.Errorf("homedir expand error %s", path)
	}

	buf := new(bytes.Buffer)
	_, _ = buf.WriteString("# Default config:\n")
	encoder := toml.NewEncoder(buf)
	if err := encoder.Encode(cfg); err != nil {
		return xerrors.Errorf("encoding config: %w", err)
	}

	return ioutil.WriteFile(path, buf.Bytes(), 0666)
}

// MinerFromReader loads config from a reader instance.
func MinerFromReader(reader interface {
	io.Reader
	io.Seeker
}) (*StorageMiner, error) {
	cfg := DefaultMainnetStorageMiner()
	meta, err := toml.DecodeReader(reader, cfg)
	if err != nil {
		return nil, err
	}

	// MarketNodeConfig of venus-sealer v1 in configuration file is define as  'Market',
	// but in v1.14.0 is 'MarketNode', the following code is just for compatible,
	// TODO: may there be a more graceful approach to do that.
	if meta.IsDefined("Market") && !meta.IsDefined("MarketNode") {
		if _, err := reader.Seek(0, io.SeekStart); err != nil {
			return nil, xerrors.Errorf("read 'Market' as 'MarketNode' failed, seek file failed:%w", err)
		}
		var tmpMinerCfg = &struct {
			Market MarketNodeConfig `toml:"Market"`
		}{}
		if _, err := toml.DecodeReader(reader, tmpMinerCfg); err != nil {
			return nil, xerrors.Errorf("read 'Market' as 'MarketNode' failed, DecodeReader failed:%w", err)
		}
		cfg.MarketNode = tmpMinerCfg.Market
	}

	err = envconfig.Process("MINER", cfg)
	if err != nil {
		return nil, fmt.Errorf("processing env vars overrides: %s", err)
	}

	return cfg, nil
}

// WorkerFromReader loads config from a reader instance.
func WorkerFromReader(reader io.Reader) (*StorageWorker, error) {
	cfg := GetDefaultWorkerConfig()
	_, err := toml.DecodeReader(reader, cfg)
	if err != nil {
		return nil, err
	}

	err = envconfig.Process("WORKER", cfg)
	if err != nil {
		return nil, fmt.Errorf("processing env vars overrides: %s", err)
	}

	return cfg, nil
}

func ConfigComment(t interface{}) ([]byte, error) {
	buf := new(bytes.Buffer)
	_, _ = buf.WriteString("# Default config:\n")
	e := toml.NewEncoder(buf)
	if err := e.Encode(t); err != nil {
		return nil, xerrors.Errorf("encoding config: %w", err)
	}
	b := buf.Bytes()
	b = bytes.ReplaceAll(b, []byte("\n"), []byte("\n#"))
	b = bytes.ReplaceAll(b, []byte("#["), []byte("["))
	return b, nil
}
