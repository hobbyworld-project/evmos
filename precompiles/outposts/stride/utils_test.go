// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)
package stride_test

import (
	"encoding/json"
	"math/big"
	"time"

	"github.com/evmos/evmos/v14/precompiles/outposts/stride"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/crypto/tmhash"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	tmtypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/testutil/mock"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	transfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	evmosapp "github.com/evmos/evmos/v14/app"
	evmosibc "github.com/evmos/evmos/v14/ibc/testing"
	"github.com/evmos/evmos/v14/precompiles/authorization"
	cmn "github.com/evmos/evmos/v14/precompiles/common"
	"github.com/evmos/evmos/v14/precompiles/ics20"
	"github.com/evmos/evmos/v14/precompiles/testutil"
	"github.com/evmos/evmos/v14/precompiles/testutil/contracts"
	evmosutil "github.com/evmos/evmos/v14/testutil"
	evmosutiltx "github.com/evmos/evmos/v14/testutil/tx"
	evmostypes "github.com/evmos/evmos/v14/types"
	"github.com/evmos/evmos/v14/utils"
	"github.com/evmos/evmos/v14/x/evm/statedb"
	evmtypes "github.com/evmos/evmos/v14/x/evm/types"
	feemarkettypes "github.com/evmos/evmos/v14/x/feemarket/types"
	inflationtypes "github.com/evmos/evmos/v14/x/inflation/types"

	. "github.com/onsi/gomega"
)

type erc20Meta struct {
	Name     string
	Symbol   string
	Decimals uint8
}

var (
	maxUint256Coins    = sdk.Coins{sdk.Coin{Denom: utils.BaseDenom, Amount: sdk.NewIntFromBigInt(abi.MaxUint256)}}
	maxUint256CmnCoins = []cmn.Coin{{Denom: utils.BaseDenom, Amount: abi.MaxUint256}}
	defaultCoins       = sdk.Coins{sdk.Coin{Denom: utils.BaseDenom, Amount: sdk.NewInt(1e18)}}
	baseDenomCmnCoin   = cmn.Coin{Denom: utils.BaseDenom, Amount: big.NewInt(1e18)}
	defaultCmnCoins    = []cmn.Coin{baseDenomCmnCoin}
	atomCoins          = sdk.Coins{sdk.Coin{Denom: "uatom", Amount: sdk.NewInt(1e18)}}
	atomCmnCoin        = cmn.Coin{Denom: "uatom", Amount: big.NewInt(1e18)}
	atomComnCoins      = []cmn.Coin{atomCmnCoin}
	mutliSpendLimit    = sdk.Coins{sdk.Coin{Denom: utils.BaseDenom, Amount: sdk.NewInt(1e18)}, sdk.Coin{Denom: "uatom", Amount: sdk.NewInt(1e18)}}
	mutliCmnCoins      = []cmn.Coin{baseDenomCmnCoin, atomCmnCoin}
	testERC20          = erc20Meta{
		Name:     "TestCoin",
		Symbol:   "TC",
		Decimals: 18,
	}
)

