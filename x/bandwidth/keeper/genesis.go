package keeper

import (
	"context"

	"github.com/cosmos/evm/x/bandwidth/types"
)

func (k Keeper) InitGenesis(ctx context.Context, genState types.GenesisState) {
	if err := k.SetParams(ctx, genState.Params); err != nil {
		panic(err)
	}
	for _, device := range genState.Devices {
		k.SetDevice(ctx, device)
	}
}

func (k Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
	return &types.GenesisState{
		Params:  k.GetParams(ctx),
		Devices: k.GetAllDevices(ctx),
	}
}
