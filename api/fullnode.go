package api

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-bitfield"
	"github.com/filecoin-project/go-jsonrpc/auth"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/go-state-types/dline"
	stnetwork "github.com/filecoin-project/go-state-types/network"

	"github.com/google/uuid"
	"github.com/ipfs/go-cid"
	metrics "github.com/libp2p/go-libp2p-core/metrics"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	protocol "github.com/libp2p/go-libp2p-core/protocol"

	"github.com/filecoin-project/venus/app/submodule/apitypes"
	paych2 "github.com/filecoin-project/venus/app/submodule/paych"
	"github.com/filecoin-project/venus/pkg/chain"
	"github.com/filecoin-project/venus/pkg/messagepool"
	"github.com/filecoin-project/venus/pkg/specactors/builtin/miner"
	"github.com/filecoin-project/venus/pkg/specactors/builtin/paych"
	"github.com/filecoin-project/venus/pkg/types"

	types2 "github.com/filecoin-project/venus-sealer/types"
)

// FullNode API is a low-level interface to the Filecoin network full node
type FullNode interface {
	Common

	// MethodGroup: Chain
	// The Chain method group contains methods for interacting with the
	// blockchain, but that do not require any form of state computation.

	// ChainNotify returns channel with chain head updates.
	// First message is guaranteed to be of len == 1, and type == 'current'.
	ChainNotify(context.Context) (<-chan []*chain.HeadChange, error)

	// ChainHead returns the current head of the chain.
	ChainHead(context.Context) (*types.TipSet, error)

	// ChainGetRandomnessFromTickets is used to sample the chain for randomness.
	ChainGetRandomnessFromTickets(ctx context.Context, tsk types.TipSetKey, personalization crypto.DomainSeparationTag, randEpoch abi.ChainEpoch, entropy []byte) (abi.Randomness, error)

	// ChainGetRandomnessFromBeacon is used to sample the beacon for randomness.
	ChainGetRandomnessFromBeacon(ctx context.Context, tsk types.TipSetKey, personalization crypto.DomainSeparationTag, randEpoch abi.ChainEpoch, entropy []byte) (abi.Randomness, error)

	// ChainGetBlock returns the block specified by the given CID.
	ChainGetBlock(context.Context, cid.Cid) (*types.BlockHeader, error)
	// ChainGetTipSet returns the tipset specified by the given TipSetKey.
	ChainGetTipSet(context.Context, types.TipSetKey) (*types.TipSet, error)

	// ChainGetBlockMessages returns messages stored in the specified block.
	ChainGetBlockMessages(ctx context.Context, blockCid cid.Cid) (*apitypes.BlockMessages, error)

	// ChainGetParentReceipts returns receipts for messages in parent tipset of
	// the specified block.
	ChainGetParentReceipts(ctx context.Context, blockCid cid.Cid) ([]*types.MessageReceipt, error)

	// ChainGetParentMessages returns messages stored in parent tipset of the
	// specified block.
	ChainGetParentMessages(ctx context.Context, blockCid cid.Cid) ([]apitypes.Message, error)

	// ChainGetTipSetByHeight looks back for a tipset at the specified epoch.
	// If there are no blocks at the specified epoch, a tipset at an earlier epoch
	// will be returned.
	ChainGetTipSetByHeight(context.Context, abi.ChainEpoch, types.TipSetKey) (*types.TipSet, error)

	// ChainReadObj reads ipld nodes referenced by the specified CID from chain
	// blockstore and returns raw bytes.
	ChainReadObj(context.Context, cid.Cid) ([]byte, error)

	// ChainDeleteObj deletes node referenced by the given CID
	ChainDeleteObj(context.Context, cid.Cid) error

	// ChainHasObj checks if a given CID exists in the chain blockstore.
	ChainHasObj(context.Context, cid.Cid) (bool, error)

	// ChainStatObj returns statistics about the graph referenced by 'obj'.
	// If 'base' is also specified, then the returned stat will be a diff
	// between the two objects.
	ChainStatObj(ctx context.Context, obj cid.Cid, base cid.Cid) (ObjStat, error)

	// ChainSetHead forcefully sets current chain head. Use with caution.
	ChainSetHead(context.Context, types.TipSetKey) error

	// ChainGetGenesis returns the genesis tipset.
	ChainGetGenesis(context.Context) (*types.TipSet, error)

	// ChainTipSetWeight computes weight for the specified tipset.
	ChainTipSetWeight(context.Context, types.TipSetKey) (types.BigInt, error)
	ChainGetNode(ctx context.Context, p string) (*IpldObject, error)

	// ChainGetMessage reads a message referenced by the specified CID from the
	// chain blockstore.
	ChainGetMessage(context.Context, cid.Cid) (*types.Message, error)

	// ChainGetPath returns a set of revert/apply operations needed to get from
	// one tipset to another, for example:
	//```
	//        to
	//         ^
	// from   tAA
	//   ^     ^
	// tBA    tAB
	//  ^---*--^
	//      ^
	//     tRR
	//```
	// Would return `[revert(tBA), apply(tAB), apply(tAA)]`
	ChainGetPath(ctx context.Context, from types.TipSetKey, to types.TipSetKey) ([]*chain.HeadChange, error)

	// ChainExport returns a stream of bytes with CAR dump of chain data.
	// The exported chain data includes the header chain from the given tipset
	// back to genesis, the entire genesis state, and the most recent 'nroots'
	// state trees.
	// If oldmsgskip is set, messages from before the requested roots are also not included.
	ChainExport(ctx context.Context, nroots abi.ChainEpoch, oldmsgskip bool, tsk types.TipSetKey) (<-chan []byte, error)

	// MethodGroup: Beacon
	// The Beacon method group contains methods for interacting with the random beacon (DRAND)

	// BeaconGetEntry returns the beacon entry for the given filecoin epoch. If
	// the entry has not yet been produced, the call will block until the entry
	// becomes available
	BeaconGetEntry(ctx context.Context, epoch abi.ChainEpoch) (*types.BeaconEntry, error)

	// GasEstimateFeeCap estimates gas fee cap
	GasEstimateFeeCap(context.Context, *types.Message, int64, types.TipSetKey) (types.BigInt, error)

	// GasEstimateGasLimit estimates gas used by the message and returns it.
	// It fails if message fails to execute.
	GasEstimateGasLimit(context.Context, *types.Message, types.TipSetKey) (int64, error)

	// GasEstimateGasPremium estimates what gas price should be used for a
	// message to have high likelihood of inclusion in `nblocksincl` epochs.

	GasEstimateGasPremium(_ context.Context, nblocksincl uint64,
		sender address.Address, gaslimit int64, tsk types.TipSetKey) (types.BigInt, error)

	// GasEstimateMessageGas estimates gas values for unset message gas fields
	GasEstimateMessageGas(context.Context, *types.Message, *types.MessageSendSpec, types.TipSetKey) (*types.Message, error)

	// MethodGroup: Sync
	// The Sync method group contains methods for interacting with and
	// observing the venus sync service.

	// SyncState returns the current status of the venus sync system.
	SyncState(context.Context) (*apitypes.SyncState, error)

	// SyncSubmitBlock can be used to submit a newly created block to the.
	// network through this node
	SyncSubmitBlock(ctx context.Context, blk *types.BlockMsg) error

	// SyncIncomingBlocks returns a channel streaming incoming, potentially not
	// yet synced block headers.
	SyncIncomingBlocks(ctx context.Context) (<-chan *types.BlockHeader, error)

	// SyncCheckpoint marks a blocks as checkpointed, meaning that it won't ever fork away from it.
	SyncCheckpoint(ctx context.Context, tsk types.TipSetKey) error

	// SyncMarkBad marks a blocks as bad, meaning that it won't ever by synced.
	// Use with extreme caution.
	SyncMarkBad(ctx context.Context, bcid cid.Cid) error

	// SyncUnmarkBad unmarks a blocks as bad, making it possible to be validated and synced again.
	SyncUnmarkBad(ctx context.Context, bcid cid.Cid) error

	// SyncUnmarkAllBad purges bad block cache, making it possible to sync to chains previously marked as bad
	SyncUnmarkAllBad(ctx context.Context) error

	// SyncCheckBad checks if a block was marked as bad, and if it was, returns
	// the reason.
	SyncCheckBad(ctx context.Context, bcid cid.Cid) (string, error)

	// SyncValidateTipset indicates whether the provided tipset is valid or not
	SyncValidateTipset(ctx context.Context, tsk types.TipSetKey) (bool, error)

	// MethodGroup: Mpool
	// The Mpool methods are for interacting with the message pool. The message pool
	// manages all incoming and outgoing 'messages' going over the network.

	// MpoolPending returns pending mempool messages.
	MpoolPending(context.Context, types.TipSetKey) ([]*types.SignedMessage, error)

	// MpoolSelect returns a list of pending messages for inclusion in the next block
	MpoolSelect(context.Context, types.TipSetKey, float64) ([]*types.SignedMessage, error)

	// MpoolPush pushes a signed message to mempool.
	MpoolPush(context.Context, *types.SignedMessage) (cid.Cid, error)

	// MpoolPushUntrusted pushes a signed message to mempool from untrusted sources.
	MpoolPushUntrusted(context.Context, *types.SignedMessage) (cid.Cid, error)

	// MpoolPushMessage atomically assigns a nonce, signs, and pushes a message
	// to mempool.
	// maxFee is only used when GasFeeCap/GasPremium fields aren't specified
	//
	// When maxFee is set to 0, MpoolPushMessage will guess appropriate fee
	// based on current chain conditions
	MpoolPushMessage(ctx context.Context, msg *types.Message, spec *types.MessageSendSpec) (*types.SignedMessage, error)

	// MpoolBatchPush batch pushes a signed message to mempool.
	MpoolBatchPush(context.Context, []*types.SignedMessage) ([]cid.Cid, error)

	// MpoolBatchPushUntrusted batch pushes a signed message to mempool from untrusted sources.
	MpoolBatchPushUntrusted(context.Context, []*types.SignedMessage) ([]cid.Cid, error)

	// type batch pushes a unsigned message to mempool.
	MpoolBatchPushMessage(context.Context, []*types.Message, *types.MessageSendSpec) ([]*types.SignedMessage, error)

	// MpoolGetNonce gets next nonce for the specified sender.
	// Note that this method may not be atomic. Use MpoolPushMessage instead.
	MpoolGetNonce(context.Context, address.Address) (uint64, error)
	MpoolSub(context.Context) (<-chan messagepool.MpoolUpdate, error)

	// MpoolClear clears pending messages from the mpool
	MpoolClear(context.Context, bool) error

	// MpoolGetConfig returns (a copy of) the current mpool config
	MpoolGetConfig(context.Context) (*messagepool.MpoolConfig, error)
	// MpoolSetConfig sets the mpool config to (a copy of) the supplied config
	MpoolSetConfig(context.Context, *messagepool.MpoolConfig) error

	// MethodGroup: Sealer

	MinerGetBaseInfo(context.Context, address.Address, abi.ChainEpoch, types.TipSetKey) (*apitypes.MiningBaseInfo, error)
	MinerCreateBlock(context.Context, *apitypes.BlockTemplate) (*types.BlockMsg, error)

	// // UX ?

	// MethodGroup: Wallet

	// WalletNew creates a new address in the wallet with the given sigType.
	// Available key types: bls, secp256k1, secp256k1-ledger
	// Support for numerical types: 1 - secp256k1, 2 - BLS is deprecated
	WalletNew(context.Context, types.KeyType) (address.Address, error)
	// WalletHas indicates whether the given address is in the wallet.
	WalletHas(context.Context, address.Address) (bool, error)
	// WalletList lists all the addresses in the wallet.
	WalletList(context.Context) ([]address.Address, error)
	// WalletBalance returns the balance of the given address at the current head of the chain.
	WalletBalance(context.Context, address.Address) (types.BigInt, error)
	// WalletSign signs the given bytes using the given address.
	WalletSign(context.Context, address.Address, []byte) (*crypto.Signature, error)
	// WalletSignMessage signs the given message using the given address.
	WalletSignMessage(context.Context, address.Address, *types.Message) (*types.SignedMessage, error)
	// WalletVerify takes an address, a signature, and some bytes, and indicates whether the signature is valid.
	// The address does not have to be in the wallet.
	WalletVerify(context.Context, address.Address, []byte, *crypto.Signature) (bool, error)
	// WalletDefaultAddress returns the address marked as default in the wallet.
	WalletDefaultAddress(context.Context) (address.Address, error)
	// WalletSetDefault marks the given address as as the default one.
	WalletSetDefault(context.Context, address.Address) error
	// WalletDelete deletes an address from the wallet.
	WalletDelete(context.Context, address.Address) error
	// WalletValidateAddress validates whether a given string can be decoded as a well-formed address
	WalletValidateAddress(context.Context, string) (address.Address, error)

	// Other
	// MethodGroup: State
	// The State methods are used to query, inspect, and interact with chain state.
	// Most methods take a TipSetKey as a parameter. The state looked up is the state at that tipset.
	// A nil TipSetKey can be provided as a param, this will cause the heaviest tipset in the chain to be used.

	// StateCall runs the given message and returns its result without any persisted changes.
	StateCall(context.Context, *types.Message, types.TipSetKey) (*apitypes.InvocResult, error)
	// StateReplay replays a given message, assuming it was included in a block in the specified tipset.
	// If no tipset key is provided, the appropriate tipset is looked up.
	StateReplay(context.Context, types.TipSetKey, cid.Cid) (*apitypes.InvocResult, error)
	// StateGetActor returns the indicated actor's nonce and balance.
	StateGetActor(ctx context.Context, actor address.Address, tsk types.TipSetKey) (*types.Actor, error)
	// StateReadState returns the indicated actor's state.
	StateReadState(ctx context.Context, actor address.Address, tsk types.TipSetKey) (*ActorState, error)
	// StateListMessages looks back and returns all messages with a matching to or from address, stopping at the given height.
	StateListMessages(ctx context.Context, match *MessageMatch, tsk types.TipSetKey, toht abi.ChainEpoch) ([]cid.Cid, error)
	// StateDecodeParams attempts to decode the provided params, based on the recipient actor address and method number.
	StateDecodeParams(ctx context.Context, toAddr address.Address, method abi.MethodNum, params []byte, tsk types.TipSetKey) (interface{}, error)

	// StateNetworkName returns the name of the network the node is synced to
	StateNetworkName(context.Context) (types2.NetworkName, error)
	// StateMinerSectors returns info about the given miner's sectors. If the filter bitfield is nil, all sectors are included.
	StateMinerSectors(context.Context, address.Address, *bitfield.BitField, types.TipSetKey) ([]*miner.SectorOnChainInfo, error)
	// StateMinerActiveSectors returns info about sectors that a given miner is actively proving.
	StateMinerActiveSectors(context.Context, address.Address, types.TipSetKey) ([]*miner.SectorOnChainInfo, error)
	// StateMinerProvingDeadline calculates the deadline at some epoch for a proving period
	// and returns the deadline-related calculations.
	StateMinerProvingDeadline(context.Context, address.Address, types.TipSetKey) (*dline.Info, error)
	// StateMinerPower returns the power of the indicated miner
	StateMinerPower(context.Context, address.Address, types.TipSetKey) (*apitypes.MinerPower, error)
	// StateMinerInfo returns info about the indicated miner
	StateMinerInfo(context.Context, address.Address, types.TipSetKey) (miner.MinerInfo, error)
	// StateMinerDeadlines returns all the proving deadlines for the given miner
	StateMinerDeadlines(context.Context, address.Address, types.TipSetKey) ([]apitypes.Deadline, error)
	// StateMinerPartitions returns all partitions in the specified deadline
	StateMinerPartitions(ctx context.Context, m address.Address, dlIdx uint64, tsk types.TipSetKey) ([]apitypes.Partition, error)
	// StateMinerFaults returns a bitfield indicating the faulty sectors of the given miner
	StateMinerFaults(context.Context, address.Address, types.TipSetKey) (bitfield.BitField, error)
	// StateAllMinerFaults returns all non-expired Faults that occur within lookback epochs of the given tipset
	StateAllMinerFaults(ctx context.Context, lookback abi.ChainEpoch, ts types.TipSetKey) ([]*Fault, error)
	// StateMinerRecoveries returns a bitfield indicating the recovering sectors of the given miner
	StateMinerRecoveries(context.Context, address.Address, types.TipSetKey) (bitfield.BitField, error)
	// StateMinerInitialPledgeCollateral returns the precommit deposit for the specified miner's sector
	StateMinerPreCommitDepositForPower(context.Context, address.Address, miner.SectorPreCommitInfo, types.TipSetKey) (types.BigInt, error)
	// StateMinerInitialPledgeCollateral returns the initial pledge collateral for the specified miner's sector
	StateMinerInitialPledgeCollateral(context.Context, address.Address, miner.SectorPreCommitInfo, types.TipSetKey) (types.BigInt, error)
	// StateMinerAvailableBalance returns the portion of a miner's balance that can be withdrawn or spent
	StateMinerAvailableBalance(context.Context, address.Address, types.TipSetKey) (types.BigInt, error)
	// StateMinerSectorAllocated checks if a sector is allocated
	StateMinerSectorAllocated(context.Context, address.Address, abi.SectorNumber, types.TipSetKey) (bool, error)
	// StateSectorPreCommitInfo returns the PreCommit info for the specified miner's sector
	StateSectorPreCommitInfo(context.Context, address.Address, abi.SectorNumber, types.TipSetKey) (miner.SectorPreCommitOnChainInfo, error)
	// StateSectorGetInfo returns the on-chain info for the specified miner's sector. Returns null in case the sector info isn't found
	// NOTE: returned info.Expiration may not be accurate in some cases, use StateSectorExpiration to get accurate
	// expiration epoch
	StateSectorGetInfo(context.Context, address.Address, abi.SectorNumber, types.TipSetKey) (*miner.SectorOnChainInfo, error)
	// StateSectorExpiration returns epoch at which given sector will expire
	StateSectorExpiration(context.Context, address.Address, abi.SectorNumber, types.TipSetKey) (*miner.SectorExpiration, error)
	// StateSectorPartition finds deadline/partition with the specified sector
	StateSectorPartition(ctx context.Context, maddr address.Address, sectorNumber abi.SectorNumber, tok types.TipSetKey) (*miner.SectorLocation, error)
	// StateSearchMsg searches for a message in the chain, and returns its receipt and the tipset where it was executed
	StateSearchMsg(ctx context.Context, from types.TipSetKey, msg cid.Cid, limit abi.ChainEpoch, allowReplaced bool) (*apitypes.MsgLookup, error)
	// StateSearchMsgLimited looks back up to limit epochs in the chain for a message, and returns its receipt and the tipset where it was executed
	StateSearchMsgLimited(ctx context.Context, msg cid.Cid, limit abi.ChainEpoch) (*chain.MsgLookup, error)
	// StateWaitMsg looks back in the chain for a message. If not found, it blocks until the
	// message arrives on chain, and gets to the indicated confidence depth.
	StateWaitMsg(ctx context.Context, cid cid.Cid, confidence uint64, limit abi.ChainEpoch, allowReplaced bool) (*apitypes.MsgLookup, error)
	// StateWaitMsgLimited looks back up to limit epochs in the chain for a message.
	// If not found, it blocks until the message arrives on chain, and gets to the
	// indicated confidence depth.
	StateWaitMsgLimited(ctx context.Context, cid cid.Cid, confidence uint64, limit abi.ChainEpoch) (*chain.MsgLookup, error)
	// StateListMiners returns the addresses of every miner that has claimed power in the Power Actor
	StateListMiners(context.Context, types.TipSetKey) ([]address.Address, error)
	// StateListActors returns the addresses of every actor in the state
	StateListActors(context.Context, types.TipSetKey) ([]address.Address, error)
	// StateMarketBalance looks up the Escrow and Locked balances of the given address in the Storage Market
	StateMarketBalance(context.Context, address.Address, types.TipSetKey) (apitypes.MarketBalance, error)
	// StateMarketParticipants returns the Escrow and Locked balances of every participant in the Storage Market
	StateMarketParticipants(context.Context, types.TipSetKey) (map[string]apitypes.MarketBalance, error)
	// StateMarketDeals returns information about every deal in the Storage Market
	StateMarketDeals(context.Context, types.TipSetKey) (map[string]apitypes.MarketDeal, error)
	// StateMarketStorageDeal returns information about the indicated deal
	StateMarketStorageDeal(context.Context, abi.DealID, types.TipSetKey) (*apitypes.MarketDeal, error)
	// StateLookupID retrieves the ID address of the given address
	StateLookupID(context.Context, address.Address, types.TipSetKey) (address.Address, error)
	// StateAccountKey returns the public key address of the given ID address
	StateAccountKey(context.Context, address.Address, types.TipSetKey) (address.Address, error)
	// StateChangedActors returns all the actors whose states change between the two given state CIDs
	// TODO: Should this take tipset keys instead?
	StateChangedActors(context.Context, cid.Cid, cid.Cid) (map[string]types.Actor, error)
	// StateGetReceipt returns the message receipt for the given message
	StateGetReceipt(context.Context, cid.Cid, types.TipSetKey) (*types.MessageReceipt, error)
	// StateMinerSectorCount returns the number of sectors in a miner's sector set and proving set
	StateMinerSectorCount(context.Context, address.Address, types.TipSetKey) (apitypes.MinerSectors, error)
	// StateCompute is a flexible command that applies the given messages on the given tipset.
	// The messages are run as though the VM were at the provided height.
	StateCompute(context.Context, abi.ChainEpoch, []*types.Message, types.TipSetKey) (*apitypes.ComputeStateOutput, error)
	// StateVerifierStatus returns the data cap for the given address.
	// Returns nil if there is no entry in the data cap table for the
	// address.
	StateVerifierStatus(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*abi.StoragePower, error)
	// StateVerifiedClientStatus returns the data cap for the given address.
	// Returns nil if there is no entry in the data cap table for the
	// address.
	StateVerifiedClientStatus(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*abi.StoragePower, error)
	// StateVerifiedClientStatus returns the address of the Verified Registry's root key
	StateVerifiedRegistryRootKey(ctx context.Context, tsk types.TipSetKey) (address.Address, error)
	// StateDealProviderCollateralBounds returns the min and max collateral a storage provider
	// can issue. It takes the deal size and verified status as parameters.
	StateDealProviderCollateralBounds(context.Context, abi.PaddedPieceSize, bool, types.TipSetKey) (DealCollateralBounds, error)

	// StateCirculatingSupply returns the exact circulating supply of Filecoin at the given tipset.
	// This is not used anywhere in the protocol itself, and is only for external consumption.
	StateCirculatingSupply(context.Context, types.TipSetKey) (abi.TokenAmount, error)
	// StateVMCirculatingSupplyInternal returns an approximation of the circulating supply of Filecoin at the given tipset.
	// This is the value reported by the runtime interface to actors code.
	StateVMCirculatingSupplyInternal(context.Context, types.TipSetKey) (chain.CirculatingSupply, error)
	// StateNetworkVersion returns the network version at the given tipset
	StateNetworkVersion(context.Context, types.TipSetKey) (stnetwork.Version, error)

	// MethodGroup: Msig
	// The Msig methods are used to interact with multisig wallets on the
	// filecoin network

	// MsigGetAvailableBalance returns the portion of a multisig's balance that can be withdrawn or spent
	MsigGetAvailableBalance(context.Context, address.Address, types.TipSetKey) (types.BigInt, error)
	// MsigGetVestingSchedule returns the vesting details of a given multisig.
	MsigGetVestingSchedule(context.Context, address.Address, types.TipSetKey) (MsigVesting, error)
	// MsigGetVested returns the amount of FIL that vested in a multisig in a certain period.
	// It takes the following params: <multisig address>, <start epoch>, <end epoch>
	MsigGetVested(context.Context, address.Address, types.TipSetKey, types.TipSetKey) (types.BigInt, error)
	// MsigCreate creates a multisig wallet
	// It takes the following params: <required number of senders>, <approving addresses>, <unlock duration>
	//<initial balance>, <sender address of the create msg>, <gas price>
	MsigCreate(context.Context, uint64, []address.Address, abi.ChainEpoch, types.BigInt, address.Address, types.BigInt) (cid.Cid, error)
	// MsigPropose proposes a multisig message
	// It takes the following params: <multisig address>, <recipient address>, <value to transfer>,
	// <sender address of the propose msg>, <method to call in the proposed message>, <params to include in the proposed message>
	MsigPropose(context.Context, address.Address, address.Address, types.BigInt, address.Address, uint64, []byte) (cid.Cid, error)

	// MsigApprove approves a previously-proposed multisig message by transaction ID
	// It takes the following params: <multisig address>, <proposed transaction ID> <signer address>
	MsigApprove(context.Context, address.Address, uint64, address.Address) (cid.Cid, error)

	// MsigApproveTxnHash approves a previously-proposed multisig message, specified
	// using both transaction ID and a hash of the parameters used in the
	// proposal. This method of approval can be used to ensure you only approve
	// exactly the transaction you think you are.
	// It takes the following params: <multisig address>, <proposed message ID>, <proposer address>, <recipient address>, <value to transfer>,
	// <sender address of the approve msg>, <method to call in the proposed message>, <params to include in the proposed message>
	MsigApproveTxnHash(context.Context, address.Address, uint64, address.Address, address.Address, types.BigInt, address.Address, uint64, []byte) (cid.Cid, error)

	// MsigCancel cancels a previously-proposed multisig message
	// It takes the following params: <multisig address>, <proposed transaction ID>, <recipient address>, <value to transfer>,
	// <sender address of the cancel msg>, <method to call in the proposed message>, <params to include in the proposed message>
	MsigCancel(context.Context, address.Address, uint64, address.Address, types.BigInt, address.Address, uint64, []byte) (cid.Cid, error)
	// MsigAddPropose proposes adding a signer in the multisig
	// It takes the following params: <multisig address>, <sender address of the propose msg>,
	// <new signer>, <whether the number of required signers should be increased>
	MsigAddPropose(context.Context, address.Address, address.Address, address.Address, bool) (cid.Cid, error)
	// MsigAddApprove approves a previously proposed AddSigner message
	// It takes the following params: <multisig address>, <sender address of the approve msg>, <proposed message ID>,
	// <proposer address>, <new signer>, <whether the number of required signers should be increased>
	MsigAddApprove(context.Context, address.Address, address.Address, uint64, address.Address, address.Address, bool) (cid.Cid, error)
	// MsigAddCancel cancels a previously proposed AddSigner message
	// It takes the following params: <multisig address>, <sender address of the cancel msg>, <proposed message ID>,
	// <new signer>, <whether the number of required signers should be increased>
	MsigAddCancel(context.Context, address.Address, address.Address, uint64, address.Address, bool) (cid.Cid, error)
	// MsigSwapPropose proposes swapping 2 signers in the multisig
	// It takes the following params: <multisig address>, <sender address of the propose msg>,
	// <old signer>, <new signer>
	MsigSwapPropose(context.Context, address.Address, address.Address, address.Address, address.Address) (cid.Cid, error)
	// MsigSwapApprove approves a previously proposed SwapSigner
	// It takes the following params: <multisig address>, <sender address of the approve msg>, <proposed message ID>,
	// <proposer address>, <old signer>, <new signer>
	MsigSwapApprove(context.Context, address.Address, address.Address, uint64, address.Address, address.Address, address.Address) (cid.Cid, error)
	// MsigSwapCancel cancels a previously proposed SwapSigner message
	// It takes the following params: <multisig address>, <sender address of the cancel msg>, <proposed message ID>,
	// <old signer>, <new signer>
	MsigSwapCancel(context.Context, address.Address, address.Address, uint64, address.Address, address.Address) (cid.Cid, error)

	// MsigRemoveSigner proposes the removal of a signer from the multisig.
	// It accepts the multisig to make the change on, the proposer address to
	// send the message from, the address to be removed, and a boolean
	// indicating whether or not the signing threshold should be lowered by one
	// along with the address removal.
	MsigRemoveSigner(ctx context.Context, msig address.Address, proposer address.Address, toRemove address.Address, decrease bool) (cid.Cid, error)

	// MarketAddBalance adds funds to the market actor
	MarketAddBalance(ctx context.Context, wallet, addr address.Address, amt types.BigInt) (cid.Cid, error)
	// MarketGetReserved gets the amount of funds that are currently reserved for the address
	MarketGetReserved(ctx context.Context, addr address.Address) (types.BigInt, error)
	// MarketReserveFunds reserves funds for a deal
	MarketReserveFunds(ctx context.Context, wallet address.Address, addr address.Address, amt types.BigInt) (cid.Cid, error)
	// MarketReleaseFunds releases funds reserved by MarketReserveFunds
	MarketReleaseFunds(ctx context.Context, addr address.Address, amt types.BigInt) error
	// MarketWithdraw withdraws unlocked funds from the market actor
	MarketWithdraw(ctx context.Context, wallet, addr address.Address, amt types.BigInt) (cid.Cid, error)

	// MethodGroup: Paych
	// The Paych methods are for interacting with and managing payment channels

	PaychGet(ctx context.Context, from, to address.Address, amt types.BigInt) (*apitypes.ChannelInfo, error)
	PaychGetWaitReady(context.Context, cid.Cid) (address.Address, error)
	PaychAvailableFunds(ctx context.Context, ch address.Address) (*apitypes.ChannelAvailableFunds, error)
	PaychAvailableFundsByFromTo(ctx context.Context, from, to address.Address) (*apitypes.ChannelAvailableFunds, error)
	PaychList(context.Context) ([]address.Address, error)
	PaychStatus(context.Context, address.Address) (*types.PaychStatus, error)
	PaychSettle(context.Context, address.Address) (cid.Cid, error)
	PaychCollect(context.Context, address.Address) (cid.Cid, error)
	PaychAllocateLane(ctx context.Context, ch address.Address) (uint64, error)
	PaychNewPayment(ctx context.Context, from, to address.Address, vouchers []apitypes.VoucherSpec) (*apitypes.PaymentInfo, error)
	PaychVoucherCheckValid(context.Context, address.Address, *paych.SignedVoucher) error
	PaychVoucherCheckSpendable(context.Context, address.Address, *paych.SignedVoucher, []byte, []byte) (bool, error)
	PaychVoucherCreate(context.Context, address.Address, types.BigInt, uint64) (*apitypes.VoucherCreateResult, error)
	PaychVoucherAdd(context.Context, address.Address, *paych.SignedVoucher, []byte, types.BigInt) (types.BigInt, error)
	PaychVoucherList(context.Context, address.Address) ([]*paych.SignedVoucher, error)
	PaychVoucherSubmit(context.Context, address.Address, *paych.SignedVoucher, []byte, []byte) (cid.Cid, error)

	// CreateBackup creates node backup onder the specified file name. The
	// method requires that the venus daemon is running with the
	// LOTUS_BACKUP_BASE_PATH environment variable set to some path, and that
	// the path specified when calling CreateBackup is within the base path
	CreateBackup(ctx context.Context, fpath string) error
}

