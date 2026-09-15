package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"context"
	"github.com/cosmos/evm/x/bandwidth/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"
)

const (
	secondsPerDay = int64(24 * 60 * 60)
	// dailyResetOffsetSeconds shifts the daily boundary so that each "day"
	// runs from 03:00 UTC to the next 03:00 UTC, instead of midnight UTC.
	dailyResetOffsetSeconds = int64(3 * 60 * 60)
)

// dayIndex returns which fixed UTC-anchored "day" a unix timestamp falls
// into, where a day is defined as [03:00 UTC, next 03:00 UTC). Two
// timestamps map to the same dayIndex iff they fall in the same daily
// window.
func dayIndex(unixTime int64) int64 {
	return (unixTime - dailyResetOffsetSeconds) / secondsPerDay
}

// EndBlocker: any wallet that (a) currently owns at least one registered
// device AND (b) was drawn to exactly zero balance by the bandwidth
// transaction fee, and who has not been topped up yet in the current fixed
// daily window (03:00 UTC boundary), gets DailyTopupAmount minted to it
// once. This is a per-wallet "never fully stuck at zero" safety net keyed
// on actual bandwidth-fee usage BY A GENUINE NETWORK PARTICIPANT.
//
// Security fix (8 Sep 2026 chain audit finding #2): the device-ownership
// requirement (b) is new. Previously ANY wallet that had been drawn to zero
// qualified, with no link to owning a registered device at all - since
// creating a new EVM wallet is free, that made the top-up trivially
// Sybil-farmable for unbounded free inflation (spin up N wallets, zero each
// one out via a cheap tx, collect N * DailyTopupAmount every single day,
// forever). Registering a device requires a real ESP32's own private-key
// signature (see RegisterDevice's proof-of-possession check, finding #1's
// fix) - a genuine, non-free cost tied to real hardware - so restricting
// eligibility to CURRENT device-owner wallets ties this safety net back to
// real participation in the network instead of free wallet creation.
func (k Keeper) EndBlocker(ctx context.Context) error {
	// Continuous per-block issuance into the pool (212B BANDWIDTH/year at
	// a fixed 5s block interval - see mint.go). Runs every block
	// unconditionally, independent of tx volume, so the pool that
	// SubmitShare draws rewards from keeps growing even during
	// zero-activity blocks.
	k.MintBlockReward(ctx)

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	now := sdkCtx.BlockTime().Unix()
	currentDay := dayIndex(now)
	params := k.GetParams(ctx)
	denom := evmtypes.GetEVMCoinDenom()

	// Built once per block, not once per pending wallet - GetAllDevices is
	// a full store scan, so this keeps EndBlocker at O(devices + pending
	// wallets) instead of O(devices * pending wallets).
	deviceOwners := make(map[string]bool)
	for _, device := range k.GetAllDevices(ctx) {
		deviceOwners[device.Owner] = true
	}

	for _, addr := range k.GetAllPendingTopupWallets(ctx) {
		balance := k.bankKeeper.GetBalance(ctx, addr, denom)
		if !balance.IsZero() {
			// wallet received funds since it was flagged - no longer needs
			// the top-up, drop it from the pending set.
			k.removePendingTopup(ctx, addr)
			continue
		}
		if !deviceOwners[addr.String()] {
			// Not (or no longer, e.g. after UnregisterDevice) the owner of
			// any registered device - never eligible for the free top-up.
			// Drop it from the pending set so it isn't re-checked every
			// block forever; it re-enters if the wallet later registers a
			// device and its balance is drawn to zero again.
			k.removePendingTopup(ctx, addr)
			continue
		}
		if k.getWalletLastTopupDay(ctx, addr) >= currentDay {
			continue // already topped up in the current daily window
		}

		coins := sdk.NewCoins(sdk.NewCoin(denom, params.DailyTopupAmount))
		if err := k.bankKeeper.MintCoins(ctx, types.ModuleName, coins); err != nil {
			sdkCtx.Logger().Error("bandwidth daily topup: mint failed", "wallet", addr.String(), "error", err)
			continue
		}
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, addr, coins); err != nil {
			sdkCtx.Logger().Error("bandwidth daily topup: send failed", "wallet", addr.String(), "error", err)
			continue
		}

		k.setWalletLastTopupDay(ctx, addr, currentDay)

		sdkCtx.EventManager().EmitEvent(
			sdk.NewEvent(
				"daily_topup",
				sdk.NewAttribute("wallet", addr.String()),
				sdk.NewAttribute("amount", params.DailyTopupAmount.String()),
				sdk.NewAttribute("mode", "EndBlock"),
			),
		)
	}
	return nil
}
