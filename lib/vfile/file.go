package vfile

import "os"

func EnsureDir(path string) error {
	_, err := os.Stat(path)
	if err == os.ErrNotExist {
		return os.MkdirAll(path, 0777)
	} else if err == nil {
		return nil
	} else {
		return err
	}
}
