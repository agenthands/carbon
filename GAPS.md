# Feature Parity Gap Analysis

This document outlines the feature gaps between the current Go implementation (`go/internal`) and the reference Python implementation (`graphiti_core`).

## 1. Core Graph Logic

### Sagas
**Python**: Supports `SagaNode` to group related episodes into a narrative arc. Handles `HAS_EPISODE` and `NEXT_EPISODE` relationships within a Saga.
**Go**: **MISSING**. Episodes are standalone or loosely grouped by `group_id`. No Saga structure.

### Community Detection
**Python**: Includes `build_communities` and `update_community` to cluster nodes (using Leipniz or similar) and generate high-level summaries.
**Go**: **MISSING**. Entities are individual; no higher-order clustering.

### Edge Invalidation (Temporal Truth)
**Python**: `resolve_extracted_edges` and `invalidated_edges` logic. When a new fact contradicts an old one (based on time), the old edge is marked invalid (`invalid_at`).
**Go**: **MISSING**. New edges are just added. Old edges remain valid indefinitely.

## 2. Search & Retrieval

### Cross-Encoder Reranking
**Python**: Supports `CrossEncoderClient` (OpenAI, Gemini, BGE) to rerank semantic search results for higher relevance.
**Go**: **MISSING**. Returns raw search results from the database (text or simple vector match).

### Advanced Search Recipes
**Python**: Configurable search strategies (Hybrid, RRF - Reciprocal Rank Fusion, Node Distance).
**Go**: **PARTIAL**. Basic Hybrid (Text + Vector) is sketched but not fully robust. No RRF.

## 3. Data & Schema

### Custom Entity/Edge Schemas
**Python**: Allows passing Pydantic models (`entity_types`, `edge_types`) to define custom nodes/edges dynamically.
**Go**: **MISSING**. `EntityNode` structure is hardcoded in `internal/core/model`.

### Bulk Operations
**Python**: `add_nodes_and_edges_bulk` for efficient batch ingestion.
**Go**: **MISSING**. `AddEpisode` processes one episode at a time.

## 4. Drivers & Integrations

### Database Drivers
**Python**: Neo4j, FalkorDB, Kuzu, Amazon Neptune.
**Go**: Memgraph (via Neo4j driver). Verified to work with Memgraph, likely works with Neo4j but untested. FalkorDB/Kuzu/Neptune unsupported.

### LLM Clients
**Python**: Extensive native client support (OpenAI, Azure, Gemini, Anthropic, Groq).
**Go**: OpenAI (supports Ollama), Gemini. Others missing native clients (though OpenAI compat helps).

## 5. Miscellaneous

- **Telemetry**: Python has anonymous usage tracking. Go does not (low priority).
- **Async/Concurrency**: Python uses `asyncio` with semaphores. Go uses synchronous calls currently (though Go's concurrency model makes this easy to add later).
