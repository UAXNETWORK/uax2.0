package bandwidth

import (
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	cmn "github.com/cosmos/evm/precompiles/common"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// GetDeviceOwnerMethod defines the ABI method name for the bandwidth
// GetDeviceOwner read-only query.
const GetDeviceOwnerMethod = "getDeviceOwner"

// GetDeviceOwner returns the on-chain registered owner address for a
// device_id, or the zero address if the device isn't registered at all.
// This is a read-only (view) method - callable via eth_call, no tx or gas
// beyond the base RequiredGas needed - and, unlike the tx methods above,
// does NOT require the caller to BE the owner: anyone can look up who owns
// a given device, the same way anyone can read any other public chain
// state.
//
// Added so off-chain services (the DePIN relayer's HTTP API in particular)
// can verify "does this wallet really own this device_id?" by asking the
// chain directly, instead of trusting a copy of that fact sitting in their
// own database. The chain stays the sole source of truth for device
// ownership - exactly as it already is for share submission (see
// x/bandwidth/keeper/msg_server.go's SubmitShare).
func (p Precompile) GetDeviceOwner(
	ctx sdk.Context,
	_ *vm.Contract,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 1, len(args))
	}

	deviceID, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("invalid deviceId: %v", args[0])
	}

	device, found := p.deviceQuerier.GetDevice(ctx, deviceID)
	if !found {
		// Unregistered device - zero address, same convention Solidity
		// itself uses for "no value" on an address type. Callers (like the
		// relayer's ownership check) should treat the zero address as
		// "not registered", never as a valid owner to compare against.
		return method.Outputs.Pack(common.Address{})
	}

	ownerAddr, err := sdk.AccAddressFromBech32(device.Owner)
	if err != nil {
		return nil, fmt.Errorf("invalid owner address stored for device %s: %w", deviceID, err)
	}

	return method.Outputs.Pack(common.BytesToAddress(ownerAddr.Bytes()))
}

