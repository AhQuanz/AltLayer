#!/bin/bash

# Start MySQL in the background
service mysql start

# Wait for MySQL to be ready (you can adjust the timeout as needed)
echo "Waiting for MySQL to be ready..."
until mysqladmin ping -h "localhost" --silent; do
  sleep 1
done
echo "MySQL is up and running!"

# Start Geth in the background
geth --datadir /root/.ethereum \
     --http --http.addr "0.0.0.0" --http.port 8545 \
     --http.api "web3,eth,net,debug,personal,miner,admin" \
     --allow-insecure-unlock --dev &

# Wait for Geth to be ready (waiting for HTTP server to start)
echo "Waiting for Geth to be ready..."
until curl -s http://localhost:8545 > /dev/null; do
  sleep 1
done
echo "Geth is up and running!"

# Finally, run the Go application
/usr/local/bin/assignment