// SetupWithGenesisValSet initializes a new EvmosApp with a validator set and genesis accounts
// that also act as delegators. For simplicity, each validator is bonded with a delegation
// of one consensus engine unit (10^6) in the default token of the simapp from first genesis
// account. A Nop logger is set in SimApp.
func (s *PrecompileTestSuite) SetupWithGenesisValSet(valSet *tmtypes.ValidatorSet, genAccs []authtypes.GenesisAccount, balances ...banktypes.Balance) {
	appI, genesisState := evmosapp.SetupTestingApp(cmn.DefaultChainID)()
	app, ok := appI.(*evmosapp.Evmos)
	s.Require().True(ok)

	// set genesis accounts
	authGenesis := authtypes.NewGenesisState(authtypes.DefaultParams(), genAccs)
	genesisState[authtypes.ModuleName] = app.AppCodec().MustMarshalJSON(authGenesis)

	validators := make([]stakingtypes.Validator, 0, len(valSet.Validators))
	delegations := make([]stakingtypes.Delegation, 0, len(valSet.Validators))

	bondAmt := sdk.TokensFromConsensusPower(1, evmostypes.PowerReduction)

	for _, val := range valSet.Validators {
		pk, err := cryptocodec.FromTmPubKeyInterface(val.PubKey)
		s.Require().NoError(err)
		pkAny, err := codectypes.NewAnyWithValue(pk)
		s.Require().NoError(err)
		validator := stakingtypes.Validator{
			OperatorAddress:   sdk.ValAddress(val.Address).String(),
			ConsensusPubkey:   pkAny,
			Jailed:            false,
			Status:            stakingtypes.Bonded,
			Tokens:            bondAmt,
			DelegatorShares:   sdk.OneDec(),
			Description:       stakingtypes.Description{},
			UnbondingHeight:   int64(0),
			UnbondingTime:     time.Unix(0, 0).UTC(),
			Commission:        stakingtypes.NewCommission(sdk.ZeroDec(), sdk.ZeroDec(), sdk.ZeroDec()),
			MinSelfDelegation: sdk.ZeroInt(),
		}
		validators = append(validators, validator)
		delegations = append(delegations, stakingtypes.NewDelegation(genAccs[0].GetAddress(), val.Address.Bytes(), sdk.OneDec()))
	}
	s.validators = validators

	// set validators and delegations
	stakingParams := stakingtypes.DefaultParams()
	// set bond demon to be aevmos
	stakingParams.BondDenom = utils.BaseDenom
	stakingGenesis := stakingtypes.NewGenesisState(stakingParams, validators, delegations)
	genesisState[stakingtypes.ModuleName] = app.AppCodec().MustMarshalJSON(stakingGenesis)

	totalBondAmt := bondAmt.Mul(sdk.NewInt(int64(len(validators))))
	totalSupply := sdk.NewCoins()
	for _, b := range balances {
		// add genesis acc tokens and delegated tokens to total supply
		totalSupply = totalSupply.Add(b.Coins.Add(sdk.NewCoin(utils.BaseDenom, totalBondAmt))...)
	}

	// add bonded amount to bonded pool module account
	balances = append(balances, banktypes.Balance{
		Address: authtypes.NewModuleAddress(stakingtypes.BondedPoolName).String(),
		Coins:   sdk.Coins{sdk.NewCoin(utils.BaseDenom, totalBondAmt)},
	})

	// update total supply
	bankGenesis := banktypes.NewGenesisState(banktypes.DefaultGenesisState().Params, balances, totalSupply, []banktypes.Metadata{}, []banktypes.SendEnabled{})
	genesisState[banktypes.ModuleName] = app.AppCodec().MustMarshalJSON(bankGenesis)

	stateBytes, err := json.MarshalIndent(genesisState, "", " ")
	s.Require().NoError(err)

	feeGenesis := feemarkettypes.NewGenesisState(feemarkettypes.DefaultGenesisState().Params, 0)
	genesisState[feemarkettypes.ModuleName] = app.AppCodec().MustMarshalJSON(feeGenesis)

	// init chain will set the validator set and initialize the genesis accounts
	app.InitChain(
		abci.RequestInitChain{
			ChainId:         cmn.DefaultChainID,
			Validators:      []abci.ValidatorUpdate{},
			ConsensusParams: evmosapp.DefaultConsensusParams,
			AppStateBytes:   stateBytes,
		},
	)

	// commit genesis changes
	app.Commit()

	// instantiate new header
	header := evmosutil.NewHeader(
		2,
		time.Now().UTC(),
		cmn.DefaultChainID,
		sdk.ConsAddress(validators[0].GetOperator()),
		tmhash.Sum([]byte("app")),
		tmhash.Sum([]byte("validators")),
	)

	app.BeginBlock(abci.RequestBeginBlock{Header: header})

	// create Contexts
	s.ctx = app.BaseApp.NewContext(false, header)
	s.app = app
}

