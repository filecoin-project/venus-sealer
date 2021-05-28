package api

import (
	"context"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/venus-messager/api/client"
	"github.com/filecoin-project/venus-messager/types"
	"github.com/filecoin-project/venus-sealer/config"
	types2 "github.com/filecoin-project/venus/pkg/types"
	"golang.org/x/xerrors"
	"net/http"
)

var ErrFailMsg = xerrors.New("Message Fail")

type IMessager interface {
	HasAddress(ctx context.Context, addr address.Address) (bool, error)
	WaitMessage(ctx context.Context, id string, confidence uint64) (*types.Message, error)
	PushMessage(ctx context.Context, msg *types2.UnsignedMessage, meta *types.MsgMeta) (string, error)
	PushMessageWithId(ctx context.Context, id string, msg *types2.UnsignedMessage, meta *types.MsgMeta) (string, error)
	GetMessageByUid(ctx context.Context, id string) (*types.Message, error)
}

var _ IMessager = (*Messager)(nil)

type Messager struct {
	in client.IMessager
}

func NewMessager(in client.IMessager) *Messager {
	return &Messager{in: in}
}

func (m *Messager) WaitMessage(ctx context.Context, id string, confidence uint64) (*types.Message, error) {
	msg, err := m.in.WaitMessage(ctx, id, confidence)
	if err != nil {
		return nil, err
	}
	if msg.State == types.FailedMsg {
		return nil, ErrFailMsg
	}
	return msg, nil
}

func (m *Messager) HasAddress(ctx context.Context, addr address.Address) (bool, error) {
	return m.in.HasAddress(ctx, addr)
}

func (m *Messager) PushMessage(ctx context.Context, msg *types2.UnsignedMessage, meta *types.MsgMeta) (string, error) {
	return m.in.PushMessage(ctx, msg, meta)
}

func (m *Messager) PushMessageWithId(ctx context.Context, id string, msg *types2.UnsignedMessage, meta *types.MsgMeta) (string, error) {
	return m.in.PushMessageWithId(ctx, id, msg, meta)
}

func (m *Messager) GetMessageByUid(ctx context.Context, id string) (*types.Message, error) {
	return m.in.GetMessageByUid(ctx, id)
}

func NewMessageRPC(messagerCfg *config.MessagerConfig) (IMessager, jsonrpc.ClientCloser, error) {
	headers := http.Header{}
	if len(messagerCfg.Token) != 0 {
		headers.Add("Authorization", "Bearer "+messagerCfg.Token)
	}

	client, closer, err := client.NewMessageRPC(context.Background(), messagerCfg.Url, headers)
	if err != nil {
		return nil, nil, err
	}

	return NewMessager(client), closer, nil
}
