package api

import (
	"context"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/venus-messager/api/client"
	"github.com/filecoin-project/venus-sealer/config"
	"github.com/filecoin-project/venus/venus-shared/types"
	"github.com/filecoin-project/venus/venus-shared/types/messager"
	"github.com/ipfs-force-community/venus-common-utils/apiinfo"
	"golang.org/x/xerrors"
	"time"
)

type IMessager interface {
	WalletHas(ctx context.Context, addr address.Address) (bool, error)
	WaitMessage(ctx context.Context, id string, confidence uint64) (*messager.Message, error)
	PushMessage(ctx context.Context, msg *types.Message, spec *messager.SendSpec) (string, error)
	PushMessageWithId(ctx context.Context, id string, msg *types.Message, spec *messager.SendSpec) (string, error)
	GetMessageByUid(ctx context.Context, id string) (*messager.Message, error)
}

var _ IMessager = (*Messager)(nil)

type Messager struct {
	in client.IMessager
}

func NewMessager(in client.IMessager) *Messager {
	return &Messager{in: in}
}

func (message *Messager) WaitMessage(ctx context.Context, id string, confidence uint64) (*messager.Message, error) {
	tm := time.NewTicker(time.Second * 30)
	defer tm.Stop()

	doneCh := make(chan struct{}, 1)
	doneCh <- struct{}{}

	for {
		select {
		case <-doneCh:
			msg, err := message.in.GetMessageByUid(ctx, id)
			if err != nil {
				log.Warnw("get message fail while wait %w", err)
				time.Sleep(time.Second * 5)
				continue
			}

			switch msg.State {
			//OffChain
			case messager.FillMsg:
				fallthrough
			case messager.UnFillMsg:
				fallthrough
			case messager.UnKnown:
				continue
			//OnChain
			case messager.ReplacedMsg:
				fallthrough
			case messager.OnChainMsg:
				if msg.Confidence > int64(confidence) {
					return msg, nil
				}
				continue
			//Error
			case messager.FailedMsg:
				var reason string
				if msg.Receipt != nil {
					reason = string(msg.Receipt.Return)
				}
				return nil, xerrors.Errorf("msg failed due to %s", reason)
			}

		case <-tm.C:
			doneCh <- struct{}{}
		case <-ctx.Done():
			return nil, xerrors.Errorf("get message fail while wait")
		}
	}
}

func (m *Messager) WalletHas(ctx context.Context, addr address.Address) (bool, error) {
	return m.in.WalletHas(ctx, addr)
}

func (m *Messager) PushMessage(ctx context.Context, msg *types.Message, spec *messager.SendSpec) (string, error) {
	return m.in.PushMessage(ctx, msg, spec)
}

func (m *Messager) PushMessageWithId(ctx context.Context, id string, msg *types.Message, spec *messager.SendSpec) (string, error) {
	return m.in.PushMessageWithId(ctx, id, msg, spec)
}

func (m *Messager) GetMessageByUid(ctx context.Context, id string) (*messager.Message, error) {
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
