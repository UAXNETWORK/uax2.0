import json
import hashlib
from web3 import Web3
from eth_account import Account
from ecdsa import SigningKey, SECP256k1
from ecdsa.util import sigencode_string_canonize

RPC_URL = "http://127.0.0.1:8545"
PRECOMPILE_ADDRESS = "0x0000000000000000000000000000000000000808"
BANDWIDTH_PER_SHARE = 8000000000000000  # atest, genesis param bandwidth_per_share

DEV0_PRIVKEY = "0x88CBEAD91AEE890D27BF06E003ADE3D4E952427E88F88D31D61D3EF5E5D54305"
DEV2_PRIVKEY = "0x3B7955D25189C99A7468192FCBC6429205C158834053EBE3F78F4512AB432DB9"

ABI = json.loads('''[
  {
    "type": "function",
    "name": "submitShare",
    "inputs": [
      { "name": "relayer", "type": "address" },
      { "name": "deviceId", "type": "string" },
      { "name": "shareCount", "type": "uint32" },
      { "name": "nonce", "type": "uint64" },
      { "name": "signature", "type": "bytes" }
    ],
    "outputs": [ { "name": "mintedAmount", "type": "string" } ],
    "stateMutability": "nonpayable"
  },
  {
    "type": "function",
    "name": "registerDevice",
    "inputs": [
      { "name": "owner", "type": "address" },
      { "name": "deviceId", "type": "string" },
      { "name": "publicKey", "type": "bytes" }
    ],
    "outputs": [ { "name": "success", "type": "bool" } ],
    "stateMutability": "nonpayable"
  }
]''')

w3 = Web3(Web3.HTTPProvider(RPC_URL))
assert w3.is_connected(), "EVM RPC se connect nahi ho raha"

dev0 = Account.from_key(DEV0_PRIVKEY)
dev2 = Account.from_key(DEV2_PRIVKEY)
print("dev0 (owner):  ", dev0.address)
print("dev2 (relayer):", dev2.address)

contract = w3.eth.contract(address=Web3.to_checksum_address(PRECOMPILE_ADDRESS), abi=ABI)

# fresh device keypair generate karo (ESP32 simulate)
device_sk = SigningKey.generate(curve=SECP256k1)
device_vk = device_sk.get_verifying_key()
compressed_pubkey = device_vk.to_string("compressed")  # 33 bytes
print("device pubkey (compressed):", compressed_pubkey.hex())

device_id = "precompile_test_device_001"

def send_tx(account, fn):
    tx = fn.build_transaction({
        "from": account.address,
        "nonce": w3.eth.get_transaction_count(account.address),
        "gas": 500000,
        "gasPrice": 0,
        "chainId": w3.eth.chain_id,
    })
    signed = account.sign_transaction(tx)
    raw = getattr(signed, "raw_transaction", None) or getattr(signed, "rawTransaction", None)
    tx_hash = w3.eth.send_raw_transaction(raw)
    return w3.eth.wait_for_transaction_receipt(tx_hash)

print("\n--- registerDevice (caller = owner = dev0) ---")
fn = contract.functions.registerDevice(dev0.address, device_id, compressed_pubkey)
receipt = send_tx(dev0, fn)
print("status:", receipt.status, "gas used:", receipt.gasUsed)
if receipt.status != 1:
    raise SystemExit("registerDevice FAIL ho gaya, yahin ruk jao")

share_count = 3
nonce = 1
sign_bytes = f"{device_id}:{share_count}:{nonce}".encode()
signature = device_sk.sign(sign_bytes, hashfunc=hashlib.sha256, sigencode=sigencode_string_canonize)
print("signature length:", len(signature), "bytes")

balance_before = w3.eth.get_balance(dev0.address)

print("\n--- submitShare (caller = relayer = dev2) ---")
fn = contract.functions.submitShare(dev2.address, device_id, share_count, nonce, signature)
receipt = send_tx(dev2, fn)
print("status:", receipt.status, "gas used:", receipt.gasUsed)

balance_after = w3.eth.get_balance(dev0.address)
diff = balance_after - balance_before
expected = BANDWIDTH_PER_SHARE * share_count
print(f"\nbalance diff : {diff}")
print(f"expected     : {expected}")
print("MATCH ✅" if diff == expected else "MISMATCH ❌")
