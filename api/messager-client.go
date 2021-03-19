package api

import (
	"context"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/venus-sealer/config"
	"github.com/ipfs-force-community/venus-messager/api/client"
	"net/http"
)

func NewMessageRPC(messagerCfg *config.MessagerConfig) (client.IMessager, jsonrpc.ClientCloser, error) {
	header := http.Header{}
	if len(messagerCfg.Token) != 0 {
		headers := http.Header{}
		headers.Add("Authorization", "Bearer "+string(messagerCfg.Token))
	}

	return client.NewMessageRPC(context.Background(), messagerCfg.Url, header)
}
