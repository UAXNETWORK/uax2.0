package types

import "cosmossdk.io/errors"

var (
	ErrInvalidParams       = errors.Register(ModuleName, 2, "invalid params")
	ErrDeviceNotFound      = errors.Register(ModuleName, 3, "device not found")
	ErrDeviceAlreadyExists = errors.Register(ModuleName, 4, "device already registered")
	ErrInvalidSignature    = errors.Register(ModuleName, 5, "invalid device signature")
	ErrInvalidNonce        = errors.Register(ModuleName, 6, "nonce must be greater than the last accepted nonce")
	ErrInvalidPublicKey    = errors.Register(ModuleName, 7, "invalid device public key")
	ErrNotDeviceOwner      = errors.Register(ModuleName, 8, "signer is not the device's registered owner")
)
