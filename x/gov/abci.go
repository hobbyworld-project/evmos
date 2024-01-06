package gov

import (
	"fmt"
	abci "github.com/cometbft/cometbft/abci/types"
	"time"

	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/evmos/evmos/v15/x/gov/keeper"
	"github.com/evmos/evmos/v15/x/gov/types"
	v1 "github.com/evmos/evmos/v15/x/gov/types/v1"
)

func BeginBlock(ctx sdk.Context, req abci.RequestBeginBlock, keeper *keeper.Keeper) {
	var validators []abci.Validator
	votes := req.LastCommitInfo.Votes
	for _, v := range votes {
		validators = append(validators, v.Validator)
	}
	if ctx.BlockHeight() > 1 {
		keeper.SettleVoterReward(ctx, validators)
		keeper.CheckValidatorVotes(ctx, validators)
	}
}

// EndBlocker called every block, process inflation, update validator set.
func EndBlocker(ctx sdk.Context, keeper *keeper.Keeper) {
	defer telemetry.ModuleMeasureSince(types.ModuleName, time.Now(), telemetry.MetricKeyEndBlocker)

	logger := keeper.Logger(ctx)
	params := keeper.GetParams(ctx)

	if params.GovErc721.EnableEvm {
		if ctx.BlockHeight() == 1 {
			// deploy erc721 contract
			contractAddr, err := keeper.DeployGovContract(ctx) //keeper.DeployERC20Contract(ctx)
			if err != nil {
				panic(err.Error())
			}
			// save erc721 contract address to kv store
			keeper.SetContractAddr(ctx, contractAddr)
			logger.Info("kv store: save erc721 contract successful", "address", contractAddr.String())
		}
		keeper.SettleNftVesting(ctx)
		return // using ERC721 contract to govern
	}

	// delete dead proposals from store and returns theirs deposits.
	// A proposal is dead when it's inactive and didn't get enough deposit on time to get into voting phase.
	keeper.IterateInactiveProposalsQueue(ctx, ctx.BlockHeader().Time, func(proposal v1.Proposal) bool {
		keeper.DeleteProposal(ctx, proposal.Id)

		if !params.BurnProposalDepositPrevote {
			keeper.RefundAndDeleteDeposits(ctx, proposal.Id) // refund deposit if proposal got removed without getting 100% of the proposal
		} else {
			keeper.DeleteAndBurnDeposits(ctx, proposal.Id) // burn the deposit if proposal got removed without getting 100% of the proposal
		}

		// called when proposal become inactive
		keeper.Hooks().AfterProposalFailedMinDeposit(ctx, proposal.Id)

		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				types.EventTypeInactiveProposal,
				sdk.NewAttribute(types.AttributeKeyProposalID, fmt.Sprintf("%d", proposal.Id)),
				sdk.NewAttribute(types.AttributeKeyProposalResult, types.AttributeValueProposalDropped),
			),
		)

		logger.Info(
			"proposal did not meet minimum deposit; deleted",
			"proposal", proposal.Id,
			"min_deposit", sdk.NewCoins(params.MinDeposit...).String(),
			"total_deposit", sdk.NewCoins(proposal.TotalDeposit...).String(),
		)

		return false
	})

	// fetch active proposals whose voting periods have ended (are passed the block time)
	keeper.IterateActiveProposalsQueue(ctx, ctx.BlockHeader().Time, func(proposal v1.Proposal) bool {
		var tagValue, logMsg string

		passes, burnDeposits, tallyResults := keeper.Tally(ctx, proposal)

		if burnDeposits {
			keeper.DeleteAndBurnDeposits(ctx, proposal.Id)
		} else {
			keeper.RefundAndDeleteDeposits(ctx, proposal.Id)
		}

		if passes {
			var (
				idx    int
				events sdk.Events
				msg    sdk.Msg
			)

			// attempt to execute all messages within the passed proposal
			// Messages may mutate state thus we use a cached context. If one of
			// the handlers fails, no state mutation is written and the error
			// message is logged.
			cacheCtx, writeCache := ctx.CacheContext()
			messages, err := proposal.GetMsgs()
			if err == nil {
				for idx, msg = range messages {
					handler := keeper.Router().Handler(msg)

					var res *sdk.Result
					res, err = handler(cacheCtx, msg)
					if err != nil {
						break
					}

					events = append(events, res.GetEvents()...)
				}
			}

			// `err == nil` when all handlers passed.
			// Or else, `idx` and `err` are populated with the msg index and error.
			if err == nil {
				proposal.Status = v1.StatusPassed
				tagValue = types.AttributeValueProposalPassed
				logMsg = "passed"

				// write state to the underlying multi-store
				writeCache()

				// propagate the msg events to the current context
				ctx.EventManager().EmitEvents(events)
			} else {
				proposal.Status = v1.StatusFailed
				tagValue = types.AttributeValueProposalFailed
				logMsg = fmt.Sprintf("passed, but msg %d (%s) failed on execution: %s", idx, sdk.MsgTypeURL(msg), err)
			}
		} else {
			proposal.Status = v1.StatusRejected
			tagValue = types.AttributeValueProposalRejected
			logMsg = "rejected"
		}

		proposal.FinalTallyResult = &tallyResults

		keeper.SetProposal(ctx, proposal)
		keeper.RemoveFromActiveProposalQueue(ctx, proposal.Id, *proposal.VotingEndTime)

		// when proposal become active
		keeper.Hooks().AfterProposalVotingPeriodEnded(ctx, proposal.Id)

		logger.Info(
			"proposal tallied",
			"proposal", proposal.Id,
			"results", logMsg,
		)

		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				types.EventTypeActiveProposal,
				sdk.NewAttribute(types.AttributeKeyProposalID, fmt.Sprintf("%d", proposal.Id)),
				sdk.NewAttribute(types.AttributeKeyProposalResult, tagValue),
			),
		)
		return false
	})
}
