package sealing

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mitchellh/go-homedir"
	"golang.org/x/xerrors"

	ffi "github.com/filecoin-project/filecoin-ffi"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/specs-storage/storage"
	"github.com/filecoin-project/venus-sealer/sector-storage/storiface"
	"github.com/filecoin-project/venus-sealer/types"
)

func (m *Sealing) CurrentSectorID(ctx context.Context) (abi.SectorNumber, error) {
	return m.sc.GetStorageCounter()
}

func (m *Sealing) PledgeSector(ctx context.Context) (storage.SectorRef, error) {
	m.startupWait.Wait()

	m.inputLk.Lock()
	defer m.inputLk.Unlock()

	cfg, err := m.getConfig()
	if err != nil {
		return storage.SectorRef{}, xerrors.Errorf("getting config: %w", err)
	}

	if cfg.MaxSealingSectors > 0 {
		if m.stats.CurSealing() >= cfg.MaxSealingSectors {
			return storage.SectorRef{}, xerrors.Errorf("too many sectors sealing (curSealing: %d, max: %d)", m.stats.CurSealing(), cfg.MaxSealingSectors)
		}
	}

	spt, err := m.currentSealProof(ctx)
	if err != nil {
		return storage.SectorRef{}, xerrors.Errorf("getting seal proof type: %w", err)
	}

	sid, err := m.createSector(ctx, cfg, spt)
	if err != nil {
		return storage.SectorRef{}, err
	}

	log.Infof("Creating CC sector %d", sid)
	return m.minerSector(spt, sid), m.sectors.Send(uint64(sid), SectorStartCC{
		ID:         sid,
		SectorType: spt,
	})
}

