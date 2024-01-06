package types

import (
	"context"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/evmos/evmos/v15/x/evm/statedb"
	evmtypes "github.com/evmos/evmos/v15/x/evm/types"
	"time"
)

// ParamSubspace defines the expected Subspace interface for parameters (noalias)
type ParamSubspace interface {
	Get(ctx sdk.Context, key []byte, ptr interface{})
	Set(ctx sdk.Context, key []byte, param interface{})
}

type DistributionKeeper interface {
	IsBurnValidator(ctx sdk.Context, validator stakingtypes.ValidatorI) bool
}

// StakingKeeper expected staking keeper (Validator and Delegator sets) (noalias)
type StakingKeeper interface {
	// iterate through bonded validators by operator address, execute func for each validator
	IterateBondedValidatorsByPower(
		sdk.Context, func(index int64, validator stakingtypes.ValidatorI) (stop bool),
	)

	TotalBondedTokens(sdk.Context) math.Int // total bonded tokens within the validator set
	IterateDelegations(
		ctx sdk.Context, delegator sdk.AccAddress,
		fn func(index int64, delegation stakingtypes.DelegationI) (stop bool),
	)
	SetEvmCallback(cb sdk.GovEventCallback)
	GetValidator(ctx sdk.Context, addr sdk.ValAddress) (validator stakingtypes.Validator, found bool)
	Undelegate(
		ctx sdk.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, sharesAmount sdk.Dec,
	) (time.Time, error)
	ValidateUnbondAmount(
		ctx sdk.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, amt math.Int,
	) (shares sdk.Dec, err error)
	BondDenom(ctx sdk.Context) string
	ValidatorByConsAddr(sdk.Context, sdk.ConsAddress) stakingtypes.ValidatorI // get a particular validator by consensus address
	CreateEvmValidator(ctx sdk.Context, valAddr sdk.ValAddress) (*stakingtypes.MsgCreateValidatorResponse, error)
}

// AccountKeeper defines the expected account keeper (noalias)
type AccountKeeper interface {
	GetAccount(ctx sdk.Context, addr sdk.AccAddress) types.AccountI

	GetModuleAddress(name string) sdk.AccAddress
	GetModuleAccount(ctx sdk.Context, name string) types.ModuleAccountI

	// TODO remove with genesis 2-phases refactor https://github.com/cosmos/cosmos-sdk/issues/2862
	SetModuleAccount(sdk.Context, types.ModuleAccountI)
	GetSequence(sdk.Context, sdk.AccAddress) (uint64, error)
}

// BankKeeper defines the expected interface needed to retrieve account balances.
type BankKeeper interface {
	GetAllBalances(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins
	GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin
	LockedCoins(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins
	SpendableCoins(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins
	SendCoinsFromModuleToModule(ctx sdk.Context, senderModule string, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx sdk.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromAccountToModule(ctx sdk.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	BurnCoins(ctx sdk.Context, name string, amt sdk.Coins) error
	MintCoins(ctx sdk.Context, moduleName string, amt sdk.Coins) error
	UndelegateCoinsFromModuleToAccount(ctx sdk.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
}

// Event Hooks
// These can be utilized to communicate between a governance keeper and another
// keepers.

// GovHooks event hooks for governance proposal object (noalias)
type GovHooks interface {
	AfterProposalSubmission(ctx sdk.Context, proposalID uint64)                            // Must be called after proposal is submitted
	AfterProposalDeposit(ctx sdk.Context, proposalID uint64, depositorAddr sdk.AccAddress) // Must be called after a deposit is made
	AfterProposalVote(ctx sdk.Context, proposalID uint64, voterAddr sdk.AccAddress)        // Must be called after a vote on a proposal is cast
	AfterProposalFailedMinDeposit(ctx sdk.Context, proposalID uint64)                      // Must be called when proposal fails to reach min deposit
	AfterProposalVotingPeriodEnded(ctx sdk.Context, proposalID uint64)                     // Must be called when proposal's finishes it's voting period
}

type GovHooksWrapper struct{ GovHooks }

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (GovHooksWrapper) IsOnePerModuleType() {}

// EVMKeeper defines the expected EVM keeper interface used on erc20
type EVMKeeper interface {
	GetParams(ctx sdk.Context) evmtypes.Params
	GetAccountWithoutBalance(ctx sdk.Context, addr common.Address) *statedb.Account
	EstimateGas(c context.Context, req *evmtypes.EthCallRequest) (*evmtypes.EstimateGasResponse, error)
	ApplyMessage(ctx sdk.Context, msg core.Message, tracer vm.EVMLogger, commit bool) (*evmtypes.MsgEthereumTxResponse, error)
}
