// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/filecoin-project/venus-sealer/sector-storage/stores (interfaces: Store)

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	ffi "github.com/filecoin-project/filecoin-ffi"
	abi "github.com/filecoin-project/go-state-types/abi"
	fsutil "github.com/filecoin-project/venus-sealer/sector-storage/fsutil"
	stores "github.com/filecoin-project/venus-sealer/sector-storage/stores"
	storiface "github.com/filecoin-project/venus-sealer/sector-storage/storiface"
	"github.com/filecoin-project/venus-sealer/sector-storage/ffiwrapper"
	storage "github.com/filecoin-project/specs-storage/storage"
	gomock "github.com/golang/mock/gomock"
)

// MockStore is a mock of Store interface.
type MockStore struct {
	ctrl     *gomock.Controller
	recorder *MockStoreMockRecorder
}

// MockStoreMockRecorder is the mock recorder for MockStore.
type MockStoreMockRecorder struct {
	mock *MockStore
}

// NewMockStore creates a new mock instance.
func NewMockStore(ctrl *gomock.Controller) *MockStore {
	mock := &MockStore{ctrl: ctrl}
	mock.recorder = &MockStoreMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockStore) EXPECT() *MockStoreMockRecorder {
	return m.recorder
}

// AcquireSector mocks base method.
func (m *MockStore) AcquireSector(arg0 context.Context, arg1 storage.SectorRef, arg2, arg3 storiface.SectorFileType, arg4 storiface.PathType, arg5 storiface.AcquireMode) (storiface.SectorPaths, storiface.SectorPaths, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AcquireSector", arg0, arg1, arg2, arg3, arg4, arg5)
	ret0, _ := ret[0].(storiface.SectorPaths)
	ret1, _ := ret[1].(storiface.SectorPaths)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// AcquireSector indicates an expected call of AcquireSector.
func (mr *MockStoreMockRecorder) AcquireSector(arg0, arg1, arg2, arg3, arg4, arg5 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AcquireSector", reflect.TypeOf((*MockStore)(nil).AcquireSector), arg0, arg1, arg2, arg3, arg4, arg5)
}

// FsStat mocks base method.
func (m *MockStore) FsStat(arg0 context.Context, arg1 stores.ID) (fsutil.FsStat, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FsStat", arg0, arg1)
	ret0, _ := ret[0].(fsutil.FsStat)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FsStat indicates an expected call of FsStat.
func (mr *MockStoreMockRecorder) FsStat(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FsStat", reflect.TypeOf((*MockStore)(nil).FsStat), arg0, arg1)
}

// MoveStorage mocks base method.
func (m *MockStore) MoveStorage(arg0 context.Context, arg1 storage.SectorRef, arg2 storiface.SectorFileType) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MoveStorage", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// MoveStorage indicates an expected call of MoveStorage.
func (mr *MockStoreMockRecorder) MoveStorage(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MoveStorage", reflect.TypeOf((*MockStore)(nil).MoveStorage), arg0, arg1, arg2)
}

// Remove mocks base method.
func (m *MockStore) Remove(arg0 context.Context, arg1 abi.SectorID, arg2 storiface.SectorFileType, arg3 bool, arg4 []stores.ID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Remove", arg0, arg1, arg2, arg3, arg4)
	ret0, _ := ret[0].(error)
	return ret0
}

// Remove indicates an expected call of Remove.
func (mr *MockStoreMockRecorder) Remove(arg0, arg1, arg2, arg3, arg4 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Remove", reflect.TypeOf((*MockStore)(nil).Remove), arg0, arg1, arg2, arg3, arg4)
}

// RemoveCopies mocks base method.
func (m *MockStore) RemoveCopies(arg0 context.Context, arg1 abi.SectorID, arg2 storiface.SectorFileType) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RemoveCopies", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// RemoveCopies indicates an expected call of RemoveCopies.
func (mr *MockStoreMockRecorder) RemoveCopies(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RemoveCopies", reflect.TypeOf((*MockStore)(nil).RemoveCopies), arg0, arg1, arg2)
}

// Reserve mocks base method.
func (m *MockStore) Reserve(arg0 context.Context, arg1 storage.SectorRef, arg2 storiface.SectorFileType, arg3 storiface.SectorPaths, arg4 map[storiface.SectorFileType]int) (func(), error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Reserve", arg0, arg1, arg2, arg3, arg4)
	ret0, _ := ret[0].(func())
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Reserve indicates an expected call of Reserve.
func (mr *MockStoreMockRecorder) Reserve(arg0, arg1, arg2, arg3, arg4 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Reserve", reflect.TypeOf((*MockStore)(nil).Reserve), arg0, arg1, arg2, arg3, arg4)
}

// GenerateWindowPoStVanilla mocks base method.
func (m *MockStore) GenerateWindowPoStVanilla(ctx context.Context, minerID abi.ActorID, privsector *ffi.PrivateSectorInfo, vanillaParams string, randomness abi.PoStRandomness) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GenerateWindowPoStVanilla", ctx, minerID, privsector, vanillaParams, randomness)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GenerateWindowPoStVanilla indicates an expected call of GenerateWindowPoStVanilla.
func (mr *MockStoreMockRecorder) GenerateWindowPoStVanilla(ctx, minerID, privsector, vanillaParams, randomness interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GenerateWindowPoStVanilla", reflect.TypeOf((*MockStore)(nil).GenerateWindowPoStVanilla), ctx, minerID, privsector, vanillaParams, randomness)
}

// GenerateWinningPoStVanilla mocks base method.
func (m *MockStore) GenerateWinningPoStVanilla(ctx context.Context, minerID abi.ActorID, privsector *ffi.PrivateSectorInfo, randomness abi.PoStRandomness) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GenerateWinningPoStVanilla", ctx, minerID, privsector, randomness)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GenerateWinningPoStVanilla indicates an expected call of GenerateWinningPoStVanilla.
func (mr *MockStoreMockRecorder) GenerateWinningPoStVanilla(ctx, minerID, privsector, randomness interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GenerateWinningPoStVanilla", reflect.TypeOf((*MockStore)(nil).GenerateWinningPoStVanilla), ctx, minerID, privsector, randomness)
}

// AcquireSectorPaths mocks base method.
func (m *MockStore) AcquireSectorPaths(ctx context.Context, s storage.SectorRef, existing storiface.SectorFileType, allocate storiface.SectorFileType, sealing storiface.PathType) (storiface.SectorPaths, storiface.SectorPaths, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AcquireSectorPaths", ctx, s, existing, allocate, sealing)
	ret0, _ := ret[0].(storiface.SectorPaths)
	ret1, _ := ret[1].(storiface.SectorPaths)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// AcquireSectorPaths indicates an expected call of AcquireSectorPaths.
func (mr *MockStoreMockRecorder) AcquireSectorPaths(ctx, s, existing, allocate, sealing interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AcquireSectorPaths", reflect.TypeOf((*MockStore)(nil).AcquireSectorPaths), ctx, s, existing, allocate, sealing)
}

// GenerateSingleVanillaProof mocks base method.
func (m *MockStore) GenerateSingleVanillaProof(ctx context.Context, minerID abi.ActorID, privsector *ffiwrapper.PrivateSectorInfo, challange []uint64) ([]byte, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GenerateSingleVanillaProof", ctx, minerID, privsector, challange)
	ret0, _ := ret[0].([]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GenerateSingleVanillaProof indicates an expected call of GenerateSingleVanillaProof.
func (mr *MockStoreMockRecorder) GenerateSingleVanillaProof(ctx, minerID, privsector, challange interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GenerateSingleVanillaProof", reflect.TypeOf((*MockStore)(nil).GenerateSingleVanillaProof), ctx, minerID, privsector, challange)
}
