// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)

package keeper

import (
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/evmos/evmos/v15/contracts"
	evmcommon "github.com/evmos/evmos/v15/precompiles/common"
	evmtypes "github.com/evmos/evmos/v15/x/evm/types"
	"github.com/evmos/evmos/v15/x/gov/types"
	v1 "github.com/evmos/evmos/v15/x/gov/types/v1"
)

var _ evmtypes.EvmHooks = Hooks{}

// Hooks wrapper struct for erc20 keeper
type Hooks struct {
	k Keeper
}

// Return the wrapper struct
func (k Keeper) EvmHooks() Hooks {
	return Hooks{k}
}

// PostTxProcessing is a wrapper for calling the EVM PostTxProcessing hook on
// the module keeper
func (h Hooks) PostTxProcessing(ctx sdk.Context, msg core.Message, receipt *ethtypes.Receipt) error {
	return h.k.PostTxProcessing(ctx, msg, receipt)
}

// PostTxProcessing implements EvmHooks.PostTxProcessing.
func (k Keeper) PostTxProcessing(
	ctx sdk.Context,
	msg core.Message,
	receipt *ethtypes.Receipt,
) error {
	logger := ctx.Logger()
	params := k.GetParams(ctx)
	if !params.GovErc721.EnableEvm {
		// no error is returned to avoid reverting the tx and allow for other post
		// processing txs to pass and
		return nil
	}
	logger.Info("[PostTxProcessing]", "gov-module-address", types.ModuleAddress.String())
	if len(msg.Data()) == 0 {
		//this is a native token transfer msg
		return nil
	}
	contractAddr, ok := k.GetContractAddr(ctx)
	if !ok {
		logger.Error("no contract address found for evm tx event processing")
		return nil
	}
	erc721 := contracts.ERC721DelegateContract.ABI
	for _, log := range receipt.Logs {
		// Check if event is included in ERC721 delegate contract
		eventID := log.Topics[0]
		event, err := erc721.EventByID(eventID)
		if err != nil {
			continue
		}
		if !params.GovErc721.AllowDeploy {
			if log.Address.String() != contractAddr.String() {
				logger.Debug("evm contract address not equal to gov contract address", "receipt-addr", log.Address.String(), "gov-addr", contractAddr.String())
				continue
			}
		}

		switch event.Name {
		case types.ContractEventNameDeploy: //contract test only
			err = k.handleContractEventDeploy(ctx, msg, receipt, log, event, erc721, params)
		case types.ContractEventNameTransfer:
			err = k.handleContractEventTransfer(ctx, msg, receipt, log, event, erc721)
		case types.ContractEventNameCreateCandidate:
			err = k.handleContractEventCreateCandidate(ctx, msg, receipt, log, event, erc721)
		case types.ContractEventNameVoteFinish:
			err = k.handleContractEventVoteFinish(ctx, msg, receipt, log, event, erc721)
		case types.ContractEventNameVote:
			err = k.handleContractEventVote(ctx, msg, receipt, log, event, erc721)
		case types.ContractEventNameUnvote:
			err = k.handleContractEventUnvote(ctx, msg, receipt, log, event, erc721)
		case types.ContractEventNameUnbond:
			err = k.handleContractEventUnbond(ctx, msg, receipt, log, event, erc721)
		case types.ContractEventNameActiveToken:
			err = k.handleContractEventActiveToken(ctx, msg, receipt, log, event, erc721)
		default:
			logger.Info("[EvmHook] can not handle event", "name", event.Name, "receipt-addr", log.Address.String())
		}
		if err != nil {
			logger.Error("[EvmHook] handle event failed", "event", event.Name, "error", err.Error())
			return err
		}
	}
	return nil
}

func (k Keeper) handleContractEventDeploy(ctx sdk.Context, msg core.Message, receipt *ethtypes.Receipt, log *ethtypes.Log, event *abi.Event, erc721 abi.ABI, params v1.Params) error {
	logger := ctx.Logger()
	contractAddr := log.Address
	_ = contractAddr
	var st types.EventDeploy

	err := evmcommon.UnpackLog(erc721, &st, event.Name, *log)
	if err != nil {
		return fmt.Errorf("[EvmHook] contract %s event %s unpack log error: %s", contractAddr, event.Name, err.Error())
	}
	logger.Info("[EvmHook]", "contract-address", contractAddr, "event", event.Name, "unpack", st)
	return k.deployContract(ctx, st, params)
}

