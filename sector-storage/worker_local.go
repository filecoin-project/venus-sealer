package sectorstorage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/elastic/go-sysinfo"
	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/ipfs/go-cid"
	"golang.org/x/xerrors"

	ffi "github.com/filecoin-project/filecoin-ffi"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-statestore"
	"github.com/filecoin-project/specs-storage/storage"

	"github.com/filecoin-project/venus-sealer/sector-storage/ffiwrapper"
	"github.com/filecoin-project/venus-sealer/sector-storage/stores"
	"github.com/filecoin-project/venus-sealer/sector-storage/storiface"
	"github.com/filecoin-project/venus-sealer/types"
)

var pathTypes = []storiface.SectorFileType{storiface.FTUnsealed, storiface.FTSealed, storiface.FTCache, storiface.FTUpdate, storiface.FTUpdateCache}

type WorkerConfig struct {
	TaskTypes []types.TaskType
	NoSwap    bool

	// IgnoreResourceFiltering enables task distribution to happen on this
	// worker regardless of its currently available resources. Used in testing
	// with the local worker.
	IgnoreResourceFiltering bool

	TaskTotal int64
}

// used do provide custom proofs impl (mostly used in testing)
type ExecutorFunc func() (ffiwrapper.Storage, error)
type EnvFunc func(string) (string, bool)

type LocalWorker struct {
	storage    stores.Store
	localStore *stores.Local
	sindex     stores.SectorIndex
	ret        storiface.WorkerReturn
	executor   ExecutorFunc
	noSwap     bool
	envLookup  EnvFunc

	// see equivalent field on WorkerConfig.
	ignoreResources bool

	ct          *workerCallTracker
	acceptTasks map[types.TaskType]struct{}
	running     sync.WaitGroup
	taskLk      sync.Mutex
	taskNumber  int64
	taskTotal   int64

	session     uuid.UUID
	testDisable int64
	closing     chan struct{}
}

func newLocalWorker(executor ExecutorFunc, wcfg WorkerConfig, envLookup EnvFunc, store stores.Store, local *stores.Local, sindex stores.SectorIndex, ret storiface.WorkerReturn, cst statestore.StateStore) *LocalWorker {
	acceptTasks := map[types.TaskType]struct{}{}
	for _, taskType := range wcfg.TaskTypes {
		acceptTasks[taskType] = struct{}{}
	}

	w := &LocalWorker{
		storage:    store,
		localStore: local,
		sindex:     sindex,
		ret:        ret,

		ct: &workerCallTracker{
			st: cst,
		},
		acceptTasks:     acceptTasks,
		taskTotal:       wcfg.TaskTotal,
		executor:        executor,
		noSwap:          wcfg.NoSwap,
		envLookup:       envLookup,
		ignoreResources: wcfg.IgnoreResourceFiltering,
		session:         uuid.New(),
		closing:         make(chan struct{}),
	}

	if w.executor == nil {
		w.executor = w.ffiExec
	}

	unfinished, err := w.ct.unfinished()
	if err != nil {
		log.Errorf("reading unfinished tasks: %+v", err)
		return w
	}

	go func() {
		for _, call := range unfinished {
			hostname, osErr := os.Hostname()
			if osErr != nil {
				log.Errorf("get hostname err: %+v", err)
				hostname = ""
			}

			err := storiface.Err(storiface.ErrTempWorkerRestart, xerrors.Errorf("worker [Hostname: %s] restarted", hostname))

			// TODO: Handle restarting PC1 once support is merged

			if doReturn(context.TODO(), call.RetType, call.ID, ret, nil, err) {
				if err := w.ct.onReturned(call.ID); err != nil {
					log.Errorf("marking call as returned failed: %s: %+v", call.RetType, err)
				}
			}
		}
	}()

	return w
}

func NewLocalWorker(wcfg WorkerConfig, store stores.Store, local *stores.Local, sindex stores.SectorIndex, ret storiface.WorkerReturn, cst statestore.StateStore) *LocalWorker {
	return newLocalWorker(nil, wcfg, os.LookupEnv, store, local, sindex, ret, cst)
}

