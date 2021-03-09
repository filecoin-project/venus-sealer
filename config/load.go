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

// FromFile loads config from a specified file overriding defaults specified in
// the def parameter. If file does not exist or is empty defaults are assumed.
func FromFile(path string) (*StorageMiner, error) {
	path, err := homedir.Expand(path)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer file.Close() //nolint:errcheck // The file is RO
	return FromReader(file)
}

// FromFile loads config from a specified file overriding defaults specified in
// the def parameter. If file does not exist or is empty defaults are assumed.
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
	file.Close()
	return true, nil
}

func SaveConfig(path string, cfg interface{}) error {
	path, err := homedir.Expand(path)
	if err != nil {
		return xerrors.Errorf("homedir expand error %s", path)
	}
	dir := filepath.Dir(path)
	err = os.MkdirAll(dir, 0777)
	if err != nil {
		return xerrors.Errorf("make dir faile: %w", err)
	}
	buf := new(bytes.Buffer)
	_, _ = buf.WriteString("# Default config:\n")
	e := toml.NewEncoder(buf)
	if err := e.Encode(cfg); err != nil {
		return xerrors.Errorf("encoding config: %w", err)
	}

	return ioutil.WriteFile(path, buf.Bytes(), 0600)
}

// FromReader loads config from a reader instance.
func FromReader(reader io.Reader) (*StorageMiner, error) {
	cfg := DefaultMainnetStorageMiner()
	_, err := toml.DecodeReader(reader, cfg)
	if err != nil {
		return nil, err
	}

	err = envconfig.Process("LOTUS", cfg)
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
