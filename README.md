# Carbon

Carbon is a complete Go port of the Graphiti library, providing a robust Knowledge Graph layer for LLM applications. It leverages **Memgraph** for graph storage and **Ollama** (via [dspy-go](https://github.com/XiaoConstantine/dspy-go)) for extraction, deduplication, and summarization.

## Features (100% Parity)

- **Entity & Edge Extraction**: Extracts structured entities (Person, Place, Organization, etc.) and semantic relationships from unstructured text using LLM prompts.
- **Intelligent Deduplication**: Resolves duplicate entities by comparing semantic similarity (embeddings) and LLM-based reasoning.
- **Iterative Summarization**: Automatically updates entity summaries as new information (mentions) is ingested.
- **Hybrid Search**: Combines full-text search with vector similarity search to retrieve the most relevant context.
- **Temporal Graph**: Tracks valid time and transaction time for all nodes and edges.

## Architecture

Carbon follows a modular architecture:
- `internal/core`: Core logic including `Graphiti` service, extraction, deduplication, and summarization modules.
- `internal/driver`: MEMGRAPH driver wrapper and Cypher query definitions.
- `internal/llm`: Interface and implementation for LLM and Embedding services (Ollama).
- `cmd/server`: HTTP server entry point.

## Prerequisites

- **Go 1.23+**
- **Memgraph Platform**:
  - Run using Docker/Podman: `podman run -p 7687:7687 -p 7444:7444 memgraph/memgraph-mage`
- **Ollama**:
  - Ensure `gpt-oss:latest` (or your preferred small/efficient model) is available.

## Getting Started

1.  **Clone the repository**:
    ```bash
    git clone https://github.com/agenthands/carbon.git
    cd carbon
    ```


2.  **Configuration**:
    Carbon uses `config/config.toml` for prompts and LLM settings.
    You can also use environment variables to override LLM settings.

    Example `config/config.toml`:
    ```toml
    [llm]
    provider = "openai" # ollama, openai, gemini, claude
    model = "gpt-4o"
    api_key = "sk-..."
    base_url = "" # Optional, for Ollama or custom OpenAI proxies
    ```

    Or environment variables:
    ```bash
    export LLM_PROVIDER="openai"
    export LLM_MODEL="gpt-4o"
    export LLM_API_KEY="sk-..."
    ```

    Memgraph connection is configured via env vars:
    ```bash
    export MEMGRAPH_URI="bolt://localhost:7687"
    export MEMGRAPH_USER=""
    export MEMGRAPH_PASSWORD=""
    export PORT="8080"
    ```

3.  **Run the Server**:
    ```bash
    go run cmd/server/main.go
    ```

4.  **Test**:
    Run unit tests to verify core logic:
    ```bash
    go test ./internal/core/...
    ```

## CLI Usage

Carbon can be used as a library or via the provided server.

### Example: Adding an Episode
Send a POST request to `/episodes` with the conversation content. The extracting, deduplication, and linking process happens automatically.

### Example: Search
Send a GET request to `/search?q=query` to retrieve relevant entities and summaries.

## Documentation

See the [docs/](docs/) directory for detailed planning and walkthrough documents:
- [Implementation Plan](docs/implementation_plan.md)
- [Walkthrough](docs/walkthrough.md)