type localWorkerPathProvider struct {
	w  *LocalWorker
	op storiface.AcquireMode
}

func (l *localWorkerPathProvider) AcquireSector(ctx context.Context, sector storage.SectorRef, existing storiface.SectorFileType, allocate storiface.SectorFileType, sealing storiface.PathType) (storiface.SectorPaths, func(), error) {
	paths, storageIDs, err := l.w.storage.AcquireSector(ctx, sector, existing, allocate, sealing, l.op)
	if err != nil {
		return storiface.SectorPaths{}, nil, err
	}

	releaseStorage, err := l.w.localStore.Reserve(ctx, sector, allocate, storageIDs, storiface.FSOverheadSeal)
	if err != nil {
		return storiface.SectorPaths{}, nil, xerrors.Errorf("reserving storage space: %w", err)
	}

	log.Debugf("acquired sector %d (e:%d; a:%d): %v", sector, existing, allocate, paths)

	return paths, func() {
		releaseStorage()

		for _, fileType := range pathTypes {
			if fileType&allocate == 0 {
				continue
			}

			sid := storiface.PathByType(storageIDs, fileType)
			if err := l.w.sindex.StorageDeclareSector(ctx, stores.ID(sid), sector.ID, fileType, l.op == storiface.AcquireMove); err != nil {
				log.Errorf("declare sector error: %+v", err)
			}
		}
	}, nil
}

func (l *LocalWorker) ffiExec() (ffiwrapper.Storage, error) {
	return ffiwrapper.New(&localWorkerPathProvider{w: l})
}

// in: func(WorkerReturn, context.Context, CallID, err string)
// in: func(WorkerReturn, context.Context, CallID, ret T, err string)
func rfunc(in interface{}) func(context.Context, types.CallID, storiface.WorkerReturn, interface{}, *storiface.CallError) error {
	rf := reflect.ValueOf(in)
	ft := rf.Type()
	withRet := ft.NumIn() == 5

	return func(ctx context.Context, ci types.CallID, wr storiface.WorkerReturn, i interface{}, err *storiface.CallError) error {
		rctx := reflect.ValueOf(ctx)
		rwr := reflect.ValueOf(wr)
		rerr := reflect.ValueOf(err)
		rci := reflect.ValueOf(ci)

		var ro []reflect.Value

		if withRet {
			ret := reflect.ValueOf(i)
			if i == nil {
				ret = reflect.Zero(rf.Type().In(3))
			}

			ro = rf.Call([]reflect.Value{rwr, rctx, rci, ret, rerr})
		} else {
			ro = rf.Call([]reflect.Value{rwr, rctx, rci, rerr})
		}

		if !ro[0].IsNil() {
			return ro[0].Interface().(error)
		}

		return nil
	}
}

var returnFunc = map[types.ReturnType]func(context.Context, types.CallID, storiface.WorkerReturn, interface{}, *storiface.CallError) error{
	types.ReturnDataCid:               rfunc(storiface.WorkerReturn.ReturnDataCid),
	types.ReturnAddPiece:              rfunc(storiface.WorkerReturn.ReturnAddPiece),
	types.ReturnSealPreCommit1:        rfunc(storiface.WorkerReturn.ReturnSealPreCommit1),
	types.ReturnSealPreCommit2:        rfunc(storiface.WorkerReturn.ReturnSealPreCommit2),
	types.ReturnSealCommit1:           rfunc(storiface.WorkerReturn.ReturnSealCommit1),
	types.ReturnSealCommit2:           rfunc(storiface.WorkerReturn.ReturnSealCommit2),
	types.ReturnFinalizeSector:        rfunc(storiface.WorkerReturn.ReturnFinalizeSector),
	types.ReturnReleaseUnsealed:       rfunc(storiface.WorkerReturn.ReturnReleaseUnsealed),
	types.ReturnReplicaUpdate:         rfunc(storiface.WorkerReturn.ReturnReplicaUpdate),
	types.ReturnProveReplicaUpdate1:   rfunc(storiface.WorkerReturn.ReturnProveReplicaUpdate1),
	types.ReturnProveReplicaUpdate2:   rfunc(storiface.WorkerReturn.ReturnProveReplicaUpdate2),
	types.ReturnGenerateSectorKey:     rfunc(storiface.WorkerReturn.ReturnGenerateSectorKeyFromData),
	types.ReturnFinalizeReplicaUpdate: rfunc(storiface.WorkerReturn.ReturnFinalizeReplicaUpdate),
	types.ReturnMoveStorage:           rfunc(storiface.WorkerReturn.ReturnMoveStorage),
	types.ReturnUnsealPiece:           rfunc(storiface.WorkerReturn.ReturnUnsealPiece),
	types.ReturnFetch:                 rfunc(storiface.WorkerReturn.ReturnFetch),
}

