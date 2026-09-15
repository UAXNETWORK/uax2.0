#!/usr/bin/env python3
"""
Chain-side revert (item 4, 8 Sep 2026): removes the GetAllDevices and
VerifyDeviceSignature precompile query methods (and their ABI-method-name
consts) added earlier today for the device-signed-handshake experiment.
Both are already fully dormant - nothing calls them any more since the
relayer's own handshake check was reverted - this just removes the dead
code instead of leaving it in place.

Leaves GetDeviceOwner (used by verify_node_owner_onchain in the relayer,
still very much in active use) and the DeviceQuerier interface entirely
untouched - only the two now-unused methods and their consts go.

Run from inside the precompiles/bandwidth directory:
    cd ~/Documents/uax-production/cosmos/evm/precompiles/bandwidth
    python3 patch_query_go_revert.py
"""
path = "query.go"

with open(path) as f:
    content = f.read()

original_content = content

# --- 1. remove the two method-name consts ---
old1 = (
    "// GetAllDevicesMethod defines the ABI method name for the bandwidth\n"
    "// GetAllDevices read-only query.\n"
    "const GetAllDevicesMethod = \"getAllDevices\"\n"
    "\n"
    "// VerifyDeviceSignatureMethod defines the ABI method name for the\n"
    "// bandwidth VerifyDeviceSignature read-only query.\n"
    "const VerifyDeviceSignatureMethod = \"verifyDeviceSignature\"\n"
    "\n"
)
count1 = content.count(old1)
assert count1 == 1, f"[consts] expected exactly 1 match, found {count1} - aborting, no changes made"
content = content.replace(old1, "")

# --- 2. remove GetAllDevices method (from its doc comment through closing brace) ---
old2 = (
    "// GetAllDevices returns every currently-registered device_id alongside its\n"
    "// owner address, in matching order (deviceIds[i] is owned by owners[i]).\n"
    "//\n"
    "// Read-only (view) method, same permissionless-by-design rationale as\n"
    "// GetDeviceOwner above: the full device_id -> owner mapping is already\n"
    "// entirely public today, just scattered across this module's on-chain\n"
    "// device_registered/device_unregistered event history instead of available\n"
    "// as one call - anyone willing to replay that history can already\n"
    "// reconstruct exactly this list. This method adds no new information\n"
    "// disclosure; it's a convenience query only (added 8 Sep 2026, so tools -\n"
    "// and people - checking \"how many devices are registered right now\" don't\n"
    "// need to stop the node and run `evmd export`, or replay every historical\n"
    "// event, to find out).\n"
    "func (p Precompile) GetAllDevices(\n"
    "\tctx sdk.Context,\n"
    "\t_ *vm.Contract,\n"
    "\tmethod *abi.Method,\n"
    "\targs []interface{},\n"
    ") ([]byte, error) {\n"
    "\tif len(args) != 0 {\n"
    "\t\treturn nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 0, len(args))\n"
    "\t}\n"
    "\n"
    "\tdevices := p.deviceQuerier.GetAllDevices(ctx)\n"
    "\n"
    "\tdeviceIDs := make([]string, len(devices))\n"
    "\towners := make([]common.Address, len(devices))\n"
    "\tfor i, device := range devices {\n"
    "\t\tdeviceIDs[i] = device.DeviceId\n"
    "\t\townerAddr, err := sdk.AccAddressFromBech32(device.Owner)\n"
    "\t\tif err != nil {\n"
    "\t\t\treturn nil, fmt.Errorf(\"invalid owner address stored for device %s: %w\", device.DeviceId, err)\n"
    "\t\t}\n"
    "\t\towners[i] = common.BytesToAddress(ownerAddr.Bytes())\n"
    "\t}\n"
    "\n"
    "\treturn method.Outputs.Pack(deviceIDs, owners)\n"
    "}\n"
    "\n"
)
count2 = content.count(old2)
assert count2 == 1, f"[GetAllDevices] expected exactly 1 match, found {count2} - aborting, no changes made"
content = content.replace(old2, "")

# --- 3. remove VerifyDeviceSignature method ---
old3 = (
    "// VerifyDeviceSignature checks whether `signature` is a valid secp256k1\n"
    "// signature over `message`, verified against device_id's CURRENTLY\n"
    "// on-chain-registered public key. Returns `false` (never an error, for an\n"
    "// unregistered device or a bad signature alike) so callers can treat both\n"
    "// cases identically: \"not proven\".\n"
    "//\n"
    "// Read-only (view) method. Added 8 Sep 2026 (Rust relayer audit, finding:\n"
    "// unauthenticated TCP handshake) as a general proof-of-possession\n"
    "// primitive - lets off-chain services (the DePIN relayer's TCP handshake\n"
    "// in particular) require a device to prove it holds the private key\n"
    "// behind its OWN registered device_id, for actions beyond\n"
    "// registerDevice/submitShare, without duplicating secp256k1 verification\n"
    "// logic off-chain. Same underlying check as RegisterDevice's\n"
    "// proof-of-possession fix (finding #1) and SubmitShare's signature check -\n"
    "// this just exposes it as a reusable, message-agnostic query.\n"
    "func (p Precompile) VerifyDeviceSignature(\n"
    "\tctx sdk.Context,\n"
    "\t_ *vm.Contract,\n"
    "\tmethod *abi.Method,\n"
    "\targs []interface{},\n"
    ") ([]byte, error) {\n"
    "\tif len(args) != 3 {\n"
    "\t\treturn nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 3, len(args))\n"
    "\t}\n"
    "\n"
    "\tdeviceID, ok := args[0].(string)\n"
    "\tif !ok {\n"
    "\t\treturn nil, fmt.Errorf(\"invalid deviceId: %v\", args[0])\n"
    "\t}\n"
    "\tmessage, ok := args[1].([]byte)\n"
    "\tif !ok {\n"
    "\t\treturn nil, fmt.Errorf(\"invalid message: %v\", args[1])\n"
    "\t}\n"
    "\tsignature, ok := args[2].([]byte)\n"
    "\tif !ok {\n"
    "\t\treturn nil, fmt.Errorf(\"invalid signature: %v\", args[2])\n"
    "\t}\n"
    "\n"
    "\tdevice, found := p.deviceQuerier.GetDevice(ctx, deviceID)\n"
    "\tif !found {\n"
    "\t\t// No on-chain public key to check against - can't be proven.\n"
    "\t\treturn method.Outputs.Pack(false)\n"
    "\t}\n"
    "\n"
    "\tpubKey := &secp256k1.PubKey{Key: device.PublicKey}\n"
    "\treturn method.Outputs.Pack(pubKey.VerifySignature(message, signature))\n"
    "}\n"
)
count3 = content.count(old3)
assert count3 == 1, f"[VerifyDeviceSignature] expected exactly 1 match, found {count3} - aborting, no changes made"
content = content.replace(old3, "")

# --- 4. secp256k1 import is now unused - remove it ---
old4 = '\t"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"\n'
count4 = content.count(old4)
assert count4 == 1, f"[secp256k1 import] expected exactly 1 match, found {count4} - aborting, no changes made"
content = content.replace(old4, "")

assert content != original_content, "No changes were made - something is wrong"

with open(path, "w") as f:
    f.write(content)

print("Chain-side query.go revert applied successfully to", path)
print(f"File shrank from {len(original_content)} to {len(content)} bytes")
