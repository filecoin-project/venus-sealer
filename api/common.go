package api

import (
	"context"
	"github.com/google/uuid"

	"github.com/filecoin-project/go-jsonrpc/auth"
)

type Common interface {

	// MethodGroup: Auth

	AuthVerify(ctx context.Context, token string) ([]auth.Permission, error)
	AuthNew(ctx context.Context, perms []auth.Permission) ([]byte, error)

	// Version provides information about API provider
	Version(context.Context) (Version, error)

	LogList(context.Context) ([]string, error)
	LogSetLevel(context.Context, string, string) error

	// trigger graceful shutdown
	Shutdown(context.Context) error

	// Session returns a random UUID of api provider session
	Session(context.Context) (uuid.UUID, error)

	Closing(context.Context) (<-chan struct{}, error)

	// Token returns api token
	Token(ctx context.Context) ([]byte, error)
}
