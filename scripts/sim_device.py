import argparse
import hashlib
import json
import os

from ecdsa import SigningKey, SECP256k1
from ecdsa.util import sigencode_string_canonize

KEYFILE = os.path.expanduser("~/uax_sim_device.json")


def gen():
    sk = SigningKey.generate(curve=SECP256k1)
    vk = sk.get_verifying_key()
    priv_hex = sk.to_string().hex()
    pub_hex = vk.to_string("compressed").hex()
    with open(KEYFILE, "w") as f:
        json.dump({"priv": priv_hex, "pub": pub_hex}, f)
    print("private_key_hex:", priv_hex)
    print("public_key_hex (compressed, 33 bytes):", pub_hex)


def sign(device_id, share_count, nonce):
    with open(KEYFILE) as f:
        data = json.load(f)
    sk = SigningKey.from_string(bytes.fromhex(data["priv"]), curve=SECP256k1)
    msg = f"{device_id}:{share_count}:{nonce}".encode()
    digest = hashlib.sha256(msg).digest()
    sig = sk.sign_digest(digest, sigencode=sigencode_string_canonize)
    print("signature_hex:", sig.hex())


if __name__ == "__main__":
    p = argparse.ArgumentParser()
    sub = p.add_subparsers(dest="cmd", required=True)
    sub.add_parser("gen")
    s = sub.add_parser("sign")
    s.add_argument("device_id")
    s.add_argument("share_count", type=int)
    s.add_argument("nonce", type=int)
    args = p.parse_args()

    if args.cmd == "gen":
        gen()
    elif args.cmd == "sign":
        sign(args.device_id, args.share_count, args.nonce)
