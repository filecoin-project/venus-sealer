package venus_sealer

import (
	"context"
	"net"
	"net/http"
	_ "net/http/pprof"

	mux "github.com/gorilla/mux"
	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"go.opencensus.io/tag"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/go-jsonrpc/auth"

	"github.com/filecoin-project/venus-sealer/api"
	"github.com/filecoin-project/venus-sealer/api/impl"
)

// ServeRPC serves an HTTP handler over the supplied listen multiaddr.
//
// This function spawns a goroutine to run the server, and returns immediately.
// It returns the stop function to be called to terminate the endpoint.
//
// The supplied ID is used in tracing, by inserting a tag in the context.
func ServeRPC(h http.Handler, id string, addr multiaddr.Multiaddr) (StopFunc, error) {
	// Start listening to the addr; if invalid or occupied, we will fail early.
	lst, err := manet.Listen(addr)
	if err != nil {
		return nil, xerrors.Errorf("could not listen: %w", err)
	}

	// Instantiate the server and start listening.
	srv := &http.Server{
		Handler: h,
		BaseContext: func(listener net.Listener) context.Context {
			key, _ := tag.NewKey("api")
			ctx, _ := tag.New(context.Background(), tag.Upsert(key, id))
			return ctx
		},
	}

	go func() {
		err = srv.Serve(manet.NetListener(lst))
		if err != http.ErrServerClosed {
			log.Warnf("rpc server failed: %s", err)
		}
	}()

	return srv.Shutdown, err
}

func MinerHandler(mapi api.StorageMiner, permissioned bool) (http.Handler, error) {
	m := mux.NewRouter()

	rpcServer := jsonrpc.NewServer()
	rpcServer.Register("Filecoin", mapi)

	mux := mux.NewRouter()
	mux.Handle("/rpc/v0", rpcServer)
	mux.PathPrefix("/remote").HandlerFunc(mapi.(*impl.StorageMinerAPI).ServeRemote)

	// debugging
	// m.Handle("/debug/metrics", metrics.Exporter())
	m.PathPrefix("/").Handler(http.DefaultServeMux) // pprof

	if !permissioned {
		return rpcServer, nil
	}

	ah := &auth.Handler{
		Verify: mapi.AuthVerify,
		Next:   mux.ServeHTTP,
	}

	return ah, nil
}

