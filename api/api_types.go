//TODO refer api types in venus project
package api

import (
	"bytes"
	"fmt"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-bitfield"
	datatransfer "github.com/filecoin-project/go-data-transfer"
	"github.com/filecoin-project/go-fil-markets/retrievalmarket"
	"github.com/filecoin-project/go-fil-markets/storagemarket"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/venus-sealer/constants"
	"github.com/filecoin-project/venus/pkg/types"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/polydawn/refmt/json"
)

type ActorState struct {
	Balance types.BigInt
	State   interface{}
}

type QueryOffer struct {
	Err string

	Root  cid.Cid
	Piece *cid.Cid

	Size                    uint64
	MinPrice                types.BigInt
	UnsealPrice             types.BigInt
	PaymentInterval         uint64
	PaymentIntervalIncrease uint64
	Miner                   address.Address
	MinerPeer               retrievalmarket.RetrievalPeer
}

func (o *QueryOffer) Order(client address.Address) RetrievalOrder {
	return RetrievalOrder{
		Root:                    o.Root,
		Piece:                   o.Piece,
		Size:                    o.Size,
		Total:                   o.MinPrice,
		UnsealPrice:             o.UnsealPrice,
		PaymentInterval:         o.PaymentInterval,
		PaymentIntervalIncrease: o.PaymentIntervalIncrease,
		Client:                  client,

		Miner:     o.Miner,
		MinerPeer: o.MinerPeer,
	}
}

type RetrievalOrder struct {
	// TODO: make this less unixfs specific
	Root  cid.Cid
	Piece *cid.Cid
	Size  uint64
	// TODO: support offset
	Total                   types.BigInt
	UnsealPrice             types.BigInt
	PaymentInterval         uint64
	PaymentIntervalIncrease uint64
	Client                  address.Address
	Miner                   address.Address
	MinerPeer               retrievalmarket.RetrievalPeer
}

type MethodCall struct {
	types.MessageReceipt
	Error string
}

type StartDealParams struct {
	Data               *storagemarket.DataRef
	Wallet             address.Address
	Miner              address.Address
	EpochPrice         types.BigInt
	MinBlocksDuration  uint64
	ProviderCollateral big.Int
	DealStartEpoch     abi.ChainEpoch
	FastRetrieval      bool
	VerifiedDeal       bool
}

func (s *StartDealParams) UnmarshalJSON(raw []byte) (err error) {
	type sdpAlias StartDealParams

	sdp := sdpAlias{
		FastRetrieval: true,
	}

	if err := json.Unmarshal(raw, &sdp); err != nil {
		return err
	}

	*s = StartDealParams(sdp)

	return nil
}

type IpldObject struct {
	Cid cid.Cid
	Obj interface{}
}

type DealCollateralBounds struct {
	Min abi.TokenAmount
	Max abi.TokenAmount
}

type DataSize struct {
	PayloadSize int64
	PieceSize   abi.PaddedPieceSize
}

type DataCIDSize struct {
	PayloadSize int64
	PieceSize   abi.PaddedPieceSize
	PieceCID    cid.Cid
}

type CommPRet struct {
	Root cid.Cid
	Size abi.UnpaddedPieceSize
}

type MsigProposeResponse int

const (
	MsigApprove MsigProposeResponse = iota
	MsigCancel
)

type Deadline struct {
	PostSubmissions      bitfield.BitField
	DisputableProofCount uint64
}

type Partition struct {
	AllSectors        bitfield.BitField
	FaultySectors     bitfield.BitField
	RecoveringSectors bitfield.BitField
	LiveSectors       bitfield.BitField
	ActiveSectors     bitfield.BitField
}

type Fault struct {
	Miner address.Address
	Epoch abi.ChainEpoch
}

var EmptyVesting = MsigVesting{
	InitialBalance: types.EmptyInt,
	StartEpoch:     -1,
	UnlockDuration: -1,
}