type CommonStruct struct {
	Internal struct {
		AuthVerify func(ctx context.Context, token string) ([]auth.Permission, error) `perm:"read"`
		AuthNew    func(ctx context.Context, perms []auth.Permission) ([]byte, error) `perm:"admin"`
		Token      func(ctx context.Context) ([]byte, error)                          `perm:"admin"`

		NetConnectedness            func(context.Context, peer.ID) (network.Connectedness, error)    `perm:"read"`
		NetPeers                    func(context.Context) ([]peer.AddrInfo, error)                   `perm:"read"`
		NetConnect                  func(context.Context, peer.AddrInfo) error                       `perm:"write"`
		NetAddrsListen              func(context.Context) (peer.AddrInfo, error)                     `perm:"read"`
		NetDisconnect               func(context.Context, peer.ID) error                             `perm:"write"`
		NetFindPeer                 func(context.Context, peer.ID) (peer.AddrInfo, error)            `perm:"read"`
		NetPubsubScores             func(context.Context) ([]PubsubScore, error)                     `perm:"read"`
		NetAutoNatStatus            func(context.Context) (NatInfo, error)                           `perm:"read"`
		NetBandwidthStats           func(ctx context.Context) (metrics.Stats, error)                 `perm:"read"`
		NetBandwidthStatsByPeer     func(ctx context.Context) (map[string]metrics.Stats, error)      `perm:"read"`
		NetBandwidthStatsByProtocol func(ctx context.Context) (map[protocol.ID]metrics.Stats, error) `perm:"read"`
		NetAgentVersion             func(ctx context.Context, p peer.ID) (string, error)             `perm:"read"`
		NetBlockAdd                 func(ctx context.Context, acl NetBlockList) error                `perm:"admin"`
		NetBlockRemove              func(ctx context.Context, acl NetBlockList) error                `perm:"admin"`
		NetBlockList                func(ctx context.Context) (NetBlockList, error)                  `perm:"read"`

		ID      func(context.Context) (peer.ID, error) `perm:"read"`
		Version func(context.Context) (Version, error) `perm:"read"`

		LogList     func(context.Context) ([]string, error)     `perm:"write"`
		LogSetLevel func(context.Context, string, string) error `perm:"write"`

		Shutdown func(context.Context) error                    `perm:"admin"`
		Session  func(context.Context) (uuid.UUID, error)       `perm:"read"`
		Closing  func(context.Context) (<-chan struct{}, error) `perm:"read"`
	}
}

