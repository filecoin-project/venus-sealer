package repo

import "github.com/filecoin-project/venus-sealer/types"

type WorkerCallRepo interface {
	GetCallByCallID(role string, callId types.CallID) (*types.Call, error)
	HasCall(role string, callId types.CallID) (bool, error)
	Save(role string, callId types.CallID, call *types.Call) error
	GetAllCall(role string) ([]*types.Call, error)
	DeleteByCallID(role string, callId types.CallID) error
	UpdateCallByCallID(role string, call *types.Call, callId types.CallID) error
}