func (m *Sealing) RedoSector(ctx context.Context, rsi storiface.SectorRedoParams) error  {
	var si types.SectorInfo
	err := m.sectors.Get(uint64(rsi.SectorNumber)).Get(&si)
	if err != nil {
		return err
	}

	go func() {
		// P1
		if err := checkPieces(ctx, m.maddr, si, m.api); err != nil { // Sanity check state
			switch err.(type) {
			case *ErrApi:
				log.Errorf("handlePreCommit1: api error in sector %d, not proceeding: %+v", si.SectorNumber, err)
				return
			case *ErrInvalidDeals:
				log.Errorf("invalid deals in sector %d: %v", si.SectorNumber, err)
				return
			case *ErrExpiredDeals: // Probably not much we can do here, maybe re-pack the sector?
				log.Errorf("expired dealIDs in sector %d: %s", si.SectorNumber, err)
			default:
				log.Errorf("checkPieces sanity check error in sector %d: %s", si.SectorNumber, err)
			}
		}

		// Copy basic unsealed to the sealed directory
		sector := m.minerSector(si.SectorType, si.SectorNumber)
		ssize, err := sector.ProofType.SectorSize()
		if err != nil {
			log.Errorf("sector %d err: %s", si.SectorNumber, err)
			return
		}

		sid := m.minerSectorID(si.SectorNumber)
		tUnsealedFile := storiface.DefaultUnsealedFile(ssize)
		paths := storiface.SectorPaths{
			ID: sid,
			Unsealed: rsi.SectorSealPath(sid, storiface.FTUnsealed),
			Sealed: rsi.SectorSealPath(sid, storiface.FTSealed),
			Cache: rsi.SectorSealPath(sid, storiface.FTCache),
		}

		log.Infof("sector %d sealPaths: %v", si.SectorNumber, paths)
		err = os.MkdirAll(filepath.Join(rsi.SealPath, storiface.FTSealed.String()), 0755)
		if err != nil {
			log.Errorf("sector %d mkdir %s err: %s", si.SectorNumber, filepath.Join(rsi.SealPath, storiface.FTSealed.String()), err)
		}
		err = os.MkdirAll(filepath.Join(rsi.SealPath, storiface.FTCache.String()), 0755)
		if err != nil {
			log.Errorf("sector %d mkdir %s err: %s", si.SectorNumber, filepath.Join(rsi.SealPath, storiface.FTCache.String()), err)
		}
		err = os.MkdirAll(filepath.Join(rsi.SealPath, storiface.FTUnsealed.String()), 0755)
		if err != nil {
			log.Errorf("sector %d mkdir %s err: %s", si.SectorNumber, filepath.Join(rsi.SealPath, storiface.FTUnsealed.String()), err)
		}

		uf, err := os.OpenFile(paths.Unsealed, os.O_RDWR|os.O_CREATE, 0644) // nolint:gosec
		if err != nil {
			log.Errorf("ensuring sealed file exists: %s", err)
			return
		}
		if err := uf.Close(); err != nil {
			log.Errorf("sector %d err: %s", si.SectorNumber, err)
			return
		}

		if bExist, _ := storiface.FileExists(tUnsealedFile); bExist {
			if bExist, _ := storiface.FileExists(paths.Unsealed); !bExist {
				err = storiface.CopyFile(tUnsealedFile, paths.Unsealed)
				if err != nil {
					log.Errorf("sector %d err: %s", si.SectorNumber, err)
					return
				}
			}
		} else {
			log.Errorf("The default unsealed does not exist,please copy a generated unsealed file to %s", tUnsealedFile)
			return
		}

		e, err := os.OpenFile(paths.Sealed, os.O_RDWR|os.O_CREATE, 0644) // nolint:gosec
		if err != nil {
			log.Errorf("ensuring sealed file exists: %s", err)
			return
		}
		if err := e.Close(); err != nil {
			log.Errorf("sector %d err: %s", si.SectorNumber, err)
			return
		}

		if err := os.Mkdir(paths.Cache, 0755); err != nil { // nolint
			if os.IsExist(err) {
				log.Warnf("existing cache in %s; removing", paths.Cache)

				if err := os.RemoveAll(paths.Cache); err != nil {
					log.Errorf("sector %d remove existing sector cache from %s err: %s", si.SectorNumber, paths.Cache, err)
					return
				}

				if err := os.Mkdir(paths.Cache, 0755); err != nil { // nolint:gosec
					log.Errorf("sector %d mkdir cache path after cleanup err: %s", si.SectorNumber, err)
					return
				}
			} else {
				log.Errorf("sector %d err: %s", si.SectorNumber, err)
				return
			}
		}

		var sum abi.UnpaddedPieceSize
		pieces := si.PieceInfos()
		for _, piece := range pieces {
			sum += piece.Size.Unpadded()
		}
		ussize := abi.PaddedPieceSize(ssize).Unpadded()
		if sum != ussize {
			log.Errorf("sector %d aggregated piece sizes don't match sector size: %d != %d (%d)", si.SectorNumber,  sum, ussize, int64(ussize-sum))
			return
		}

		// TODO: context cancellation respect
		p1o, err := ffi.SealPreCommitPhase1(
			sector.ProofType,
			paths.Cache,
			paths.Unsealed,
			paths.Sealed,
			sector.ID.Number,
			sector.ID.Miner,
			si.TicketValue,
			pieces,
		)
		if err != nil {
			log.Errorf("sector %d presealing_01 %s: %s", si.SectorNumber,  paths.Unsealed, err)
			return
		}

		// P2
		_, _, err = ffi.SealPreCommitPhase2(p1o, paths.Cache, paths.Sealed)  // sealedCID, unsealedCID
		if err != nil {
			log.Errorf("sector %d presealing_02 err: %s", si.SectorNumber,  paths.Unsealed, err)
			return
		}

		// C1
		//_, err = ffi.SealCommitPhase1( // output
		//	sector.ProofType,
		//	sealedCID,
		//	unsealedCID,
		//	paths.Cache,
		//	paths.Sealed,
		//	sector.ID.Number,
		//	sector.ID.Miner,
		//	si.TicketValue,
		//	si.SeedValue,
		//	pieces,
		//)
		//if err != nil {
		//	log.Errorf("sector %d StandaloneSealCommit_01 error: %s", si.SectorNumber, err)
		//	return
		//}

		//// C2
		//_, err = ffi.SealCommitPhase2(output, sector.ID.Number, sector.ID.Miner)
		//if err != nil {
		//	log.Errorf("sector %d StandaloneSealCommit_02 error: %s", si.SectorNumber, err)
		//	return
		//}

		// 拷贝和删除文件
		storePaths := storiface.SectorPaths{
			ID: sid,
			Sealed: rsi.SectorStorePath(sid, storiface.FTSealed),
			Cache: rsi.SectorStorePath(sid, storiface.FTCache),
		}

		err = os.MkdirAll(filepath.Join(rsi.StorePath, storiface.FTSealed.String()), 0755)
		if err != nil {
			log.Errorf("sector %d mkdir %s err: %s", si.SectorNumber, filepath.Join(rsi.StorePath, storiface.FTSealed.String()), err)
		}
		err = os.MkdirAll(filepath.Join(rsi.StorePath, storiface.FTCache.String()), 0755)
		if err != nil {
			log.Errorf("sector %d mkdir %s err: %s", si.SectorNumber, filepath.Join(rsi.StorePath, storiface.FTCache.String()), err)
		}

		log.Infof("sector %d storePaths: %v", si.SectorNumber, storePaths)
		err = storiface.CopyFile(paths.Sealed, storePaths.Sealed)
		if err != nil {
			log.Errorf("sector %d copy sealed file error: %s", si.SectorNumber, err)
		}

		// 删除目录下sc-02-data-layer-*.dat,sc-02-data-tree-c.dat,sc-02-data-tree-d.dat
		files, errDir := ioutil.ReadDir(paths.Cache)
		if errDir != nil {
			log.Fatal(errDir)
		}

		for _, file := range files {
			if file.IsDir() {

			} else {
				// 输出绝对路径
				strPath := filepath.Join(paths.Cache, file.Name())
				if strings.Contains(strPath, "sc-02-data-layer") || strings.Contains(strPath, "sc-02-data-tree-c") || strings.Contains(strPath, "sc-02-data-tree-d") {
					log.Infof("remove %s", strPath)
					os.Remove(strPath)
				}
			}
		}

		err = move(paths.Cache, storePaths.Cache)
		if err != nil {
			log.Errorf("sector %d copy cache file error: %s", si.SectorNumber, err)
		}

		// 删除seal目录
		os.RemoveAll(paths.Sealed)
		os.RemoveAll(paths.Unsealed)
		os.RemoveAll(paths.Cache)

		log.Infof("sector %d redo success ...", si.SectorNumber)
	}()

	return nil
}