func (s *PrecompileTestSuite) DoSetupTest() {
	s.defaultExpirationDuration = s.ctx.BlockTime().Add(cmn.DefaultExpirationDuration).UTC()

	// generate validators private/public key
	var (
		validatorsPerChain = 2
		validators         []*tmtypes.Validator
		signersByAddress   = make(map[string]tmtypes.PrivValidator, validatorsPerChain)
	)

	for i := 0; i < validatorsPerChain; i++ {
		privVal := mock.NewPV()
		pubKey, err := privVal.GetPubKey()
		s.Require().NoError(err)
		validators = append(validators, tmtypes.NewValidator(pubKey, 1))
		signersByAddress[pubKey.Address().String()] = privVal
	}

	// construct validator set;
	// Note that the validators are sorted by voting power
	// or, if equal, by address lexical order
	s.valSet = tmtypes.NewValidatorSet(validators)

	// Create a coordinator and 2 test chains that will be used in the testing suite
	chains := make(map[string]*ibctesting.TestChain)
	s.coordinator = &ibctesting.Coordinator{
		T: s.T(),
		// NOTE: This year has to be updated otherwise the client will be shown as expired
		CurrentTime: time.Date(time.Now().Year()+1, 1, 2, 0, 0, 0, 0, time.UTC),
	}
	// Create 2 Evmos chains
	chains[cmn.DefaultChainID] = s.NewTestChainWithValSet(s.coordinator, s.valSet, signersByAddress)
	// TODO: Figure out if we want to make the second chain keepers accessible to the tests to check the state
	chainID2 := utils.MainnetChainID + "-2"
	chains[chainID2] = ibctesting.NewTestChain(s.T(), s.coordinator, chainID2)
	s.coordinator.Chains = chains

	// Setup Chains in the testing suite
	s.chainA = s.coordinator.GetChain(cmn.DefaultChainID)
	s.chainB = s.coordinator.GetChain(chainID2)

	if s.suiteIBCTesting {
		s.setupIBCTest()
	}
}

func (s *PrecompileTestSuite) NewTestChainWithValSet(coord *ibctesting.Coordinator, valSet *tmtypes.ValidatorSet, signers map[string]tmtypes.PrivValidator) *ibctesting.TestChain {
	// generate genesis account
	addr, priv := evmosutiltx.NewAddrKey()
	s.privKey = priv
	s.address = addr
	// differentAddr is an address generated for testing purposes that e.g. raises the different origin error
	s.differentAddr = evmosutiltx.GenerateAddress()
	s.signer = evmosutiltx.NewSigner(priv)

	baseAcc := authtypes.NewBaseAccount(priv.PubKey().Address().Bytes(), priv.PubKey(), 0, 0)

	acc := &evmostypes.EthAccount{
		BaseAccount: baseAcc,
		CodeHash:    common.BytesToHash(evmtypes.EmptyCodeHash).Hex(),
	}

	amount := sdk.TokensFromConsensusPower(5, evmostypes.PowerReduction)

	balance := banktypes.Balance{
		Address: acc.GetAddress().String(),
		Coins:   sdk.NewCoins(sdk.NewCoin(utils.BaseDenom, amount)),
	}

	s.SetupWithGenesisValSet(s.valSet, []authtypes.GenesisAccount{acc}, balance)

	// create current header and call begin block
	header := tmproto.Header{
		ChainID: cmn.DefaultChainID,
		Height:  1,
		Time:    coord.CurrentTime.UTC(),
	}

	txConfig := s.app.GetTxConfig()

	// Create StateDB
	s.stateDB = statedb.New(s.ctx, s.app.EvmKeeper, statedb.NewEmptyTxConfig(common.BytesToHash(s.ctx.HeaderHash().Bytes())))

	// bond denom
	stakingParams := s.app.StakingKeeper.GetParams(s.ctx)
	stakingParams.BondDenom = utils.BaseDenom
	s.bondDenom = stakingParams.BondDenom
	err := s.app.StakingKeeper.SetParams(s.ctx, stakingParams)
	s.Require().NoError(err)

	s.ethSigner = ethtypes.LatestSignerForChainID(s.app.EvmKeeper.ChainID())

	// Setting up the fee market to 0 so the transactions don't fail in IBC testing
	s.app.FeeMarketKeeper.SetBaseFee(s.ctx, big.NewInt(0))
	s.app.FeeMarketKeeper.SetBlockGasWanted(s.ctx, 0)
	s.app.FeeMarketKeeper.SetTransientBlockGasWanted(s.ctx, 0)

	precompile, err := stride.NewPrecompile(s.app.TransferKeeper, s.app.IBCKeeper.ChannelKeeper, s.app.AuthzKeeper, s.app.StakingKeeper)
	s.Require().NoError(err)
	s.precompile = precompile

	queryHelperEvm := baseapp.NewQueryServerTestHelper(s.ctx, s.app.InterfaceRegistry())
	evmtypes.RegisterQueryServer(queryHelperEvm, s.app.EvmKeeper)
	s.queryClientEVM = evmtypes.NewQueryClient(queryHelperEvm)

	// create an account to send transactions from
	chain := &ibctesting.TestChain{
		T:              s.T(),
		Coordinator:    coord,
		ChainID:        cmn.DefaultChainID,
		App:            s.app,
		CurrentHeader:  header,
		QueryServer:    s.app.GetIBCKeeper(),
		TxConfig:       txConfig,
		Codec:          s.app.AppCodec(),
		Vals:           valSet,
		NextVals:       valSet,
		Signers:        signers,
		SenderPrivKey:  priv,
		SenderAccount:  acc,
		SenderAccounts: []ibctesting.SenderAccount{{SenderPrivKey: priv, SenderAccount: acc}},
	}

	coord.CommitBlock(chain)

	return chain
}