// FullNodeStruct implements API passing calls to user-provided function values.
type FullNodeStruct struct {
	CommonStruct

	Internal struct {
		ChainNotify                   func(context.Context) (<-chan []*chain.HeadChange, error)                                                          `perm:"read"`
		ChainHead                     func(context.Context) (*types.TipSet, error)                                                                       `perm:"read"`
		ChainGetRandomnessFromTickets func(context.Context, types.TipSetKey, crypto.DomainSeparationTag, abi.ChainEpoch, []byte) (abi.Randomness, error) `perm:"read"`
		ChainGetRandomnessFromBeacon  func(context.Context, types.TipSetKey, crypto.DomainSeparationTag, abi.ChainEpoch, []byte) (abi.Randomness, error) `perm:"read"`
		ChainGetBlock                 func(context.Context, cid.Cid) (*types.BlockHeader, error)                                                         `perm:"read"`
		ChainGetTipSet                func(context.Context, types.TipSetKey) (*types.TipSet, error)                                                      `perm:"read"`
		ChainGetBlockMessages         func(context.Context, cid.Cid) (*apitypes.BlockMessages, error)                                                    `perm:"read"`
		ChainGetParentReceipts        func(context.Context, cid.Cid) ([]*types.MessageReceipt, error)                                                    `perm:"read"`
		ChainGetParentMessages        func(context.Context, cid.Cid) ([]apitypes.Message, error)                                                         `perm:"read"`
		ChainGetTipSetByHeight        func(context.Context, abi.ChainEpoch, types.TipSetKey) (*types.TipSet, error)                                      `perm:"read"`
		ChainReadObj                  func(context.Context, cid.Cid) ([]byte, error)                                                                     `perm:"read"`
		ChainDeleteObj                func(context.Context, cid.Cid) error                                                                               `perm:"admin"`
		ChainHasObj                   func(context.Context, cid.Cid) (bool, error)                                                                       `perm:"read"`
		ChainStatObj                  func(context.Context, cid.Cid, cid.Cid) (ObjStat, error)                                                           `perm:"read"`
		ChainSetHead                  func(context.Context, types.TipSetKey) error                                                                       `perm:"admin"`
		ChainGetGenesis               func(context.Context) (*types.TipSet, error)                                                                       `perm:"read"`
		ChainTipSetWeight             func(context.Context, types.TipSetKey) (types.BigInt, error)                                                       `perm:"read"`
		ChainGetNode                  func(ctx context.Context, p string) (*IpldObject, error)                                                           `perm:"read"`
		ChainGetMessage               func(context.Context, cid.Cid) (*types.Message, error)                                                             `perm:"read"`
		ChainGetPath                  func(context.Context, types.TipSetKey, types.TipSetKey) ([]*chain.HeadChange, error)                               `perm:"read"`
		ChainExport                   func(context.Context, abi.ChainEpoch, bool, types.TipSetKey) (<-chan []byte, error)                                `perm:"read"`

		BeaconGetEntry func(ctx context.Context, epoch abi.ChainEpoch) (*types.BeaconEntry, error) `perm:"read"`

		GasEstimateGasPremium func(context.Context, uint64, address.Address, int64, types.TipSetKey) (types.BigInt, error)           `perm:"read"`
		GasEstimateGasLimit   func(context.Context, *types.Message, types.TipSetKey) (int64, error)                                  `perm:"read"`
		GasEstimateFeeCap     func(context.Context, *types.Message, int64, types.TipSetKey) (types.BigInt, error)                    `perm:"read"`
		GasEstimateMessageGas func(context.Context, *types.Message, *types.MessageSendSpec, types.TipSetKey) (*types.Message, error) `perm:"read"`

		SyncState          func(context.Context) (*apitypes.SyncState, error)           `perm:"read"`
		SyncSubmitBlock    func(ctx context.Context, blk *types.BlockMsg) error         `perm:"write"`
		SyncIncomingBlocks func(ctx context.Context) (<-chan *types.BlockHeader, error) `perm:"read"`
		SyncCheckpoint     func(ctx context.Context, key types.TipSetKey) error         `perm:"admin"`
		SyncMarkBad        func(ctx context.Context, bcid cid.Cid) error                `perm:"admin"`
		SyncUnmarkBad      func(ctx context.Context, bcid cid.Cid) error                `perm:"admin"`
		SyncUnmarkAllBad   func(ctx context.Context) error                              `perm:"admin"`
		SyncCheckBad       func(ctx context.Context, bcid cid.Cid) (string, error)      `perm:"read"`
		SyncValidateTipset func(ctx context.Context, tsk types.TipSetKey) (bool, error) `perm:"read"`

		MpoolGetConfig func(context.Context) (*messagepool.MpoolConfig, error) `perm:"read"`
		MpoolSetConfig func(context.Context, *messagepool.MpoolConfig) error   `perm:"write"`

		MpoolSelect func(context.Context, types.TipSetKey, float64) ([]*types.SignedMessage, error) `perm:"read"`

		MpoolPending func(context.Context, types.TipSetKey) ([]*types.SignedMessage, error) `perm:"read"`
		MpoolClear   func(context.Context, bool) error                                      `perm:"write"`

		MpoolPush          func(context.Context, *types.SignedMessage) (cid.Cid, error) `perm:"write"`
		MpoolPushUntrusted func(context.Context, *types.SignedMessage) (cid.Cid, error) `perm:"write"`

		MpoolPushMessage func(context.Context, *types.Message, *types.MessageSendSpec) (*types.SignedMessage, error) `perm:"sign"`
		MpoolGetNonce    func(context.Context, address.Address) (uint64, error)                                      `perm:"read"`
		MpoolSub         func(context.Context) (<-chan messagepool.MpoolUpdate, error)                               `perm:"read"`

		MpoolBatchPush          func(ctx context.Context, smsgs []*types.SignedMessage) ([]cid.Cid, error)                                    `perm:"write"`
		MpoolBatchPushUntrusted func(ctx context.Context, smsgs []*types.SignedMessage) ([]cid.Cid, error)                                    `perm:"write"`
		MpoolBatchPushMessage   func(ctx context.Context, msgs []*types.Message, spec *types.MessageSendSpec) ([]*types.SignedMessage, error) `perm:"sign"`

		MinerGetBaseInfo func(context.Context, address.Address, abi.ChainEpoch, types.TipSetKey) (*apitypes.MiningBaseInfo, error) `perm:"read"`
		MinerCreateBlock func(context.Context, *apitypes.BlockTemplate) (*types.BlockMsg, error)                                   `perm:"write"`

		WalletNew             func(context.Context, types.KeyType) (address.Address, error)                        `perm:"write"`
		WalletHas             func(context.Context, address.Address) (bool, error)                                 `perm:"write"`
		WalletList            func(context.Context) ([]address.Address, error)                                     `perm:"write"`
		WalletBalance         func(context.Context, address.Address) (types.BigInt, error)                         `perm:"read"`
		WalletSign            func(context.Context, address.Address, []byte) (*crypto.Signature, error)            `perm:"sign"`
		WalletSignMessage     func(context.Context, address.Address, *types.Message) (*types.SignedMessage, error) `perm:"sign"`
		WalletVerify          func(context.Context, address.Address, []byte, *crypto.Signature) (bool, error)      `perm:"read"`
		WalletDefaultAddress  func(context.Context) (address.Address, error)                                       `perm:"write"`
		WalletSetDefault      func(context.Context, address.Address) error                                         `perm:"admin"`
		WalletDelete          func(context.Context, address.Address) error                                         `perm:"write"`
		WalletValidateAddress func(context.Context, string) (address.Address, error)                               `perm:"read"`

		StateNetworkName                   func(context.Context) (types2.NetworkName, error)                                                                   `perm:"read"`
		StateMinerSectors                  func(context.Context, address.Address, *bitfield.BitField, types.TipSetKey) ([]*miner.SectorOnChainInfo, error)     `perm:"read"`
		StateMinerActiveSectors            func(context.Context, address.Address, types.TipSetKey) ([]*miner.SectorOnChainInfo, error)                         `perm:"read"`
		StateMinerProvingDeadline          func(context.Context, address.Address, types.TipSetKey) (*dline.Info, error)                                        `perm:"read"`
		StateMinerPower                    func(context.Context, address.Address, types.TipSetKey) (*apitypes.MinerPower, error)                               `perm:"read"`
		StateMinerInfo                     func(context.Context, address.Address, types.TipSetKey) (miner.MinerInfo, error)                                    `perm:"read"`
		StateMinerDeadlines                func(context.Context, address.Address, types.TipSetKey) ([]apitypes.Deadline, error)                                `perm:"read"`
		StateMinerPartitions               func(ctx context.Context, m address.Address, dlIdx uint64, tsk types.TipSetKey) ([]apitypes.Partition, error)       `perm:"read"`
		StateMinerFaults                   func(context.Context, address.Address, types.TipSetKey) (bitfield.BitField, error)                                  `perm:"read"`
		StateAllMinerFaults                func(context.Context, abi.ChainEpoch, types.TipSetKey) ([]*Fault, error)                                            `perm:"read"`
		StateMinerRecoveries               func(context.Context, address.Address, types.TipSetKey) (bitfield.BitField, error)                                  `perm:"read"`
		StateMinerPreCommitDepositForPower func(context.Context, address.Address, miner.SectorPreCommitInfo, types.TipSetKey) (types.BigInt, error)            `perm:"read"`
		StateMinerInitialPledgeCollateral  func(context.Context, address.Address, miner.SectorPreCommitInfo, types.TipSetKey) (types.BigInt, error)            `perm:"read"`
		StateMinerAvailableBalance         func(context.Context, address.Address, types.TipSetKey) (types.BigInt, error)                                       `perm:"read"`
		StateMinerSectorAllocated          func(context.Context, address.Address, abi.SectorNumber, types.TipSetKey) (bool, error)                             `perm:"read"`
		StateSectorPreCommitInfo           func(context.Context, address.Address, abi.SectorNumber, types.TipSetKey) (miner.SectorPreCommitOnChainInfo, error) `perm:"read"`
		StateSectorGetInfo                 func(context.Context, address.Address, abi.SectorNumber, types.TipSetKey) (*miner.SectorOnChainInfo, error)         `perm:"read"`
		StateSectorExpiration              func(context.Context, address.Address, abi.SectorNumber, types.TipSetKey) (*miner.SectorExpiration, error)          `perm:"read"`
		StateSectorPartition               func(context.Context, address.Address, abi.SectorNumber, types.TipSetKey) (*miner.SectorLocation, error)            `perm:"read"`
		StateCall                          func(context.Context, *types.Message, types.TipSetKey) (*apitypes.InvocResult, error)                               `perm:"read"`
		StateReplay                        func(context.Context, types.TipSetKey, cid.Cid) (*apitypes.InvocResult, error)                                      `perm:"read"`
		StateGetActor                      func(context.Context, address.Address, types.TipSetKey) (*types.Actor, error)                                       `perm:"read"`
		StateReadState                     func(context.Context, address.Address, types.TipSetKey) (*ActorState, error)                                        `perm:"read"`
		StateWaitMsg                       func(context.Context, cid.Cid, uint64, abi.ChainEpoch, bool) (*apitypes.MsgLookup, error)                           `perm:"read"`
		StateWaitMsgLimited                func(context.Context, cid.Cid, uint64, abi.ChainEpoch) (*chain.MsgLookup, error)                                    `perm:"read"`
		StateSearchMsg                     func(context.Context, types.TipSetKey, cid.Cid, abi.ChainEpoch, bool) (*apitypes.MsgLookup, error)                  `perm:"read"`
		StateSearchMsgLimited              func(context.Context, cid.Cid, abi.ChainEpoch) (*chain.MsgLookup, error)                                            `perm:"read"`
		StateListMiners                    func(context.Context, types.TipSetKey) ([]address.Address, error)                                                   `perm:"read"`
		StateListActors                    func(context.Context, types.TipSetKey) ([]address.Address, error)                                                   `perm:"read"`
		StateMarketBalance                 func(context.Context, address.Address, types.TipSetKey) (apitypes.MarketBalance, error)                             `perm:"read"`
		StateMarketParticipants            func(context.Context, types.TipSetKey) (map[string]apitypes.MarketBalance, error)                                   `perm:"read"`
		StateMarketDeals                   func(context.Context, types.TipSetKey) (map[string]apitypes.MarketDeal, error)                                      `perm:"read"`
		StateMarketStorageDeal             func(context.Context, abi.DealID, types.TipSetKey) (*apitypes.MarketDeal, error)                                    `perm:"read"`
		StateLookupID                      func(ctx context.Context, addr address.Address, tsk types.TipSetKey) (address.Address, error)                       `perm:"read"`
		StateAccountKey                    func(context.Context, address.Address, types.TipSetKey) (address.Address, error)                                    `perm:"read"`
		StateChangedActors                 func(context.Context, cid.Cid, cid.Cid) (map[string]types.Actor, error)                                             `perm:"read"`
		StateGetReceipt                    func(context.Context, cid.Cid, types.TipSetKey) (*types.MessageReceipt, error)                                      `perm:"read"`
		StateMinerSectorCount              func(context.Context, address.Address, types.TipSetKey) (apitypes.MinerSectors, error)                              `perm:"read"`
		StateListMessages                  func(ctx context.Context, match *MessageMatch, tsk types.TipSetKey, toht abi.ChainEpoch) ([]cid.Cid, error)         `perm:"read"`
		StateDecodeParams                  func(context.Context, address.Address, abi.MethodNum, []byte, types.TipSetKey) (interface{}, error)                 `perm:"read"`
		StateCompute                       func(context.Context, abi.ChainEpoch, []*types.Message, types.TipSetKey) (*apitypes.ComputeStateOutput, error)      `perm:"read"`
		StateVerifierStatus                func(context.Context, address.Address, types.TipSetKey) (*abi.StoragePower, error)                                  `perm:"read"`
		StateVerifiedClientStatus          func(context.Context, address.Address, types.TipSetKey) (*abi.StoragePower, error)                                  `perm:"read"`
		StateVerifiedRegistryRootKey       func(ctx context.Context, tsk types.TipSetKey) (address.Address, error)                                             `perm:"read"`
		StateDealProviderCollateralBounds  func(context.Context, abi.PaddedPieceSize, bool, types.TipSetKey) (DealCollateralBounds, error)                     `perm:"read"`
		StateCirculatingSupply             func(context.Context, types.TipSetKey) (abi.TokenAmount, error)                                                     `perm:"read"`
		StateVMCirculatingSupplyInternal   func(context.Context, types.TipSetKey) (chain.CirculatingSupply, error)                                             `perm:"read"`
		StateNetworkVersion                func(context.Context, types.TipSetKey) (stnetwork.Version, error)                                                   `perm:"read"`

		MsigGetAvailableBalance func(context.Context, address.Address, types.TipSetKey) (types.BigInt, error)                                                                    `perm:"read"`
		MsigGetVestingSchedule  func(context.Context, address.Address, types.TipSetKey) (MsigVesting, error)                                                                     `perm:"read"`
		MsigGetVested           func(context.Context, address.Address, types.TipSetKey, types.TipSetKey) (types.BigInt, error)                                                   `perm:"read"`
		MsigCreate              func(context.Context, uint64, []address.Address, abi.ChainEpoch, types.BigInt, address.Address, types.BigInt) (cid.Cid, error)                   `perm:"sign"`
		MsigPropose             func(context.Context, address.Address, address.Address, types.BigInt, address.Address, uint64, []byte) (cid.Cid, error)                          `perm:"sign"`
		MsigApprove             func(context.Context, address.Address, uint64, address.Address) (cid.Cid, error)                                                                 `perm:"sign"`
		MsigApproveTxnHash      func(context.Context, address.Address, uint64, address.Address, address.Address, types.BigInt, address.Address, uint64, []byte) (cid.Cid, error) `perm:"sign"`
		MsigCancel              func(context.Context, address.Address, uint64, address.Address, types.BigInt, address.Address, uint64, []byte) (cid.Cid, error)                  `perm:"sign"`
		MsigAddPropose          func(context.Context, address.Address, address.Address, address.Address, bool) (cid.Cid, error)                                                  `perm:"sign"`
		MsigAddApprove          func(context.Context, address.Address, address.Address, uint64, address.Address, address.Address, bool) (cid.Cid, error)                         `perm:"sign"`
		MsigAddCancel           func(context.Context, address.Address, address.Address, uint64, address.Address, bool) (cid.Cid, error)                                          `perm:"sign"`
		MsigSwapPropose         func(context.Context, address.Address, address.Address, address.Address, address.Address) (cid.Cid, error)                                       `perm:"sign"`
		MsigSwapApprove         func(context.Context, address.Address, address.Address, uint64, address.Address, address.Address, address.Address) (cid.Cid, error)              `perm:"sign"`
		MsigSwapCancel          func(context.Context, address.Address, address.Address, uint64, address.Address, address.Address) (cid.Cid, error)                               `perm:"sign"`
		MsigRemoveSigner        func(ctx context.Context, msig address.Address, proposer address.Address, toRemove address.Address, decrease bool) (cid.Cid, error)              `perm:"sign"`

		MarketAddBalance   func(ctx context.Context, wallet, addr address.Address, amt types.BigInt) (cid.Cid, error)                 `perm:"sign"`
		MarketGetReserved  func(ctx context.Context, addr address.Address) (types.BigInt, error)                                      `perm:"sign"`
		MarketReserveFunds func(ctx context.Context, wallet address.Address, addr address.Address, amt types.BigInt) (cid.Cid, error) `perm:"sign"`
		MarketReleaseFunds func(ctx context.Context, addr address.Address, amt types.BigInt) error                                    `perm:"sign"`
		MarketWithdraw     func(ctx context.Context, wallet, addr address.Address, amt types.BigInt) (cid.Cid, error)                 `perm:"sign"`

		PaychGet                    func(ctx context.Context, from, to address.Address, amt types.BigInt) (*apitypes.ChannelInfo, error)                `perm:"sign"`
		PaychGetWaitReady           func(context.Context, cid.Cid) (address.Address, error)                                                             `perm:"sign"`
		PaychAvailableFunds         func(context.Context, address.Address) (*paych2.ChannelAvailableFunds, error)                                       `perm:"sign"`
		PaychAvailableFundsByFromTo func(context.Context, address.Address, address.Address) (*paych2.ChannelAvailableFunds, error)                      `perm:"sign"`
		PaychList                   func(context.Context) ([]address.Address, error)                                                                    `perm:"read"`
		PaychStatus                 func(context.Context, address.Address) (*paych2.PaychStatus, error)                                                 `perm:"read"`
		PaychSettle                 func(context.Context, address.Address) (cid.Cid, error)                                                             `perm:"sign"`
		PaychCollect                func(context.Context, address.Address) (cid.Cid, error)                                                             `perm:"sign"`
		PaychAllocateLane           func(context.Context, address.Address) (uint64, error)                                                              `perm:"sign"`
		PaychNewPayment             func(ctx context.Context, from, to address.Address, vouchers []apitypes.VoucherSpec) (*apitypes.PaymentInfo, error) `perm:"sign"`
		PaychVoucherCheck           func(context.Context, *paych.SignedVoucher) error                                                                   `perm:"read"`
		PaychVoucherCheckValid      func(context.Context, address.Address, *paych.SignedVoucher) error                                                  `perm:"read"`
		PaychVoucherCheckSpendable  func(context.Context, address.Address, *paych.SignedVoucher, []byte, []byte) (bool, error)                          `perm:"read"`
		PaychVoucherAdd             func(context.Context, address.Address, *paych.SignedVoucher, []byte, types.BigInt) (types.BigInt, error)            `perm:"write"`
		PaychVoucherCreate          func(context.Context, address.Address, big.Int, uint64) (*apitypes.VoucherCreateResult, error)                      `perm:"sign"`
		PaychVoucherList            func(context.Context, address.Address) ([]*paych.SignedVoucher, error)                                              `perm:"write"`
		PaychVoucherSubmit          func(context.Context, address.Address, *paych.SignedVoucher, []byte, []byte) (cid.Cid, error)                       `perm:"sign"`

		CreateBackup func(ctx context.Context, fpath string) error `perm:"admin"`
	}
}

