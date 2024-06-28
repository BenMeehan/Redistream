# Distributed Key-Value Store (Redis Clone)

A Key-Value Store implementing the Redis Server protocol, RDB Persistence, Distributed clustering, and Streaming.

## Table of Contents

1. [Introduction](#introduction)
2. [Features](#features)
3. [Installation](#installation)
4. [Usage](#usage)
5. [Supported Commands](#supported-commands)

## Introduction

This project is a Key-Value Store designed to implement the Redis Server protocol. It provides functionalities including RDB persistence, distributed clustering, and data streaming, making it suitable for high-performance and scalable applications.

## Features

- **Redis Server Protocol**: Fully compatible with the Redis server protocol, allowing seamless integration with existing Redis clients.
- **RDB Persistence**: Efficiently persists data to disk using the RDB format, ensuring data durability and reliability.
- **Distributed Clustering**: Supports distributed clustering using master-slave architecture for high availability and fault tolerance.
- **Data Streaming**: Implements data streaming capabilities for real-time data processing.

## Installation

To install and set up the project, follow these steps:

```bash
# Clone the repository
git clone https://github.com/yourusername/projectname.git

# Navigate to the project directory
cd projectname

# Install dependencies (if any)
go mod tidy
```

## Usage

Instructions on how to build and run the project:

```bash
# Build the project
go build -o kvstore

# Run the project
./kvstore
```
## Supported Commands

1. **SET**: Sets the value of a key.
    - **Usage**: `SET key value`
    - **Example**: `SET mykey "Hello, World!"`

2. **GET**: Gets the value of a key.
    - **Usage**: `GET key`
    - **Example**: `GET mykey`

3. **DEL**: Deletes one or more keys.
    - **Usage**: `DEL key [key ...]`
    - **Example**: `DEL mykey`

4. **EXISTS**: Checks if a key exists.
    - **Usage**: `EXISTS key`
    - **Example**: `EXISTS mykey`

5. **PING**: Tests if the server is running.
    - **Usage**: `PING`
    - **Example**: `PING`

6. **SAVE**: Synchronously saves the dataset to disk.
    - **Usage**: `SAVE`
    - **Example**: `SAVE`

7. **INFO**: Provides information and statistics about the server.
    - **Usage**: `INFO`
    - **Example**: `INFO`

8. **LPUSH**: Inserts a value at the head of a list.
    - **Usage**: `LPUSH key value`
    - **Example**: `LPUSH mylist "Hello"`

9. **RPUSH**: Inserts a value at the tail of a list.
    - **Usage**: `RPUSH key value`
    - **Example**: `RPUSH mylist "World"`

10. **LPOP**: Removes and returns the first element of a list.
    - **Usage**: `LPOP key`
    - **Example**: `LPOP mylist`

11. **RPOP**: Removes and returns the last element of a list.
    - **Usage**: `RPOP key`
    - **Example**: `RPOP mylist`

12. **XADD**: Appends a new entry to a stream.
    - **Usage**: `XADD stream key value`
    - **Example**: `XADD mystream * field1 value1`

13. **XRANGE**: Returns a range of elements in a stream.
    - **Usage**: `XRANGE stream start end`
    - **Example**: `XRANGE mystream - +`
