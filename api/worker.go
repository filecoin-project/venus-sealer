package api

import (
	"context"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/specs-storage/storage"
	"github.com/filecoin-project/venus-sealer/extern/sector-storage/sealtasks"
	"github.com/filecoin-project/venus-sealer/extern/sector-storage/stores"
	"github.com/filecoin-project/venus-sealer/extern/sector-storage/storiface"
	"github.com/google/uuid"
	"github.com/ipfs/go-cid"
	"io"
)

type WorkerStruct struct {
	Internal struct {
		// TODO: lower perms

		Version func(context.Context) (Version, error) `perm:"admin"`

		TaskTypes func(context.Context) (map[sealtasks.TaskType]struct{}, error) `perm:"admin"`
		Paths     func(context.Context) ([]stores.StoragePath, error)            `perm:"admin"`
		Info      func(context.Context) (storiface.WorkerInfo, error)            `perm:"admin"`

		AddPiece        func(ctx context.Context, sector storage.SectorRef, pieceSizes []abi.UnpaddedPieceSize, newPieceSize abi.UnpaddedPieceSize, pieceData storage.Data) (storiface.CallID, error)                 `perm:"admin"`
		SealPreCommit1  func(ctx context.Context, sector storage.SectorRef, ticket abi.SealRandomness, pieces []abi.PieceInfo) (storiface.CallID, error)                                                              `perm:"admin"`
		SealPreCommit2  func(ctx context.Context, sector storage.SectorRef, pc1o storage.PreCommit1Out) (storiface.CallID, error)                                                                                     `perm:"admin"`
		SealCommit1     func(ctx context.Context, sector storage.SectorRef, ticket abi.SealRandomness, seed abi.InteractiveSealRandomness, pieces []abi.PieceInfo, cids storage.SectorCids) (storiface.CallID, error) `perm:"admin"`
		SealCommit2     func(ctx context.Context, sector storage.SectorRef, c1o storage.Commit1Out) (storiface.CallID, error)                                                                                         `perm:"admin"`
		FinalizeSector  func(ctx context.Context, sector storage.SectorRef, keepUnsealed []storage.Range) (storiface.CallID, error)                                                                                   `perm:"admin"`
		ReleaseUnsealed func(ctx context.Context, sector storage.SectorRef, safeToFree []storage.Range) (storiface.CallID, error)                                                                                     `perm:"admin"`
		MoveStorage     func(ctx context.Context, sector storage.SectorRef, types storiface.SectorFileType) (storiface.CallID, error)                                                                                 `perm:"admin"`
		UnsealPiece     func(context.Context, storage.SectorRef, storiface.UnpaddedByteIndex, abi.UnpaddedPieceSize, abi.SealRandomness, cid.Cid) (storiface.CallID, error)                                           `perm:"admin"`
		ReadPiece       func(context.Context, io.Writer, storage.SectorRef, storiface.UnpaddedByteIndex, abi.UnpaddedPieceSize) (storiface.CallID, error)                                                             `perm:"admin"`
		Fetch           func(context.Context, storage.SectorRef, storiface.SectorFileType, storiface.PathType, storiface.AcquireMode) (storiface.CallID, error)                                                       `perm:"admin"`

		TaskDisable func(ctx context.Context, tt sealtasks.TaskType) error `perm:"admin"`
		TaskEnable  func(ctx context.Context, tt sealtasks.TaskType) error `perm:"admin"`

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

func (w *WorkerStruct) Version(ctx context.Context) (Version, error) {
	return w.Internal.Version(ctx)
}

func (w *WorkerStruct) TaskTypes(ctx context.Context) (map[sealtasks.TaskType]struct{}, error) {
	return w.Internal.TaskTypes(ctx)
}

func (w *WorkerStruct) Paths(ctx context.Context) ([]stores.StoragePath, error) {
	return w.Internal.Paths(ctx)
}

func (w *WorkerStruct) Info(ctx context.Context) (storiface.WorkerInfo, error) {
	return w.Internal.Info(ctx)
}

func (w *WorkerStruct) AddPiece(ctx context.Context, sector storage.SectorRef, pieceSizes []abi.UnpaddedPieceSize, newPieceSize abi.UnpaddedPieceSize, pieceData storage.Data) (storiface.CallID, error) {
	return w.Internal.AddPiece(ctx, sector, pieceSizes, newPieceSize, pieceData)
}

func (w *WorkerStruct) SealPreCommit1(ctx context.Context, sector storage.SectorRef, ticket abi.SealRandomness, pieces []abi.PieceInfo) (storiface.CallID, error) {
	return w.Internal.SealPreCommit1(ctx, sector, ticket, pieces)
}

func (w *WorkerStruct) SealPreCommit2(ctx context.Context, sector storage.SectorRef, pc1o storage.PreCommit1Out) (storiface.CallID, error) {
	return w.Internal.SealPreCommit2(ctx, sector, pc1o)
}

func (w *WorkerStruct) SealCommit1(ctx context.Context, sector storage.SectorRef, ticket abi.SealRandomness, seed abi.InteractiveSealRandomness, pieces []abi.PieceInfo, cids storage.SectorCids) (storiface.CallID, error) {
	return w.Internal.SealCommit1(ctx, sector, ticket, seed, pieces, cids)
}

func (w *WorkerStruct) SealCommit2(ctx context.Context, sector storage.SectorRef, c1o storage.Commit1Out) (storiface.CallID, error) {
	return w.Internal.SealCommit2(ctx, sector, c1o)
}

func (w *WorkerStruct) FinalizeSector(ctx context.Context, sector storage.SectorRef, keepUnsealed []storage.Range) (storiface.CallID, error) {
	return w.Internal.FinalizeSector(ctx, sector, keepUnsealed)
}

func (w *WorkerStruct) ReleaseUnsealed(ctx context.Context, sector storage.SectorRef, safeToFree []storage.Range) (storiface.CallID, error) {
	return w.Internal.ReleaseUnsealed(ctx, sector, safeToFree)
}

func (w *WorkerStruct) MoveStorage(ctx context.Context, sector storage.SectorRef, types storiface.SectorFileType) (storiface.CallID, error) {
	return w.Internal.MoveStorage(ctx, sector, types)
}

func (w *WorkerStruct) UnsealPiece(ctx context.Context, sector storage.SectorRef, offset storiface.UnpaddedByteIndex, size abi.UnpaddedPieceSize, ticket abi.SealRandomness, c cid.Cid) (storiface.CallID, error) {
	return w.Internal.UnsealPiece(ctx, sector, offset, size, ticket, c)
}

func (w *WorkerStruct) ReadPiece(ctx context.Context, sink io.Writer, sector storage.SectorRef, offset storiface.UnpaddedByteIndex, size abi.UnpaddedPieceSize) (storiface.CallID, error) {
	return w.Internal.ReadPiece(ctx, sink, sector, offset, size)
}

func (w *WorkerStruct) Fetch(ctx context.Context, id storage.SectorRef, fileType storiface.SectorFileType, ptype storiface.PathType, am storiface.AcquireMode) (storiface.CallID, error) {
	return w.Internal.Fetch(ctx, id, fileType, ptype, am)
}

func (w *WorkerStruct) TaskDisable(ctx context.Context, tt sealtasks.TaskType) error {
	return w.Internal.TaskDisable(ctx, tt)
}

func (w *WorkerStruct) TaskEnable(ctx context.Context, tt sealtasks.TaskType) error {
	return w.Internal.TaskEnable(ctx, tt)
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