func (c *FullNodeStruct) StateMinerSectorCount(ctx context.Context, addr address.Address, tsk types.TipSetKey) (apitypes.MinerSectors, error) {
	return c.Internal.StateMinerSectorCount(ctx, addr, tsk)
}

// CommonStruct

func (c *CommonStruct) AuthVerify(ctx context.Context, token string) ([]auth.Permission, error) {
	return c.Internal.AuthVerify(ctx, token)
}

func (c *CommonStruct) AuthNew(ctx context.Context, perms []auth.Permission) ([]byte, error) {
	return c.Internal.AuthNew(ctx, perms)
}

func (c *CommonStruct) Token(ctx context.Context) ([]byte, error) {
	return c.Internal.Token(ctx)
}

func (c *CommonStruct) NetPubsubScores(ctx context.Context) ([]PubsubScore, error) {
	return c.Internal.NetPubsubScores(ctx)
}

func (c *CommonStruct) NetConnectedness(ctx context.Context, pid peer.ID) (network.Connectedness, error) {
	return c.Internal.NetConnectedness(ctx, pid)
}

func (c *CommonStruct) NetPeers(ctx context.Context) ([]peer.AddrInfo, error) {
	return c.Internal.NetPeers(ctx)
}

func (c *CommonStruct) NetConnect(ctx context.Context, p peer.AddrInfo) error {
	return c.Internal.NetConnect(ctx, p)
}

