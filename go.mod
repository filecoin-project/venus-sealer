module github.com/filecoin-project/venus-sealer

go 1.15

require (
	contrib.go.opencensus.io/exporter/jaeger v0.1.0
	github.com/BurntSushi/toml v0.3.1
	github.com/acarl005/stripansi v0.0.0-20180116102854-5a71ef0e047d
	github.com/detailyang/go-fallocate v0.0.0-20180908115635-432fa640bd2e
	github.com/dgraph-io/badger/v2 v2.2007.2
	github.com/docker/go-units v0.4.0
	github.com/elastic/go-sysinfo v1.5.0
	github.com/fatih/color v1.10.0
	github.com/filecoin-project/filecoin-ffi v0.30.4-0.20200910194244-f640612a1a1f
	github.com/filecoin-project/go-address v0.0.5
	github.com/filecoin-project/go-bitfield v0.2.4
	github.com/filecoin-project/go-cbor-util v0.0.0-20201016124514-d0bbec7bfcc4
	github.com/filecoin-project/go-commp-utils v0.0.0-20201119054358-b88f7a96a434
	github.com/filecoin-project/go-data-transfer v1.2.7
	github.com/filecoin-project/go-fil-commcid v0.0.0-20201016201715-d41df56b4f6a
	github.com/filecoin-project/go-fil-markets v1.1.7
	github.com/filecoin-project/go-jsonrpc v0.1.4-0.20210217175800-45ea43ac2bec
	github.com/filecoin-project/go-padreader v0.0.0-20201016201355-9c5eb1faedb5
	github.com/filecoin-project/go-paramfetch v0.0.2-0.20200701152213-3e0f0afdc261
	github.com/filecoin-project/go-state-types v0.1.0
	github.com/filecoin-project/go-statemachine v0.0.0-20200925172917-aaed5359be39
	github.com/filecoin-project/go-statestore v0.1.0
	github.com/filecoin-project/go-storedcounter v0.0.0-20200421200003-1c99c62e8a5b
	github.com/filecoin-project/specs-actors v0.9.13
	github.com/filecoin-project/specs-actors/v2 v2.3.4
	github.com/filecoin-project/specs-actors/v3 v3.0.3
	github.com/filecoin-project/specs-storage v0.1.1-0.20201105051918-5188d9774506
	github.com/filecoin-project/venus v0.9.1
	github.com/gbrlsnchs/jwt/v3 v3.0.0
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/snappy v0.0.2-0.20190904063534-ff6b7dc882cf // indirect
	github.com/google/go-cmp v0.5.4 // indirect
	github.com/google/uuid v1.2.0
	github.com/gorilla/mux v1.8.0
	github.com/hako/durafmt v0.0.0-20200710122514-c0fb7b4da026
	github.com/hashicorp/go-multierror v1.1.0
	github.com/ipfs-force-community/venus-messager v0.0.0-20210325080111-009a063e8c97
	github.com/ipfs/go-block-format v0.0.3
	github.com/ipfs/go-cid v0.0.7
	github.com/ipfs/go-datastore v0.4.5
	github.com/ipfs/go-ipfs-blockstore v1.0.3
	github.com/ipfs/go-ipfs-ds-help v1.0.0
	github.com/ipfs/go-ipfs-util v0.0.2
	github.com/ipfs/go-ipld-cbor v0.0.5
	github.com/ipfs/go-log v1.0.4
	github.com/ipfs/go-log/v2 v2.1.2
	github.com/ipfs/go-metrics-interface v0.0.1
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/kr/text v0.2.0 // indirect
	github.com/libp2p/go-buffer-pool v0.0.2
	github.com/libp2p/go-libp2p-core v0.7.0
	github.com/libp2p/go-libp2p-noise v0.1.2 // indirect
	github.com/libp2p/go-libp2p-pubsub v0.4.1
	github.com/minio/sha256-simd v1.0.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/multiformats/go-base32 v0.0.3
	github.com/multiformats/go-multiaddr v0.3.1
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/onsi/ginkgo v1.14.2 // indirect
	github.com/onsi/gomega v1.10.4 // indirect
	github.com/polydawn/refmt v0.0.0-20201211092308-30ac6d18308e
	github.com/raulk/clock v1.1.0
	github.com/shopspring/decimal v1.2.0 // indirect
	github.com/stretchr/testify v1.7.0
	github.com/urfave/cli/v2 v2.3.0
	github.com/whyrusleeping/cbor-gen v0.0.0-20210219115102-f37d292932f2
	github.com/xorcare/golden v0.6.1-0.20191112154924-b87f686d7542 // indirect
	github.com/zbiljic/go-filelock v0.0.0-20170914061330-1dbf7103ab7d
	go.opencensus.io v0.22.6
	go.uber.org/dig v1.10.0 // indirect
	go.uber.org/fx v1.13.1
	go.uber.org/multierr v1.6.0
	go.uber.org/zap v1.16.0
	golang.org/x/crypto v0.0.0-20210220033148-5ea612d1eb83 // indirect
	golang.org/x/exp v0.0.0-20200513190911-00229845015e // indirect
	golang.org/x/mod v0.4.0 // indirect
	golang.org/x/net v0.0.0-20201202161906-c7110b5ffcbb
	golang.org/x/sys v0.0.0-20210301091718-77cc2087c03b
	golang.org/x/text v0.3.4 // indirect
	golang.org/x/tools v0.1.0 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1
	gopkg.in/check.v1 v1.0.0-20200227125254-8fa46927fb4f // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gorm.io/driver/mysql v1.0.4
	gorm.io/driver/sqlite v1.1.4
	gorm.io/gorm v1.20.12
	gotest.tools v2.2.0+incompatible
)

replace github.com/filecoin-project/filecoin-ffi => ./extern/filecoin-ffi

replace github.com/filecoin-project/go-statemachine => github.com/hunjixin/go-statemachine v0.0.0-20210309063909-a65119041e93

replace github.com/filecoin-project/go-statestore => github.com/hunjixin/go-statestore v0.1.1-0.20210309025204-01c69afffbd7
