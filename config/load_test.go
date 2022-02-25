package config

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestMinerConfigCompatible(t *testing.T) {
	var exampleCfg = MarketNodeConfig{
		Url:   "/ip4/192.168.200.19/tcp/41235",
		Token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoiMS0xMjUiLCJwZXJtIjoic2lnbiIsImV4dCI6IiJ9.JenwgK0JZcxFDin3cyhBUN41VXNvYw-_0UUT2ZOohM0"}
	var configToml = fmt.Sprintf("[Market]\n\tUrl = \"%s\"\n\tToken = \"%s\"\n", exampleCfg.Url, exampleCfg.Token)
	cfg, err := MinerFromReader(bytes.NewReader([]byte(configToml)))
	require.NoError(t, err)
	require.Equal(t, cfg.MarketNode, exampleCfg)
}
