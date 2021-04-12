package api

import (
	"context"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/venus-sealer/config"
	types2 "github.com/filecoin-project/venus/pkg/types"
	"github.com/ipfs-force-community/venus-messager/api/client"
	"github.com/ipfs-force-community/venus-messager/types"
	"net/http"
)

type IMessager interface {
	HasWalletAddress(ctx context.Context, addr address.Address) (bool, error)
	WaitMessage(ctx context.Context, id string, confidence uint64) (*types.Message, error)
	PushMessage(ctx context.Context, msg *types2.UnsignedMessage, meta *types.MsgMeta) (string, error)
	PushMessageWithId(ctx context.Context, id string, msg *types2.UnsignedMessage, meta *types.MsgMeta) (string, error)
	GetMessageByUid(ctx context.Context, id string) (*types.Message, error)
}

var _ IMessager = (*Messager)(nil)

type Messager struct {
	in         client.IMessager
	walletName string
}

func NewMessager(in client.IMessager, walletName string) *Messager {
	return &Messager{in: in, walletName: walletName}
}

func (m *Messager) WaitMessage(ctx context.Context, id string, confidence uint64) (*types.Message, error) {
	return m.in.WaitMessage(ctx, id, confidence)
}

func (m *Messager) HasWalletAddress(ctx context.Context, addr address.Address) (bool, error) {
	return m.in.HasWalletAddress(ctx, m.walletName, addr)
}

func (m *Messager) PushMessage(ctx context.Context, msg *types2.UnsignedMessage, meta *types.MsgMeta) (string, error) {
	return m.in.PushMessage(ctx, msg, meta, m.walletName)
}

func (m *Messager) PushMessageWithId(ctx context.Context, id string, msg *types2.UnsignedMessage, meta *types.MsgMeta) (string, error) {
	return m.in.PushMessageWithId(ctx, id, msg, meta, m.walletName)
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

	return NewMessager(client, messagerCfg.Wallet), closer, nil
}