func (c *CommonStruct) NetAddrsListen(ctx context.Context) (peer.AddrInfo, error) {
	return c.Internal.NetAddrsListen(ctx)
}

func (c *CommonStruct) NetDisconnect(ctx context.Context, p peer.ID) error {
	return c.Internal.NetDisconnect(ctx, p)
}

func (c *CommonStruct) NetFindPeer(ctx context.Context, p peer.ID) (peer.AddrInfo, error) {
	return c.Internal.NetFindPeer(ctx, p)
}

func (c *CommonStruct) NetAutoNatStatus(ctx context.Context) (NatInfo, error) {
	return c.Internal.NetAutoNatStatus(ctx)
}

func (c *CommonStruct) NetBandwidthStats(ctx context.Context) (metrics.Stats, error) {
	return c.Internal.NetBandwidthStats(ctx)
}

func (c *CommonStruct) NetBandwidthStatsByPeer(ctx context.Context) (map[string]metrics.Stats, error) {
	return c.Internal.NetBandwidthStatsByPeer(ctx)
}

func (c *CommonStruct) NetBandwidthStatsByProtocol(ctx context.Context) (map[protocol.ID]metrics.Stats, error) {
	return c.Internal.NetBandwidthStatsByProtocol(ctx)
}

func (c *CommonStruct) NetBlockAdd(ctx context.Context, acl NetBlockList) error {
	return c.Internal.NetBlockAdd(ctx, acl)
}

func (c *CommonStruct) NetBlockRemove(ctx context.Context, acl NetBlockList) error {
	return c.Internal.NetBlockRemove(ctx, acl)
}

func (c *CommonStruct) NetBlockList(ctx context.Context) (NetBlockList, error) {
	return c.Internal.NetBlockList(ctx)
}

func (c *CommonStruct) NetAgentVersion(ctx context.Context, p peer.ID) (string, error) {
	return c.Internal.NetAgentVersion(ctx, p)
}

