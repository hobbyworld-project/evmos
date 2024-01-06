package v1

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
	authzcodec "github.com/cosmos/cosmos-sdk/x/authz/codec"
	groupcodec "github.com/cosmos/cosmos-sdk/x/group/codec"
	govcodec "github.com/evmos/evmos/v15/x/gov/codec"
)

// RegisterLegacyAminoCodec registers all the necessary types and interfaces for the
// governance module.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	legacy.RegisterAminoMsg(cdc, &MsgSubmitProposal{}, "evmos/v1/MsgSubmitProposal")
	legacy.RegisterAminoMsg(cdc, &MsgDeposit{}, "evmos/v1/MsgDeposit")
	legacy.RegisterAminoMsg(cdc, &MsgVote{}, "evmos/v1/MsgVote")
	legacy.RegisterAminoMsg(cdc, &MsgVoteWeighted{}, "evmos/v1/MsgVoteWeighted")
	legacy.RegisterAminoMsg(cdc, &MsgExecLegacyContent{}, "evmos/v1/MsgExecLegacyContent")
	legacy.RegisterAminoMsg(cdc, &MsgUpdateParams{}, "evmos/x/gov/v1/MsgUpdateParams")
}

// RegisterInterfaces registers the interfaces types with the Interface Registry.
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgSubmitProposal{},
		&MsgVote{},
		&MsgVoteWeighted{},
		&MsgDeposit{},
		&MsgExecLegacyContent{},
		&MsgUpdateParams{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

func init() {
	// Register all Amino interfaces and concrete types on the authz and gov Amino codec so that this can later be
	// used to properly serialize MsgGrant, MsgExec and MsgSubmitProposal instances
	RegisterLegacyAminoCodec(authzcodec.Amino)
	RegisterLegacyAminoCodec(govcodec.Amino)
	RegisterLegacyAminoCodec(groupcodec.Amino)
}
