# Walkthrough - Porting Graphiti to Go (Carbon)

This document outlines the changes made to port Graphiti to Go, now named **Carbon**, and how to run and verify the new implementation.

## 1. Project Structure

The project is located in `go/` and is initialized as the module `github.com/agenthands/carbon`.

- `cmd/server/main.go`: The entry point for the API server.
- `internal/core/model`: Defines the core domain models (`EntityNode`, `EpisodicNode`, etc.).
- `internal/driver`: Contains the `MemgraphDriver` wrapper and Cypher queries.
- `internal/llm`: Contains the `OllamaClient` using `dspy-go`.
- `internal/server`: Implements the REST API handlers (`AddMessages`, `Search`).

## 2. Prerequisites

- **Go 1.23+**
- **Memgraph**: Ensure a Memgraph instance is running.
  - Using Podman: `podman run -p 7687:7687 -p 7444:7444 memgraph/memgraph-mage`
- **Ollama**: Ensure Ollama is running with `gpt-oss:latest`.
  - `ollama run gpt-oss:latest`

## 3. Running the Server

1.  Navigate to the `go` directory:
    ```bash
    cd go
    ```
2.  Set environment variables (optional, defaults provided):
    ```bash
    export MEMGRAPH_URI="bolt://localhost:7687"
    export OLLAMA_BASE_URL="http://localhost:11434"
    ```
3.  Run the server:
    ```bash
    go run cmd/server/main.go
    ```

## 4. Verification

### Automated Integration Test
A script is provided to test ingestion and search end-to-end.
```bash
go run cmd/test_integration/main.go
```

**Note:** The integration test requires a running Memgraph instance.

### Manual Verification Steps

1.  **Ingest a Message**:
    ```bash
    curl -X POST http://localhost:8080/messages \
      -H "Content-Type: application/json" \
      -d '{
        "group_id": "test-group",
        "messages": [
          {"role": "user", "content": "Hello, my name is Bob."}
        ]
      }'
    ```
2.  **Search**:
    ```bash
    curl -X POST http://localhost:8080/search \
      -H "Content-Type: application/json" \
      -d '{
        "group_id": "test-group",
        "query": "Bob"
      }'
    ```

## 5. Repository

The code has been pushed to [agenthands/carbon](https://github.com/agenthands/carbon).
