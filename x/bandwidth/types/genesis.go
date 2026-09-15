package types

// DefaultGenesisState returns the default bandwidth module genesis state.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params:  DefaultParams(),
		Devices: []Device{},
	}
}

// Validate performs basic genesis state validation.
func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return err
	}
	seen := make(map[string]bool)
	for _, d := range gs.Devices {
		if seen[d.DeviceId] {
			return ErrDeviceAlreadyExists.Wrapf("duplicate device_id %s in genesis", d.DeviceId)
		}
		seen[d.DeviceId] = true
	}
	return nil
}
