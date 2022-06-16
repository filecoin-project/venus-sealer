package api

import (
	"context"
	"io"

	"github.com/google/uuid"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/specs-storage/storage"

	"github.com/filecoin-project/venus-sealer/constants"
	"github.com/filecoin-project/venus-sealer/sector-storage/stores"
	"github.com/filecoin-project/venus-sealer/sector-storage/storiface"
	"github.com/filecoin-project/venus-sealer/types"
)

type WorkerStruct struct {
	Internal struct {
		// TODO: lower perms

		Version func(context.Context) (constants.Version, error) `perm:"admin"`

		TaskTypes func(context.Context) (map[types.TaskType]struct{}, error) `perm:"admin"`
		Paths     func(context.Context) ([]stores.StoragePath, error)        `perm:"admin"`
		Info      func(context.Context) (storiface.WorkerInfo, error)        `perm:"admin"`

		DataCid                   func(ctx context.Context, pieceSize abi.UnpaddedPieceSize, pieceData storage.Data) (types.CallID, error)                                                                                  `perm:"admin"`
		AddPiece                  func(ctx context.Context, sector storage.SectorRef, pieceSizes []abi.UnpaddedPieceSize, newPieceSize abi.UnpaddedPieceSize, pieceData storage.Data) (types.CallID, error)                 `perm:"admin"`
		SealPreCommit1            func(ctx context.Context, sector storage.SectorRef, ticket abi.SealRandomness, pieces []abi.PieceInfo) (types.CallID, error)                                                              `perm:"admin"`
		SealPreCommit2            func(ctx context.Context, sector storage.SectorRef, pc1o storage.PreCommit1Out) (types.CallID, error)                                                                                     `perm:"admin"`
		SealCommit1               func(ctx context.Context, sector storage.SectorRef, ticket abi.SealRandomness, seed abi.InteractiveSealRandomness, pieces []abi.PieceInfo, cids storage.SectorCids) (types.CallID, error) `perm:"admin"`
		SealCommit2               func(ctx context.Context, sector storage.SectorRef, c1o storage.Commit1Out) (types.CallID, error)                                                                                         `perm:"admin"`
		FinalizeReplicaUpdate     func(p0 context.Context, p1 storage.SectorRef, p2 []storage.Range) (types.CallID, error)                                                                                                  `perm:"admin"`
		FinalizeSector            func(ctx context.Context, sector storage.SectorRef, keepUnsealed []storage.Range) (types.CallID, error)                                                                                   `perm:"admin"`
		ReplicaUpdate             func(ctx context.Context, sector storage.SectorRef, pieces []abi.PieceInfo) (types.CallID, error)                                                                                         `perm:"admin"`
		ProveReplicaUpdate1       func(ctx context.Context, sector storage.SectorRef, sectorKey, newSealed, newUnsealed cid.Cid) (types.CallID, error)                                                                      `perm:"admin"`
		ProveReplicaUpdate2       func(ctx context.Context, sector storage.SectorRef, sectorKey, newSealed, newUnsealed cid.Cid, vanillaProofs storage.ReplicaVanillaProofs) (types.CallID, error)                          `perm:"admin"`
		GenerateSectorKeyFromData func(ctx context.Context, sector storage.SectorRef, commD cid.Cid) (types.CallID, error)                                                                                                  `perm:"admin"`
		ReleaseUnsealed           func(ctx context.Context, sector storage.SectorRef, safeToFree []storage.Range) (types.CallID, error)                                                                                     `perm:"admin"`
		MoveStorage               func(ctx context.Context, sector storage.SectorRef, types storiface.SectorFileType) (types.CallID, error)                                                                                 `perm:"admin"`
		UnsealPiece               func(context.Context, storage.SectorRef, storiface.UnpaddedByteIndex, abi.UnpaddedPieceSize, abi.SealRandomness, cid.Cid) (types.CallID, error)                                           `perm:"admin"`
		ReadPiece                 func(context.Context, io.Writer, storage.SectorRef, storiface.UnpaddedByteIndex, abi.UnpaddedPieceSize) (types.CallID, error)                                                             `perm:"admin"`
		Fetch                     func(context.Context, storage.SectorRef, storiface.SectorFileType, storiface.PathType, storiface.AcquireMode) (types.CallID, error)                                                       `perm:"admin"`

		TaskDisable func(ctx context.Context, tt types.TaskType) error `perm:"admin"`
		TaskEnable  func(ctx context.Context, tt types.TaskType) error `perm:"admin"`

		TaskNumbers func(context.Context) (string, error) `perm:"admin"`

		Remove          func(ctx context.Context, sector abi.SectorID) error `perm:"admin"`
		StorageAddLocal func(ctx context.Context, path string) error         `perm:"admin"`

		SetEnabled func(ctx context.Context, enabled bool) error `perm:"admin"`
		Enabled    func(ctx context.Context) (bool, error)       `perm:"admin"`

		WaitQuiet func(ctx context.Context) error `perm:"admin"`

		ProcessSession func(context.Context) (uuid.UUID, error) `perm:"admin"`
		Session        func(context.Context) (uuid.UUID, error) `perm:"admin"`
	}
}

