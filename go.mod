module github.com/filecoin-project/venus-sealer

go 1.16

require (
	contrib.go.opencensus.io/exporter/jaeger v0.2.1
	github.com/BurntSushi/toml v0.3.1
	github.com/acarl005/stripansi v0.0.0-20180116102854-5a71ef0e047d
	github.com/detailyang/go-fallocate v0.0.0-20180908115635-432fa640bd2e
	github.com/dgraph-io/badger/v2 v2.2007.2
	github.com/docker/go-units v0.4.0
	github.com/elastic/go-sysinfo v1.7.0
	github.com/fatih/color v1.10.0
	github.com/filecoin-project/filecoin-ffi v0.30.4-0.20200910194244-f640612a1a1f
	github.com/filecoin-project/go-address v0.0.5
	github.com/filecoin-project/go-bitfield v0.2.4
	github.com/filecoin-project/go-cbor-util v0.0.0-20201016124514-d0bbec7bfcc4
	github.com/filecoin-project/go-commp-utils v0.1.1-0.20210427191551-70bf140d31c7
	github.com/filecoin-project/go-data-transfer v1.10.1
	github.com/filecoin-project/go-fil-commcid v0.1.0
	github.com/filecoin-project/go-fil-markets v1.12.0
	github.com/filecoin-project/go-jsonrpc v0.1.4-0.20210217175800-45ea43ac2bec
	github.com/filecoin-project/go-padreader v0.0.0-20210723183308-812a16dc01b1
	github.com/filecoin-project/go-paramfetch v0.0.2
	github.com/filecoin-project/go-state-types v0.1.1-0.20210915140513-d354ccf10379
	github.com/filecoin-project/go-statemachine v1.0.1
	github.com/filecoin-project/go-statestore v0.1.1
	github.com/filecoin-project/go-storedcounter v0.0.0-20200421200003-1c99c62e8a5b
	github.com/filecoin-project/specs-actors v0.9.14
	github.com/filecoin-project/specs-actors/v2 v2.3.5
	github.com/filecoin-project/specs-actors/v3 v3.1.1
	github.com/filecoin-project/specs-actors/v5 v5.0.4
	github.com/filecoin-project/specs-actors/v6 v6.0.0
	github.com/filecoin-project/specs-storage v0.1.1-0.20201105051918-5188d9774506
	github.com/filecoin-project/venus v1.1.1-rc1
	github.com/filecoin-project/venus-market v1.0.1
	github.com/filecoin-project/venus-messager v1.2.0
	github.com/gbrlsnchs/jwt/v3 v3.0.0
	github.com/golang/mock v1.6.0
	github.com/google/uuid v1.2.0
	github.com/gorilla/mux v1.8.0
	github.com/hako/durafmt v0.0.0-20200710122514-c0fb7b4da026
	github.com/hashicorp/go-multierror v1.1.1
	github.com/icza/backscanner v0.0.0-20210726202459-ac2ffc679f94
	github.com/ipfs-force-community/venus-common-utils v0.0.0-20210924063144-1d3a5b30de87
	github.com/ipfs-force-community/venus-gateway v1.1.2-0.20210924083450-c55d3300dfc9
	github.com/ipfs/go-block-format v0.0.3
	github.com/ipfs/go-cid v0.1.0
	github.com/ipfs/go-datastore v0.4.6
	github.com/ipfs/go-ds-leveldb v0.4.2
	github.com/ipfs/go-ipfs-blockstore v1.0.4
	github.com/ipfs/go-ipfs-ds-help v1.0.0
	github.com/ipfs/go-ipfs-util v0.0.2
	github.com/ipfs/go-ipld-cbor v0.0.5
	github.com/ipfs/go-log/v2 v2.3.0
	github.com/ipfs/go-metrics-interface v0.0.1
	github.com/ipld/go-car v0.3.2-0.20211001225732-32d0d9933823 // indirect
	github.com/ipld/go-ipld-prime v0.12.3 // indirect
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/libp2p/go-buffer-pool v0.0.2
	github.com/libp2p/go-libp2p-core v0.8.6
	github.com/libp2p/go-libp2p-pubsub v0.5.4
	github.com/mitchellh/go-homedir v1.1.0
	github.com/modern-go/reflect2 v1.0.1
	github.com/multiformats/go-base32 v0.0.3
	github.com/multiformats/go-multiaddr v0.3.3
	github.com/polydawn/refmt v0.0.0-20201211092308-30ac6d18308e
	github.com/raulk/clock v1.1.0
	github.com/stretchr/testify v1.7.0
	github.com/syndtr/goleveldb v1.0.0
	github.com/urfave/cli/v2 v2.3.0
	github.com/whyrusleeping/cbor-gen v0.0.0-20210713220151-be142a5ae1a8
	github.com/zbiljic/go-filelock v0.0.0-20170914061330-1dbf7103ab7d
	go.opencensus.io v0.23.0
	go.uber.org/fx v1.13.1
	go.uber.org/multierr v1.7.0
	go.uber.org/zap v1.16.0
	golang.org/x/net v0.0.0-20210525063256-abc453219eb5
	golang.org/x/sys v0.0.0-20210927094055-39ccf1dd6fa6
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1
	gorm.io/driver/mysql v1.1.1
	gorm.io/driver/sqlite v1.1.4
	gorm.io/gorm v1.21.12
	gotest.tools v2.2.0+incompatible
)

replace github.com/filecoin-project/filecoin-ffi => ./extern/filecoin-ffi

replace github.com/ipfs/go-ipfs-cmds => github.com/ipfs-force-community/go-ipfs-cmds v0.6.1-0.20210521090123-4587df7fa0ab

replace github.com/filecoin-project/go-jsonrpc => github.com/ipfs-force-community/go-jsonrpc v0.1.4-0.20211222073806-e895f2de9729

replace github.com/filecoin-project/go-statemachine => github.com/hunjixin/go-statemachine v0.0.0-20210309063909-a65119041e93

replace github.com/filecoin-project/go-statestore => github.com/hunjixin/go-statestore v0.1.1-0.20210309025204-01c69afffbd7
