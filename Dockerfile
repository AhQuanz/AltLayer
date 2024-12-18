# Stage 1: Build the Go application
FROM golang:1.23 AS builder

WORKDIR /app

# Copy Go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod tidy

# Copy the source code and build the Go application
COPY . .
# RUN go build -o assignment main.go

# Stage 2: Run the Go application
FROM ubuntu:22.04

# Install dependencies for runtime
RUN apt-get update && apt-get install -y \
    curl \
    bash \
    && rm -rf /var/lib/apt/lists/*

# Copy the built binary from Stage 1
# COPY --from=builder /app/assignment /usr/local/bin/assignment

# Copy the genesis.json file from the local machine into the container
COPY genesis.json /root/genesis.json
COPY init-genesis.sh /root/init-genesis.sh

RUN chmod +x /root/init-genesis.sh

COPY init_db.sql /docker-entrypoint-initdb.d/
COPY init_db.sh /docker-entrypoint-initdb.d/

# Expose the port for the Go app
# EXPOSE 8080

# Run the Go application
# CMD ["/usr/local/bin/assignment"]
