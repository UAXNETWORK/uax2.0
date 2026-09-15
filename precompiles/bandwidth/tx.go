package bandwidth

import (
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	cmn "github.com/cosmos/evm/precompiles/common"
	bandwidthtypes "github.com/cosmos/evm/x/bandwidth/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// SubmitShareMethod defines the ABI method name for the bandwidth SubmitShare transaction.
	SubmitShareMethod = "submitShare"
	// RegisterDeviceMethod defines the ABI method name for the bandwidth RegisterDevice transaction.
	RegisterDeviceMethod = "registerDevice"
	// UnregisterDeviceMethod defines the ABI method name for the bandwidth UnregisterDevice transaction.
	UnregisterDeviceMethod = "unregisterDevice"
)

// SubmitShare relays a device-signed share submission on-chain. The caller
// (msg.sender) must equal the relayer address passed as the first argument,
// so the bandwidth-fee AnteHandler's relayer exemption lines up with
// whoever actually authorizes this call.
func (p *Precompile) SubmitShare(
	ctx sdk.Context,
	contract *vm.Contract,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	if len(args) != 5 {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 5, len(args))
	}

	relayer, ok := args[0].(common.Address)
	if !ok || relayer == (common.Address{}) {
		return nil, fmt.Errorf("invalid relayer address: %v", args[0])
	}

	msgSender := contract.Caller()
	if msgSender != relayer {
		return nil, fmt.Errorf(cmn.ErrRequesterIsNotMsgSender, msgSender.String(), relayer.String())
	}

	deviceID, ok := args[1].(string)
	if !ok {
		return nil, fmt.Errorf("invalid deviceId: %v", args[1])
	}
	shareCount, ok := args[2].(uint32)
	if !ok {
		return nil, fmt.Errorf("invalid shareCount: %v", args[2])
	}
	nonce, ok := args[3].(uint64)
	if !ok {
		return nil, fmt.Errorf("invalid nonce: %v", args[3])
	}
	signature, ok := args[4].([]byte)
	if !ok {
		return nil, fmt.Errorf("invalid signature: %v", args[4])
	}

	relayerAddr, err := p.addrCdc.BytesToString(relayer.Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to decode relayer address: %w", err)
	}

	msg := &bandwidthtypes.MsgSubmitShare{
		Relayer:    relayerAddr,
		DeviceId:   deviceID,
		ShareCount: shareCount,
		Nonce:      nonce,
		Signature:  signature,
	}

	res, err := p.bandwidthMsgServer.SubmitShare(ctx, msg)
	if err != nil {
		return nil, err
	}

	return method.Outputs.Pack(res.MintedAmount)
}

// RegisterDevice registers a device's public key on-chain. The caller
// (msg.sender) must equal the owner address passed as the first argument.
// deviceSignature is a proof-of-possession check (security fix, 8 Sep 2026
// chain audit finding #1) - see MsgRegisterDevice's doc comment and
// x/bandwidth/keeper/msg_server.go's RegisterDevice for the verified
// payload format; without it, anyone who merely observed deviceId +
// publicKey (both printed in plaintext on the device's own setup page)
// could front-run registration and hijack the device's future rewards.
func (p *Precompile) RegisterDevice(
	ctx sdk.Context,
	contract *vm.Contract,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	if len(args) != 4 {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 4, len(args))
	}

	owner, ok := args[0].(common.Address)
	if !ok || owner == (common.Address{}) {
		return nil, fmt.Errorf("invalid owner address: %v", args[0])
	}

	// uax: msg.sender is deliberately NOT required to equal owner here
	// (unlike UnregisterDevice below, which still enforces it).
	// deviceSignature (verified below, inside the keeper's RegisterDevice
	// handler) is the real proof-of-possession: it's a signature made by
	// the DEVICE's own key over "register:{deviceId}:{owner}", which only
	// someone holding the physical device could produce. That already
	// authenticates the (deviceId, owner) binding regardless of which
	// wallet actually broadcasts the transaction, so a trusted relayer
	// can submit this on behalf of the real owner using its own gas.

	deviceID, ok := args[1].(string)
	if !ok {
		return nil, fmt.Errorf("invalid deviceId: %v", args[1])
	}
	publicKey, ok := args[2].([]byte)
	if !ok {
		return nil, fmt.Errorf("invalid publicKey: %v", args[2])
	}
	deviceSignature, ok := args[3].([]byte)
	if !ok {
		return nil, fmt.Errorf("invalid deviceSignature: %v", args[3])
	}

	ownerAddr, err := p.addrCdc.BytesToString(owner.Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to decode owner address: %w", err)
	}

	msg := &bandwidthtypes.MsgRegisterDevice{
		Owner:           ownerAddr,
		DeviceId:        deviceID,
		PublicKey:       publicKey,
		DeviceSignature: deviceSignature,
	}

	if _, err := p.bandwidthMsgServer.RegisterDevice(ctx, msg); err != nil {
		return nil, err
	}

	return method.Outputs.Pack(true)
}

// UnregisterDevice removes a device's on-chain registration. The caller
// (msg.sender) must equal the owner address passed as the first argument -
// the keeper itself also independently checks that this address matches the
// device's currently-registered owner (see x/bandwidth/keeper/msg_server.go),
// so a stale/forged owner argument still can't unregister someone else's
// device.
func (p *Precompile) UnregisterDevice(
	ctx sdk.Context,
	contract *vm.Contract,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 2, len(args))
	}

	owner, ok := args[0].(common.Address)
	if !ok || owner == (common.Address{}) {
		return nil, fmt.Errorf("invalid owner address: %v", args[0])
	}

	msgSender := contract.Caller()
	if msgSender != owner {
		return nil, fmt.Errorf(cmn.ErrRequesterIsNotMsgSender, msgSender.String(), owner.String())
	}

	deviceID, ok := args[1].(string)
	if !ok {
		return nil, fmt.Errorf("invalid deviceId: %v", args[1])
	}

	ownerAddr, err := p.addrCdc.BytesToString(owner.Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to decode owner address: %w", err)
	}

	msg := &bandwidthtypes.MsgUnregisterDevice{
		Owner:    ownerAddr,
		DeviceId: deviceID,
	}

	if _, err := p.bandwidthMsgServer.UnregisterDevice(ctx, msg); err != nil {
		return nil, err
	}

	return method.Outputs.Pack(true)
}
