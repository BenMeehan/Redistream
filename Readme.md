# Redis Server Clone

This is a simple implementation of a Redis server in Go. It supports basic commands such as `PING`, `ECHO`, `SET`, and `GET`, along with the ability to set key expiry using the `PX` argument to the `SET` command.

> This project was done as a part of CodeCrafters *Build Your Own Redis* Challenge.

## Prerequisites

- Go (Golang) installed on your machine
- Basic knowledge of the Redis protocol (RESP)

## Usage

1. Clone the repository:

   ```bash
   git clone https://github.com/yourusername/redis-server.git
   ```

2. Navigate to the project directory:

   ```bash
   cd redis-server
   ```

3. Build the executable:

   ```bash
   go build
   ```

4. Run the server:

   ```bash
   ./redis-server
   ```

5. Connect to the server using a Redis client (e.g., `redis-cli`) and start issuing commands:

   ```bash
   redis-cli
   ```

   Example commands:

   ```bash
   set mykey myvalue px 1000
   get mykey
   ```

## Supported Commands

- `PING`: Responds with `PONG` to indicate that the server is running.
- `ECHO`: Echoes back the provided argument.
- `SET`: Sets a key to a value. Supports optional expiry using the `PX` argument.
- `GET`: Retrieves the value of a key. Handles key expiry automatically.

## License
This project is licensed under the MIT License. See the [LICENSE](./LICENSE) file for details.