# Gap Analysis: Graphiti (Python) vs. Carbon (Go) - Review 2

This document outlines the gaps identified after a second comprehensive review of the reference implementation (`graphiti_core`) against the current Go implementation.

## 1. Core Logic & Architecture

| Feature | Python (`graphiti_core`) | Go (`carbon`) | Gap Severity | Description |
| :--- | :--- | :--- | :--- | :--- |
| **Bulk Operations** | True Batch Processing | Concurrent Simple Operations | **High** | Python performs bulk extraction, deduplication across the batch, and resolution before writing. Go simply runs `AddEpisode` concurrently, which risks race conditions for shared entities in the batch. |
| **Context Awareness** | Stateful | Stateless | **Medium** | Python passes `previous_episodes` to `extract_nodes` for coreference resolution. Go treats each episode in isolation. |

## 2. Community Detection & Summarization

| Feature | Python (`graphiti_core`) | Go (`carbon`) | Gap Severity | Description |
| :--- | :--- | :--- | :--- | :--- |
| **Detection Algorithm** | Label Propagation (LPA) | Simple Connected Components (DFS) | **High** | Python uses a robust community detection algo (LPA). Go uses a very basic DFS approach which only finds disconnected islands, not dense clusters. |
| **Summarization Logic** | Iterative Pair-wise | Single Pass | **Medium** | Python summarizes large communities iteratively (MapReduce style) to fit context windows. Go attempts to summarize all nodes at once, which will fail for large communities. |
| **Community Naming** | LLM Generated | Generic ("Community X") | **Low** | Python generates descriptive names for communities. Go uses generic numbering. |

## 3. Search & Retrieval

| Feature | Python (`graphiti_core`) | Go (`carbon`) | Gap Severity | Description |
| :--- | :--- | :--- | :--- | :--- |
| **Return Type** | structured `EntityEdge` objects | list of `string` facts | **Medium** | Python's `search_` returns graph objects allowing traversal. Go returns pre-formatted strings. |
| **Ranking** | RRF / Cross-Encoder | Simple Hybrid | **Medium** | Python supports Reciprocal Rank Fusion and Cross-Encoder re-ranking. Go uses a simple dot product + text match hybrid. |
| **Filters** | robust `SearchFilters` | Basic | **Low** | Python has more advanced filtering capabilities. |

## 4. Data Models

| Feature | Python (`graphiti_core`) | Go (`carbon`) | Gap Severity | Description |
| :--- | :--- | :--- | :--- | :--- |
| **Edge Attributes** | Flattened Properties | JSON String | **Low** | Python flattens `attributes` dict into edge properties. Go serializes `attributes` as a single JSON string property. This affects query capability on attributes. |

## Recommendations for Next Steps

1.  **Refactor Bulk Ops**: Implement true batch extraction and deduplication to match Python's robustness.
2.  **Upgrade Community Detection**: Implement Label Propagation or use Memgraph's built-in MAGE library.
3.  **Enhance Search**: Return structured `EntityEdge` objects and implement RRF ranking.
4.  **Add Context**: Update `AddEpisode` to accept and use previous episode context.
