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
	e := toml.NewEncoder(buf)
	if err := e.Encode(cfg); err != nil {
		return xerrors.Errorf("encoding config: %w", err)
	}

	return ioutil.WriteFile(path, buf.Bytes(), 0666)
}

// MinerFromReader loads config from a reader instance.
func MinerFromReader(reader io.Reader) (*StorageMiner, error) {
	cfg := DefaultMainnetStorageMiner()
	_, err := toml.DecodeReader(reader, cfg)
	if err != nil {
		return nil, err
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
