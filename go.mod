module github.com/filecoin-project/venus-sealer

go 1.15

require (
	github.com/detailyang/go-fallocate v0.0.0-20180908115635-432fa640bd2e
	github.com/elastic/go-sysinfo v1.5.0
	github.com/filecoin-project/filecoin-ffi v0.30.4-0.20200910194244-f640612a1a1f
	github.com/filecoin-project/go-address v0.0.5-0.20201103152444-f2023ef3f5bb
	github.com/filecoin-project/go-bitfield v0.2.3-0.20201110211213-fe2c1862e816
	github.com/filecoin-project/go-cbor-util v0.0.0-20201016124514-d0bbec7bfcc4
	github.com/filecoin-project/go-commp-utils v0.0.0-20201119054358-b88f7a96a434
	github.com/filecoin-project/go-fil-commcid v0.0.0-20201016201715-d41df56b4f6a
	github.com/filecoin-project/go-padreader v0.0.0-20201016201355-9c5eb1faedb5
	github.com/filecoin-project/go-paramfetch v0.0.2-0.20200701152213-3e0f0afdc261
	github.com/filecoin-project/go-state-types v0.0.0-20201102161440-c8033295a1fc
	github.com/filecoin-project/go-statemachine v0.0.0-20200925172917-aaed5359be39
	github.com/filecoin-project/go-statestore v0.1.0
	github.com/filecoin-project/lotus v1.4.1
	github.com/filecoin-project/specs-actors v0.9.13
	github.com/filecoin-project/specs-actors/v2 v2.3.4
	github.com/filecoin-project/specs-storage v0.1.1-0.20201105051918-5188d9774506
	github.com/google/uuid v1.2.0
	github.com/gorilla/mux v1.8.0
	github.com/hashicorp/go-multierror v1.1.0
	github.com/ipfs/go-cid v0.0.7
	github.com/ipfs/go-datastore v0.4.5
	github.com/ipfs/go-log v1.0.4
	github.com/ipfs/go-log/v2 v2.1.2-0.20200626104915-0016c0b4b3e4
	github.com/mitchellh/go-homedir v1.1.0
	github.com/stretchr/testify v1.6.1
	github.com/whyrusleeping/cbor-gen v0.0.0-20200826160007-0b9f6c5fb163
	go.opencensus.io v0.22.6
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1
	gotest.tools v2.2.0+incompatible
)

replace github.com/filecoin-project/filecoin-ffi => ./extern/filecoin-ffi
