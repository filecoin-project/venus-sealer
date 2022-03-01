package modules

import (
	"bytes"
	"context"
	"github.com/filecoin-project/venus-sealer/tool/convert-with-lotus/types"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	levelds "github.com/ipfs/go-ds-leveldb"
	ldbopts "github.com/syndtr/goleveldb/leveldb/opt"
	"go.uber.org/multierr"
	"golang.org/x/xerrors"
	"path/filepath"
)

const (
	fsDatastore       = "datastore"
	SectorStorePrefix = "/sectors"
)

var SectorPrefixKey = datastore.NewKey(SectorStorePrefix)
var MaxSIdKey = datastore.NewKey("/storage/nextid")

func OpenLotusDatastore(repoPath string, readonly bool) (datastore.Batching, error) {
	path := filepath.Join(repoPath, fsDatastore, "metadata")
	return levelds.NewDatastore(path, &levelds.Options{
		Compression: ldbopts.NoCompression,
		NoSync:      false,
		Strict:      ldbopts.StrictAll,
		ReadOnly:    readonly,
	})
}

func ForEachSector(ds datastore.Batching, cb func(info *types.SectorInfo) error) error {
	res, err := ds.Query(context.TODO(), query.Query{})
	if err != nil {
		return err
	}

	defer res.Close()

	var errs error
	for v, isOk := res.NextSync(); isOk; v, isOk = res.NextSync() {
		var sector types.SectorInfo
		if err := sector.UnmarshalCBOR(bytes.NewReader(v.Value)); err != nil {
			errs = multierr.Append(errs, xerrors.Errorf("decoding state for key '%s': %w", v.Key, err))
			continue
		}
		if err := cb(&sector); err != nil {
			return err
		}
	}
	return errs
}
