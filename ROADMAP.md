## Phase 1: Narrative Structure (Sagas) - **COMPLETED**
**Objective**: Group episodes into cohesive narratives to support long-term context and sequential reasoning.

- [x] Create `SagaNode` struct.
- [x] Add Cypher queries for Sagas.
- [x] Update `Graphiti.AddEpisode` to handle `saga` argument.
- [x] Implement logic to link episodes and sagas.

## Phase 2: Temporal Truth (Edge Invalidation) - **PARTIALLY COMPLETED**
**Objective**: Maintain a consistent view of facts over time.

- [x] Add `ValidAt`/`InvalidAt` fields (already in model).
- [x] Implement `InvalidateEdgeQuery`.
- [x] Implement `dedupe` logic to prevent exact duplicate edges.
- [x] Implement LLM-based contradiction detection.

## Phase 3: Search Relevance (Reranking) - **COMPLETED**
**Objective**: Improve search result quality using cross-encoders.

- [x] Define `RerankerClient` interface.
- [x] Implement `SimpleLLMReranker`.
- [x] Update `Graphiti.Search` to use reranker and fetch more candidates.

## Phase 4: High-Level Insight (Community Detection) - **COMPLETED**
**Objective**: Generate summary insights for clusters of related entities.

1.  **Algorithm**:
    -   Implemented client-side Community Detection (Simple Connected Components) in `internal/core/community`.
    -   Added API endpoint `/communities/detect` to trigger detection.
2.  **Community Node**:
    -   Defined `CommunityNode` model.
    -   Implemented logic to save Community nodes and `HAS_MEMBER` edges.
3.  **Summarization**:
    -   Implemented `SummarizeCommunity` in `summarizer.go`.
    -   Added `Did DetectAndSummarizeCommunities` method to Graphiti.

## Phase 5: Flexibility (Custom Schemas) - **PENDING**
**Objective**: Allow users to define their own entity types and attributes.

1.  **Refactor EntityNode**:
    -   Change `Attributes` from `string` (JSON) to `map[string]interface{}` in the struct.
    -   Update `Extractor` to accept a JSON Schema or list of fields to extract.
2.  **Dynamic Extraction**:
    -   Modify `ExtractionPrompts` to inject the user's custom schema definition into the LLM prompt.
    -   Ensure validation of dynamic attributes before saving.

## Phase 6: Bulk Operations & Optimization - **PENDING**
**Objective**: Improve performance for large-scale ingestion.

1.  **Bulk API**:
    -   Add `AddEpisodes_Bulk` method.
    -   Use `UNWIND` Cypher clauses to batch insert nodes/edges in single transactions.
2.  **Parallelism**:
    -   Utilize Go's goroutines for parallel extraction of multiple episodes (with semaphore limiting).