func move(from, to string) error {
	from, err := homedir.Expand(from)
	if err != nil {
		return xerrors.Errorf("move: expanding from: %w", err)
	}

	to, err = homedir.Expand(to)
	if err != nil {
		return xerrors.Errorf("move: expanding to: %w", err)
	}

	if filepath.Base(from) != filepath.Base(to) {
		return xerrors.Errorf("move: base names must match ('%s' != '%s')", filepath.Base(from), filepath.Base(to))
	}

	log.Debugw("move sector data", "from", from, "to", to)

	toDir := filepath.Dir(to)

	// `mv` has decades of experience in moving files quickly; don't pretend we
	//  can do better

	var errOut bytes.Buffer

	var cmd *exec.Cmd
	if runtime.GOOS == "darwin" {
		if err := os.MkdirAll(toDir, 0777); err != nil {
			return xerrors.Errorf("failed exec MkdirAll: %s", err)
		}

		cmd = exec.Command("/usr/bin/env", "mv", from, toDir) // nolint
	} else {
		cmd = exec.Command("/usr/bin/env", "mv", "-t", toDir, from) // nolint
	}

	cmd.Stderr = &errOut
	if err := cmd.Run(); err != nil {
		return xerrors.Errorf("exec mv (stderr: %s): %w", strings.TrimSpace(errOut.String()), err)
	}

	return nil
}
