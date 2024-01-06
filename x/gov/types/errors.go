package types

import (
	errorsmod "cosmossdk.io/errors"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const (
	codespace = "evmos-gov"
)

var (
	// x/gov module sentinel errors
	ErrUnknownProposal       = sdkerrors.Register(codespace, 2, "unknown proposal")
	ErrInactiveProposal      = sdkerrors.Register(codespace, 3, "inactive proposal")
	ErrAlreadyActiveProposal = sdkerrors.Register(codespace, 4, "proposal already active")
	// Errors 5 & 6 are legacy errors related to v1beta1.Proposal.
	ErrInvalidProposalContent  = sdkerrors.Register(codespace, 5, "invalid proposal content")
	ErrInvalidProposalType     = sdkerrors.Register(codespace, 6, "invalid proposal type")
	ErrInvalidVote             = sdkerrors.Register(codespace, 7, "invalid vote option")
	ErrInvalidGenesis          = sdkerrors.Register(codespace, 8, "invalid genesis state")
	ErrNoProposalHandlerExists = sdkerrors.Register(codespace, 9, "no handler exists for proposal type")
	ErrUnroutableProposalMsg   = sdkerrors.Register(codespace, 10, "proposal message not recognized by router")
	ErrNoProposalMsgs          = sdkerrors.Register(codespace, 11, "no messages proposed")
	ErrInvalidProposalMsg      = sdkerrors.Register(codespace, 12, "invalid proposal message")
	ErrInvalidSigner           = sdkerrors.Register(codespace, 13, "expected gov account as only signer for proposal message")
	ErrInvalidSignalMsg        = sdkerrors.Register(codespace, 14, "signal message is invalid")
	ErrMetadataTooLong         = sdkerrors.Register(codespace, 15, "metadata too long")
	ErrMinDepositTooSmall      = sdkerrors.Register(codespace, 16, "minimum deposit is too small")

	// evm errors
	ErrERC721Disabled        = errorsmod.Register(codespace, 17, "erc20 module is disabled")
	ErrInternalTokenPair     = errorsmod.Register(codespace, 18, "internal ethereum token mapping error")
	ErrContractNotFound      = errorsmod.Register(codespace, 19, "contract not found")
	ErrContractAlreadyExists = errorsmod.Register(codespace, 20, "contract already exists")
	ErrUndefinedOwner        = errorsmod.Register(codespace, 21, "undefined owner of contract pair")
	ErrBalanceInvariance     = errorsmod.Register(codespace, 22, "post transfer balance invariant failed")
	ErrUnexpectedEvent       = errorsmod.Register(codespace, 23, "unexpected event")
	ErrABIPack               = errorsmod.Register(codespace, 24, "contract ABI pack failed")
	ErrABIUnpack             = errorsmod.Register(codespace, 25, "contract ABI unpack failed")
	ErrEVMDenom              = errorsmod.Register(codespace, 26, "EVM denomination registration")
	ErrEVMCall               = errorsmod.Register(codespace, 27, "EVM call unexpected error")
	ErrAccessDenied          = errorsmod.Register(codespace, 28, "access denied")
)

func init() {

}