func (l *LocalWorker) asyncCall(ctx context.Context, sector storage.SectorRef, rt types.ReturnType, work func(ctx context.Context, ci types.CallID) (interface{}, error)) (types.CallID, error) {
	taskNumber := atomic.AddInt64(&l.taskNumber, 1)
	if l.taskTotal >= 0 && taskNumber > l.taskTotal {
		atomic.AddInt64(&l.taskNumber, -1)
		log.Errorf("task number [%d-%d]", taskNumber-1, l.taskTotal)
		return types.CallID{}, xerrors.Errorf("The number of tasks has reached the upper limit [%d-%d] ", taskNumber-1, l.taskTotal)
	}

	ci := types.CallID{
		Sector: sector.ID,
		ID:     uuid.New(),
	}

	if err := l.ct.onStart(ci, rt); err != nil {
		log.Errorf("tracking call (start): %+v", err)
	}

	l.running.Add(1)

	go func() {
		defer func() {
			log.Infof("task [%s] complete for sector %d", rt, sector.ID.Number)
			atomic.AddInt64(&l.taskNumber, -1)
			l.running.Done()
		}()

		ctx := &wctx{
			vals:    ctx,
			closing: l.closing,
		}

		res, err := work(ctx, ci)
		if err != nil {
			rb, err := json.Marshal(res)
			if err != nil {
				log.Errorf("tracking call (marshaling results): %+v", err)
			} else {
				if err := l.ct.onDone(ci, rb); err != nil {
					log.Errorf("tracking call (done): %+v", err)
				}
			}
		}

		if err != nil {
			hostname, osErr := os.Hostname()
			if osErr != nil {
				log.Errorf("get hostname err: %+v", err)
			}

			err = xerrors.Errorf("%w [Hostname: %s]", err, hostname)
		}

		if doReturn(ctx, rt, ci, l.ret, res, toCallError(err)) {
			if err := l.ct.onReturned(ci); err != nil {
				log.Errorf("tracking call (done): %+v", err)
			}
		}
	}()
	return ci, nil
}

func toCallError(err error) *storiface.CallError {
	var serr *storiface.CallError
	if err != nil && !xerrors.As(err, &serr) {
		serr = storiface.Err(storiface.ErrUnknown, err)
	}

	return serr
}

// doReturn tries to send the result to manager, returns true if successful
func doReturn(ctx context.Context, rt types.ReturnType, ci types.CallID, ret storiface.WorkerReturn, res interface{}, rerr *storiface.CallError) bool {
	for {
		err := returnFunc[rt](ctx, ci, ret, res, rerr)
		if err == nil {
			break
		}

		log.Errorf("return error, will retry in 5s: %s: %+v", rt, err)
		select {
		case <-time.After(5 * time.Second):
		case <-ctx.Done():
			log.Errorf("failed to return results: %s", ctx.Err())

			// fine to just return, worker is most likely shutting down, and
			// we didn't mark the result as returned yet, so we'll try to
			// re-submit it on restart
			return false
		}
	}

	return true
}

func (l *LocalWorker) NewSector(ctx context.Context, sector storage.SectorRef) error {
	sb, err := l.executor()
	if err != nil {
		return err
	}

	return sb.NewSector(ctx, sector)
}