// NewPrecompileContract creates a new precompile contract and sets the gas meter
func (s *PrecompileTestSuite) NewPrecompileContract(gas uint64) *vm.Contract {
	contract := vm.NewContract(vm.AccountRef(s.address), s.precompile, big.NewInt(0), gas)

	s.ctx = s.ctx.WithGasMeter(sdk.NewInfiniteGasMeter())
	initialGas := s.ctx.GasMeter().GasConsumed()
	s.Require().Zero(initialGas)

	return contract
}

// NewTransferAuthorizationWithAllocations creates a new allocation for the given grantee and granter and the given coins
func (s *PrecompileTestSuite) NewTransferAuthorizationWithAllocations(ctx sdk.Context, app *evmosapp.Evmos, grantee, granter common.Address, allocations []transfertypes.Allocation) error {
	transferAuthz := &transfertypes.TransferAuthorization{Allocations: allocations}
	if err := transferAuthz.ValidateBasic(); err != nil {
		return err
	}

	// create the authorization
	return app.AuthzKeeper.SaveGrant(ctx, grantee.Bytes(), granter.Bytes(), transferAuthz, &s.defaultExpirationDuration)
}

// NewTransferAuthorization creates a new transfer authorization for the given grantee and granter and the given coins
func (s *PrecompileTestSuite) NewTransferAuthorization(ctx sdk.Context, app *evmosapp.Evmos, grantee, granter common.Address, path *ibctesting.Path, coins sdk.Coins, allowList []string) error {
	allocations := []transfertypes.Allocation{
		{
			SourcePort:    path.EndpointA.ChannelConfig.PortID,
			SourceChannel: path.EndpointA.ChannelID,
			SpendLimit:    coins,
			AllowList:     allowList,
		},
	}

	transferAuthz := &transfertypes.TransferAuthorization{Allocations: allocations}
	if err := transferAuthz.ValidateBasic(); err != nil {
		return err
	}

	// create the authorization
	return app.AuthzKeeper.SaveGrant(ctx, grantee.Bytes(), granter.Bytes(), transferAuthz, &s.defaultExpirationDuration)
}

// GetTransferAuthorization returns the transfer authorization for the given grantee and granter
func (s *PrecompileTestSuite) GetTransferAuthorization(ctx sdk.Context, grantee, granter common.Address) *transfertypes.TransferAuthorization {
	grant, _ := s.app.AuthzKeeper.GetAuthorization(ctx, grantee.Bytes(), granter.Bytes(), ics20.TransferMsg)
	s.Require().NotNil(grant)
	transferAuthz, ok := grant.(*transfertypes.TransferAuthorization)
	s.Require().True(ok)
	s.Require().NotNil(transferAuthz)
	return transferAuthz
}

