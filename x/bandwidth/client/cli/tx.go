package cli

import (
	"encoding/hex"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"

	"github.com/cosmos/evm/x/bandwidth/types"
)

func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "bandwidth transactions subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		CmdRegisterDevice(),
		CmdSubmitShare(),
		CmdUnregisterDevice(),
	)

	return cmd
}

func CmdRegisterDevice() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register-device [device-id] [public-key-hex]",
		Short: "Register a device with its compressed secp256k1 public key",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			pubKeyBytes, err := hex.DecodeString(args[1])
			if err != nil {
				return err
			}

			msg := &types.MsgRegisterDevice{
				Owner:     clientCtx.GetFromAddress().String(),
				DeviceId:  args[0],
				PublicKey: pubKeyBytes,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdUnregisterDevice() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unregister-device [device-id]",
		Short: "Remove a device's on-chain registration (must be signed by its current owner)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgUnregisterDevice{
				Owner:    clientCtx.GetFromAddress().String(),
				DeviceId: args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdSubmitShare() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit-share [device-id] [share-count] [nonce] [signature-hex]",
		Short: "Submit a device-signed share on behalf of a device",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			shareCount, err := strconv.ParseUint(args[1], 10, 32)
			if err != nil {
				return err
			}
			nonce, err := strconv.ParseUint(args[2], 10, 64)
			if err != nil {
				return err
			}
			sigBytes, err := hex.DecodeString(args[3])
			if err != nil {
				return err
			}

			msg := &types.MsgSubmitShare{
				Relayer:    clientCtx.GetFromAddress().String(),
				DeviceId:   args[0],
				ShareCount: uint32(shareCount),
				Nonce:      nonce,
				Signature:  sigBytes,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
