#!/bin/bash
# ============================================================================
# uax_node.sh — single script for the FIRST node of a UAX Chain network.
#
# Usage:
#   ./uax_node.sh reset     # wipe and create a brand-new genesis (irreversible)
#   ./uax_node.sh start     # start the node in the background
#   ./uax_node.sh stop      # stop it
#   ./uax_node.sh status    # check if it's running / current block height
#   ./uax_node.sh logs      # tail the log
#   ./uax_node.sh info      # reprint address / node ID (does NOT show the
#                           #   private key or mnemonic again — those are
#                           #   only ever shown once, at reset time)
#
# After 'reset', all information a new node operator or relayer needs is
# printed ONCE in a single block: wallet address (bech32 and 0x), private
# key, mnemonic, and node ID. Save all of it immediately — none of it can
# be recovered later except the mnemonic-based wallet recovery.
# ============================================================================
set -e

CHAINID="9001"
MONIKER="uaxvalidator"
KEYRING="file"
KEYALGO="eth_secp256k1"
CHAINDIR="$HOME/.evmd"
BASEFEE=10000000
VAL_KEY="uaxvalidator"
TOTAL_SUPPLY_ABANDWIDTH="1000000000000000000000000"   # 1,000,000 BANDWIDTH
SELF_DELEGATE_ABANDWIDTH="100000000000000000000000"    # 100,000 BANDWIDTH

CONFIG_TOML="$CHAINDIR/config/config.toml"
APP_TOML="$CHAINDIR/config/app.toml"
GENESIS="$CHAINDIR/config/genesis.json"
TMP_GENESIS="$CHAINDIR/config/tmp_genesis.json"
PIDFILE="$CHAINDIR/node.pid"
LOGFILE="$CHAINDIR/node.log"

cmd="${1:-}"

print_info() {
  echo ""
  echo "============================================================================"
  echo "NODE INFO"
  echo "============================================================================"
  BECH32=$(evmd keys show "$VAL_KEY" -a --keyring-backend "$KEYRING" --home "$CHAINDIR")
  echo "Bech32 address: $BECH32"
  echo "Ethereum (0x) address:"
  evmd debug addr "$BECH32" --home "$CHAINDIR"
  echo ""
  echo "Node ID:"
  evmd comet show-node-id --home "$CHAINDIR"
  echo "============================================================================"
}

