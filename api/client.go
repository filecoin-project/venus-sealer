package api

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/filecoin-project/go-jsonrpc"

	"github.com/ipfs-force-community/venus-common-utils/apiinfo"

	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/venus-sealer/config"
)

var log = logging.Logger("api")

const (
	metadataTraceContext = "traceContext"
)

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

func GetFullNodeFromNodeConfig(ctx context.Context, cfg *config.NodeConfig) (FullNode, jsonrpc.ClientCloser, error) {
	apiInfo := apiinfo.NewAPIInfo(cfg.Url, cfg.Token)
	addr, err := apiInfo.DialArgs("v1")
	if err != nil {
		return nil, nil, xerrors.Errorf("could not get DialArgs: %w", err)
	}
	return NewFullNodeRPC(ctx, addr, apiInfo.AuthHeader())
}

func GetFullNodeAPIFromConfig(cctx *cli.Context) (apiinfo.APIInfo, error) {
	repoPath := cctx.String("repo")
	cfgPath := config.FsConfig(repoPath)
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
		sig := <-sigChan
		log.Warnf("receive sig: %v", sig)
		done()
	}()
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)

	return ctx
}