type MsigVesting struct {
	InitialBalance abi.TokenAmount
	StartEpoch     abi.ChainEpoch
	UnlockDuration abi.ChainEpoch
}

type MessageMatch struct {
	To   address.Address
	From address.Address
}

type PubsubScore struct {
	ID    peer.ID
	Score *pubsub.PeerScoreSnapshot
}

type NetBlockList struct {
	Peers     []peer.ID
	IPAddrs   []string
	IPSubnets []string
}

type ObjStat struct {
	Size  uint64
	Links uint64
}

type MessageSendSpec struct {
	MaxFee abi.TokenAmount
}

type DataTransferChannel struct {
	TransferID  datatransfer.TransferID
	Status      datatransfer.Status
	BaseCID     cid.Cid
	IsInitiator bool
	IsSender    bool
	Voucher     string
	Message     string
	OtherPeer   peer.ID
	Transferred uint64
}

// Version provides various build-time information
type Version struct {
	Version string

	// APIVersion is a binary encoded semver version of the remote implementing
	// this api
	//
	// See APIVersion in build/version.go
	APIVersion constants.Version

	// TODO: git commit / os / genesis cid?

	// Seconds
	BlockDelay uint64
}

func (v Version) String() string {
	return fmt.Sprintf("%s+api%s", v.Version, v.APIVersion.String())
}

type NatInfo struct {
	Reachability network.Reachability
	PublicAddr   string
}

//storage
type SealRes struct {
	Err   string
	GoErr error `json:"-"`

	Proof []byte
}

type SectorLog struct {
	Kind      string
	Timestamp uint64

	Trace string

	Message string
}

type SectorInfo struct {
	SectorID     abi.SectorNumber
	State        SectorState
	CommD        *cid.Cid
	CommR        *cid.Cid
	Proof        []byte
	Deals        []abi.DealID
	Ticket       SealTicket
	Seed         SealSeed
	PreCommitMsg *cid.Cid
	CommitMsg    *cid.Cid
	Retries      uint64
	ToUpgrade    bool

	LastErr string

	Log []SectorLog

	// On Chain Info
	SealProof          abi.RegisteredSealProof // The seal proof type implies the PoSt proof/s
	Activation         abi.ChainEpoch          // Epoch during which the sector proof was accepted
	Expiration         abi.ChainEpoch          // Epoch during which the sector expires
	DealWeight         abi.DealWeight          // Integral of active deals over sector lifetime
	VerifiedDealWeight abi.DealWeight          // Integral of active verified deals over sector lifetime
	InitialPledge      abi.TokenAmount         // Pledge collected to commit this sector
	// Expiration Info
	OnTime abi.ChainEpoch
	// non-zero if sector is faulty, epoch at which it will be permanently
	// removed if it doesn't recover
	Early abi.ChainEpoch
}

type SealedRef struct {
	SectorID abi.SectorNumber
	Offset   abi.PaddedPieceSize
	Size     abi.UnpaddedPieceSize
}

type SealedRefs struct {
	Refs []SealedRef
}

type SealTicket struct {
	Value abi.SealRandomness
	Epoch abi.ChainEpoch
}

type SealSeed struct {
	Value abi.InteractiveSealRandomness
	Epoch abi.ChainEpoch
}

func (st *SealTicket) Equals(ost *SealTicket) bool {
	return bytes.Equal(st.Value, ost.Value) && st.Epoch == ost.Epoch
}

func (st *SealSeed) Equals(ost *SealSeed) bool {
	return bytes.Equal(st.Value, ost.Value) && st.Epoch == ost.Epoch
}

type SectorState string

type AddrUse int

const (
	PreCommitAddr AddrUse = iota
	CommitAddr
	PoStAddr

	TerminateSectorsAddr
)

type AddressConfig struct {
	PreCommitControl []address.Address
	CommitControl    []address.Address
	TerminateControl []address.Address
}