do_reset() {
  command -v jq >/dev/null 2>&1 || { echo "Install jq first: sudo apt install jq"; exit 1; }

  if [ -d "$CHAINDIR" ]; then
    echo "============================================================================"
    echo "  $CHAINDIR already exists — it will be DELETED (no backup)."
    echo "  All existing chain data will be IRREVERSIBLY lost."
    echo "============================================================================"
    read -r -p "Really delete it? Type 'yes' to continue: " confirm
    [ "$confirm" != "yes" ] && { echo "Cancelled."; exit 1; }
    do_stop || true
    rm -rf "$CHAINDIR"
  fi

  evmd config set client chain-id "$CHAINID" --home "$CHAINDIR"
  evmd config set client keyring-backend "$KEYRING" --home "$CHAINDIR"

  echo ""
  echo ">>> Generating a new validator/treasury key. You will be asked to set a"
  echo ">>> keyring passphrase, and a mnemonic will be printed further below —"
  echo ">>> both are shown only once."
  echo ""
  evmd keys add "$VAL_KEY" --keyring-backend "$KEYRING" --algo "$KEYALGO" --home "$CHAINDIR"

  evmd init "$MONIKER" -o --chain-id "$CHAINID" --home "$CHAINDIR"

  jq '.app_state["staking"]["params"]["bond_denom"]="abandwidth"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
  jq '.app_state["gov"]["deposit_params"]["min_deposit"][0]["denom"]="abandwidth"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
  jq '.app_state["gov"]["params"]["min_deposit"][0]["denom"]="abandwidth"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
  jq '.app_state["gov"]["params"]["expedited_min_deposit"][0]["denom"]="abandwidth"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
  jq '.app_state["evm"]["params"]["evm_denom"]="abandwidth"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
  jq '.app_state["mint"]["params"]["mint_denom"]="abandwidth"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
  jq '.app_state["mint"]["params"]["blocks_per_year"]="6311520"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

  jq '.app_state["bank"]["denom_metadata"]=[{"description":"The native staking asset for evmd.","denom_units":[{"denom":"abandwidth","exponent":0,"aliases":["attobandwidth"]},{"denom":"bandwidth","exponent":18,"aliases":[]}],"base":"abandwidth","display":"bandwidth","name":"Bandwidth","symbol":"BANDWIDTH","uri":"","uri_hash":""}]' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

  jq '.app_state["evm"]["params"]["active_static_precompiles"]=["0x0000000000000000000000000000000000000100","0x0000000000000000000000000000000000000400","0x0000000000000000000000000000000000000800","0x0000000000000000000000000000000000000801","0x0000000000000000000000000000000000000802","0x0000000000000000000000000000000000000803","0x0000000000000000000000000000000000000804","0x0000000000000000000000000000000000000805","0x0000000000000000000000000000000000000806","0x0000000000000000000000000000000000000807","0x0000000000000000000000000000000000000808"]' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

  jq '.app_state.erc20.native_precompiles=["0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE"]' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
  jq '.app_state.erc20.token_pairs=[{contract_owner:1,erc20_address:"0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE",denom:"abandwidth",enabled:true}]' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

  jq '.consensus.params.block.max_gas="10000000"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

  jq '.app_state["gov"]["params"]["min_deposit"][0]["amount"]="1000000000000000000000"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
  jq '.app_state["gov"]["params"]["expedited_min_deposit"][0]["amount"]="5000000000000000000000"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
  jq '.app_state["gov"]["deposit_params"]["min_deposit"][0]["amount"]="1000000000000000000000"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

  evmd genesis add-genesis-account "$VAL_KEY" "${TOTAL_SUPPLY_ABANDWIDTH}abandwidth" --keyring-backend "$KEYRING" --home "$CHAINDIR"

  sed -i.bak 's/type = "flood"/type = "app"/g' "$CONFIG_TOML"
  sed -i.bak 's/prometheus = false/prometheus = true/' "$CONFIG_TOML"
  sed -i.bak 's/prometheus-retention-time  = "0"/prometheus-retention-time  = "1000000000000"/g' "$APP_TOML"
  sed -i.bak 's/enabled = false/enabled = true/g' "$APP_TOML"
  sed -i.bak 's/enable = false/enable = true/g' "$APP_TOML"

  evmd genesis gentx "$VAL_KEY" "${SELF_DELEGATE_ABANDWIDTH}abandwidth" --gas-prices ${BASEFEE}abandwidth --keyring-backend "$KEYRING" --chain-id "$CHAINID" --home "$CHAINDIR"
  evmd genesis collect-gentxs --home "$CHAINDIR"
  evmd genesis validate-genesis --home "$CHAINDIR"

  echo ""
  echo "DONE. Genesis ready."

  echo ""
  echo "============================================================================"
  echo "SAVE THE FOLLOWING NOW — shown only this once:"
  echo "============================================================================"
  BECH32=$(evmd keys show "$VAL_KEY" -a --keyring-backend "$KEYRING" --home "$CHAINDIR")
  echo ""
  echo "--- Bech32 address ---"
  echo "$BECH32"
  echo ""
  echo "--- Ethereum (0x) address ---"
  evmd debug addr "$BECH32" --home "$CHAINDIR"
  echo ""
  echo "--- Private key (hex) — you will be asked for the keyring passphrase you just set ---"
  evmd keys export "$VAL_KEY" --unarmored-hex --unsafe --keyring-backend "$KEYRING" --home "$CHAINDIR"
  echo ""
  echo "--- Node ID (for peers to connect) ---"
  evmd comet show-node-id --home "$CHAINDIR"
  echo ""
  echo "--- Mnemonic ---"
  echo "(printed above, during key creation — scroll up if you missed it)"
  echo "============================================================================"
  echo ""
  echo "Next: ./uax_node.sh start"
}

do_start() {
  if [ -f "$PIDFILE" ] && kill -0 "$(cat "$PIDFILE")" 2>/dev/null; then
    echo ">>> Already running (pid $(cat "$PIDFILE"))."
    return 0
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
  if [ -f "$PIDFILE" ] && kill -0 "$(cat "$PIDFILE")" 2>/dev/null; then
    echo "Running (pid $(cat "$PIDFILE"))"
  else
    echo "Not running"
  fi
  curl -s http://localhost:26657/status 2>/dev/null | python3 -c "
import json,sys
try:
    d = json.load(sys.stdin)['result']['sync_info']
    print('latest_block_height:', d['latest_block_height'])
    print('catching_up:', d['catching_up'])
except Exception:
    print('(RPC not responding yet)')
" 2>/dev/null || echo "(RPC not responding yet)"
}

do_logs() {
  tail -f "$LOGFILE"
}

case "$cmd" in
  reset)  do_reset ;;
  start)  do_start ;;
  stop)   do_stop ;;
  status) do_status ;;
  logs)   do_logs ;;
  info)   print_info ;;
  *) echo "Usage: $0 {reset|start|stop|status|logs|info}"; exit 1 ;;
esac
