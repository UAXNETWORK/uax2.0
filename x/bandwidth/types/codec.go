package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgRegisterDevice{}, "cosmos/evm/x/bandwidth/MsgRegisterDevice", nil)
	cdc.RegisterConcrete(&MsgSubmitShare{}, "cosmos/evm/x/bandwidth/MsgSubmitShare", nil)
	cdc.RegisterConcrete(&MsgUnregisterDevice{}, "cosmos/evm/x/bandwidth/MsgUnregisterDevice", nil)
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgRegisterDevice{},
		&MsgSubmitShare{},
		&MsgUnregisterDevice{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
