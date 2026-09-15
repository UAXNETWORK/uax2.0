package keeper

import (
	"context"
	"encoding/hex"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/cosmos/evm/x/bandwidth/types"
)

const PubKeyLength = 33 // compressed secp256k1 public key

type msgServer struct {
	Keeper
}

func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

func (k msgServer) RegisterDevice(goCtx context.Context, msg *types.MsgRegisterDevice) (*types.MsgRegisterDeviceResponse, error) {
	if k.HasDevice(goCtx, msg.DeviceId) {
		return nil, types.ErrDeviceAlreadyExists.Wrapf("device_id %s already registered", msg.DeviceId)
	}
	if len(msg.PublicKey) != PubKeyLength {
		return nil, types.ErrInvalidPublicKey.Wrapf("expected %d-byte compressed secp256k1 public key, got %d bytes", PubKeyLength, len(msg.PublicKey))
	}
	ownerAddr, err := sdk.AccAddressFromBech32(msg.Owner)
	if err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidAddress, "invalid owner address")
	}

	// Proof-of-possession check (security fix, 8 Sep 2026 chain audit
	// finding #1): device_id + public_key are both printed in plaintext on
	// the device's own setup page and boot log, so without this check
	// anyone who merely observed them could front-run registration and
	// hijack the device's future rewards. The device itself must sign
	// "register:{device_id}:{owner}" (owner as "0x" + lowercase hex - the
	// format the ESP32 setup page produces, since it can't do bech32) with
	// its own private key; we verify that signature here against the
	// public_key being registered. Combined with the precompile's existing
	// msgSender==owner check, only the wallet that both (a) controls the
	// owner address AND (b) possesses a signature from the device's own
	// private key can complete registration.
	ownerHex := "0x" + hex.EncodeToString(ownerAddr.Bytes())
	registerSignBytes := []byte(fmt.Sprintf("register:%s:%s", msg.DeviceId, ownerHex))
	registerPubKey := &secp256k1.PubKey{Key: msg.PublicKey}
	if !registerPubKey.VerifySignature(registerSignBytes, msg.DeviceSignature) {
		return nil, types.ErrInvalidSignature.Wrap(
			"device_signature does not match device_id+owner - get a fresh signature from the device's own " +
				"setup page for this exact owner address before registering",
		)
	}

	device := types.Device{
		DeviceId:      msg.DeviceId,
		Owner:         msg.Owner,
		PublicKey:     msg.PublicKey,
		LastNonce:     0,
		LastTopupTime: 0,
	}
	k.SetDevice(goCtx, device)

	sdkCtx := sdk.UnwrapSDKContext(goCtx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"device_registered",
			sdk.NewAttribute("device_id", msg.DeviceId),
			sdk.NewAttribute("owner", msg.Owner),
		),
	)

	return &types.MsgRegisterDeviceResponse{}, nil
}

