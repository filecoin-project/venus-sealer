package api

import (
	"context"
	"fmt"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/venus-sealer/api/repo"
	"github.com/filecoin-project/venus-sealer/config"
	"github.com/ipfs-force-community/venus-common-utils/apiinfo"
	logging "github.com/ipfs/go-log/v2"
	"github.com/mitchellh/go-homedir"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

var log = logging.Logger("api")

const (
	metadataTraceContext = "traceContext"
)

// The flag passed on the command line with the listen address of the API
// server (only used by the tests)
func flagForAPI(t repo.RepoType) string {
	switch t {
	case repo.FullNode:
		return "api-url"
	case repo.StorageMiner:
		return "miner-api-url"
	case repo.Worker:
		return "worker-api-url"
	default:
		panic(fmt.Sprintf("Unknown repo type: %v", t))
	}
}

func flagForRepo(t repo.RepoType) string {
	switch t {
	case repo.FullNode:
		return "repo"
	case repo.StorageMiner:
		return "miner-repo"
	case repo.Worker:
		return "worker-repo"
	default:
		panic(fmt.Sprintf("Unknown repo type: %v", t))
	}
}

func envForRepo(t repo.RepoType) string {
	switch t {
	case repo.FullNode:
		return "FULLNODE_API_INFO"
	case repo.StorageMiner:
		return "MINER_API_INFO"
	case repo.Worker:
		return "WORKER_API_INFO"
	default:
		panic(fmt.Sprintf("Unknown repo type: %v", t))
	}
}

// TODO remove after deprecation period
func envForRepoDeprecation(t repo.RepoType) string {
	switch t {
	case repo.FullNode:
		return "FULLNODE_API_INFO"
	case repo.StorageMiner:
		return "STORAGE_API_INFO"
	case repo.Worker:
		return "WORKER_API_INFO"
	default:
		panic(fmt.Sprintf("Unknown repo type: %v", t))
	}
}

func GetAPIInfo(ctx *cli.Context, t repo.RepoType) (apiinfo.APIInfo, error) {
	// Check if there was a flag passed with the listen address of the API
	// server (only used by the tests)
	apiFlag := flagForAPI(t)
	if ctx.IsSet(apiFlag) {
		strma := ctx.String(apiFlag)
		strma = strings.TrimSpace(strma)

		return apiinfo.APIInfo{Addr: strma}, nil
	}

	envKey := envForRepo(t)
	env, ok := os.LookupEnv(envKey)
	if !ok {
		// TODO remove after deprecation period
		envKey = envForRepoDeprecation(t)
		env, ok = os.LookupEnv(envKey)
		if ok {
			log.Warnf("Use deprecation env(%s) value, please use env(%s) instead.", envKey, envForRepo(t))
		}
	}
	if ok {
		return apiinfo.ParseApiInfo(env), nil
	}

	repoFlag := flagForRepo(t)

	p, err := homedir.Expand(ctx.String(repoFlag))
	if err != nil {
		return apiinfo.APIInfo{}, xerrors.Errorf("could not expand home dir (%s): %w", repoFlag, err)
	}

	cfg := config.NewLocalStorage(p, "")

	ma, err := cfg.APIEndpoint()
	if err != nil {
		return apiinfo.APIInfo{}, xerrors.Errorf("could not get api endpoint: %w", err)
	}

	token, err := cfg.APIToken()
	if err != nil {
		log.Warnf("Couldn't load CLI token, capabilities may be limited: %v", err)
	}

	return apiinfo.APIInfo{
		Addr:  ma.String(),
		Token: token,
	}, nil
}

func GetRawAPI(ctx *cli.Context, t repo.RepoType) (string, http.Header, error) {
	ainfo, err := GetAPIInfo(ctx, t)
	if err != nil {
		return "", nil, xerrors.Errorf("could not get API info: %w", err)
	}

	addr, err := ainfo.DialArgs("v0")
	if err != nil {
		return "", nil, xerrors.Errorf("could not get DialArgs: %w", err)
	}

	return addr, ainfo.AuthHeader(), nil
}

func GetAPI(ctx *cli.Context) (Common, jsonrpc.ClientCloser, error) {
	ti, ok := ctx.App.Metadata["repoType"]
	if !ok {
		log.Errorf("unknown repo type, are you sure you want to use GetAPI?")
		ti = repo.FullNode
	}
	t, ok := ti.(repo.RepoType)
	if !ok {
		log.Errorf("repoType type does not match the type of repo.RepoType")
	}

	if tn, ok := ctx.App.Metadata["testnode-storage"]; ok {
		return tn.(StorageMiner), func() {}, nil
	}
	if tn, ok := ctx.App.Metadata["testnode-full"]; ok {
		return tn.(FullNode), func() {}, nil
	}

	addr, headers, err := GetRawAPI(ctx, t)
	if err != nil {
		return nil, nil, err
	}

	return NewCommonRPC(ctx.Context, addr, headers)
}

// NewCommonRPC creates a new http jsonrpc client.
func NewCommonRPC(ctx context.Context, addr string, requestHeader http.Header) (Common, jsonrpc.ClientCloser, error) {
	var res CommonStruct
	closer, err := jsonrpc.NewMergeClient(ctx, addr, "Filecoin",
		[]interface{}{
			&res.Internal,
		},
		requestHeader,
	)

	return &res, closer, err
}

func GetFullNodeAPI(ctx *cli.Context) (FullNode, jsonrpc.ClientCloser, error) {
	if tn, ok := ctx.App.Metadata["testnode-full"]; ok {
		return tn.(FullNode), func() {}, nil
	}

	addr, headers, err := GetRawAPI(ctx, repo.FullNode)
	if err != nil {
		return nil, nil, err
	}

	return NewFullNodeRPC(ctx.Context, addr, headers)
}

func GetFullNodeAPIV2(cctx *cli.Context) (FullNode, jsonrpc.ClientCloser, error) {
	apiInfo, err := GetFullNodeAPIFromConfig(cctx)
	if err != nil {
		if !cctx.IsSet("node-url") || !cctx.IsSet("node-token") {
			return nil, nil, xerrors.New("must set url or token")
		}

		apiInfo = apiinfo.APIInfo{
			Addr:  cctx.String("node-url"),
			Token: []byte(cctx.String("node-token")),
		}
	}

	addr, err := apiInfo.DialArgs("v1")
	if err != nil {
		return nil, nil, xerrors.Errorf("could not get DialArgs: %w", err)
	}
	return NewFullNodeRPC(cctx.Context, addr, apiInfo.AuthHeader())
}

func GetFullNodeAPIFromConfig(cctx *cli.Context) (apiinfo.APIInfo, error) {
	cfgPath := cctx.String("config")
	cfg, err := config.MinerFromFile(cfgPath)
	if err != nil {
		return apiinfo.APIInfo{}, err
	}
	cfg.ConfigPath = cfgPath

	return apiinfo.APIInfo{
		Addr:  cfg.Node.Url,
		Token: []byte(cfg.Node.Token),
	}, nil
}

// N
//ewFullNodeRPC creates a new http jsonrpc client.
func NewFullNodeRPC(ctx context.Context, addr string, requestHeader http.Header) (FullNode, jsonrpc.ClientCloser, error) {
	var res FullNodeStruct
	closer, err := jsonrpc.NewMergeClient(ctx, addr, "Filecoin",
		[]interface{}{
			&res.CommonStruct.Internal,
			&res.Internal,
		}, requestHeader)

	return &res, closer, err
}

func DaemonContext(cctx *cli.Context) context.Context {
	if mtCtx, ok := cctx.App.Metadata[metadataTraceContext]; ok {
		return mtCtx.(context.Context)
	}

	return context.Background()
}

// ReqContext returns context for cli execution. Calling it for the first time
// installs SIGTERM handler that will close returned context.
// Not safe for concurrent execution.
func ReqContext(cctx *cli.Context) context.Context {
	tCtx := DaemonContext(cctx)

	ctx, done := context.WithCancel(tCtx)
	sigChan := make(chan os.Signal, 2)
	go func() {
		<-sigChan
		done()
	}()
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)

	return ctx
}
