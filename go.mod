module github.com/filecoin-project/venus-sealer

go 1.16

require (
	contrib.go.opencensus.io/exporter/jaeger v0.2.1
	github.com/BurntSushi/toml v0.4.1
	github.com/acarl005/stripansi v0.0.0-20180116102854-5a71ef0e047d
	github.com/containerd/cgroups v0.0.0-20201119153540-4cbc285b3327
	github.com/detailyang/go-fallocate v0.0.0-20180908115635-432fa640bd2e
	github.com/dgraph-io/badger/v2 v2.2007.3
	github.com/docker/go-units v0.4.0
	github.com/elastic/go-sysinfo v1.7.0
	github.com/fatih/color v1.13.0
	github.com/filecoin-project/dagstore v0.4.4
	github.com/filecoin-project/filecoin-ffi v0.30.4-0.20200910194244-f640612a1a1f
	github.com/filecoin-project/go-address v0.0.6
	github.com/filecoin-project/go-bitfield v0.2.4
	github.com/filecoin-project/go-cbor-util v0.0.1
	github.com/filecoin-project/go-commp-utils v0.1.3
	github.com/filecoin-project/go-crypto v0.0.1 // indirect
	github.com/filecoin-project/go-data-transfer v1.12.1
	github.com/filecoin-project/go-fil-commcid v0.1.0
	github.com/filecoin-project/go-fil-markets v1.14.1
	github.com/filecoin-project/go-jsonrpc v0.1.5
	github.com/filecoin-project/go-padreader v0.0.1
	github.com/filecoin-project/go-paramfetch v0.0.3-0.20220111000201-e42866db1a53
	github.com/filecoin-project/go-state-types v0.1.3
	github.com/filecoin-project/go-statemachine v1.0.1
	github.com/filecoin-project/go-statestore v0.2.0
	github.com/filecoin-project/go-storedcounter v0.1.0
	github.com/filecoin-project/specs-actors v0.9.14
	github.com/filecoin-project/specs-actors/v2 v2.3.6
	github.com/filecoin-project/specs-actors/v3 v3.1.1
	github.com/filecoin-project/specs-actors/v5 v5.0.4
	github.com/filecoin-project/specs-actors/v6 v6.0.1
	github.com/filecoin-project/specs-actors/v7 v7.0.0-20211230214648-aeae366b083a
	github.com/filecoin-project/specs-storage v0.1.1-0.20211228030229-6d460d25a0c9
	github.com/filecoin-project/venus v1.1.3-rc1.0.20220112052852-049c87c57362
	github.com/filecoin-project/venus-market v1.0.2-0.20220106080434-e8470650617d
	github.com/filecoin-project/venus-messager v1.2.2-rc1.0.20220106055402-ca00521b9a61
	github.com/gbrlsnchs/jwt/v3 v3.0.1
	github.com/go-kit/kit v0.12.0 // indirect
	github.com/golang/mock v1.6.0
	github.com/google/uuid v1.3.0
	github.com/gorilla/mux v1.8.0
	github.com/hako/durafmt v0.0.0-20200710122514-c0fb7b4da026
	github.com/hashicorp/go-multierror v1.1.1
	github.com/icza/backscanner v0.0.0-20210726202459-ac2ffc679f94
	github.com/ipfs-force-community/venus-common-utils v0.0.0-20211122032945-eb6cab79c62a
	github.com/ipfs-force-community/venus-gateway v1.1.2-0.20220112095107-6e37950f9f47
	github.com/ipfs/go-block-format v0.0.3
	github.com/ipfs/go-cid v0.1.0
	github.com/ipfs/go-datastore v0.5.1
	github.com/ipfs/go-ds-leveldb v0.5.0
	github.com/ipfs/go-graphsync v0.11.5
	github.com/ipfs/go-ipfs-blockstore v1.1.2
	github.com/ipfs/go-ipfs-ds-help v1.1.0
	github.com/ipfs/go-ipfs-files v0.0.9 // indirect
	github.com/ipfs/go-ipfs-util v0.0.2
	github.com/ipfs/go-ipld-cbor v0.0.6
	github.com/ipfs/go-ipld-legacy v0.1.1 // indirect
	github.com/ipfs/go-log/v2 v2.4.0
	github.com/ipfs/go-metrics-interface v0.0.1
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/libp2p/go-buffer-pool v0.0.2
	github.com/libp2p/go-libp2p-core v0.13.0
	github.com/libp2p/go-libp2p-pubsub v0.6.0
	github.com/mattn/go-runewidth v0.0.10 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/modern-go/reflect2 v1.0.2
	github.com/multiformats/go-base32 v0.0.4
	github.com/multiformats/go-multiaddr v0.4.1
	github.com/multiformats/go-multihash v0.1.0
	github.com/polydawn/refmt v0.0.0-20201211092308-30ac6d18308e
	github.com/raulk/clock v1.1.0
	github.com/stretchr/testify v1.7.0
	github.com/syndtr/goleveldb v1.0.0
	github.com/urfave/cli/v2 v2.3.0
	github.com/whyrusleeping/cbor-gen v0.0.0-20210713220151-be142a5ae1a8
	github.com/zbiljic/go-filelock v0.0.0-20170914061330-1dbf7103ab7d
	go.opencensus.io v0.23.0
	go.uber.org/fx v1.15.0
	go.uber.org/multierr v1.7.0
	go.uber.org/zap v1.19.1
	golang.org/x/crypto v0.0.0-20211209193657-4570a0811e8b // indirect
	golang.org/x/net v0.0.0-20211112202133-69e39bad7dc2
	golang.org/x/sys v0.0.0-20211209171907-798191bca915
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1
	gorm.io/driver/mysql v1.1.1
	gorm.io/driver/sqlite v1.1.4
	gorm.io/gorm v1.21.12
	gotest.tools v2.2.0+incompatible
	lukechampine.com/blake3 v1.1.7 // indirect
)

replace github.com/filecoin-project/filecoin-ffi => ./extern/filecoin-ffi

replace github.com/ipfs/go-ipfs-cmds => github.com/ipfs-force-community/go-ipfs-cmds v0.6.1-0.20210521090123-4587df7fa0ab

replace github.com/filecoin-project/go-jsonrpc => github.com/ipfs-force-community/go-jsonrpc v0.1.4-0.20210721095535-a67dff16de21

replace github.com/filecoin-project/go-statemachine => github.com/hunjixin/go-statemachine v0.0.0-20220110084945-5867c28ba08a

replace github.com/filecoin-project/go-statestore => github.com/hunjixin/go-statestore v0.1.1-0.20211229093043-b4de7dc02a01