// CheckAllowanceChangeEvent is a helper function used to check the allowance change event arguments.
func (s *PrecompileTestSuite) CheckAllowanceChangeEvent(log *ethtypes.Log, methods []string, amounts []*big.Int) {
	// Check event signature matches the one emitted
	event := s.precompile.ABI.Events[authorization.EventTypeAllowanceChange]
	s.Require().Equal(event.ID, common.HexToHash(log.Topics[0].Hex()))
	s.Require().Equal(log.BlockNumber, uint64(s.ctx.BlockHeight()))

	var approvalEvent authorization.EventAllowanceChange
	err := cmn.UnpackLog(s.precompile.ABI, &approvalEvent, authorization.EventTypeAllowanceChange, *log)
	s.Require().NoError(err)
	s.Require().Equal(s.address, approvalEvent.Grantee)
	s.Require().Equal(s.address, approvalEvent.Granter)
	s.Require().Equal(len(methods), len(approvalEvent.Methods))

	for i, method := range methods {
		s.Require().Equal(method, approvalEvent.Methods[i])
		s.Require().Equal(amounts[i], approvalEvent.Values[i])
	}
}

// NewTransferPath creates a new path between two chains with the specified portIds and version.
func NewTransferPath(chainA, chainB *ibctesting.TestChain) *ibctesting.Path {
	path := ibctesting.NewPath(chainA, chainB)
	path.EndpointA.ChannelConfig.PortID = transfertypes.PortID
	path.EndpointB.ChannelConfig.PortID = transfertypes.PortID
	path.EndpointA.ChannelConfig.Version = transfertypes.Version
	path.EndpointB.ChannelConfig.Version = transfertypes.Version

	return path
}

// setupIBCTest makes the necessary setup of chains A & B
// for integration tests
func (s *PrecompileTestSuite) setupIBCTest() {
	s.coordinator.CommitNBlocks(s.chainA, 2)
	s.coordinator.CommitNBlocks(s.chainB, 2)

	s.app = s.chainA.App.(*evmosapp.Evmos)
	evmParams := s.app.EvmKeeper.GetParams(s.chainA.GetContext())
	evmParams.EvmDenom = utils.BaseDenom
	err := s.app.EvmKeeper.SetParams(s.chainA.GetContext(), evmParams)
	s.Require().NoError(err)

	// Set block proposer once, so its carried over on the ibc-go-testing suite
	validators := s.app.StakingKeeper.GetValidators(s.chainA.GetContext(), 2)
	cons, err := validators[0].GetConsAddr()
	s.Require().NoError(err)
	s.chainA.CurrentHeader.ProposerAddress = cons.Bytes()

	err = s.app.StakingKeeper.SetValidatorByConsAddr(s.chainA.GetContext(), validators[0])
	s.Require().NoError(err)

	_, err = s.app.EvmKeeper.GetCoinbaseAddress(s.chainA.GetContext(), sdk.ConsAddress(s.chainA.CurrentHeader.ProposerAddress))
	s.Require().NoError(err)

	// Mint coins locked on the evmos account generated with secp.
	amt, ok := sdk.NewIntFromString("1000000000000000000000")
	s.Require().True(ok)
	coinEvmos := sdk.NewCoin(utils.BaseDenom, amt)
	coins := sdk.NewCoins(coinEvmos)
	err = s.app.BankKeeper.MintCoins(s.chainA.GetContext(), inflationtypes.ModuleName, coins)
	s.Require().NoError(err)
	err = s.app.BankKeeper.SendCoinsFromModuleToAccount(s.chainA.GetContext(), inflationtypes.ModuleName, s.chainA.SenderAccount.GetAddress(), coins)
	s.Require().NoError(err)

	s.transferPath = evmosibc.NewTransferPath(s.chainA, s.chainB) // clientID, connectionID, channelID empty
	evmosibc.SetupPath(s.coordinator, s.transferPath)             // clientID, connectionID, channelID filled
	s.Require().Equal("07-tendermint-0", s.transferPath.EndpointA.ClientID)
	s.Require().Equal("connection-0", s.transferPath.EndpointA.ConnectionID)
	s.Require().Equal("channel-0", s.transferPath.EndpointA.ChannelID)
}