// ID implements ID
func (c *CommonStruct) ID(ctx context.Context) (peer.ID, error) {
	return c.Internal.ID(ctx)
}

// Version implements Version
func (c *CommonStruct) Version(ctx context.Context) (Version, error) {
	return c.Internal.Version(ctx)
}

func (c *CommonStruct) LogList(ctx context.Context) ([]string, error) {
	return c.Internal.LogList(ctx)
}

func (c *CommonStruct) LogSetLevel(ctx context.Context, group, level string) error {
	return c.Internal.LogSetLevel(ctx, group, level)
}

func (c *CommonStruct) Shutdown(ctx context.Context) error {
	return c.Internal.Shutdown(ctx)
}

func (c *CommonStruct) Session(ctx context.Context) (uuid.UUID, error) {
	return c.Internal.Session(ctx)
}

func (c *CommonStruct) Closing(ctx context.Context) (<-chan struct{}, error) {
	return c.Internal.Closing(ctx)
}

// FullNodeStruct
func (c *FullNodeStruct) GasEstimateGasPremium(ctx context.Context, nblocksincl uint64, sender address.Address, gaslimit int64, tsk types.TipSetKey) (types.BigInt, error) {
	return c.Internal.GasEstimateGasPremium(ctx, nblocksincl, sender, gaslimit, tsk)
}

func (c *FullNodeStruct) GasEstimateFeeCap(ctx context.Context, msg *types.Message, maxqueueblks int64, tsk types.TipSetKey) (types.BigInt, error) {
	return c.Internal.GasEstimateFeeCap(ctx, msg, maxqueueblks, tsk)
}

func (c *FullNodeStruct) GasEstimateMessageGas(ctx context.Context, msg *types.Message, spec *types.MessageSendSpec, tsk types.TipSetKey) (*types.Message, error) {
	return c.Internal.GasEstimateMessageGas(ctx, msg, spec, tsk)
}

func (c *FullNodeStruct) GasEstimateGasLimit(ctx context.Context, msg *types.Message, tsk types.TipSetKey) (int64, error) {
	return c.Internal.GasEstimateGasLimit(ctx, msg, tsk)
}

func (c *FullNodeStruct) MpoolGetConfig(ctx context.Context) (*messagepool.MpoolConfig, error) {
	return c.Internal.MpoolGetConfig(ctx)
}

func (c *FullNodeStruct) MpoolSetConfig(ctx context.Context, cfg *messagepool.MpoolConfig) error {
	return c.Internal.MpoolSetConfig(ctx, cfg)
}

func (c *FullNodeStruct) MpoolSelect(ctx context.Context, tsk types.TipSetKey, tq float64) ([]*types.SignedMessage, error) {
	return c.Internal.MpoolSelect(ctx, tsk, tq)
}

func (c *FullNodeStruct) MpoolPending(ctx context.Context, tsk types.TipSetKey) ([]*types.SignedMessage, error) {
	return c.Internal.MpoolPending(ctx, tsk)
}

func (c *FullNodeStruct) MpoolClear(ctx context.Context, local bool) error {
	return c.Internal.MpoolClear(ctx, local)
}

func (c *FullNodeStruct) MpoolPush(ctx context.Context, smsg *types.SignedMessage) (cid.Cid, error) {
	return c.Internal.MpoolPush(ctx, smsg)
}

func (c *FullNodeStruct) MpoolPushUntrusted(ctx context.Context, smsg *types.SignedMessage) (cid.Cid, error) {
	return c.Internal.MpoolPushUntrusted(ctx, smsg)
}

func (c *FullNodeStruct) MpoolPushMessage(ctx context.Context, msg *types.Message, spec *types.MessageSendSpec) (*types.SignedMessage, error) {
	return c.Internal.MpoolPushMessage(ctx, msg, spec)
}

func (c *FullNodeStruct) MpoolBatchPush(ctx context.Context, smsgs []*types.SignedMessage) ([]cid.Cid, error) {
	return c.Internal.MpoolBatchPush(ctx, smsgs)
}

func (c *FullNodeStruct) MpoolBatchPushUntrusted(ctx context.Context, smsgs []*types.SignedMessage) ([]cid.Cid, error) {
	return c.Internal.MpoolBatchPushUntrusted(ctx, smsgs)
}

func (c *FullNodeStruct) MpoolBatchPushMessage(ctx context.Context, msgs []*types.Message, spec *types.MessageSendSpec) ([]*types.SignedMessage, error) {
	return c.Internal.MpoolBatchPushMessage(ctx, msgs, spec)
}

func (c *FullNodeStruct) MpoolSub(ctx context.Context) (<-chan messagepool.MpoolUpdate, error) {
	return c.Internal.MpoolSub(ctx)
}

func (c *FullNodeStruct) MinerGetBaseInfo(ctx context.Context, maddr address.Address, epoch abi.ChainEpoch, tsk types.TipSetKey) (*apitypes.MiningBaseInfo, error) {
	return c.Internal.MinerGetBaseInfo(ctx, maddr, epoch, tsk)
}

func (c *FullNodeStruct) MinerCreateBlock(ctx context.Context, bt *apitypes.BlockTemplate) (*types.BlockMsg, error) {
	return c.Internal.MinerCreateBlock(ctx, bt)
}

func (c *FullNodeStruct) ChainHead(ctx context.Context) (*types.TipSet, error) {
	return c.Internal.ChainHead(ctx)
}

func (c *FullNodeStruct) ChainGetRandomnessFromTickets(ctx context.Context, tsk types.TipSetKey, personalization crypto.DomainSeparationTag, randEpoch abi.ChainEpoch, entropy []byte) (abi.Randomness, error) {
	return c.Internal.ChainGetRandomnessFromTickets(ctx, tsk, personalization, randEpoch, entropy)
}

func (c *FullNodeStruct) ChainGetRandomnessFromBeacon(ctx context.Context, tsk types.TipSetKey, personalization crypto.DomainSeparationTag, randEpoch abi.ChainEpoch, entropy []byte) (abi.Randomness, error) {
	return c.Internal.ChainGetRandomnessFromBeacon(ctx, tsk, personalization, randEpoch, entropy)
}

func (c *FullNodeStruct) ChainGetTipSetByHeight(ctx context.Context, h abi.ChainEpoch, tsk types.TipSetKey) (*types.TipSet, error) {
	return c.Internal.ChainGetTipSetByHeight(ctx, h, tsk)
}

func (c *FullNodeStruct) WalletNew(ctx context.Context, typ types.KeyType) (address.Address, error) {
	return c.Internal.WalletNew(ctx, typ)
}

func (c *FullNodeStruct) WalletHas(ctx context.Context, addr address.Address) (bool, error) {
	return c.Internal.WalletHas(ctx, addr)
}

func (c *FullNodeStruct) WalletList(ctx context.Context) ([]address.Address, error) {
	return c.Internal.WalletList(ctx)
}

func (c *FullNodeStruct) WalletBalance(ctx context.Context, a address.Address) (types.BigInt, error) {
	return c.Internal.WalletBalance(ctx, a)
}

func (c *FullNodeStruct) WalletSign(ctx context.Context, k address.Address, msg []byte) (*crypto.Signature, error) {
	return c.Internal.WalletSign(ctx, k, msg)
}

func (c *FullNodeStruct) WalletSignMessage(ctx context.Context, k address.Address, msg *types.Message) (*types.SignedMessage, error) {
	return c.Internal.WalletSignMessage(ctx, k, msg)
}

func (c *FullNodeStruct) WalletVerify(ctx context.Context, k address.Address, msg []byte, sig *crypto.Signature) (bool, error) {
	return c.Internal.WalletVerify(ctx, k, msg, sig)
}

func (c *FullNodeStruct) WalletDefaultAddress(ctx context.Context) (address.Address, error) {
	return c.Internal.WalletDefaultAddress(ctx)
}

func (c *FullNodeStruct) WalletSetDefault(ctx context.Context, a address.Address) error {
	return c.Internal.WalletSetDefault(ctx, a)
}

func (c *FullNodeStruct) WalletDelete(ctx context.Context, addr address.Address) error {
	return c.Internal.WalletDelete(ctx, addr)
}

func (c *FullNodeStruct) WalletValidateAddress(ctx context.Context, str string) (address.Address, error) {
	return c.Internal.WalletValidateAddress(ctx, str)
}

func (c *FullNodeStruct) MpoolGetNonce(ctx context.Context, addr address.Address) (uint64, error) {
	return c.Internal.MpoolGetNonce(ctx, addr)
}

func (c *FullNodeStruct) ChainGetBlock(ctx context.Context, b cid.Cid) (*types.BlockHeader, error) {
	return c.Internal.ChainGetBlock(ctx, b)
}

func (c *FullNodeStruct) ChainGetTipSet(ctx context.Context, key types.TipSetKey) (*types.TipSet, error) {
	return c.Internal.ChainGetTipSet(ctx, key)
}

func (c *FullNodeStruct) ChainGetBlockMessages(ctx context.Context, b cid.Cid) (*apitypes.BlockMessages, error) {
	return c.Internal.ChainGetBlockMessages(ctx, b)
}

func (c *FullNodeStruct) ChainGetParentReceipts(ctx context.Context, b cid.Cid) ([]*types.MessageReceipt, error) {
	return c.Internal.ChainGetParentReceipts(ctx, b)
}

func (c *FullNodeStruct) ChainGetParentMessages(ctx context.Context, b cid.Cid) ([]apitypes.Message, error) {
	return c.Internal.ChainGetParentMessages(ctx, b)
}

func (c *FullNodeStruct) ChainNotify(ctx context.Context) (<-chan []*chain.HeadChange, error) {
	return c.Internal.ChainNotify(ctx)
}

func (c *FullNodeStruct) ChainReadObj(ctx context.Context, obj cid.Cid) ([]byte, error) {
	return c.Internal.ChainReadObj(ctx, obj)
}

func (c *FullNodeStruct) ChainDeleteObj(ctx context.Context, obj cid.Cid) error {
	return c.Internal.ChainDeleteObj(ctx, obj)
}

func (c *FullNodeStruct) ChainHasObj(ctx context.Context, o cid.Cid) (bool, error) {
	return c.Internal.ChainHasObj(ctx, o)
}

func (c *FullNodeStruct) ChainStatObj(ctx context.Context, obj, base cid.Cid) (ObjStat, error) {
	return c.Internal.ChainStatObj(ctx, obj, base)
}

func (c *FullNodeStruct) ChainSetHead(ctx context.Context, tsk types.TipSetKey) error {
	return c.Internal.ChainSetHead(ctx, tsk)
}

func (c *FullNodeStruct) ChainGetGenesis(ctx context.Context) (*types.TipSet, error) {
	return c.Internal.ChainGetGenesis(ctx)
}

func (c *FullNodeStruct) ChainTipSetWeight(ctx context.Context, tsk types.TipSetKey) (types.BigInt, error) {
	return c.Internal.ChainTipSetWeight(ctx, tsk)
}

func (c *FullNodeStruct) ChainGetNode(ctx context.Context, p string) (*IpldObject, error) {
	return c.Internal.ChainGetNode(ctx, p)
}

func (c *FullNodeStruct) ChainGetMessage(ctx context.Context, mc cid.Cid) (*types.Message, error) {
	return c.Internal.ChainGetMessage(ctx, mc)
}

func (c *FullNodeStruct) ChainGetPath(ctx context.Context, from types.TipSetKey, to types.TipSetKey) ([]*chain.HeadChange, error) {
	return c.Internal.ChainGetPath(ctx, from, to)
}

