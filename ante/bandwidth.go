package ante

import (
	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	bandwidthtypes "github.com/cosmos/evm/x/bandwidth/types"

	anteinterfaces "github.com/cosmos/evm/ante/interfaces"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	"github.com/ethereum/go-ethereum/common"
)

// BandwidthCostPerTx is the flat bandwidth charge per transaction, in atest
// base units. "212 bandwidth units" = 212 * 1e18 atest, matching the same
// 1e18-scaling convention used for bandwidth credits server-side.
var BandwidthCostPerTx = sdkmath.NewInt(212).MulRaw(1_000_000_000_000_000_000)

// bandwidthExemptAddresses lists accounts NOT charged the flat bandwidth
// fee - e.g. the treasury account, whose outgoing txs ARE the bandwidth
// reward payouts themselves, not user activity that should be gated.
// Add more 0x (EVM hex) addresses here, comma-separated, as needed.
var bandwidthExemptAddresses = []string{
	"0xA0EbF3f5C60143aBd5cEE0F9DAC14C36Bfb575b3", // treasury
	"0x04B5266D4da80ef6276a574Db0fF51DF96aCB44c", // bandwidth relayer (production)
}

func isBandwidthExempt(addr sdk.AccAddress) bool {
	evmAddr := common.BytesToAddress(addr.Bytes())
	for _, exempt := range bandwidthExemptAddresses {
		if evmAddr == common.HexToAddress(exempt) {
			return true
		}
	}
	return false
}

// DeductBandwidthDecorator charges a flat bandwidth fee (in atest) on every
// non-exempt transaction, regardless of gas used - separate from the normal
// gas fee, representing the UAX Network's DePIN bandwidth cost per tx.
type DeductBandwidthDecorator struct {
	bankKeeper      anteinterfaces.BankKeeper
	bandwidthKeeper anteinterfaces.BandwidthKeeper
}

func NewDeductBandwidthDecorator(bk anteinterfaces.BankKeeper, bwk anteinterfaces.BandwidthKeeper) DeductBandwidthDecorator {
	return DeductBandwidthDecorator{bankKeeper: bk, bandwidthKeeper: bwk}
}

func (d DeductBandwidthDecorator) AnteHandle(
	ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler,
) (sdk.Context, error) {
	payer, err := bandwidthFeePayer(tx)
	if err != nil {
		return ctx, err
	}

	if isBandwidthExempt(payer) {
		return next(ctx, tx, simulate)
	}

	denom := evmtypes.GetEVMCoinDenom()
	cost := sdk.NewCoin(denom, BandwidthCostPerTx)

	balance := d.bankKeeper.GetBalance(ctx, payer, denom)
	if balance.IsLT(cost) {
		return ctx, errorsmod.Wrapf(
			sdkerrors.ErrInsufficientFunds,
			"insufficient bandwidth: need %s, have %s", cost, balance,
		)
	}

	// This fee is deposited into the bandwidth module's own account and,
	// as of the "real bandwidth generation" change, deliberately left
	// there instead of being burned. That accumulated balance is REAL
	// supply collected from genuine chain activity, and x/bandwidth's
	// SubmitShare handler (msg_server.go) now pays device rewards out of
	// this same pool first (only minting fresh coins for whatever the
	// pool can't cover) - so this is the real, on-chain "bandwidth" that
	// device liveness-proof shares actually draw from.
	if err := d.bankKeeper.SendCoinsFromAccountToModule(
		ctx, payer, bandwidthtypes.ModuleName, sdk.NewCoins(cost),
	); err != nil {
		return ctx, errorsmod.Wrap(err, "failed to deduct bandwidth fee")
	}

	if d.bankKeeper.GetBalance(ctx, payer, denom).IsZero() {
		d.bandwidthKeeper.MarkPendingTopup(ctx, payer)
	}

	return next(ctx, tx, simulate)
}

// bandwidthFeePayer figures out who pays the bandwidth fee: for EVM txs it's
// the `from` address of the unpacked eth message; for plain Cosmos-native
// txs it's the standard FeeTx payer.
func bandwidthFeePayer(tx sdk.Tx) (sdk.AccAddress, error) {
	msgs := tx.GetMsgs()
	if len(msgs) > 0 {
		if ethMsg, _, err := evmtypes.UnpackEthMsg(msgs[0]); err == nil {
			return sdk.AccAddress(ethMsg.GetFrom()), nil
		}
	}

	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "tx must be an EVM tx or implement FeeTx")
	}
	return feeTx.FeePayer(), nil
}
