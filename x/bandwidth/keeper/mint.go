package keeper

import (
	"context"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/evm/x/bandwidth/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"
)

// AnnualMintCap is the target annual issuance: 212,000,000,000 BANDWIDTH,
// expressed in atest (1e18 base units) to match the 18-decimal EVM coin
// convention used everywhere else in this module (see BandwidthCostPerTx
// in ante/bandwidth.go and DefaultDailyTopupAmount in types/params.go).
var AnnualMintCap = sdkmath.NewInt(212_000_000_000).MulRaw(1_000_000_000_000_000_000)

// BlocksPerYear assumes a fixed 5-second block interval over a 365.25-day
// year (365.25*24*3600/5 = 6,311,520). Must stay in sync with this chain's
// actual timeout_commit (uax_fresh_setup.sh) and with the standard x/mint
// module's own blocks_per_year genesis param - all switched back to 5s
// together (9 Sep 2026).
const BlocksPerYear = int64(6_311_520)

// MintPerBlock is the fixed amount minted into the bandwidth module's pool
// account every single block, regardless of chain activity - this is what
// eliminates the "empty block, no bandwidth generated" problem: the pool
// now grows continuously purely from block production, independent of
// SubmitShare/fee volume.
//
// floor(AnnualMintCap / BlocksPerYear); the ~0.37 BANDWIDTH/block
// remainder this drops (~2,331 BANDWIDTH/year, an ~0.0000011% difference
// from the nominal annual figure) is intentionally left untracked - at
// atest-level precision it has no economic significance, and tracking a
// remainder accumulator would add state and complexity for no real gain.
var MintPerBlock = AnnualMintCap.QuoRaw(BlocksPerYear)

// MintBlockReward mints MintPerBlock atest into the bandwidth module's own
// pool account every block. This is pure block-linked issuance - it does
// NOT pay any device or user directly; it only grows the same shared pool
// that SubmitShare (msg_server.go) already draws down first before minting
// any share-reward shortfall. Net effect: as this pool fills continuously,
// a growing share of device rewards gets paid from real accumulated supply
// instead of fresh per-share minting - but WHO gets paid, and how much per
// share, is unchanged: that still requires an actual valid, nonce-ordered,
// signed SubmitShare from a registered device. This function only funds
// the pool; it never distributes it.
//
// Errors are logged, not returned - a transient mint failure here should
// not halt block production (same non-halting posture as the daily top-up
// loop below), since this pool top-up is a background subsidy, not a
// user-facing state transition that needs atomicity with anything else in
// the block.
func (k Keeper) MintBlockReward(ctx context.Context) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	denom := evmtypes.GetEVMCoinDenom()
	coins := sdk.NewCoins(sdk.NewCoin(denom, MintPerBlock))

	if err := k.bankKeeper.MintCoins(ctx, types.ModuleName, coins); err != nil {
		sdkCtx.Logger().Error("bandwidth block mint failed", "amount", MintPerBlock.String(), "error", err)
		return
	}

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"bandwidth_block_mint",
			sdk.NewAttribute("amount", MintPerBlock.String()),
		),
	)
}
