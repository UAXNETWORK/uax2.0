#!/usr/bin/env python3
"""
Chain-side revert (item 4, 8 Sep 2026), companion to patch_query_go_revert.py:
removes the two now-deleted methods' entries from bandwidth.go's dispatch
table (Execute()) and read-only-method switch (IsTransaction()). Apply
AFTER patch_query_go_revert.py (that one deletes the methods/consts these
reference).

Run from inside the precompiles/bandwidth directory:
    cd ~/Documents/uax-production/cosmos/evm/precompiles/bandwidth
    python3 patch_bandwidth_go_revert.py
"""
path = "bandwidth.go"

with open(path) as f:
    content = f.read()

original_content = content

# --- 1. Execute()'s dispatch switch ---
old1 = (
    "\tcase GetDeviceOwnerMethod:\n"
    "\t\tbz, err = p.GetDeviceOwner(ctx, contract, method, args)\n"
    "\tcase GetAllDevicesMethod:\n"
    "\t\tbz, err = p.GetAllDevices(ctx, contract, method, args)\n"
    "\tcase VerifyDeviceSignatureMethod:\n"
    "\t\tbz, err = p.VerifyDeviceSignature(ctx, contract, method, args)\n"
    "\tdefault:\n"
)
new1 = (
    "\tcase GetDeviceOwnerMethod:\n"
    "\t\tbz, err = p.GetDeviceOwner(ctx, contract, method, args)\n"
    "\tdefault:\n"
)
count1 = content.count(old1)
assert count1 == 1, f"[Execute switch] expected exactly 1 match, found {count1} - aborting, no changes made"
content = content.replace(old1, new1)

# --- 2. IsTransaction()'s read-only-methods switch ---
old2 = (
    "// IsTransaction returns whether a method mutates state. Every bandwidth\n"
    "// method is a transaction EXCEPT getDeviceOwner, getAllDevices and\n"
    "// verifyDeviceSignature, which are plain read-only queries - callable via\n"
    "// eth_call with no tx/gas required beyond the base RequiredGas, same as\n"
    "// reading any other public chain state.\n"
    "func (Precompile) IsTransaction(method *abi.Method) bool {\n"
    "\tswitch method.Name {\n"
    "\tcase GetDeviceOwnerMethod, GetAllDevicesMethod, VerifyDeviceSignatureMethod:\n"
    "\t\treturn false\n"
    "\tdefault:\n"
    "\t\treturn true\n"
    "\t}\n"
    "}\n"
)
new2 = (
    "// IsTransaction returns whether a method mutates state. Every bandwidth\n"
    "// method is a transaction EXCEPT getDeviceOwner, which is a plain\n"
    "// read-only query - callable via eth_call with no tx/gas required beyond\n"
    "// the base RequiredGas, same as reading any other public chain state.\n"
    "func (Precompile) IsTransaction(method *abi.Method) bool {\n"
    "\tswitch method.Name {\n"
    "\tcase GetDeviceOwnerMethod:\n"
    "\t\treturn false\n"
    "\tdefault:\n"
    "\t\treturn true\n"
    "\t}\n"
    "}\n"
)
count2 = content.count(old2)
assert count2 == 1, f"[IsTransaction switch] expected exactly 1 match, found {count2} - aborting, no changes made"
content = content.replace(old2, new2)

assert content != original_content, "No changes were made - something is wrong"

with open(path, "w") as f:
    f.write(content)

print("Chain-side bandwidth.go revert applied successfully to", path)
print(f"File shrank from {len(original_content)} to {len(content)} bytes")
