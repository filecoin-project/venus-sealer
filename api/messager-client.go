package api

import (
	"context"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/venus-messager/api/client"
	"github.com/filecoin-project/venus-messager/types"
	"github.com/filecoin-project/venus-sealer/config"
	types2 "github.com/filecoin-project/venus/pkg/types"
	"github.com/ipfs-force-community/venus-common-utils/apiinfo"
	"golang.org/x/xerrors"
	"time"
)

var ErrFailMsg = xerrors.New("Message Fail")

type IMessager interface {
	WalletHas(ctx context.Context, addr address.Address) (bool, error)
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

func (message *Messager) WaitMessage(ctx context.Context, id string, confidence uint64) (*types.Message, error) {
	tm := time.NewTicker(time.Second * 30)
	defer tm.Stop()

	doneCh := make(chan struct{}, 1)
	doneCh <- struct{}{}

	for {
		select {
		case <-doneCh:
			msg, err := message.in.GetMessageByUid(ctx, id)
			if err != nil {
				return nil, xerrors.Errorf("get message fail while wait %w", ErrFailMsg)
			}

			switch msg.State {
			//OffChain
			case types.FillMsg:
				fallthrough
			case types.UnFillMsg:
				fallthrough
			case types.UnKnown:
				continue
			//OnChain
			case types.ReplacedMsg:
				fallthrough
			case types.OnChainMsg:
				if msg.Confidence > int64(confidence) {
					return msg, nil
				}
				continue
			//Error
			case types.FailedMsg:
				var reason string
				if msg.Receipt != nil {
					reason = string(msg.Receipt.ReturnValue)
				}
				return nil, xerrors.Errorf("msg failed due to %s %w", reason, ErrFailMsg)
			}

		case <-tm.C:
			doneCh <- struct{}{}
		case <-ctx.Done():
			return nil, xerrors.Errorf("get message fail while wait %w", ErrFailMsg)
		}
	}
}

func (m *Messager) WalletHas(ctx context.Context, addr address.Address) (bool, error) {
	return m.in.WalletHas(ctx, addr)
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
	apiInfo := apiinfo.APIInfo{
		Addr:  messagerCfg.Url,
		Token: []byte(messagerCfg.Token),
	}

	addr, err := apiInfo.DialArgs("v0")
	if err != nil {
		return nil, nil, err
	}
	client, closer, err := client.NewMessageRPC(context.Background(), addr, apiInfo.AuthHeader())
	if err != nil {
		return nil, nil, err
	}

	return NewMessager(client), closer, nil
}
