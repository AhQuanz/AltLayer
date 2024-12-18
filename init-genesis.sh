#!/bin/sh
rm -rf /root/.ethereum/geth
rm -rf /root/.ethereum/chaindata

# Initialize Geth with genesis.json if not already initialized
echo "Initializing Geth with genesis.json..."
geth --datadir /root/.ethereum init /root/genesis.json

# Start Geth with the desired configuration
echo "Starting Geth..."
geth --datadir /root/.ethereum \
      --http --http.addr "0.0.0.0" --http.port 8545 \
      --http.api "web3,eth,net,debug,personal,miner,admin" \
      --allow-insecure-unlock \
      --networkid 12345