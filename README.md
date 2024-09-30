
# SageSlayerServer

SageSlayerServer is a TCP server that implements a Proof-of-Work (PoW) mechanism to protect against DDoS attacks. 
The server sends a challenge to clients, who must solve the challenge before receiving a "Word of Wisdom" quote. 
This project includes rate limiting, error tracking, and dynamic difficulty adjustment based on client behavior.

## Features

- **Proof-of-Work (PoW) Challenge**: The server uses PoW to ensure that clients solve a computational puzzle before receiving a response.
- **Rate Limiting**: Limits the number of requests and connections from individual clients based on a time-decaying model.
- **Error Tracking**: Tracks client errors and increases the difficulty for clients with excessive errors.
- **Ban Mechanism**: Clients are temporarily banned if they generate too many errors or requests within a short period.
- **Protobuf Serialization**: Messages between the server and client are serialized using Protocol Buffers.
- **Dockerized Setup**: The server and client are containerized using Docker for ease of deployment.

## Installation

1. **Clone the repository**:

   ```bash
   git clone https://github.com/painhardcore/SageSlayerServer.git
   cd SageSlayerServer
   ```

2. **Build the server and client**:

   ```bash
   go build -o server ./cmd/server
   go build -o client ./cmd/client
   ```

3. **Run the server**:

   ```bash
   ./server
   ```
   Possible flags:
   ```bash
   Usage of ./server:
    -addr string
        TCP address to listen on (default ":8000")
   ```

4. **Run the client**:

   ```bash
   ./client -server-addr localhost:8000
   ```
   Possible flags:
   ```bash
   Usage of ./client:
   -attack
         Enable attack mode to simulate constant request
   -interval duration
         Interval between requests in attack mode (e.g., 10s, 500ms)
   -server-addr string
         Server address (default "localhost:8000")
   ```

## Docker Setup

To run the server and client in Docker containers:

### Build Docker Images

```bash
docker build -t sss-server -f Dockerfile.server .
docker build -t sss-client -f Dockerfile.client .
```

### Create Docker Network

```bash
docker network create sss-network
```

### Run Server Container

```bash
docker run -d --name sss-server --network sss-network -p 8000:8000 sss-server
```

### Run Client Container

```bash
docker run --rm --network sss-network sss-client -server-addr sss-server:8000
```

### Run Legitimate Client (1 Request Every 10 Seconds)

```bash
docker run --rm --network sss-network sss-client -server-addr sss-server:8000 -attack -interval 10s
```