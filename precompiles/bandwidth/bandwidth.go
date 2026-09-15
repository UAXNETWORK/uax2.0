// Package bandwidth contains the implementation of the x/bandwidth module
// precompile: an EVM-callable interface for submitting device shares (which
// triggers real, structural on-chain minting), registering/unregistering
// devices, and querying device ownership/signatures (single device, the
// full list, or proof-of-possession for an arbitrary message).
package bandwidth

import (
	"bytes"
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	_ "embed"

	"cosmossdk.io/core/address"

	cmn "github.com/cosmos/evm/precompiles/common"
	bandwidthtypes "github.com/cosmos/evm/x/bandwidth/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ vm.PrecompiledContract = &Precompile{}

var (
	//go:embed abi.json
	f   []byte
	ABI abi.ABI
)

func init() {
	var err error
	ABI, err = abi.JSON(bytes.NewReader(f))
	if err != nil {
		panic(err)
	}
}

// DeviceQuerier is the minimal read-only keeper surface the bandwidth
// precompile needs for query-type (non-transaction) methods like
// getDeviceOwner, getAllDevices and verifyDeviceSignature. Kept as its own
// small interface (rather than importing the full x/bandwidth/keeper.Keeper
// type here) to avoid a precompile -> keeper package dependency -
// bandwidthkeeper.Keeper already satisfies this structurally (see
// x/bandwidth/keeper/device.go's GetDevice and GetAllDevices), so the real
// Keeper can be passed straight into NewPrecompile with no adapter needed.
type DeviceQuerier interface {
	GetDevice(ctx context.Context, deviceID string) (bandwidthtypes.Device, bool)
	GetAllDevices(ctx context.Context) []bandwidthtypes.Device
}

// Precompile defines the precompiled contract for the bandwidth module.
type Precompile struct {
	cmn.Precompile

	abi.ABI
	bandwidthMsgServer bandwidthtypes.MsgServer
	deviceQuerier      DeviceQuerier
	addrCdc            address.Codec
}

// NewPrecompile creates a new bandwidth Precompile instance implementing the
// PrecompiledContract interface.
func NewPrecompile(
	bandwidthMsgServer bandwidthtypes.MsgServer,
	deviceQuerier DeviceQuerier,
	addrCdc address.Codec,
) *Precompile {
	return &Precompile{
		Precompile: cmn.Precompile{
			KvGasConfig:          storetypes.KVGasConfig(),
			TransientKVGasConfig: storetypes.TransientGasConfig(),
			ContractAddress:      common.HexToAddress(evmtypes.BandwidthPrecompileAddress),
		},
		ABI:                ABI,
		bandwidthMsgServer: bandwidthMsgServer,
		deviceQuerier:      deviceQuerier,
		addrCdc:            addrCdc,
	}
}

func (Precompile) Name() string {
	return "bandwidth"
}

// RequiredGas calculates the precompiled contract's base gas rate.
func (p Precompile) RequiredGas(input []byte) uint64 {
	if len(input) < 4 {
		return 0
	}
	methodID := input[:4]

	method, err := p.MethodById(methodID)
	if err != nil {
		return 0
	}

	return p.Precompile.RequiredGas(input, p.IsTransaction(method))
}

func (p Precompile) Run(evm *vm.EVM, contract *vm.Contract, readonly bool) ([]byte, error) {
	return p.RunNativeAction(evm, contract, func(ctx sdk.Context) ([]byte, error) {
		return p.Execute(ctx, contract, readonly)
	})
}

// Execute executes the precompiled contract's bandwidth transaction and
// query methods defined in the ABI.
func (p Precompile) Execute(ctx sdk.Context, contract *vm.Contract, readOnly bool) ([]byte, error) {
	method, args, err := cmn.SetupABI(p.ABI, contract, readOnly, p.IsTransaction)
	if err != nil {
		return nil, err
	}

	var bz []byte
	switch method.Name {
	case SubmitShareMethod:
		bz, err = p.SubmitShare(ctx, contract, method, args)
	case RegisterDeviceMethod:
		bz, err = p.RegisterDevice(ctx, contract, method, args)
	case UnregisterDeviceMethod:
		bz, err = p.UnregisterDevice(ctx, contract, method, args)
	case GetDeviceOwnerMethod:
		bz, err = p.GetDeviceOwner(ctx, contract, method, args)
	default:
		return nil, fmt.Errorf(cmn.ErrUnknownMethod, method.Name)
	}

	return bz, err
}

// IsTransaction returns whether a method mutates state. Every bandwidth
// method is a transaction EXCEPT getDeviceOwner, which is a plain
// read-only query - callable via eth_call with no tx/gas required beyond
// the base RequiredGas, same as reading any other public chain state.
func (Precompile) IsTransaction(method *abi.Method) bool {
	switch method.Name {
	case GetDeviceOwnerMethod:
		return false
	default:
		return true
	}
}
