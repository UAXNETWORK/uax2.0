#!/bin/bash
# ============================================================================
# uax_join_node.sh — Join the UAX Chain network as a full node (and later,
# optionally, as a validator). Give this file + genesis.json to ANY node
# operator — they don't need anything else from the original chain creator.
#
# Prerequisites (once, before running this script):
#   1. This repo cloned and built:
#        git clone https://github.com/UAXNETWORK/uax2.0.git
#        cd uax2.0 && make install
#      (make sure `evmd` is on your PATH, e.g. copy to /usr/local/bin/evmd)
#   2. genesis.json from an existing node, placed in the SAME folder as this
#      script (i.e. next to uax_join_node.sh).
#
# Usage:
#   ./uax_join_node.sh init      # one-time: create home dir, apply shared
#                                #   genesis, connect to network
#   ./uax_join_node.sh start     # start the node in background
#   ./uax_join_node.sh stop      # stop it
#   ./uax_join_node.sh status    # check sync progress / block height
#   ./uax_join_node.sh logs      # tail the log
#
# After 'init' + 'start', your node will sync from genesis (this takes time
# proportional to how many blocks already exist). Once fully synced (see
# 'status' — catching_up should say false), you can OPTIONALLY become a
# validator with the 'become-validator' command below.
# ============================================================================
set -e

# ---- Edit these if your setup differs -------------------------------------
CHAINID="9001"
MONIKER="${UAX_MONIKER:-uax-node-$(hostname -s)}"
CHAINDIR="$HOME/.evmd"
KEYRING="file"
PEERS="8b5974d86ad9dead42af47a86652480d0d887e9d@134.209.102.138:26656,ff9f38e0c93566b3730bbde0d81a01ff5b89f558@146.190.95.50:26656"
# -----------------------------------------------------------------------------

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GENESIS_SRC="$SCRIPT_DIR/genesis.json"
CONFIG_TOML="$CHAINDIR/config/config.toml"
GENESIS_DST="$CHAINDIR/config/genesis.json"
LOGFILE="$CHAINDIR/node.log"
PIDFILE="$CHAINDIR/node.pid"

cmd="${1:-}"

do_init() {
  if [ ! -f "$GENESIS_SRC" ]; then
    echo "ERROR: genesis.json not found next to this script ($GENESIS_SRC)."
    echo "Get it from an existing node operator and place it here first."
    exit 1
  fi
  if [ -d "$CHAINDIR" ]; then
    echo "$CHAINDIR already exists. Delete it first if you want to re-init (rm -rf $CHAINDIR)."
    exit 1
  fi

  evmd config set client chain-id "$CHAINID" --home "$CHAINDIR"
  evmd config set client keyring-backend "$KEYRING" --home "$CHAINDIR"
  evmd init "$MONIKER" --chain-id "$CHAINID" --home "$CHAINDIR"

  cp "$GENESIS_SRC" "$GENESIS_DST"

  # Connect to the network
  sed -i.bak "s/^persistent_peers *=.*/persistent_peers = \"${PEERS}\"/" "$CONFIG_TOML"

  # Same mempool fix every node in this network needs
  sed -i.bak 's/type = "flood"/type = "app"/g' "$CONFIG_TOML"

  echo ""
  echo ">>> Initialized. Moniker: $MONIKER"
  echo ">>> Genesis copied from: $GENESIS_SRC"
  echo ">>> Peers: ${PEERS}"
  echo ">>> Next: ./uax_join_node.sh start"
}

do_start() {
  if [ -f "$PIDFILE" ] && kill -0 "$(cat "$PIDFILE")" 2>/dev/null; then
    echo ">>> Already running (pid $(cat "$PIDFILE"))."
    exit 0
  fi
  nohup evmd start --home "$CHAINDIR" \
    --json-rpc.enable --json-rpc.api eth,txpool,personal,net,debug,web3 \
    --json-rpc.address 0.0.0.0:8545 \
    > "$LOGFILE" 2>&1 &
  echo $! > "$PIDFILE"
  sleep 2
  if kill -0 "$(cat "$PIDFILE")" 2>/dev/null; then
    echo ">>> Running (pid $(cat "$PIDFILE")). Logs: tail -f $LOGFILE"
  else
    echo ">>> Failed to start — check $LOGFILE"; exit 1
  fi
}