func (l *LocalWorker) DataCid(ctx context.Context, pieceSize abi.UnpaddedPieceSize, pieceData storage.Data) (types.CallID, error) {
	sb, err := l.executor()
	if err != nil {
		return types.UndefCall, err
	}

	return l.asyncCall(ctx, storage.NoSectorRef, types.ReturnDataCid, func(ctx context.Context, ci types.CallID) (interface{}, error) {
		return sb.DataCid(ctx, pieceSize, pieceData)
	})
}


func (l *LocalWorker) AddPiece(ctx context.Context, sector storage.SectorRef, epcs []abi.UnpaddedPieceSize, sz abi.UnpaddedPieceSize, r io.Reader) (types.CallID, error) {
	sb, err := l.executor()
	if err != nil {
		return types.UndefCall, err
	}

	return l.asyncCall(ctx, sector, types.ReturnAddPiece, func(ctx context.Context, ci types.CallID) (interface{}, error) {
		return sb.AddPiece(ctx, sector, epcs, sz, r)
	})
}

func (l *LocalWorker) Fetch(ctx context.Context, sector storage.SectorRef, fileType storiface.SectorFileType, ptype storiface.PathType, am storiface.AcquireMode) (types.CallID, error) {
	return l.asyncCall(ctx, sector, types.ReturnFetch, func(ctx context.Context, ci types.CallID) (interface{}, error) {
		_, done, err := (&localWorkerPathProvider{w: l, op: am}).AcquireSector(ctx, sector, fileType, storiface.FTNone, ptype)
		if err == nil {
			done()
		}

		return nil, err
	})
}

func (l *LocalWorker) SealPreCommit1(ctx context.Context, sector storage.SectorRef, ticket abi.SealRandomness, pieces []abi.PieceInfo) (types.CallID, error) {
	return l.asyncCall(ctx, sector, types.ReturnSealPreCommit1, func(ctx context.Context, ci types.CallID) (interface{}, error) {

		{
			// cleanup previous failed attempts if they exist
			if err := l.storage.Remove(ctx, sector.ID, storiface.FTSealed, true, nil); err != nil {
				return nil, xerrors.Errorf("cleaning up sealed data: %w", err)
			}

			if err := l.storage.Remove(ctx, sector.ID, storiface.FTCache, true, nil); err != nil {
				return nil, xerrors.Errorf("cleaning up cache data: %w", err)
			}
		}

		sb, err := l.executor()
		if err != nil {
			return nil, err
		}

		return sb.SealPreCommit1(ctx, sector, ticket, pieces)
	})
}

func (l *LocalWorker) SealPreCommit2(ctx context.Context, sector storage.SectorRef, phase1Out storage.PreCommit1Out) (types.CallID, error) {
	sb, err := l.executor()
	if err != nil {
		return types.UndefCall, err
	}

	return l.asyncCall(ctx, sector, types.ReturnSealPreCommit2, func(ctx context.Context, ci types.CallID) (interface{}, error) {
		return sb.SealPreCommit2(ctx, sector, phase1Out)
	})
}

func (l *LocalWorker) SealCommit1(ctx context.Context, sector storage.SectorRef, ticket abi.SealRandomness, seed abi.InteractiveSealRandomness, pieces []abi.PieceInfo, cids storage.SectorCids) (types.CallID, error) {
	sb, err := l.executor()
	if err != nil {
		return types.UndefCall, err
	}

	return l.asyncCall(ctx, sector, types.ReturnSealCommit1, func(ctx context.Context, ci types.CallID) (interface{}, error) {
		return sb.SealCommit1(ctx, sector, ticket, seed, pieces, cids)
	})
}

func (l *LocalWorker) SealCommit2(ctx context.Context, sector storage.SectorRef, phase1Out storage.Commit1Out) (types.CallID, error) {
	sb, err := l.executor()
	if err != nil {
		return types.UndefCall, err
	}

	return l.asyncCall(ctx, sector, types.ReturnSealCommit2, func(ctx context.Context, ci types.CallID) (interface{}, error) {
		return sb.SealCommit2(ctx, sector, phase1Out)
	})
}

