package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/evm/x/bandwidth/types"
)

func (k Keeper) GetDevice(ctx context.Context, deviceID string) (types.Device, bool) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(k.storeKey)
	bz := store.Get(types.DeviceKey(deviceID))
	if bz == nil {
		return types.Device{}, false
	}
	var device types.Device
	k.cdc.MustUnmarshal(bz, &device)
	return device, true
}

func (k Keeper) SetDevice(ctx context.Context, device types.Device) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(&device)
	store.Set(types.DeviceKey(device.DeviceId), bz)
}

func (k Keeper) HasDevice(ctx context.Context, deviceID string) bool {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(k.storeKey)
	return store.Has(types.DeviceKey(deviceID))
}

// DeleteDevice removes a device's on-chain registration record. It does not
// touch any coins already minted to the owner - only future SubmitShare
// calls for this device_id are affected (they will fail with
// ErrDeviceNotFound until the device is registered again).
func (k Keeper) DeleteDevice(ctx context.Context, deviceID string) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(k.storeKey)
	store.Delete(types.DeviceKey(deviceID))
}

func (k Keeper) GetAllDevices(ctx context.Context) []types.Device {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(k.storeKey)
	iterator := store.Iterator(types.DeviceKeyPrefix, prefixEndBytes(types.DeviceKeyPrefix))
	defer iterator.Close()

	var devices []types.Device
	for ; iterator.Valid(); iterator.Next() {
		var device types.Device
		k.cdc.MustUnmarshal(iterator.Value(), &device)
		devices = append(devices, device)
	}
	return devices
}

// prefixEndBytes returns the smallest key strictly greater than every key
// with the given prefix, i.e. the exclusive upper bound for a prefix scan.
// (Standard algorithm, inlined here so this file has no dependency on
// whichever store-types package happens to export it in this SDK version.)
func prefixEndBytes(prefix []byte) []byte {
	if len(prefix) == 0 {
		return nil
	}
	end := make([]byte, len(prefix))
	copy(end, prefix)
	for {
		if end[len(end)-1] != byte(255) {
			end[len(end)-1]++
			break
		}
		end = end[:len(end)-1]
		if len(end) == 0 {
			end = nil
			break
		}
	}
	return end
}
