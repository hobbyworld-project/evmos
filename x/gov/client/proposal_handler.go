package client

import (
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	paramscli "github.com/cosmos/cosmos-sdk/x/params/client/cli"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramsproposal "github.com/cosmos/cosmos-sdk/x/params/types/proposal"
	upgradecli "github.com/cosmos/cosmos-sdk/x/upgrade/client/cli"
	upgradekeeper "github.com/cosmos/cosmos-sdk/x/upgrade/keeper"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	ibccli "github.com/cosmos/ibc-go/v7/modules/core/02-client/client/cli"
	corekeeper "github.com/cosmos/ibc-go/v7/modules/core/02-client/keeper"
	coretypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	govtypes "github.com/evmos/evmos/v15/x/gov/types/v1beta1"
	"github.com/spf13/cobra"
)

// function to create the cli handler
type CLIHandlerFn func() *cobra.Command

// ProposalHandler wraps CLIHandlerFn
type ProposalHandler struct {
	CLIHandler CLIHandlerFn
}

// NewProposalHandler creates a new ProposalHandler object
func NewProposalHandler(cliHandler CLIHandlerFn) ProposalHandler {
	return ProposalHandler{
		CLIHandler: cliHandler,
	}
}

// ParamsProposalHandler is the param change proposal handler.
var ParamsProposalHandler = NewProposalHandler(paramscli.NewSubmitParamChangeProposalTxCmd)
var UpgradeLegacyProposalHandler = NewProposalHandler(upgradecli.NewCmdSubmitLegacyUpgradeProposal)
var UpgradeLegacyCancelProposalHandler = NewProposalHandler(upgradecli.NewCmdSubmitLegacyCancelUpgradeProposal)
var UpdateClientProposalHandler = NewProposalHandler(ibccli.NewCmdSubmitUpdateClientProposal)
var UpgradeProposalHandler = NewProposalHandler(ibccli.NewCmdSubmitUpgradeProposal)

// NewParamChangeProposalHandler creates a new governance Handler for a ParamChangeProposal
func NewParamChangeProposalHandler(k paramskeeper.Keeper) govtypes.Handler {
	return func(ctx sdk.Context, content govtypes.Content) error {
		switch c := content.(type) {
		case *paramsproposal.ParameterChangeProposal:
			return handleParameterChangeProposal(ctx, k, c)

		default:
			return sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "unrecognized param proposal content type: %T", c)
		}
	}
}

func handleParameterChangeProposal(ctx sdk.Context, k paramskeeper.Keeper, p *paramsproposal.ParameterChangeProposal) error {
	for _, c := range p.Changes {
		ss, ok := k.GetSubspace(c.Subspace)
		if !ok {
			return sdkerrors.Wrap(paramsproposal.ErrUnknownSubspace, c.Subspace)
		}

		k.Logger(ctx).Info(
			fmt.Sprintf("attempt to set new parameter value; key: %s, value: %s", c.Key, c.Value),
		)

		if err := ss.Update(ctx, []byte(c.Key), []byte(c.Value)); err != nil {
			return sdkerrors.Wrapf(paramsproposal.ErrSettingParameter, "key: %s, value: %s, err: %s", c.Key, c.Value, err.Error())
		}
	}
	return nil
}

// NewSoftwareUpgradeProposalHandler creates a governance handler to manage new proposal types.
// It enables SoftwareUpgradeProposal to propose an Upgrade, and CancelSoftwareUpgradeProposal
// to abort a previously voted upgrade.
//
//nolint:staticcheck // we are intentionally using a deprecated proposal here.
func NewSoftwareUpgradeProposalHandler(k *upgradekeeper.Keeper) govtypes.Handler {
	return func(ctx sdk.Context, content govtypes.Content) error {
		switch c := content.(type) {
		case *upgradetypes.SoftwareUpgradeProposal:
			return handleSoftwareUpgradeProposal(ctx, k, c)

		case *upgradetypes.CancelSoftwareUpgradeProposal:
			return handleCancelSoftwareUpgradeProposal(ctx, k, c)

		default:
			return sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "unrecognized software upgrade proposal content type: %T", c)
		}
	}
}

//nolint:staticcheck // we are intentionally using a deprecated proposal here.
func handleSoftwareUpgradeProposal(ctx sdk.Context, k *upgradekeeper.Keeper, p *upgradetypes.SoftwareUpgradeProposal) error {
	return k.ScheduleUpgrade(ctx, p.Plan)
}

//nolint:staticcheck // we are intentionally using a deprecated proposal here.
func handleCancelSoftwareUpgradeProposal(ctx sdk.Context, k *upgradekeeper.Keeper, _ *upgradetypes.CancelSoftwareUpgradeProposal) error {
	k.ClearUpgradePlan(ctx)
	return nil
}

// NewClientProposalHandler defines the 02-client proposal handler
func NewClientProposalHandler(k corekeeper.Keeper) govtypes.Handler {
	return func(ctx sdk.Context, content govtypes.Content) error {
		switch c := content.(type) {
		case *coretypes.ClientUpdateProposal:
			return k.ClientUpdateProposal(ctx, c)
		case *coretypes.UpgradeProposal:
			return k.HandleUpgradeProposal(ctx, c)

		default:
			return sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "unrecognized ibc proposal content type: %T", c)
		}
	}
}