do_stop() {
  if [ -f "$PIDFILE" ] && kill -0 "$(cat "$PIDFILE")" 2>/dev/null; then
    kill -SIGTERM "$(cat "$PIDFILE")"
    rm -f "$PIDFILE"
    echo ">>> Stopped."
  else
    echo ">>> Not running."
  fi
}

do_status() {
  echo "--- Process ---"
  if [ -f "$PIDFILE" ] && kill -0 "$(cat "$PIDFILE")" 2>/dev/null; then
    echo "Running (pid $(cat "$PIDFILE"))"
  else
    echo "Not running"
  fi
  echo "--- Sync status ---"
  curl -s http://localhost:26657/status | python3 -c "
import json,sys
d = json.load(sys.stdin)['result']['sync_info']
print('latest_block_height:', d['latest_block_height'])
print('catching_up:', d['catching_up'])
"
}

do_logs() {
  tail -f "$LOGFILE"
}

do_become_validator() {
  echo "Prerequisites before this works:"
  echo "  1. Your node must be fully synced (run 'status', catching_up must be false)."
  echo "  2. You need a funded wallet (ask the chain operator to send you BANDWIDTH)."
  echo ""
  read -r -p "Wallet key name to create/use (in this node's keyring): " KEYNAME
  read -r -p "Amount to self-delegate, e.g. 1000000000000000000000 (1000 BANDWIDTH, in abandwidth): " AMOUNT
  read -r -p "Moniker for your validator (public name): " VALMONIKER

  if [ -f "$CHAINDIR/keyring-file/$KEYNAME.info" ]; then
    echo "(key '$KEYNAME' already exists on disk, reusing it — not touching it)"
  else
    evmd keys add "$KEYNAME" --keyring-backend "$KEYRING" --algo eth_secp256k1 --home "$CHAINDIR"
  fi

  PUBKEY=$(evmd comet show-validator --home "$CHAINDIR")
  VALJSON="$CHAINDIR/validator.json"
  cat > "$VALJSON" <<JSON
{
  "pubkey": ${PUBKEY},
  "amount": "${AMOUNT}abandwidth",
  "moniker": "${VALMONIKER}",
  "commission-rate": "0.10",
  "commission-max-rate": "0.20",
  "commission-max-change-rate": "0.01",
  "min-self-delegation": "1"
}
JSON

  evmd tx staking create-validator "$VALJSON" \
    --chain-id "$CHAINID" \
    --from "$KEYNAME" \
    --keyring-backend "$KEYRING" \
    --home "$CHAINDIR" \
    --gas auto --gas-adjustment 1.4 --gas-prices 10000000abandwidth \
    -y
}

do_newkey() {
  read -r -p "Name for this key (e.g. vps-node2-key): " KEYNAME
  evmd keys add "$KEYNAME" --keyring-backend "$KEYRING" --algo eth_secp256k1 --home "$CHAINDIR"
  BECH32=$(evmd keys show "$KEYNAME" -a --keyring-backend "$KEYRING" --home "$CHAINDIR")
  echo ""
  echo ">>> Bech32 address: $BECH32"
  echo ">>> 0x (EVM) address:"
  evmd debug addr "$BECH32" --home "$CHAINDIR" | grep -i "EVM\|0x" || evmd debug addr "$BECH32" --home "$CHAINDIR"
}

do_info() {
  echo ""
  echo "============================================================================"
  echo "NODE INFO"
  echo "============================================================================"
  echo "Node ID:"
  evmd comet show-node-id --home "$CHAINDIR"
  echo ""
  echo "Wallet keys in this node's keyring:"
  for name in $(evmd keys list --keyring-backend "$KEYRING" --home "$CHAINDIR" --output json | python3 -c "import json,sys; [print(k['name']) for k in json.load(sys.stdin)]"); do
    BECH32=$(evmd keys show "$name" -a --keyring-backend "$KEYRING" --home "$CHAINDIR")
    echo "  - $name"
    echo "    Bech32: $BECH32"
    echo -n "    0x:     "
    evmd debug addr "$BECH32" --home "$CHAINDIR" | grep -i "0x" | sed 's/^/ /'
  done
  echo "============================================================================"
}

case "$cmd" in
  init)             do_init ;;
  start)            do_start ;;
  stop)             do_stop ;;
  status)           do_status ;;
  logs)             do_logs ;;
  newkey)           do_newkey ;;
  info)             do_info ;;
  become-validator) do_become_validator ;;
  *) echo "Usage: $0 {init|start|stop|status|logs|newkey|info|become-validator}"; exit 1 ;;
esac
