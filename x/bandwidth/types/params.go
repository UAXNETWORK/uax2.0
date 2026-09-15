package types

import "cosmossdk.io/math"

// DefaultBandwidthPerShare is 0.008 test = 8_000_000_000_000_000 atest per share.
var DefaultBandwidthPerShare = math.NewInt(8_000_000_000_000_000)

// DefaultDailyTopupAmount is 212 test = 212 * 1e18 atest.
var DefaultDailyTopupAmount = math.NewInt(212).MulRaw(1_000_000_000_000_000_000)

// DefaultParams returns default bandwidth module parameters.
func DefaultParams() Params {
	return Params{
		BandwidthPerShare: DefaultBandwidthPerShare,
		DailyTopupAmount:  DefaultDailyTopupAmount,
	}
}

// Validate performs basic validation of bandwidth module parameters.
func (p Params) Validate() error {
	if p.BandwidthPerShare.IsNil() || p.BandwidthPerShare.IsNegative() {
		return ErrInvalidParams.Wrap("bandwidth_per_share must be non-negative")
	}
	if p.DailyTopupAmount.IsNil() || p.DailyTopupAmount.IsNegative() {
		return ErrInvalidParams.Wrap("daily_topup_amount must be non-negative")
	}
	return nil
}