// WorkerStruct

func (w *WorkerStruct) Version(ctx context.Context) (constants.Version, error) {
	return w.Internal.Version(ctx)
}

func (w *WorkerStruct) TaskTypes(ctx context.Context) (map[types.TaskType]struct{}, error) {
	return w.Internal.TaskTypes(ctx)
}

func (w *WorkerStruct) Paths(ctx context.Context) ([]stores.StoragePath, error) {
	return w.Internal.Paths(ctx)
}

func (w *WorkerStruct) Info(ctx context.Context) (storiface.WorkerInfo, error) {
	return w.Internal.Info(ctx)
}

func (w *WorkerStruct) DataCid(ctx context.Context, pieceSize abi.UnpaddedPieceSize, pieceData storage.Data) (types.CallID, error) {
	return w.Internal.DataCid(ctx, pieceSize, pieceData)
}

func (w *WorkerStruct) AddPiece(ctx context.Context, sector storage.SectorRef, pieceSizes []abi.UnpaddedPieceSize, newPieceSize abi.UnpaddedPieceSize, pieceData storage.Data) (types.CallID, error) {
	return w.Internal.AddPiece(ctx, sector, pieceSizes, newPieceSize, pieceData)
}

func (w *WorkerStruct) SealPreCommit1(ctx context.Context, sector storage.SectorRef, ticket abi.SealRandomness, pieces []abi.PieceInfo) (types.CallID, error) {
	return w.Internal.SealPreCommit1(ctx, sector, ticket, pieces)
}

func (w *WorkerStruct) SealPreCommit2(ctx context.Context, sector storage.SectorRef, pc1o storage.PreCommit1Out) (types.CallID, error) {
	return w.Internal.SealPreCommit2(ctx, sector, pc1o)
}

func (w *WorkerStruct) SealCommit1(ctx context.Context, sector storage.SectorRef, ticket abi.SealRandomness, seed abi.InteractiveSealRandomness, pieces []abi.PieceInfo, cids storage.SectorCids) (types.CallID, error) {
	return w.Internal.SealCommit1(ctx, sector, ticket, seed, pieces, cids)
}

func (w *WorkerStruct) SealCommit2(ctx context.Context, sector storage.SectorRef, c1o storage.Commit1Out) (types.CallID, error) {
	return w.Internal.SealCommit2(ctx, sector, c1o)
}

func (s *WorkerStruct) FinalizeReplicaUpdate(p0 context.Context, p1 storage.SectorRef, p2 []storage.Range) (types.CallID, error) {
	if s.Internal.FinalizeReplicaUpdate == nil {
		return *new(types.CallID), ErrNotSupported
	}
	return s.Internal.FinalizeReplicaUpdate(p0, p1, p2)
}

func (w *WorkerStruct) FinalizeSector(ctx context.Context, sector storage.SectorRef, keepUnsealed []storage.Range) (types.CallID, error) {
	return w.Internal.FinalizeSector(ctx, sector, keepUnsealed)
}