func (l *LocalWorker) ReplicaUpdate(ctx context.Context, sector storage.SectorRef, pieces []abi.PieceInfo) (types.CallID, error) {
	sb, err := l.executor()
	if err != nil {
		return types.UndefCall, err
	}

	return l.asyncCall(ctx, sector, types.ReturnReplicaUpdate, func(ctx context.Context, ci types.CallID) (interface{}, error) {
		sealerOut, err := sb.ReplicaUpdate(ctx, sector, pieces)
		return sealerOut, err
	})
}

func (l *LocalWorker) ProveReplicaUpdate1(ctx context.Context, sector storage.SectorRef, sectorKey, newSealed, newUnsealed cid.Cid) (types.CallID, error) {
	sb, err := l.executor()
	if err != nil {
		return types.UndefCall, err
	}

	return l.asyncCall(ctx, sector, types.ReturnProveReplicaUpdate1, func(ctx context.Context, ci types.CallID) (interface{}, error) {
		return sb.ProveReplicaUpdate1(ctx, sector, sectorKey, newSealed, newUnsealed)
	})
}

func (l *LocalWorker) ProveReplicaUpdate2(ctx context.Context, sector storage.SectorRef, sectorKey, newSealed, newUnsealed cid.Cid, vanillaProofs storage.ReplicaVanillaProofs) (types.CallID, error) {
	sb, err := l.executor()
	if err != nil {
		return types.UndefCall, err
	}

	return l.asyncCall(ctx, sector, types.ReturnProveReplicaUpdate2, func(ctx context.Context, ci types.CallID) (interface{}, error) {
		return sb.ProveReplicaUpdate2(ctx, sector, sectorKey, newSealed, newUnsealed, vanillaProofs)
	})
}

func (l *LocalWorker) GenerateSectorKeyFromData(ctx context.Context, sector storage.SectorRef, commD cid.Cid) (types.CallID, error) {
	sb, err := l.executor()
	if err != nil {
		return types.UndefCall, err
	}

	return l.asyncCall(ctx, sector, types.ReturnGenerateSectorKey, func(ctx context.Context, ci types.CallID) (interface{}, error) {
		return nil, sb.GenerateSectorKeyFromData(ctx, sector, commD)
	})
}

func (l *LocalWorker) FinalizeSector(ctx context.Context, sector storage.SectorRef, keepUnsealed []storage.Range) (types.CallID, error) {
	sb, err := l.executor()
	if err != nil {
		return types.UndefCall, err
	}

	return l.asyncCall(ctx, sector, types.ReturnFinalizeSector, func(ctx context.Context, ci types.CallID) (interface{}, error) {
		if err := sb.FinalizeSector(ctx, sector, keepUnsealed); err != nil {
			return nil, xerrors.Errorf("finalizing sector: %w", err)
		}

		if len(keepUnsealed) == 0 {
			if err := l.storage.Remove(ctx, sector.ID, storiface.FTUnsealed, true, nil); err != nil {
				return nil, xerrors.Errorf("removing unsealed data: %w", err)
			}
		}

		return nil, err
	})
}

func (l *LocalWorker) FinalizeReplicaUpdate(ctx context.Context, sector storage.SectorRef, keepUnsealed []storage.Range) (types.CallID, error) {
	sb, err := l.executor()
	if err != nil {
		return types.UndefCall, err
	}

	return l.asyncCall(ctx, sector, types.ReturnFinalizeReplicaUpdate, func(ctx context.Context, ci types.CallID) (interface{}, error) {
		if err := sb.FinalizeReplicaUpdate(ctx, sector, keepUnsealed); err != nil {
			return nil, xerrors.Errorf("finalizing sector: %w", err)
		}

		if len(keepUnsealed) == 0 {
			if err := l.storage.Remove(ctx, sector.ID, storiface.FTUnsealed, true, nil); err != nil {
				return nil, xerrors.Errorf("removing unsealed data: %w", err)
			}
		}

		return nil, err
	})
}
func (l *LocalWorker) ReleaseUnsealed(ctx context.Context, sector storage.SectorRef, safeToFree []storage.Range) (types.CallID, error) {
	return types.UndefCall, xerrors.Errorf("implement me")
}

