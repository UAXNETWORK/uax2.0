package types

const (
	// ModuleName defines the module name
	ModuleName = "bandwidth"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName
)

var (
	// ParamsKey is the key for module params
	ParamsKey = []byte{0x01}

	// DeviceKeyPrefix is the prefix for device records, keyed by device_id
	DeviceKeyPrefix = []byte{0x02}

	// PendingTopupKeyPrefix marks wallets whose balance has been drawn to
	// zero by the bandwidth fee, making them eligible for the once-daily
	// flat top-up.
	PendingTopupKeyPrefix = []byte{0x03}

	// WalletLastTopupDayKeyPrefix stores, per wallet, the dayIndex on which
	// that wallet was last topped up (at most once per UTC day).
	WalletLastTopupDayKeyPrefix = []byte{0x04}
)

// DeviceKey returns the store key for a given device id.
func DeviceKey(deviceID string) []byte {
	return append(DeviceKeyPrefix, []byte(deviceID)...)
}

// PendingTopupKey returns the store key marking a wallet as pending its
// once-daily flat top-up.
func PendingTopupKey(addr []byte) []byte {
	return append(PendingTopupKeyPrefix, addr...)
}

// WalletLastTopupDayKey returns the store key for a wallet's last-topup
// dayIndex.
func WalletLastTopupDayKey(addr []byte) []byte {
	return append(WalletLastTopupDayKeyPrefix, addr...)
}