func (w *WorkerStruct) ReplicaUpdate(ctx context.Context, sector storage.SectorRef, pieces []abi.PieceInfo) (types.CallID, error) {
	return w.Internal.ReplicaUpdate(ctx, sector, pieces)
}

func (w *WorkerStruct) ProveReplicaUpdate1(ctx context.Context, sector storage.SectorRef, sectorKey, newSealed, newUnsealed cid.Cid) (types.CallID, error) {
	return w.Internal.ProveReplicaUpdate1(ctx, sector, sectorKey, newSealed, newUnsealed)
}

func (w *WorkerStruct) ProveReplicaUpdate2(ctx context.Context, sector storage.SectorRef, sectorKey, newSealed, newUnsealed cid.Cid, vanillaProofs storage.ReplicaVanillaProofs) (types.CallID, error) {
	return w.Internal.ProveReplicaUpdate2(ctx, sector, sectorKey, newSealed, newUnsealed, vanillaProofs)
}

func (w *WorkerStruct) GenerateSectorKeyFromData(ctx context.Context, sector storage.SectorRef, commD cid.Cid) (types.CallID, error) {
	return w.Internal.GenerateSectorKeyFromData(ctx, sector, commD)
}

func (w *WorkerStruct) ReleaseUnsealed(ctx context.Context, sector storage.SectorRef, safeToFree []storage.Range) (types.CallID, error) {
	return w.Internal.ReleaseUnsealed(ctx, sector, safeToFree)
}

func (w *WorkerStruct) MoveStorage(ctx context.Context, sector storage.SectorRef, types storiface.SectorFileType) (types.CallID, error) {
	return w.Internal.MoveStorage(ctx, sector, types)
}

func (w *WorkerStruct) UnsealPiece(ctx context.Context, sector storage.SectorRef, offset storiface.UnpaddedByteIndex, size abi.UnpaddedPieceSize, ticket abi.SealRandomness, c cid.Cid) (types.CallID, error) {
	return w.Internal.UnsealPiece(ctx, sector, offset, size, ticket, c)
}

func (w *WorkerStruct) ReadPiece(ctx context.Context, sink io.Writer, sector storage.SectorRef, offset storiface.UnpaddedByteIndex, size abi.UnpaddedPieceSize) (types.CallID, error) {
	return w.Internal.ReadPiece(ctx, sink, sector, offset, size)
}

func (w *WorkerStruct) Fetch(ctx context.Context, id storage.SectorRef, fileType storiface.SectorFileType, ptype storiface.PathType, am storiface.AcquireMode) (types.CallID, error) {
	return w.Internal.Fetch(ctx, id, fileType, ptype, am)
}

func (w *WorkerStruct) TaskDisable(ctx context.Context, tt types.TaskType) error {
	return w.Internal.TaskDisable(ctx, tt)
}

func (w *WorkerStruct) TaskEnable(ctx context.Context, tt types.TaskType) error {
	return w.Internal.TaskEnable(ctx, tt)
}

func (w *WorkerStruct) TaskNumbers(ctx context.Context) (string, error) {
	return w.Internal.TaskNumbers(ctx)
}

func (w *WorkerStruct) Remove(ctx context.Context, sector abi.SectorID) error {
	return w.Internal.Remove(ctx, sector)
}

func (w *WorkerStruct) StorageAddLocal(ctx context.Context, path string) error {
	return w.Internal.StorageAddLocal(ctx, path)
}

func (w *WorkerStruct) SetEnabled(ctx context.Context, enabled bool) error {
	return w.Internal.SetEnabled(ctx, enabled)
}

func (w *WorkerStruct) Enabled(ctx context.Context) (bool, error) {
	return w.Internal.Enabled(ctx)
}

func (w *WorkerStruct) WaitQuiet(ctx context.Context) error {
	return w.Internal.WaitQuiet(ctx)
}

func (w *WorkerStruct) ProcessSession(ctx context.Context) (uuid.UUID, error) {
	return w.Internal.ProcessSession(ctx)
}

func (w *WorkerStruct) Session(ctx context.Context) (uuid.UUID, error) {
	return w.Internal.Session(ctx)
}