func (c *FullNodeStruct) ChainExport(ctx context.Context, nroots abi.ChainEpoch, iom bool, tsk types.TipSetKey) (<-chan []byte, error) {
	return c.Internal.ChainExport(ctx, nroots, iom, tsk)
}

func (c *FullNodeStruct) BeaconGetEntry(ctx context.Context, epoch abi.ChainEpoch) (*types.BeaconEntry, error) {
	return c.Internal.BeaconGetEntry(ctx, epoch)
}

func (c *FullNodeStruct) SyncState(ctx context.Context) (*apitypes.SyncState, error) {
	return c.Internal.SyncState(ctx)
}

func (c *FullNodeStruct) SyncSubmitBlock(ctx context.Context, blk *types.BlockMsg) error {
	return c.Internal.SyncSubmitBlock(ctx, blk)
}

func (c *FullNodeStruct) SyncIncomingBlocks(ctx context.Context) (<-chan *types.BlockHeader, error) {
	return c.Internal.SyncIncomingBlocks(ctx)
}

func (c *FullNodeStruct) SyncCheckpoint(ctx context.Context, tsk types.TipSetKey) error {
	return c.Internal.SyncCheckpoint(ctx, tsk)
}

func (c *FullNodeStruct) SyncMarkBad(ctx context.Context, bcid cid.Cid) error {
	return c.Internal.SyncMarkBad(ctx, bcid)
}

func (c *FullNodeStruct) SyncUnmarkBad(ctx context.Context, bcid cid.Cid) error {
	return c.Internal.SyncUnmarkBad(ctx, bcid)
}

func (c *FullNodeStruct) SyncUnmarkAllBad(ctx context.Context) error {
	return c.Internal.SyncUnmarkAllBad(ctx)
}

func (c *FullNodeStruct) SyncCheckBad(ctx context.Context, bcid cid.Cid) (string, error) {
	return c.Internal.SyncCheckBad(ctx, bcid)
}

func (c *FullNodeStruct) SyncValidateTipset(ctx context.Context, tsk types.TipSetKey) (bool, error) {
	return c.Internal.SyncValidateTipset(ctx, tsk)
}

func (c *FullNodeStruct) StateNetworkName(ctx context.Context) (types2.NetworkName, error) {
	return c.Internal.StateNetworkName(ctx)
}

func (c *FullNodeStruct) StateMinerSectors(ctx context.Context, addr address.Address, sectorNos *bitfield.BitField, tsk types.TipSetKey) ([]*miner.SectorOnChainInfo, error) {
	return c.Internal.StateMinerSectors(ctx, addr, sectorNos, tsk)
}

func (c *FullNodeStruct) StateMinerActiveSectors(ctx context.Context, addr address.Address, tsk types.TipSetKey) ([]*miner.SectorOnChainInfo, error) {
	return c.Internal.StateMinerActiveSectors(ctx, addr, tsk)
}

func (c *FullNodeStruct) StateMinerProvingDeadline(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*dline.Info, error) {
	return c.Internal.StateMinerProvingDeadline(ctx, addr, tsk)
}

func (c *FullNodeStruct) StateMinerPower(ctx context.Context, a address.Address, tsk types.TipSetKey) (*apitypes.MinerPower, error) {
	return c.Internal.StateMinerPower(ctx, a, tsk)
}

func (c *FullNodeStruct) StateMinerInfo(ctx context.Context, actor address.Address, tsk types.TipSetKey) (miner.MinerInfo, error) {
	return c.Internal.StateMinerInfo(ctx, actor, tsk)
}

func (c *FullNodeStruct) StateMinerDeadlines(ctx context.Context, actor address.Address, tsk types.TipSetKey) ([]apitypes.Deadline, error) {
	return c.Internal.StateMinerDeadlines(ctx, actor, tsk)
}

func (c *FullNodeStruct) StateMinerPartitions(ctx context.Context, m address.Address, dlIdx uint64, tsk types.TipSetKey) ([]apitypes.Partition, error) {
	return c.Internal.StateMinerPartitions(ctx, m, dlIdx, tsk)
}

func (c *FullNodeStruct) StateMinerFaults(ctx context.Context, actor address.Address, tsk types.TipSetKey) (bitfield.BitField, error) {
	return c.Internal.StateMinerFaults(ctx, actor, tsk)
}

func (c *FullNodeStruct) StateAllMinerFaults(ctx context.Context, cutoff abi.ChainEpoch, endTsk types.TipSetKey) ([]*Fault, error) {
	return c.Internal.StateAllMinerFaults(ctx, cutoff, endTsk)
}

func (c *FullNodeStruct) StateMinerRecoveries(ctx context.Context, actor address.Address, tsk types.TipSetKey) (bitfield.BitField, error) {
	return c.Internal.StateMinerRecoveries(ctx, actor, tsk)
}

func (c *FullNodeStruct) StateMinerPreCommitDepositForPower(ctx context.Context, maddr address.Address, pci miner.SectorPreCommitInfo, tsk types.TipSetKey) (types.BigInt, error) {
	return c.Internal.StateMinerPreCommitDepositForPower(ctx, maddr, pci, tsk)
}

func (c *FullNodeStruct) StateMinerInitialPledgeCollateral(ctx context.Context, maddr address.Address, pci miner.SectorPreCommitInfo, tsk types.TipSetKey) (types.BigInt, error) {
	return c.Internal.StateMinerInitialPledgeCollateral(ctx, maddr, pci, tsk)
}

func (c *FullNodeStruct) StateMinerAvailableBalance(ctx context.Context, maddr address.Address, tsk types.TipSetKey) (types.BigInt, error) {
	return c.Internal.StateMinerAvailableBalance(ctx, maddr, tsk)
}

func (c *FullNodeStruct) StateMinerSectorAllocated(ctx context.Context, maddr address.Address, s abi.SectorNumber, tsk types.TipSetKey) (bool, error) {
	return c.Internal.StateMinerSectorAllocated(ctx, maddr, s, tsk)
}

func (c *FullNodeStruct) StateSectorPreCommitInfo(ctx context.Context, maddr address.Address, n abi.SectorNumber, tsk types.TipSetKey) (miner.SectorPreCommitOnChainInfo, error) {
	return c.Internal.StateSectorPreCommitInfo(ctx, maddr, n, tsk)
}

func (c *FullNodeStruct) StateSectorGetInfo(ctx context.Context, maddr address.Address, n abi.SectorNumber, tsk types.TipSetKey) (*miner.SectorOnChainInfo, error) {
	return c.Internal.StateSectorGetInfo(ctx, maddr, n, tsk)
}

func (c *FullNodeStruct) StateSectorExpiration(ctx context.Context, maddr address.Address, n abi.SectorNumber, tsk types.TipSetKey) (*miner.SectorExpiration, error) {
	return c.Internal.StateSectorExpiration(ctx, maddr, n, tsk)
}

func (c *FullNodeStruct) StateSectorPartition(ctx context.Context, maddr address.Address, sectorNumber abi.SectorNumber, tok types.TipSetKey) (*miner.SectorLocation, error) {
	return c.Internal.StateSectorPartition(ctx, maddr, sectorNumber, tok)
}

func (c *FullNodeStruct) StateCall(ctx context.Context, msg *types.Message, tsk types.TipSetKey) (*apitypes.InvocResult, error) {
	return c.Internal.StateCall(ctx, msg, tsk)
}

func (c *FullNodeStruct) StateReplay(ctx context.Context, tsk types.TipSetKey, mc cid.Cid) (*apitypes.InvocResult, error) {
	return c.Internal.StateReplay(ctx, tsk, mc)
}

func (c *FullNodeStruct) StateGetActor(ctx context.Context, actor address.Address, tsk types.TipSetKey) (*types.Actor, error) {
	return c.Internal.StateGetActor(ctx, actor, tsk)
}

func (c *FullNodeStruct) StateReadState(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*ActorState, error) {
	return c.Internal.StateReadState(ctx, addr, tsk)
}

func (c *FullNodeStruct) StateWaitMsg(ctx context.Context, msgc cid.Cid, confidence uint64, limit abi.ChainEpoch, allowReplaced bool) (*apitypes.MsgLookup, error) {
	return c.Internal.StateWaitMsg(ctx, msgc, confidence, limit, allowReplaced)
}

func (c *FullNodeStruct) StateWaitMsgLimited(ctx context.Context, msgc cid.Cid, confidence uint64, limit abi.ChainEpoch) (*chain.MsgLookup, error) {
	return c.Internal.StateWaitMsgLimited(ctx, msgc, confidence, limit)
}

func (c *FullNodeStruct) StateSearchMsg(ctx context.Context, from types.TipSetKey, msg cid.Cid, limit abi.ChainEpoch, allowReplaced bool) (*apitypes.MsgLookup, error) {
	return c.Internal.StateSearchMsg(ctx, from, msg, limit, allowReplaced)
}

func (c *FullNodeStruct) StateSearchMsgLimited(ctx context.Context, msgc cid.Cid, limit abi.ChainEpoch) (*chain.MsgLookup, error) {
	return c.Internal.StateSearchMsgLimited(ctx, msgc, limit)
}

func (c *FullNodeStruct) StateListMiners(ctx context.Context, tsk types.TipSetKey) ([]address.Address, error) {
	return c.Internal.StateListMiners(ctx, tsk)
}

func (c *FullNodeStruct) StateListActors(ctx context.Context, tsk types.TipSetKey) ([]address.Address, error) {
	return c.Internal.StateListActors(ctx, tsk)
}

func (c *FullNodeStruct) StateMarketBalance(ctx context.Context, addr address.Address, tsk types.TipSetKey) (apitypes.MarketBalance, error) {
	return c.Internal.StateMarketBalance(ctx, addr, tsk)
}

func (c *FullNodeStruct) StateMarketParticipants(ctx context.Context, tsk types.TipSetKey) (map[string]apitypes.MarketBalance, error) {
	return c.Internal.StateMarketParticipants(ctx, tsk)
}

func (c *FullNodeStruct) StateMarketDeals(ctx context.Context, tsk types.TipSetKey) (map[string]apitypes.MarketDeal, error) {
	return c.Internal.StateMarketDeals(ctx, tsk)
}

func (c *FullNodeStruct) StateMarketStorageDeal(ctx context.Context, dealid abi.DealID, tsk types.TipSetKey) (*apitypes.MarketDeal, error) {
	return c.Internal.StateMarketStorageDeal(ctx, dealid, tsk)
}

func (c *FullNodeStruct) StateLookupID(ctx context.Context, addr address.Address, tsk types.TipSetKey) (address.Address, error) {
	return c.Internal.StateLookupID(ctx, addr, tsk)
}

func (c *FullNodeStruct) StateAccountKey(ctx context.Context, addr address.Address, tsk types.TipSetKey) (address.Address, error) {
	return c.Internal.StateAccountKey(ctx, addr, tsk)
}

func (c *FullNodeStruct) StateChangedActors(ctx context.Context, olnstate cid.Cid, newstate cid.Cid) (map[string]types.Actor, error) {
	return c.Internal.StateChangedActors(ctx, olnstate, newstate)
}

func (c *FullNodeStruct) StateGetReceipt(ctx context.Context, msg cid.Cid, tsk types.TipSetKey) (*types.MessageReceipt, error) {
	return c.Internal.StateGetReceipt(ctx, msg, tsk)
}