func (l *LocalWorker) Remove(ctx context.Context, sector abi.SectorID) error {
	var err error

	if rerr := l.storage.Remove(ctx, sector, storiface.FTSealed, true, nil); rerr != nil {
		err = multierror.Append(err, xerrors.Errorf("removing sector (sealed): %w", rerr))
	}
	if rerr := l.storage.Remove(ctx, sector, storiface.FTCache, true, nil); rerr != nil {
		err = multierror.Append(err, xerrors.Errorf("removing sector (cache): %w", rerr))
	}
	if rerr := l.storage.Remove(ctx, sector, storiface.FTUnsealed, true, nil); rerr != nil {
		err = multierror.Append(err, xerrors.Errorf("removing sector (unsealed): %w", rerr))
	}

	return err
}

func (l *LocalWorker) MoveStorage(ctx context.Context, sector storage.SectorRef, stypes storiface.SectorFileType) (types.CallID, error) {
	return l.asyncCall(ctx, sector, types.ReturnMoveStorage, func(ctx context.Context, ci types.CallID) (interface{}, error) {
		if err := l.storage.MoveStorage(ctx, sector, stypes); err != nil {
			return nil, xerrors.Errorf("move to storage: %w", err)
		}

		for _, fileType := range storiface.PathTypes {
			if fileType&stypes == 0 {
				continue
			}

			if err := l.storage.RemoveCopies(ctx, sector.ID, fileType); err != nil {
				return nil, xerrors.Errorf("rm copies (t:%s, s:%v): %w", fileType, sector, err)
			}
		}
		return nil, nil
	})
}

func (l *LocalWorker) UnsealPiece(ctx context.Context, sector storage.SectorRef, index storiface.UnpaddedByteIndex, size abi.UnpaddedPieceSize, randomness abi.SealRandomness, cid cid.Cid) (types.CallID, error) {
	sb, err := l.executor()
	if err != nil {
		return types.UndefCall, err
	}

	return l.asyncCall(ctx, sector, types.ReturnUnsealPiece, func(ctx context.Context, ci types.CallID) (interface{}, error) {
		log.Debugf("worker will unseal piece now, sector=%+v", sector.ID)
		if err = sb.UnsealPiece(ctx, sector, index, size, randomness, cid); err != nil {
			return nil, xerrors.Errorf("unsealing sector: %w", err)
		}

		if err = l.storage.RemoveCopies(ctx, sector.ID, storiface.FTSealed); err != nil {
			return nil, xerrors.Errorf("removing source data: %w", err)
		}

		if err = l.storage.RemoveCopies(ctx, sector.ID, storiface.FTCache); err != nil {
			return nil, xerrors.Errorf("removing source data: %w", err)
		}

		log.Debugf("worker has unsealed piece, sector=%+v", sector.ID)

		return nil, nil
	})
}

func (l *LocalWorker) TaskTypes(context.Context) (map[types.TaskType]struct{}, error) {
	l.taskLk.Lock()
	defer l.taskLk.Unlock()

	return l.acceptTasks, nil
}

func (l *LocalWorker) TaskDisable(ctx context.Context, tt types.TaskType) error {
	l.taskLk.Lock()
	defer l.taskLk.Unlock()

	delete(l.acceptTasks, tt)
	return nil
}

func (l *LocalWorker) TaskEnable(ctx context.Context, tt types.TaskType) error {
	l.taskLk.Lock()
	defer l.taskLk.Unlock()

	l.acceptTasks[tt] = struct{}{}
	return nil
}

func (l *LocalWorker) Paths(ctx context.Context) ([]stores.StoragePath, error) {
	return l.localStore.Local(ctx)
}

func (l *LocalWorker) TaskNumbers(context.Context) (string, error) {
	str := fmt.Sprintf("%d-%d", l.taskNumber, l.taskTotal)
	return str, nil
}