func (k msgServer) SubmitShare(goCtx context.Context, msg *types.MsgSubmitShare) (*types.MsgSubmitShareResponse, error) {
	device, found := k.GetDevice(goCtx, msg.DeviceId)
	if !found {
		return nil, types.ErrDeviceNotFound.Wrapf("device_id %s is not registered", msg.DeviceId)
	}

	if msg.Nonce <= device.LastNonce {
		return nil, types.ErrInvalidNonce.Wrapf("nonce %d must be greater than last accepted nonce %d", msg.Nonce, device.LastNonce)
	}
	if msg.ShareCount == 0 {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "share_count must be greater than zero")
	}

	// Signed payload: deviceID:shareCount:nonce -- raw bytes, NOT pre-hashed
	// (secp256k1.PubKey.VerifySignature SHA256-hashes internally).
	signBytes := []byte(fmt.Sprintf("%s:%d:%d", msg.DeviceId, msg.ShareCount, msg.Nonce))
	pubKey := &secp256k1.PubKey{Key: device.PublicKey}
	if !pubKey.VerifySignature(signBytes, msg.Signature) {
		return nil, types.ErrInvalidSignature.Wrap("device signature does not match share payload")
	}

	params := k.GetParams(goCtx)
	rewardAmount := params.BandwidthPerShare.MulRaw(int64(msg.ShareCount))

	denom := evmtypes.GetEVMCoinDenom()

	// --- Real bandwidth generation -----------------------------------
	// The bandwidth fee ante handler (ante/bandwidth.go) charges every
	// non-exempt transaction a flat fee and deposits it into this
	// module's own account - that pool of coins is REAL, already-existing
	// supply collected from genuine chain activity (it used to be burned
	// immediately; it no longer is, precisely so it can fund this).
	//
	// Pay the share reward from that real pool first. Only mint fresh
	// coins for whatever the pool can't cover, so as real transaction
	// volume (and therefore real fee revenue) grows, this reward becomes
	// increasingly - and eventually fully - backed by genuine economic
	// activity instead of unconditional inflation. A device is never
	// blocked/rejected for a valid share just because the pool is thin;
	// the shortfall (if any) is always minted so liveness proof always
	// pays out, but the split is fully recorded on-chain below for
	// transparency/audit.
	moduleAddr := authtypes.NewModuleAddress(types.ModuleName)
	poolBalance := k.bankKeeper.GetBalance(goCtx, moduleAddr, denom).Amount

	fromPool := rewardAmount
	if poolBalance.LT(rewardAmount) {
		fromPool = poolBalance
	}
	shortfall := rewardAmount.Sub(fromPool)

	if shortfall.IsPositive() {
		shortfallCoins := sdk.NewCoins(sdk.NewCoin(denom, shortfall))
		if err := k.bankKeeper.MintCoins(goCtx, types.ModuleName, shortfallCoins); err != nil {
			return nil, errorsmod.Wrap(err, "failed to mint bandwidth reward shortfall")
		}
	}

	ownerAddr, err := sdk.AccAddressFromBech32(device.Owner)
	if err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidAddress, "invalid device owner address")
	}
	payoutCoins := sdk.NewCoins(sdk.NewCoin(denom, rewardAmount))
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(goCtx, types.ModuleName, ownerAddr, payoutCoins); err != nil {
		return nil, errorsmod.Wrap(err, "failed to pay out bandwidth reward")
	}

	device.LastNonce = msg.Nonce
	k.SetDevice(goCtx, device)

	sdkCtx := sdk.UnwrapSDKContext(goCtx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"share_submitted",
			sdk.NewAttribute("device_id", msg.DeviceId),
			sdk.NewAttribute("owner", device.Owner),
			sdk.NewAttribute("share_count", fmt.Sprintf("%d", msg.ShareCount)),
			sdk.NewAttribute("nonce", fmt.Sprintf("%d", msg.Nonce)),
			sdk.NewAttribute("minted_amount", rewardAmount.String()),
			sdk.NewAttribute("from_fee_pool_amount", fromPool.String()),
			sdk.NewAttribute("newly_minted_amount", shortfall.String()),
		),
	)

	return &types.MsgSubmitShareResponse{MintedAmount: rewardAmount.String()}, nil
}

func (k msgServer) UnregisterDevice(goCtx context.Context, msg *types.MsgUnregisterDevice) (*types.MsgUnregisterDeviceResponse, error) {
	if _, err := sdk.AccAddressFromBech32(msg.Owner); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidAddress, "invalid owner address")
	}

	device, found := k.GetDevice(goCtx, msg.DeviceId)
	if !found {
		return nil, types.ErrDeviceNotFound.Wrapf("device_id %s is not registered", msg.DeviceId)
	}

	// Only the device's currently-registered owner may unregister it - this
	// is the same signer check pattern used implicitly by RegisterDevice
	// (cosmos.msg.v1.signer = "owner"), made explicit here since unregistering
	// someone else's device would otherwise just silently no-op their device.
	if msg.Owner != device.Owner {
		return nil, types.ErrNotDeviceOwner.Wrapf("device_id %s is owned by %s, not %s", msg.DeviceId, device.Owner, msg.Owner)
	}

	// This only removes the on-chain registration record. It does NOT:
	//   - claw back any bandwidth already minted to the owner (irreversible,
	//     same as every other minted reward in this module)
	//   - affect the physical device in any way - it keeps running, and can
	//     be registered again later (e.g. under a new owner) with a fresh
	//     RegisterDevice transaction
	// Until re-registered, SubmitShare for this device_id fails with
	// ErrDeviceNotFound, so no further rewards can be minted for it.
	k.DeleteDevice(goCtx, msg.DeviceId)

	sdkCtx := sdk.UnwrapSDKContext(goCtx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"device_unregistered",
			sdk.NewAttribute("device_id", msg.DeviceId),
			sdk.NewAttribute("owner", msg.Owner),
		),
	)

	return &types.MsgUnregisterDeviceResponse{}, nil
}
