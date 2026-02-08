# Carbon

Carbon is a Go port of the Graphiti library, designed to work with **Memgraph** as the graph database backend and **Ollama** (via [dspy-go](https://github.com/XiaoConstantine/dspy-go)) for LLM and embedding capabilities.

## Overview

Carbon provides a knowledge graph layer for LLM applications, allowing for the ingestion of conversation history, extraction of entities and relationships, and semantic search.

## Prerequisites

- **Go 1.23+**
- **Memgraph Platform**:
  - Run using Docker/Podman: `podman run -p 7687:7687 -p 7444:7444 memgraph/memgraph-mage`
- **Ollama**:
  - Ensure `gpt-oss:latest` (or your preferred model) is available: `ollama run gpt-oss:latest`

## Getting Started

1.  **Clone the repository**:
    ```bash
    git clone https://github.com/agenthands/carbon.git
    cd carbon
    ```

2.  **Configuration**:
    Set the following environment variables (optional, defaults shown):
    ```bash
    export MEMGRAPH_URI="bolt://localhost:7687"
    export MEMGRAPH_USER=""
    export MEMGRAPH_PASSWORD=""
    export OLLAMA_BASE_URL="http://localhost:11434"
    export LLM_MODEL="gpt-oss:latest"
    export PORT="8080"
    ```

3.  **Run the Server**:
    ```bash
    go run cmd/server/main.go
    ```

4.  **Test**:
    Use the provided integration test script:
    ```bash
    go run cmd/test_integration/main.go
    ```

## Documentation

See the [docs/](docs/) directory for detailed planning and walkthrough documents:
- [Implementation Plan](docs/implementation_plan.md)
- [Walkthrough](docs/walkthrough.md)