func (l *LocalWorker) memInfo() (memPhysical, memUsed, memSwap, memSwapUsed uint64, err error) {
	h, err := sysinfo.Host()
	if err != nil {
		return 0, 0, 0, 0, err
	}

	mem, err := h.Memory()
	if err != nil {
		return 0, 0, 0, 0, err
	}
	memPhysical = mem.Total
	// mem.Available is memory available without swapping, it is more relevant for this calculation
	memUsed = mem.Total - mem.Available
	memSwap = mem.VirtualTotal
	memSwapUsed = mem.VirtualUsed

	if cgMemMax, cgMemUsed, cgSwapMax, cgSwapUsed, err := cgroupV1Mem(); err == nil {
		if cgMemMax > 0 && cgMemMax < memPhysical {
			memPhysical = cgMemMax
			memUsed = cgMemUsed
		}
		if cgSwapMax > 0 && cgSwapMax < memSwap {
			memSwap = cgSwapMax
			memSwapUsed = cgSwapUsed
		}
	}

	if cgMemMax, cgMemUsed, cgSwapMax, cgSwapUsed, err := cgroupV2Mem(); err == nil {
		if cgMemMax > 0 && cgMemMax < memPhysical {
			memPhysical = cgMemMax
			memUsed = cgMemUsed
		}
		if cgSwapMax > 0 && cgSwapMax < memSwap {
			memSwap = cgSwapMax
			memSwapUsed = cgSwapUsed
		}
	}

	if l.noSwap {
		memSwap = 0
		memSwapUsed = 0
	}

	return memPhysical, memUsed, memSwap, memSwapUsed, nil
}

func (l *LocalWorker) Info(context.Context) (storiface.WorkerInfo, error) {
	hostname, err := os.Hostname() // TODO: allow overriding from config
	if err != nil {
		panic(err)
	}

	gpus, err := ffi.GetGPUDevices()
	if err != nil {
		log.Errorf("getting gpu devices failed: %+v", err)
	}

	memPhysical, memUsed, memSwap, memSwapUsed, err := l.memInfo()
	if err != nil {
		return storiface.WorkerInfo{}, xerrors.Errorf("getting memory info: %w", err)
	}

	resEnv, err := storiface.ParseResourceEnv(func(key, def string) (string, bool) {
		return l.envLookup(key)
	})
	if err != nil {
		return storiface.WorkerInfo{}, xerrors.Errorf("interpreting resource env vars: %w", err)
	}

	return storiface.WorkerInfo{
		Hostname:        hostname,
		IgnoreResources: l.ignoreResources,
		Resources: storiface.WorkerResources{
			MemPhysical: memPhysical,
			MemUsed:     memUsed,
			MemSwap:     memSwap,
			MemSwapUsed: memSwapUsed,
			CPUs:        uint64(runtime.NumCPU()),
			GPUs:        gpus,
			Resources:   resEnv,
		},
	}, nil
}

func (l *LocalWorker) Session(ctx context.Context) (uuid.UUID, error) {
	if atomic.LoadInt64(&l.testDisable) == 1 {
		return uuid.UUID{}, xerrors.Errorf("disabled")
	}

	select {
	case <-l.closing:
		return ClosedWorkerID, nil
	default:
		return l.session, nil
	}
}

func (l *LocalWorker) Close() error {
	close(l.closing)
	return nil
}

// WaitQuiet blocks as long as there are tasks running
func (l *LocalWorker) WaitQuiet() {
	l.running.Wait()
}

type wctx struct {
	vals    context.Context
	closing chan struct{}
}

func (w *wctx) Deadline() (time.Time, bool) {
	return time.Time{}, false
}

func (w *wctx) Done() <-chan struct{} {
	return w.closing
}

func (w *wctx) Err() error {
	select {
	case <-w.closing:
		return context.Canceled
	default:
		return nil
	}
}

func (w *wctx) Value(key interface{}) interface{} {
	return w.vals.Value(key)
}

var _ context.Context = &wctx{}

var _ Worker = &LocalWorker{}