func (c *FullNodeStruct) StateListMessages(ctx context.Context, match *MessageMatch, tsk types.TipSetKey, toht abi.ChainEpoch) ([]cid.Cid, error) {
	return c.Internal.StateListMessages(ctx, match, tsk, toht)
}

func (c *FullNodeStruct) StateDecodeParams(ctx context.Context, toAddr address.Address, method abi.MethodNum, params []byte, tsk types.TipSetKey) (interface{}, error) {
	return c.Internal.StateDecodeParams(ctx, toAddr, method, params, tsk)
}

func (c *FullNodeStruct) StateCompute(ctx context.Context, height abi.ChainEpoch, msgs []*types.Message, tsk types.TipSetKey) (*apitypes.ComputeStateOutput, error) {
	return c.Internal.StateCompute(ctx, height, msgs, tsk)
}

func (c *FullNodeStruct) StateVerifierStatus(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*abi.StoragePower, error) {
	return c.Internal.StateVerifierStatus(ctx, addr, tsk)
}

func (c *FullNodeStruct) StateVerifiedClientStatus(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*abi.StoragePower, error) {
	return c.Internal.StateVerifiedClientStatus(ctx, addr, tsk)
}

func (c *FullNodeStruct) StateVerifiedRegistryRootKey(ctx context.Context, tsk types.TipSetKey) (address.Address, error) {
	return c.Internal.StateVerifiedRegistryRootKey(ctx, tsk)
}

func (c *FullNodeStruct) StateDealProviderCollateralBounds(ctx context.Context, size abi.PaddedPieceSize, verified bool, tsk types.TipSetKey) (DealCollateralBounds, error) {
	return c.Internal.StateDealProviderCollateralBounds(ctx, size, verified, tsk)
}

func (c *FullNodeStruct) StateCirculatingSupply(ctx context.Context, tsk types.TipSetKey) (abi.TokenAmount, error) {
	return c.Internal.StateCirculatingSupply(ctx, tsk)
}

func (c *FullNodeStruct) StateVMCirculatingSupplyInternal(ctx context.Context, tsk types.TipSetKey) (chain.CirculatingSupply, error) {
	return c.Internal.StateVMCirculatingSupplyInternal(ctx, tsk)
}

func (c *FullNodeStruct) StateNetworkVersion(ctx context.Context, tsk types.TipSetKey) (stnetwork.Version, error) {
	return c.Internal.StateNetworkVersion(ctx, tsk)
}

func (c *FullNodeStruct) MsigGetAvailableBalance(ctx context.Context, a address.Address, tsk types.TipSetKey) (types.BigInt, error) {
	return c.Internal.MsigGetAvailableBalance(ctx, a, tsk)
}

func (c *FullNodeStruct) MsigGetVestingSchedule(ctx context.Context, a address.Address, tsk types.TipSetKey) (MsigVesting, error) {
	return c.Internal.MsigGetVestingSchedule(ctx, a, tsk)
}

func (c *FullNodeStruct) MsigGetVested(ctx context.Context, a address.Address, sTsk types.TipSetKey, eTsk types.TipSetKey) (types.BigInt, error) {
	return c.Internal.MsigGetVested(ctx, a, sTsk, eTsk)
}

func (c *FullNodeStruct) MsigCreate(ctx context.Context, req uint64, addrs []address.Address, duration abi.ChainEpoch, val types.BigInt, src address.Address, gp types.BigInt) (cid.Cid, error) {
	return c.Internal.MsigCreate(ctx, req, addrs, duration, val, src, gp)
}

func (c *FullNodeStruct) MsigPropose(ctx context.Context, msig address.Address, to address.Address, amt types.BigInt, src address.Address, method uint64, params []byte) (cid.Cid, error) {
	return c.Internal.MsigPropose(ctx, msig, to, amt, src, method, params)
}

func (c *FullNodeStruct) MsigApprove(ctx context.Context, msig address.Address, txID uint64, signer address.Address) (cid.Cid, error) {
	return c.Internal.MsigApprove(ctx, msig, txID, signer)
}

func (c *FullNodeStruct) MsigApproveTxnHash(ctx context.Context, msig address.Address, txID uint64, proposer address.Address, to address.Address, amt types.BigInt, src address.Address, method uint64, params []byte) (cid.Cid, error) {
	return c.Internal.MsigApproveTxnHash(ctx, msig, txID, proposer, to, amt, src, method, params)
}

func (c *FullNodeStruct) MsigCancel(ctx context.Context, msig address.Address, txID uint64, to address.Address, amt types.BigInt, src address.Address, method uint64, params []byte) (cid.Cid, error) {
	return c.Internal.MsigCancel(ctx, msig, txID, to, amt, src, method, params)
}

func (c *FullNodeStruct) MsigAddPropose(ctx context.Context, msig address.Address, src address.Address, newAdd address.Address, inc bool) (cid.Cid, error) {
	return c.Internal.MsigAddPropose(ctx, msig, src, newAdd, inc)
}

func (c *FullNodeStruct) MsigAddApprove(ctx context.Context, msig address.Address, src address.Address, txID uint64, proposer address.Address, newAdd address.Address, inc bool) (cid.Cid, error) {
	return c.Internal.MsigAddApprove(ctx, msig, src, txID, proposer, newAdd, inc)
}

func (c *FullNodeStruct) MsigAddCancel(ctx context.Context, msig address.Address, src address.Address, txID uint64, newAdd address.Address, inc bool) (cid.Cid, error) {
	return c.Internal.MsigAddCancel(ctx, msig, src, txID, newAdd, inc)
}

func (c *FullNodeStruct) MsigSwapPropose(ctx context.Context, msig address.Address, src address.Address, oldAdd address.Address, newAdd address.Address) (cid.Cid, error) {
	return c.Internal.MsigSwapPropose(ctx, msig, src, oldAdd, newAdd)
}

func (c *FullNodeStruct) MsigSwapApprove(ctx context.Context, msig address.Address, src address.Address, txID uint64, proposer address.Address, oldAdd address.Address, newAdd address.Address) (cid.Cid, error) {
	return c.Internal.MsigSwapApprove(ctx, msig, src, txID, proposer, oldAdd, newAdd)
}

func (c *FullNodeStruct) MsigSwapCancel(ctx context.Context, msig address.Address, src address.Address, txID uint64, oldAdd address.Address, newAdd address.Address) (cid.Cid, error) {
	return c.Internal.MsigSwapCancel(ctx, msig, src, txID, oldAdd, newAdd)
}

func (c *FullNodeStruct) MsigRemoveSigner(ctx context.Context, msig address.Address, proposer address.Address, toRemove address.Address, decrease bool) (cid.Cid, error) {
	return c.Internal.MsigRemoveSigner(ctx, msig, proposer, toRemove, decrease)
}

func (c *FullNodeStruct) MarketAddBalance(ctx context.Context, wallet address.Address, addr address.Address, amt types.BigInt) (cid.Cid, error) {
	return c.Internal.MarketAddBalance(ctx, wallet, addr, amt)
}

func (c *FullNodeStruct) MarketGetReserved(ctx context.Context, addr address.Address) (types.BigInt, error) {
	return c.Internal.MarketGetReserved(ctx, addr)
}

func (c *FullNodeStruct) MarketReserveFunds(ctx context.Context, wallet address.Address, addr address.Address, amt types.BigInt) (cid.Cid, error) {
	return c.Internal.MarketReserveFunds(ctx, wallet, addr, amt)
}

func (c *FullNodeStruct) MarketReleaseFunds(ctx context.Context, addr address.Address, amt types.BigInt) error {
	return c.Internal.MarketReleaseFunds(ctx, addr, amt)
}

func (c *FullNodeStruct) MarketWithdraw(ctx context.Context, wallet, addr address.Address, amt types.BigInt) (cid.Cid, error) {
	return c.Internal.MarketWithdraw(ctx, wallet, addr, amt)
}

func (c *FullNodeStruct) PaychGet(ctx context.Context, from, to address.Address, amt types.BigInt) (*apitypes.ChannelInfo, error) {
	return c.Internal.PaychGet(ctx, from, to, amt)
}

func (c *FullNodeStruct) PaychGetWaitReady(ctx context.Context, sentinel cid.Cid) (address.Address, error) {
	return c.Internal.PaychGetWaitReady(ctx, sentinel)
}

func (c *FullNodeStruct) PaychAvailableFunds(ctx context.Context, ch address.Address) (*paych2.ChannelAvailableFunds, error) {
	return c.Internal.PaychAvailableFunds(ctx, ch)
}

func (c *FullNodeStruct) PaychAvailableFundsByFromTo(ctx context.Context, from, to address.Address) (*paych2.ChannelAvailableFunds, error) {
	return c.Internal.PaychAvailableFundsByFromTo(ctx, from, to)
}

func (c *FullNodeStruct) PaychList(ctx context.Context) ([]address.Address, error) {
	return c.Internal.PaychList(ctx)
}

func (c *FullNodeStruct) PaychStatus(ctx context.Context, pch address.Address) (*paych2.PaychStatus, error) {
	return c.Internal.PaychStatus(ctx, pch)
}

func (c *FullNodeStruct) PaychVoucherCheckValid(ctx context.Context, addr address.Address, sv *paych.SignedVoucher) error {
	return c.Internal.PaychVoucherCheckValid(ctx, addr, sv)
}

func (c *FullNodeStruct) PaychVoucherCheckSpendable(ctx context.Context, addr address.Address, sv *paych.SignedVoucher, secret []byte, proof []byte) (bool, error) {
	return c.Internal.PaychVoucherCheckSpendable(ctx, addr, sv, secret, proof)
}

func (c *FullNodeStruct) PaychVoucherAdd(ctx context.Context, addr address.Address, sv *paych.SignedVoucher, proof []byte, minDelta types.BigInt) (types.BigInt, error) {
	return c.Internal.PaychVoucherAdd(ctx, addr, sv, proof, minDelta)
}

func (c *FullNodeStruct) PaychVoucherCreate(ctx context.Context, pch address.Address, amt types.BigInt, lane uint64) (*apitypes.VoucherCreateResult, error) {
	return c.Internal.PaychVoucherCreate(ctx, pch, amt, lane)
}

func (c *FullNodeStruct) PaychVoucherList(ctx context.Context, pch address.Address) ([]*paych.SignedVoucher, error) {
	return c.Internal.PaychVoucherList(ctx, pch)
}

func (c *FullNodeStruct) PaychSettle(ctx context.Context, a address.Address) (cid.Cid, error) {
	return c.Internal.PaychSettle(ctx, a)
}

func (c *FullNodeStruct) PaychCollect(ctx context.Context, a address.Address) (cid.Cid, error) {
	return c.Internal.PaychCollect(ctx, a)
}

func (c *FullNodeStruct) PaychAllocateLane(ctx context.Context, ch address.Address) (uint64, error) {
	return c.Internal.PaychAllocateLane(ctx, ch)
}

func (c *FullNodeStruct) PaychNewPayment(ctx context.Context, from, to address.Address, vouchers []apitypes.VoucherSpec) (*apitypes.PaymentInfo, error) {
	return c.Internal.PaychNewPayment(ctx, from, to, vouchers)
}

func (c *FullNodeStruct) PaychVoucherSubmit(ctx context.Context, ch address.Address, sv *paych.SignedVoucher, secret []byte, proof []byte) (cid.Cid, error) {
	return c.Internal.PaychVoucherSubmit(ctx, ch, sv, secret, proof)
}

func (c *FullNodeStruct) CreateBackup(ctx context.Context, fpath string) error {
	return c.Internal.CreateBackup(ctx, fpath)
}
