package v1

import (
	"fmt"
	"time"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Default period for deposits & voting
const (
	DefaultPeriod                  time.Duration = time.Hour * 24 * 2 // 2 days
	DefaultGenesisNftVestingEpochs               = 1126286            // about 3 months
	DefaultSettleIntervalEpochs                  = 12343              // about 1 day
)

// Default governance params
var (
	DefaultMinDepositTokens       = sdk.NewInt(10000000)
	DefaultQuorum                 = sdk.NewDecWithPrec(334, 3)
	DefaultThreshold              = sdk.NewDecWithPrec(5, 1)
	DefaultVetoThreshold          = sdk.NewDecWithPrec(334, 3)
	DefaultMinInitialDepositRatio = sdk.ZeroDec()
	DefaultBurnProposalPrevote    = false                                               // set to false to replicate behavior of when this change was made (0.47)
	DefaultBurnVoteQuorom         = false                                               // set to false to  replicate behavior of when this change was made (0.47)
	DefaultBurnVoteVeto           = false                                               // set to true to replicate behavior of when this change was made (0.47)
	DefaultMasterVestingReward    = sdk.MustNewDecFromStr("20000000000000000000000")    //vesting tokens
	DefaultSlaveVestingReward     = sdk.MustNewDecFromStr("10000000000000000000000")    //vesting token
	DefaultCommonVestingReward    = sdk.MustNewDecFromStr("4973000000000000000000")     //vesting token
	DefaultMintQuota              = sdk.MustNewDecFromStr("50000000000000000000000000") //mint quota
)

// Deprecated: NewDepositParams creates a new DepositParams object
func NewDepositParams(minDeposit sdk.Coins, maxDepositPeriod *time.Duration) DepositParams {
	return DepositParams{
		MinDeposit:       minDeposit,
		MaxDepositPeriod: maxDepositPeriod,
	}
}

// Deprecated: NewTallyParams creates a new TallyParams object
func NewTallyParams(quorum, threshold, vetoThreshold string) TallyParams {
	return TallyParams{
		Quorum:        quorum,
		Threshold:     threshold,
		VetoThreshold: vetoThreshold,
	}
}

// Deprecated: NewVotingParams creates a new VotingParams object
func NewVotingParams(votingPeriod *time.Duration) VotingParams {
	return VotingParams{
		VotingPeriod: votingPeriod,
	}
}

// NewParams creates a new Params instance with given values.
func NewParams(
	minDeposit sdk.Coins, maxDepositPeriod, votingPeriod time.Duration,
	quorum, threshold, vetoThreshold, minInitialDepositRatio string, burnProposalDeposit, burnVoteQuorum, burnVoteVeto bool,
) Params {
	return Params{
		MinDeposit:                 minDeposit,
		MaxDepositPeriod:           &maxDepositPeriod,
		VotingPeriod:               &votingPeriod,
		Quorum:                     quorum,
		Threshold:                  threshold,
		VetoThreshold:              vetoThreshold,
		MinInitialDepositRatio:     minInitialDepositRatio,
		BurnProposalDepositPrevote: burnProposalDeposit,
		BurnVoteQuorum:             burnVoteQuorum,
		BurnVoteVeto:               burnVoteVeto,
		GovErc721: GovErc721{
			EnableEvm:            true,
			Denom:                "uhby",
			MintQuota:            &DefaultMintQuota,
			MasterVestingReward:  &DefaultMasterVestingReward,
			MasterVestingEpochs:  DefaultGenesisNftVestingEpochs, // default 3month
			SlaveVestingReward:   &DefaultSlaveVestingReward,
			SlaveVestingEpochs:   DefaultGenesisNftVestingEpochs, // default 3month
			CommonVestingReward:  &DefaultCommonVestingReward,
			CommonVestingEpochs:  DefaultGenesisNftVestingEpochs, // default 3month
			SettleIntervalEpochs: DefaultSettleIntervalEpochs,    // settle interval epochs (default 1day)
		},
	}
}

// DefaultParams returns the default governance params
func DefaultParams() Params {
	return NewParams(
		sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, DefaultMinDepositTokens)),
		DefaultPeriod,
		DefaultPeriod,
		DefaultQuorum.String(),
		DefaultThreshold.String(),
		DefaultVetoThreshold.String(),
		DefaultMinInitialDepositRatio.String(),
		DefaultBurnProposalPrevote,
		DefaultBurnVoteQuorom,
		DefaultBurnVoteVeto,
	)
}

// ValidateBasic performs basic validation on governance parameters.
func (p Params) ValidateBasic() error {
	if minDeposit := sdk.Coins(p.MinDeposit); minDeposit.Empty() || !minDeposit.IsValid() {
		return fmt.Errorf("invalid minimum deposit: %s", minDeposit)
	}

	if p.MaxDepositPeriod == nil {
		return fmt.Errorf("maximum deposit period must not be nil: %d", p.MaxDepositPeriod)
	}

	if p.MaxDepositPeriod.Seconds() <= 0 {
		return fmt.Errorf("maximum deposit period must be positive: %d", p.MaxDepositPeriod)
	}

	quorum, err := sdk.NewDecFromStr(p.Quorum)
	if err != nil {
		return fmt.Errorf("invalid quorum string: %w", err)
	}
	if quorum.IsNegative() {
		return fmt.Errorf("quorom cannot be negative: %s", quorum)
	}
	if quorum.GT(math.LegacyOneDec()) {
		return fmt.Errorf("quorom too large: %s", p.Quorum)
	}

	threshold, err := sdk.NewDecFromStr(p.Threshold)
	if err != nil {
		return fmt.Errorf("invalid threshold string: %w", err)
	}
	if !threshold.IsPositive() {
		return fmt.Errorf("vote threshold must be positive: %s", threshold)
	}
	if threshold.GT(math.LegacyOneDec()) {
		return fmt.Errorf("vote threshold too large: %s", threshold)
	}

	vetoThreshold, err := sdk.NewDecFromStr(p.VetoThreshold)
	if err != nil {
		return fmt.Errorf("invalid vetoThreshold string: %w", err)
	}
	if !vetoThreshold.IsPositive() {
		return fmt.Errorf("veto threshold must be positive: %s", vetoThreshold)
	}
	if vetoThreshold.GT(math.LegacyOneDec()) {
		return fmt.Errorf("veto threshold too large: %s", vetoThreshold)
	}

	if p.VotingPeriod == nil {
		return fmt.Errorf("voting period must not be nil: %d", p.VotingPeriod)
	}

	if p.VotingPeriod.Seconds() <= 0 {
		return fmt.Errorf("voting period must be positive: %s", p.VotingPeriod)
	}
	if p.GovErc721.EnableEvm {
		if p.GovErc721.MintQuota == nil {
			return fmt.Errorf("mint quota must not be nil")
		}
	}
	return nil
}
