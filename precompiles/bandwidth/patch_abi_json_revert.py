#!/usr/bin/env python3
"""
Chain-side revert (item 4, 8 Sep 2026), companion to the query.go/
bandwidth.go revert patches: removes the getAllDevices and
verifyDeviceSignature entries from the ABI (exact text removal, so the
file's existing formatting/ordering is otherwise untouched - getDeviceOwner
becomes the last array entry again, same as before today). Order relative
to the other two patches doesn't matter (different file).

Run from inside the precompiles/bandwidth directory:
    cd ~/Documents/uax-production/cosmos/evm/precompiles/bandwidth
    python3 patch_abi_json_revert.py
"""
import json

path = "abi.json"

with open(path) as f:
    content = f.read()

original_content = content

old = (
    "  },\n"
    "  {\n"
    "    \"type\": \"function\",\n"
    "    \"name\": \"getAllDevices\",\n"
    "    \"inputs\": [],\n"
    "    \"outputs\": [\n"
    "      { \"name\": \"deviceIds\", \"type\": \"string[]\" },\n"
    "      { \"name\": \"owners\", \"type\": \"address[]\" }\n"
    "    ],\n"
    "    \"stateMutability\": \"view\"\n"
    "  },\n"
    "  {\n"
    "    \"type\": \"function\",\n"
    "    \"name\": \"verifyDeviceSignature\",\n"
    "    \"inputs\": [\n"
    "      { \"name\": \"deviceId\", \"type\": \"string\" },\n"
    "      { \"name\": \"message\", \"type\": \"bytes\" },\n"
    "      { \"name\": \"signature\", \"type\": \"bytes\" }\n"
    "    ],\n"
    "    \"outputs\": [\n"
    "      { \"name\": \"valid\", \"type\": \"bool\" }\n"
    "    ],\n"
    "    \"stateMutability\": \"view\"\n"
    "  }\n"
)
new = "  }\n"
count = content.count(old)
assert count == 1, f"expected exactly 1 match, found {count} - aborting, no changes made"
content = content.replace(old, new)

assert content != original_content, "No changes were made - something is wrong"

# Sanity check: still valid JSON, and exactly 2 fewer entries than before.
before_entries = json.loads(original_content)
after_entries = json.loads(content)
assert len(after_entries) == len(before_entries) - 2, (
    f"expected exactly 2 fewer entries ({len(before_entries)} -> {len(before_entries) - 2}), "
    f"got {len(after_entries)} - aborting write, no changes made"
)

with open(path, "w") as f:
    f.write(content)

print("abi.json revert applied successfully - 2 entries removed")
print(f"Entry count: {len(before_entries)} -> {len(after_entries)}")
