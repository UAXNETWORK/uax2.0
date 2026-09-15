package keeper

import (
	"context"
	"encoding/binary"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/evm/x/bandwidth/types"
)

// MarkPendingTopup flags a wallet as having had its balance drawn to zero by
// the bandwidth transaction fee, making it eligible for the once-daily flat
// top-up (see EndBlocker) once it hasn't been topped up yet in the current
// UTC day.
func (k Keeper) MarkPendingTopup(ctx context.Context, addr sdk.AccAddress) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(k.storeKey)
	store.Set(types.PendingTopupKey(addr.Bytes()), []byte{1})
}

func (k Keeper) removePendingTopup(ctx context.Context, addr sdk.AccAddress) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(k.storeKey)
	store.Delete(types.PendingTopupKey(addr.Bytes()))
}

// GetAllPendingTopupWallets returns every wallet currently flagged as
// pending a top-up.
func (k Keeper) GetAllPendingTopupWallets(ctx context.Context) []sdk.AccAddress {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(k.storeKey)
	iterator := store.Iterator(types.PendingTopupKeyPrefix, prefixEndBytes(types.PendingTopupKeyPrefix))
	defer iterator.Close()

	var wallets []sdk.AccAddress
	for ; iterator.Valid(); iterator.Next() {
		addrBytes := iterator.Key()[len(types.PendingTopupKeyPrefix):]
		wallets = append(wallets, sdk.AccAddress(addrBytes))
	}
	return wallets
}

func (k Keeper) getWalletLastTopupDay(ctx context.Context, addr sdk.AccAddress) int64 {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(k.storeKey)
	bz := store.Get(types.WalletLastTopupDayKey(addr.Bytes()))
	if bz == nil {
		return -1
	}
	return int64(binary.BigEndian.Uint64(bz))
}

func (k Keeper) setWalletLastTopupDay(ctx context.Context, addr sdk.AccAddress, day int64) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(k.storeKey)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, uint64(day))
	store.Set(types.WalletLastTopupDayKey(addr.Bytes()), bz)
}
