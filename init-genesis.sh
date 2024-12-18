#!/bin/sh
MARKER_FILE="/root/.ethereum/init.marker"

# Initialize Geth with genesis.json if not already initialized
if [ ! -f "$MARKER_FILE" ]; then
    echo "Initializing Geth with genesis.json..."
    rm -rf /root/.ethereum/geth
    rm -rf /root/.ethereum/chaindata
    geth --datadir /root/.ethereum init /root/genesis.json
    # Create the marker file after successful initialization
    touch "$MARKER_FILE"
else
    echo "Geth already initialized. Skipping initialization."
fi

# Start Geth with the desired configuration
echo "Starting Geth..."

exec geth --datadir /root/.ethereum \
      --http --http.addr "0.0.0.0" --http.port 8545 \
      --http.corsdomain "*" \
      --http.api "web3,eth,net,debug,personal,miner,admin" \
      --allow-insecure-unlock \
      --networkid 12345
      --evm-version cancun \
      --mine --miner.threads=1