// setTransferApproval sets the transfer approval for the given grantee and allocations
func (s *PrecompileTestSuite) setTransferApproval(
	args contracts.CallArgs,
	grantee common.Address,
	allocations []ics20.Allocation,
) {
	args.MethodName = authorization.ApproveMethod
	args.Args = []interface{}{
		grantee,
		allocations,
	}

	logCheckArgs := testutil.LogCheckArgs{
		ABIEvents: s.precompile.Events,
		ExpEvents: []string{ics20.EventTypeIBCTransferAuthorization},
		ExpPass:   true,
	}

	_, _, err := contracts.CallContractAndCheckLogs(s.chainA.GetContext(), s.app, args, logCheckArgs)
	Expect(err).To(BeNil(), "error while calling the contract to approve")

	s.chainA.NextBlock()

	// check auth created successfully
	authz, _ := s.app.AuthzKeeper.GetAuthorization(s.chainA.GetContext(), grantee.Bytes(), args.PrivKey.PubKey().Address().Bytes(), ics20.TransferMsg)
	Expect(authz).NotTo(BeNil())
	transferAuthz, ok := authz.(*transfertypes.TransferAuthorization)
	Expect(ok).To(BeTrue())
	Expect(len(transferAuthz.Allocations[0].SpendLimit)).To(Equal(len(allocations[0].SpendLimit)))
	for i, sl := range transferAuthz.Allocations[0].SpendLimit {
		// NOTE order may change if there're more than one coin
		Expect(sl.Denom).To(Equal(allocations[0].SpendLimit[i].Denom))
		Expect(sl.Amount.BigInt()).To(Equal(allocations[0].SpendLimit[i].Amount))
	}
}

// setTransferApprovalForContract sets the transfer approval for the given contract
func (s *PrecompileTestSuite) setTransferApprovalForContract(args contracts.CallArgs) {
	logCheckArgs := testutil.LogCheckArgs{
		ABIEvents: s.precompile.Events,
		ExpEvents: []string{ics20.EventTypeIBCTransferAuthorization},
		ExpPass:   true,
	}

	_, _, err := contracts.CallContractAndCheckLogs(s.chainA.GetContext(), s.app, args, logCheckArgs)
	Expect(err).To(BeNil(), "error while calling the contract to approve")

	s.chainA.NextBlock()

	// check auth created successfully
	authz, _ := s.app.AuthzKeeper.GetAuthorization(s.chainA.GetContext(), args.ContractAddr.Bytes(), args.PrivKey.PubKey().Address().Bytes(), ics20.TransferMsg)
	Expect(authz).NotTo(BeNil())
	transferAuthz, ok := authz.(*transfertypes.TransferAuthorization)
	Expect(ok).To(BeTrue())
	Expect(len(transferAuthz.Allocations) > 0).To(BeTrue())
}

// makePacket is a helper function to build the sent IBC packet
// to perform an ICS20 tranfer.
// This packet is then used to test the IBC callbacks (Timeout, Ack)
func (s *PrecompileTestSuite) makePacket(
	senderAddr,
	receiverAddr,
	denom,
	memo string,
	amt *big.Int,
	seq uint64,
	timeoutHeight clienttypes.Height,
) channeltypes.Packet {
	packetData := transfertypes.NewFungibleTokenPacketData(
		denom,
		amt.String(),
		senderAddr,
		receiverAddr,
		memo,
	)

	return channeltypes.NewPacket(
		packetData.GetBytes(),
		seq,
		s.transferPath.EndpointA.ChannelConfig.PortID,
		s.transferPath.EndpointA.ChannelID,
		s.transferPath.EndpointB.ChannelConfig.PortID,
		s.transferPath.EndpointB.ChannelID,
		timeoutHeight,
		0,
	)
}