#!/bin/sh
rm -rf /root/.ethereum/geth
rm -rf /root/.ethereum/chaindata
MARKER_FILE="/root/.ethereum/init.marker"

# Initialize Geth with genesis.json if not already initialized
if [ ! -f "$MARKER_FILE" ]; then
    echo "Initializing Geth with genesis.json..."
    geth --datadir /root/.ethereum init /root/genesis.json
    # Create the marker file after successful initialization
    touch "$MARKER_FILE"
else
    echo "Geth already initialized. Skipping initialization."
fi

# Start Geth with the desired configuration
echo "Starting Geth..."

geth --datadir /root/.ethereum \
      --http --http.addr "0.0.0.0" --http.port 8545 \
      --http.corsdomain "https://remix.ethereum.org" \
      --http.api "web3,eth,net,debug,personal,miner,admin" \
      --allow-insecure-unlock \
      --networkid 12345

if [! -f "$MARKER_FILE"]; then
  echo "DEPLOYING CONTRACT"