func (k Keeper) handleContractEventCreateCandidate(ctx sdk.Context, msg core.Message, receipt *ethtypes.Receipt, log *ethtypes.Log, event *abi.Event, erc721 abi.ABI) error {
	logger := ctx.Logger()
	contractAddr := log.Address
	_ = contractAddr
	var st types.EventCreateCandidate

	err := evmcommon.UnpackLog(erc721, &st, event.Name, *log)
	if err != nil {
		return fmt.Errorf("[EvmHook] contract %s event %s unpack log error: %s", contractAddr, event.Name, err.Error())
	}
	logger.Info("[EvmHook]", "contract-address", contractAddr, "event", event.Name, "unpack", st)
	return k.createCandidate(ctx, st)
}

func (k Keeper) handleContractEventVoteFinish(ctx sdk.Context, msg core.Message, receipt *ethtypes.Receipt, log *ethtypes.Log, event *abi.Event, erc721 abi.ABI) error {
	logger := ctx.Logger()
	contractAddr := log.Address
	_ = contractAddr
	var st types.EventVoteFinished

	err := evmcommon.UnpackLog(erc721, &st, event.Name, *log)
	if err != nil {
		return fmt.Errorf("[EvmHook] contract %s event %s unpack log error: %s", contractAddr, event.Name, err.Error())
	}
	logger.Info("[EvmHook]", "contract-address", contractAddr, "event", event.Name, "unpack", st)
	return k.voteFinish(ctx, st)
}

func (k Keeper) handleContractEventVote(ctx sdk.Context, msg core.Message, receipt *ethtypes.Receipt, log *ethtypes.Log, event *abi.Event, erc721 abi.ABI) error {
	logger := ctx.Logger()
	contractAddr := log.Address
	_ = contractAddr
	var st types.EventVote
	err := evmcommon.UnpackLog(erc721, &st, event.Name, *log)
	if err != nil {
		return fmt.Errorf("[EvmHook] contract %s event %s unpack log error: %s", contractAddr, event.Name, err.Error())
	}
	logger.Info("[EvmHook]", "contract-address", contractAddr, "event", event.Name, "unpack", st)
	return k.userVote(ctx, st)
}

func (k Keeper) handleContractEventUnvote(ctx sdk.Context, msg core.Message, receipt *ethtypes.Receipt, log *ethtypes.Log, event *abi.Event, erc721 abi.ABI) error {
	logger := ctx.Logger()
	contractAddr := log.Address
	_ = contractAddr
	var st types.EventUnvote

	err := evmcommon.UnpackLog(erc721, &st, event.Name, *log)
	if err != nil {
		return fmt.Errorf("[EvmHook] contract %s event %s unpack log error: %s", contractAddr, event.Name, err.Error())
	}
	logger.Info("[EvmHook]", "contract-address", contractAddr, "event", event.Name, "unpack", st)
	return k.userUnvote(ctx, st)
}

func (k Keeper) handleContractEventUnbond(ctx sdk.Context, msg core.Message, receipt *ethtypes.Receipt, log *ethtypes.Log, event *abi.Event, erc721 abi.ABI) error {
	logger := ctx.Logger()
	contractAddr := log.Address
	_ = contractAddr
	var st types.EventUnbond

	err := evmcommon.UnpackLog(erc721, &st, event.Name, *log)
	if err != nil {
		return fmt.Errorf("[EvmHook] contract %s event %s unpack log error: %s", contractAddr, event.Name, err.Error())
	}
	logger.Info("[EvmHook]", "contract-address", contractAddr, "event", event.Name, "unpack", st)
	return k.validatorUnbond(ctx, st)
}

func (k Keeper) handleContractEventActiveToken(ctx sdk.Context, msg core.Message, receipt *ethtypes.Receipt, log *ethtypes.Log, event *abi.Event, erc721 abi.ABI) error {
	logger := ctx.Logger()
	contractAddr := log.Address
	_ = contractAddr
	var st types.EventActiveToken

	err := evmcommon.UnpackLog(erc721, &st, event.Name, *log)
	if err != nil {
		return fmt.Errorf("[EvmHook] contract %s event %s unpack log error: %s", contractAddr, event.Name, err.Error())
	}
	logger.Info("[EvmHook]", "contract-address", contractAddr, "event", event.Name, "unpack", st)
	return k.activeToken(ctx, st)
}

func (k Keeper) handleContractEventTransfer(ctx sdk.Context, msg core.Message, receipt *ethtypes.Receipt, log *ethtypes.Log, event *abi.Event, erc721 abi.ABI) error {
	logger := ctx.Logger()
	contractAddr := log.Address
	_ = contractAddr
	var st types.EventTransfer

	err := evmcommon.UnpackLog(erc721, &st, event.Name, *log)
	if err != nil {
		logger.Info("[EvmHook] unpack log failed", "contract-address", contractAddr, "event", event.Name, "error", err.Error())
		return nil //do not return error
	}
	logger.Info("[EvmHook]", "contract-address", contractAddr, "event", event.Name, "unpack", st)
	return nil
}
