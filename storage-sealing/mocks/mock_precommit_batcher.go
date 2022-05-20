// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/filecoin-project/venus-sealer/storage-sealing (interfaces: PreCommitBatcherApi)

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	address "github.com/filecoin-project/go-address"
	abi "github.com/filecoin-project/go-state-types/abi"
	big "github.com/filecoin-project/go-state-types/big"
	network "github.com/filecoin-project/go-state-types/network"
	types "github.com/filecoin-project/venus-sealer/types"
	types2 "github.com/filecoin-project/venus/venus-shared/types"
	gomock "github.com/golang/mock/gomock"
)

// MockPreCommitBatcherApi is a mock of PreCommitBatcherApi interface.
type MockPreCommitBatcherApi struct {
	ctrl     *gomock.Controller
	recorder *MockPreCommitBatcherApiMockRecorder
}

// MockPreCommitBatcherApiMockRecorder is the mock recorder for MockPreCommitBatcherApi.
type MockPreCommitBatcherApiMockRecorder struct {
	mock *MockPreCommitBatcherApi
}

// NewMockPreCommitBatcherApi creates a new mock instance.
func NewMockPreCommitBatcherApi(ctrl *gomock.Controller) *MockPreCommitBatcherApi {
	mock := &MockPreCommitBatcherApi{ctrl: ctrl}
	mock.recorder = &MockPreCommitBatcherApiMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockPreCommitBatcherApi) EXPECT() *MockPreCommitBatcherApiMockRecorder {
	return m.recorder
}

// ChainBaseFee mocks base method.
func (m *MockPreCommitBatcherApi) ChainBaseFee(arg0 context.Context, arg1 types.TipSetToken) (big.Int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ChainBaseFee", arg0, arg1)
	ret0, _ := ret[0].(big.Int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ChainBaseFee indicates an expected call of ChainBaseFee.
func (mr *MockPreCommitBatcherApiMockRecorder) ChainBaseFee(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ChainBaseFee", reflect.TypeOf((*MockPreCommitBatcherApi)(nil).ChainBaseFee), arg0, arg1)
}

// ChainHead mocks base method.
func (m *MockPreCommitBatcherApi) ChainHead(arg0 context.Context) (types.TipSetToken, abi.ChainEpoch, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ChainHead", arg0)
	ret0, _ := ret[0].(types.TipSetToken)
	ret1, _ := ret[1].(abi.ChainEpoch)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// ChainHead indicates an expected call of ChainHead.
func (mr *MockPreCommitBatcherApiMockRecorder) ChainHead(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ChainHead", reflect.TypeOf((*MockPreCommitBatcherApi)(nil).ChainHead), arg0)
}

// MessagerSendMsg mocks base method.
func (m *MockPreCommitBatcherApi) MessagerSendMsg(arg0 context.Context, arg1, arg2 address.Address, arg3 abi.MethodNum, arg4, arg5 big.Int, arg6 []byte) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MessagerSendMsg", arg0, arg1, arg2, arg3, arg4, arg5, arg6)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// MessagerSendMsg indicates an expected call of MessagerSendMsg.
func (mr *MockPreCommitBatcherApiMockRecorder) MessagerSendMsg(arg0, arg1, arg2, arg3, arg4, arg5, arg6 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MessagerSendMsg", reflect.TypeOf((*MockPreCommitBatcherApi)(nil).MessagerSendMsg), arg0, arg1, arg2, arg3, arg4, arg5, arg6)
}

// StateMinerAvailableBalance mocks base method.
func (m *MockPreCommitBatcherApi) StateMinerAvailableBalance(arg0 context.Context, arg1 address.Address, arg2 types.TipSetToken) (big.Int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StateMinerAvailableBalance", arg0, arg1, arg2)
	ret0, _ := ret[0].(big.Int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// StateMinerAvailableBalance indicates an expected call of StateMinerAvailableBalance.
func (mr *MockPreCommitBatcherApiMockRecorder) StateMinerAvailableBalance(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StateMinerAvailableBalance", reflect.TypeOf((*MockPreCommitBatcherApi)(nil).StateMinerAvailableBalance), arg0, arg1, arg2)
}

// StateMinerInfo mocks base method.
func (m *MockPreCommitBatcherApi) StateMinerInfo(arg0 context.Context, arg1 address.Address, arg2 types.TipSetToken) (types2.MinerInfo, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StateMinerInfo", arg0, arg1, arg2)
	ret0, _ := ret[0].(types2.MinerInfo)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// StateMinerInfo indicates an expected call of StateMinerInfo.
func (mr *MockPreCommitBatcherApiMockRecorder) StateMinerInfo(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StateMinerInfo", reflect.TypeOf((*MockPreCommitBatcherApi)(nil).StateMinerInfo), arg0, arg1, arg2)
}

// StateNetworkVersion mocks base method.
func (m *MockPreCommitBatcherApi) StateNetworkVersion(arg0 context.Context, arg1 types.TipSetToken) (network.Version, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StateNetworkVersion", arg0, arg1)
	ret0, _ := ret[0].(network.Version)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// StateNetworkVersion indicates an expected call of StateNetworkVersion.
func (mr *MockPreCommitBatcherApiMockRecorder) StateNetworkVersion(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StateNetworkVersion", reflect.TypeOf((*MockPreCommitBatcherApi)(nil).StateNetworkVersion), arg0, arg1)
}
