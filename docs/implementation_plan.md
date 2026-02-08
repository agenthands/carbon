# Implementation Plan - Port Graphiti to Go

This plan outlines the steps to port the Graphiti application from Python (Neo4j) to Go (Memgraph).

## User Review Required
> [!IMPORTANT]
> **Database Selection:** The port will target **Memgraph** as the graph database. Memgraph is Bolt-compatible, so we will use the standard Neo4j Go driver (`github.com/neo4j/neo4j-go-driver/v5`).
> **Project Structure:** We will create a `go` directory within the existing repository to house the Go implementation, keeping it isolated from the Python code during the transition.
> **LLM/Embedder:** We will use `github.com/agenthands/dspy-go` for LLM and embedding interactions, connecting to a local Ollama instance running `gpt-oss:latest`.

## Proposed Changes

### 1. Project Initialization & Structure
We will establish a standard Go project structure within a `go/` directory.

#### [NEW] [go/go.mod](file:///Users/Janis_Vizulis/go/src/github.com/getzep/graphiti/go/go.mod)
- Initialize module `github.com/getzep/graphiti/go`.
- Dependencies: `neo4j-go-driver`, `dspy-go`, `gin` (or `echo`), `godotenv`.

#### [NEW] [go/cmd/server/main.go](file:///Users/Janis_Vizulis/go/src/github.com/getzep/graphiti/go/cmd/server/main.go)
- Entry point for the server.

### 2. Core Domain Models (`go/internal/core/model`)
Define the structs representing the graph elements.

#### [NEW] [go/internal/core/model/node.go](file:///Users/Janis_Vizulis/go/src/github.com/getzep/graphiti/go/internal/core/model/node.go)
- Structs: `EntityNode`, `EpisodicNode`, `CommunityNode`, `SagaNode`.

#### [NEW] [go/internal/core/model/edge.go](file:///Users/Janis_Vizulis/go/src/github.com/getzep/graphiti/go/internal/core/model/edge.go)
- Structs: `EntityEdge` (RELATES_TO), `EpisodicEdge` (MENTIONS), `CommunityEdge` (HAS_MEMBER).

### 3. Database Driver (`go/internal/driver`)
Implement the database connection and query execution using the Neo4j Go driver, tailored for Memgraph.

#### [NEW] [go/internal/driver/memgraph.go](file:///Users/Janis_Vizulis/go/src/github.com/getzep/graphiti/go/internal/driver/memgraph.go)
- Wrapper around `neo4j.DriverWithContext`.
- Methods for `ExecuteQuery`, `BuildIndices`.
- Memgraph-specific vector index creation syntax.

#### [NEW] [go/internal/driver/queries.go](file:///Users/Janis_Vizulis/go/src/github.com/getzep/graphiti/go/internal/driver/queries.go)
- Port usage of Cypher queries from `graphiti_core/graph_queries.py`.
- Ensure compatibility with Memgraph.

### 4. LLM & Embedder Clients (`go/internal/llm`)
Implement clients for generating embeddings and extracting graph data using `dspy-go`.

#### [NEW] [go/internal/llm/client.go](file:///Users/Janis_Vizulis/go/src/github.com/getzep/graphiti/go/internal/llm/client.go)
- Interfaces for `LLMClient` and `EmbedderClient` (wrappers around `dspy-go`'s `OllamaLLM`).
- Implementation using `dspy-go/pkg/llms` configured for Ollama (`gpt-oss:latest`).

### 5. Graphiti Logic (`go/internal/core`)
Port the main business logic from `graphiti.py`.

#### [NEW] [go/internal/core/graphiti.go](file:///Users/Janis_Vizulis/go/src/github.com/getzep/graphiti/go/internal/core/graphiti.go)
- Struct `Graphiti`.
- Methods:
    - `AddEpisode`: Orchestrates extraction, deduplication, and saving.
    - `ExtractNodesAndEdges`: Calls LLM to parse text (using DSPy modules if applicable, or direct prompt).
    - `Search`: Implements hybrid search (Text + Vector).

### 6. API Server (`go/internal/server`)
Implement the REST API.

#### [NEW] [go/internal/server/router.go](file:///Users/Janis_Vizulis/go/src/github.com/getzep/graphiti/go/internal/server/router.go)
- Setup Gin/Echo router.
- Middleware (Logger, Auth if needed).

#### [NEW] [go/internal/server/handlers.go](file:///Users/Janis_Vizulis/go/src/github.com/getzep/graphiti/go/internal/server/handlers.go)
- Handlers for:
    - `POST /messages` (Ingest)
    - `POST /search`
    - `GET /episodes/{group_id}`
    - `DELETE /...`

## Verification Plan

### Automated Tests
1.  **Unit Tests**:
    - Test `ExtractNodesAndEdges` with mocked LLM responses.
    - Test Model serialization/deserialization.
2.  **Integration Tests**:
    - Spin up a Memgraph Podman container.
    - Test `Driver` connection and basic queries.
    - Test `AddEpisode` flow end-to-end (Real Ollama, Real DB).

### Manual Verification
1.  **Setup**:
    - Run `podman run -p 7687:7687 -p 7444:7444 memgraph/memgraph-mage`.
    - Run the Go server: `go run cmd/server/main.go`.
2.  **Ingest Test**:
    - Send a POST request to `/messages` with a sample conversation.
    - Verify data in Memgraph using Memgraph Lab or Cypher shell.
3.  **Search Test**:
    - Send a POST request to `/search` with a query.
    - Verify relevant results are returned.
