package api

import (
	"context"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/venus-sealer/config"
	"github.com/filecoin-project/venus-sealer/lib/rpcenc"
	"github.com/ipfs-force-community/venus-common-utils/apiinfo"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"
	"net/http"
	"net/url"
	"path"
	"time"
)

type GetStorageOptions struct {
	PreferHttp bool
	Address    string
	Token      string
}

type GetStorageOption func(*GetStorageOptions)

func StorageMinerUseHttp(opts *GetStorageOptions) {
	opts.PreferHttp = true
}

func StorageSealerAddr(address string) func(*GetStorageOptions) {
	return func(opts *GetStorageOptions) {
		opts.Address = address
	}
}

func StorageSealerToken(token string) func(*GetStorageOptions) {
	return func(opts *GetStorageOptions) {
		opts.Token = token
	}
}

func GetWorkerAPI(ctx *cli.Context, opts ...GetStorageOption) (WorkerAPI, jsonrpc.ClientCloser, error) {
	var options GetStorageOptions
	for _, opt := range opts {
		opt(&options)
	}

	addr, headers, err := GetConfigAPI(ctx, options)
	if err != nil {
		return nil, nil, err
	}

	return NewWorkerRPC(ctx.Context, addr, headers)
}

func NewWorkerRPC(ctx context.Context, addr string, requestHeader http.Header) (WorkerAPI, jsonrpc.ClientCloser, error) {
	u, err := url.Parse(addr)
	if err != nil {
		return nil, nil, err
	}
	switch u.Scheme {
	case "ws":
		u.Scheme = "http"
	case "wss":
		u.Scheme = "https"
	}
	///rpc/v0 -> /rpc/streams/v0/push
	u.Path = path.Join(u.Path, "../streams/v0/push")

	var res WorkerStruct
	closer, err := jsonrpc.NewMergeClient(ctx, addr, "Filecoin",
		[]interface{}{
			&res.Internal,
		},
		requestHeader,
		rpcenc.ReaderParamEncoder(u.String()),
		jsonrpc.WithNoReconnect(),
		jsonrpc.WithTimeout(30*time.Second),
	)

	return &res, closer, err
}

// NewStorageMinerRPC creates a new http jsonrpc client for miner
func NewStorageMinerRPC(ctx context.Context, addr string, requestHeader http.Header, opts ...jsonrpc.Option) (StorageMiner, jsonrpc.ClientCloser, error) {
	var res StorageMinerStruct
	closer, err := jsonrpc.NewMergeClient(ctx, addr, "Filecoin",
		[]interface{}{
			&res.CommonStruct.Internal,
			&res.Internal,
		},
		requestHeader,
		opts...,
	)

	return &res, closer, err
}

func GetStorageMinerAPI(ctx *cli.Context, opts ...GetStorageOption) (StorageMiner, jsonrpc.ClientCloser, error) {
	var options GetStorageOptions
	for _, opt := range opts {
		opt(&options)
	}

	addr, headers, err := GetConfigAPI(ctx, options)
	if err != nil {
		return nil, nil, err
	}

	if options.PreferHttp {
		u, err := url.Parse(addr)
		if err != nil {
			return nil, nil, xerrors.Errorf("parsing miner api URL: %w", err)
		}

		switch u.Scheme {
		case "ws":
			u.Scheme = "http"
		case "wss":
			u.Scheme = "https"
		}

		addr = u.String()
	}

	return NewStorageMinerRPC(ctx.Context, addr, headers)
}

func GetConfigAPI(cctx *cli.Context, options GetStorageOptions) (string, http.Header, error) {
	var apiInfo apiinfo.APIInfo
	if len(options.Address) > 0 {
		apiInfo.Addr = options.Address
		apiInfo.Token = []byte(options.Token)
	} else {
		cfgPath := cctx.String("config")
		cfg, err := config.MinerFromFile(cfgPath)
		if err != nil {
			return "", nil, err
		}

		if cctx.IsSet("data") {
			cfg.DataDir = cctx.String("data")
		}

		ls := config.NewLocalStorage(cfg.DataDir, "")

		ma, err := ls.APIEndpoint()
		if err != nil {
			return "", nil, err
		}
		apiInfo.Addr = ma.String()

		apiInfo.Token, err = ls.APIToken()
		if err != nil {
			log.Warnf("Couldn't load CLI token, capabilities may be limited: %v", err)
		}
	}

	headers := http.Header{}
	if len(apiInfo.Token) != 0 {
		headers.Add("Authorization", "Bearer "+string(apiInfo.Token))
	} else {
		log.Warn("API Token not set and requested, capabilities might be limited.")
	}

	addr, err := apiInfo.DialArgs("v0")
	if err != nil {
		return "", nil, xerrors.Errorf("could not get API info: %w", err)
	}

	return addr, headers, nil
}
