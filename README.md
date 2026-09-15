# UAX Chain

A Cosmos EVM fork implementing the `x/bandwidth` module for the UAX DePIN
bandwidth-reward network: device registration, proof-of-bandwidth share
submission, and continuous block-reward minting.

- Chain ID: `9001` (Cosmos) / `262144` (EVM/EIP-155)
- Native token: `BANDWIDTH` (denom `abandwidth`, 18 decimals)
- Block time: 5 seconds

---

## Running the first node (network creator only)

Use `uax_node.sh` only if you are standing up a brand-new network. If you're
joining an existing one, skip to "Joining an existing network" below.

```bash
chmod +x uax_node.sh

./uax_node.sh reset     # create a brand-new genesis (irreversible — wipes
                         #   any existing chain data on this machine)
./uax_node.sh start     # start the node in the background
./uax_node.sh stop      # stop it
./uax_node.sh status    # check if it's running and the current block height
./uax_node.sh logs      # tail the live log
./uax_node.sh info      # reprint your wallet address and node ID (does NOT
                         #   show the private key or mnemonic again — those
                         #   are only ever shown once, right after 'reset')
```

After `reset`, save the printed private key, mnemonic, and node ID
immediately — none of it can be recovered later except by mnemonic.

---

## Joining an existing network

Use `uax_join_node.sh`. You need this repo (cloned and built) plus a
`genesis.json` from an existing node operator, placed in the same folder as
the script.

```bash
git clone https://github.com/UAXNETWORK/uax2.0.git
cd uax2.0
make install
sudo cp $(go env GOPATH)/bin/evmd /usr/local/bin/evmd

# place genesis.json (from an existing node operator) in this same folder,
# then:
chmod +x uax_join_node.sh

./uax_join_node.sh init      # one-time: create home dir, apply the shared
                              #   genesis, connect to the network
./uax_join_node.sh start     # start the node in the background
./uax_join_node.sh stop      # stop it
./uax_join_node.sh status    # check sync progress / block height
./uax_join_node.sh logs      # tail the live log
```

Wait until `status` shows `catching_up: False` before continuing.

### Becoming a validator (optional)

```bash
./uax_join_node.sh newkey             # create a new wallet; prints its
                                       #   bech32 and 0x address
# ask the chain operator to fund this address with BANDWIDTH

./uax_join_node.sh become-validator   # stake and register as a validator
```

`become-validator` will ask for: the wallet key name (from `newkey`), the
amount to self-delegate (in `abandwidth`, e.g. `50000000000000000000000`
for 50,000 BANDWIDTH), and a public moniker.

---

## Notes

- `PEER_NODE_ID` and `PEER_ADDRESS` near the top of `uax_join_node.sh`
  point at the network's known node — edit them if that node's address
  changes.
- Rewards from staking (chain inflation + tx fees) are separate from
  `x/bandwidth` device/share rewards — see the module source under
  `x/bandwidth/` for that mechanism.
