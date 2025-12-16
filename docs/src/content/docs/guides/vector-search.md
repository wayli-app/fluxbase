---
title: "Vector Search"
description: "Semantic search with pgvector embeddings"
---

Fluxbase provides vector search capabilities powered by [pgvector](https://github.com/pgvector/pgvector), enabling semantic similarity search with automatic text embeddings.

## Overview

Vector search in Fluxbase enables:

- **Semantic Search**: Find similar content based on meaning, not just keywords
- **Automatic Embeddings**: Convert text to vectors using OpenAI, Azure, or Ollama
- **Multiple Distance Metrics**: L2 (Euclidean), Cosine, and Inner Product similarity
- **Combined Filtering**: Use vector search alongside standard SQL filters
- **Index Support**: HNSW and IVFFlat indexes for fast similarity search

Common use cases include document search, recommendation systems, question answering, image similarity, and any application requiring "find similar" functionality.

## Prerequisites

Vector search requires the pgvector extension to be installed on your PostgreSQL database:

```sql
-- Enable pgvector extension
CREATE EXTENSION IF NOT EXISTS vector;
```

:::tip[Check Extension Status]
You can check if pgvector is installed via the admin API:
```bash
curl http://localhost:8080/api/v1/admin/extensions/vector/status \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN"
```
Or use the capabilities endpoint:
```bash
curl http://localhost:8080/api/v1/capabilities/vector
```
:::

## Configuration

### Embedding Provider

To enable automatic text-to-vector conversion, configure an embedding provider:

```yaml
# fluxbase.yaml
ai:
  embedding_enabled: true
  embedding_provider: openai  # openai, azure, or ollama
  embedding_model: text-embedding-3-small
  openai_api_key: ${OPENAI_API_KEY}
```

Or via environment variables:

```bash
FLUXBASE_AI_EMBEDDING_ENABLED=true
FLUXBASE_AI_EMBEDDING_PROVIDER=openai
FLUXBASE_AI_EMBEDDING_MODEL=text-embedding-3-small
FLUXBASE_AI_OPENAI_API_KEY=sk-...
```

### Supported Providers

| Provider | Models | Dimensions |
|----------|--------|------------|
| OpenAI | `text-embedding-3-small`, `text-embedding-3-large`, `text-embedding-ada-002` | 1536, 3072, 1536 |
| Azure OpenAI | Deployment-based | Varies |
| Ollama | `nomic-embed-text`, `mxbai-embed-large`, `all-minilm` | 768, 1024, 384 |

## Quick Start

### 1. Create a Table with Vector Column

```sql
-- Create a documents table with a vector column
CREATE TABLE documents (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  title TEXT NOT NULL,
  content TEXT NOT NULL,
  embedding vector(1536),  -- 1536 dimensions for OpenAI text-embedding-3-small
  created_at TIMESTAMPTZ DEFAULT now()
);

-- Create an index for fast similarity search
CREATE INDEX ON documents USING hnsw (embedding vector_cosine_ops);
```

### 2. Generate and Store Embeddings

```typescript
import { createClient } from '@fluxbase/sdk'

const client = createClient('http://localhost:8080', 'your-api-key')

// Generate embedding for a document
const { data: embedResult } = await client.vector.embed({
  text: 'Introduction to machine learning and neural networks'
})

// Insert document with embedding
await client.from('documents').insert({
  title: 'ML Basics',
  content: 'Introduction to machine learning and neural networks',
  embedding: embedResult.embeddings[0]
})
```

### 3. Search for Similar Documents

```typescript
// Search using text query (auto-embedded)
const { data: results } = await client.vector.search({
  table: 'documents',
  column: 'embedding',
  query: 'How do neural networks work?',
  match_count: 10,
  match_threshold: 0.7
})

console.log('Similar documents:', results.data)
console.log('Distances:', results.distances)
```

## SDK Usage

Fluxbase provides three approaches for vector search:

### 1. Convenience API (`client.vector`)

The simplest approach with automatic embedding:

```typescript
// Generate embeddings
const { data: embedResult } = await client.vector.embed({
  text: 'Hello world'
})
console.log('Embedding:', embedResult.embeddings[0])
console.log('Dimensions:', embedResult.dimensions)

// Batch embedding
const { data: batchResult } = await client.vector.embed({
  texts: ['Hello', 'World', 'Fluxbase']
})

// Semantic search with auto-embedding
const { data: searchResult } = await client.vector.search({
  table: 'documents',
  column: 'embedding',
  query: 'machine learning tutorials',  // Will be auto-embedded
  metric: 'cosine',
  match_count: 10,
  match_threshold: 0.78,
  select: 'id, title, content',
  filters: [
    { column: 'created_at', operator: 'gte', value: '2024-01-01' }
  ]
})
```

### 2. Tables API (`client.from`)

For more control, use vector operators directly in queries:

```typescript
// First, get the query vector
const { data: embed } = await client.vector.embed({ text: 'search query' })
const queryVector = embed.embeddings[0]

// Search using Tables API with vector operators
const { data } = await client
  .from('documents')
  .select('id, title, content')
  .filter('embedding', 'vec_cos', queryVector)  // Cosine distance
  .order('embedding', {
    vector: queryVector,
    metric: 'cosine'
  })
  .limit(10)
  .execute()
```

#### Vector Operators

| Operator | Description | SQL Operator |
|----------|-------------|--------------|
| `vec_l2` | L2/Euclidean distance | `<->` |
| `vec_cos` | Cosine distance | `<=>` |
| `vec_ip` | Negative inner product | `<#>` |

All operators return a distance where **lower values = more similar**.

### 3. RPC Pattern (Supabase-compatible)

For complex queries, create a PostgreSQL function:

```sql
-- Create a match_documents function
CREATE OR REPLACE FUNCTION match_documents(
  query_embedding vector(1536),
  match_threshold float DEFAULT 0.78,
  match_count int DEFAULT 10
)
RETURNS TABLE (
  id UUID,
  title TEXT,
  content TEXT,
  similarity float
)
LANGUAGE plpgsql
AS $$
BEGIN
  RETURN QUERY
  SELECT
    d.id,
    d.title,
    d.content,
    1 - (d.embedding <=> query_embedding) AS similarity
  FROM documents d
  WHERE 1 - (d.embedding <=> query_embedding) > match_threshold
  ORDER BY d.embedding <=> query_embedding
  LIMIT match_count;
END;
$$;
```

Then create an RPC procedure (`rpc/default/match_documents.sql`):

```sql
-- @fluxbase:name match_documents
-- @fluxbase:public true
-- @fluxbase:input query_embedding:vector, match_threshold:float, match_count:int
-- @fluxbase:output id:uuid, title:text, content:text, similarity:float

SELECT * FROM match_documents($query_embedding::vector, $match_threshold, $match_count);
```

Call via SDK:

```typescript
// Get embedding for query
const { data: embed } = await client.vector.embed({ text: 'search query' })

// Call RPC function
const { data } = await client.rpc('match_documents', {
  query_embedding: embed.embeddings[0],
  match_threshold: 0.78,
  match_count: 10
})
```

## Indexing

For production workloads, create indexes on vector columns:

### HNSW Index (Recommended)

Hierarchical Navigable Small World - best for most use cases:

```sql
-- For cosine distance (most common)
CREATE INDEX ON documents
  USING hnsw (embedding vector_cosine_ops);

-- For L2 distance
CREATE INDEX ON documents
  USING hnsw (embedding vector_l2_ops);

-- For inner product
CREATE INDEX ON documents
  USING hnsw (embedding vector_ip_ops);

-- With tuning parameters
CREATE INDEX ON documents
  USING hnsw (embedding vector_cosine_ops)
  WITH (m = 16, ef_construction = 64);
```

### IVFFlat Index

Good for very large datasets (millions of vectors):

```sql
-- Create IVFFlat index
CREATE INDEX ON documents
  USING ivfflat (embedding vector_cosine_ops)
  WITH (lists = 100);

-- Before querying, set probes for accuracy/speed tradeoff
SET ivfflat.probes = 10;
```

### Index Selection Guide

| Index | Best For | Build Time | Query Speed | Accuracy |
|-------|----------|------------|-------------|----------|
| HNSW | Most use cases, < 10M vectors | Slower | Fast | High |
| IVFFlat | Large datasets, > 10M vectors | Fast | Medium | Medium |

## Distance Metrics

Choose the right metric based on your embedding model:

| Metric | Use Case | Index Ops |
|--------|----------|-----------|
| **Cosine** | Text embeddings (OpenAI, etc.) | `vector_cosine_ops` |
| **L2** | Image embeddings, when magnitude matters | `vector_l2_ops` |
| **Inner Product** | Normalized vectors, dot product similarity | `vector_ip_ops` |

:::tip[Cosine vs L2]
Most text embedding models (OpenAI, Cohere, etc.) work best with **cosine distance**. L2 distance is more appropriate when the vector magnitude carries meaning.
:::

## Advanced Topics

### Hybrid Search (Vector + Full-Text)

Combine vector similarity with PostgreSQL full-text search:

```sql
-- Add full-text search column
ALTER TABLE documents ADD COLUMN search_vector tsvector;
CREATE INDEX ON documents USING gin(search_vector);

-- Update with searchable content
UPDATE documents SET search_vector = to_tsvector('english', title || ' ' || content);

-- Hybrid search function
CREATE OR REPLACE FUNCTION hybrid_search(
  query_text TEXT,
  query_embedding vector(1536),
  keyword_weight float DEFAULT 0.3,
  semantic_weight float DEFAULT 0.7,
  match_count int DEFAULT 10
)
RETURNS TABLE (id UUID, title TEXT, score float)
LANGUAGE plpgsql AS $$
BEGIN
  RETURN QUERY
  SELECT
    d.id,
    d.title,
    (keyword_weight * ts_rank(d.search_vector, plainto_tsquery('english', query_text))
     + semantic_weight * (1 - (d.embedding <=> query_embedding))) AS score
  FROM documents d
  WHERE d.search_vector @@ plainto_tsquery('english', query_text)
     OR (d.embedding <=> query_embedding) < 0.5
  ORDER BY score DESC
  LIMIT match_count;
END;
$$;
```

### Filtering with Vector Search

Combine vector search with standard filters:

```typescript
const { data } = await client.vector.search({
  table: 'documents',
  column: 'embedding',
  query: 'machine learning',
  match_count: 10,
  filters: [
    { column: 'category', operator: 'eq', value: 'tutorials' },
    { column: 'published', operator: 'eq', value: true },
    { column: 'created_at', operator: 'gte', value: '2024-01-01' }
  ]
})
```

### Batch Embeddings

For inserting many documents:

```typescript
// Batch embed
const texts = documents.map(d => d.content)
const { data: embedResult } = await client.vector.embed({ texts })

// Insert with embeddings
const docsWithEmbeddings = documents.map((doc, i) => ({
  ...doc,
  embedding: embedResult.embeddings[i]
}))

await client.from('documents').insert(docsWithEmbeddings)
```

### Dimension Reduction

If storage is a concern, use lower-dimensional models:

| Model | Dimensions | Quality | Storage per 1M vectors |
|-------|------------|---------|------------------------|
| `text-embedding-3-large` | 3072 | Best | ~12 GB |
| `text-embedding-3-small` | 1536 | Good | ~6 GB |
| `all-minilm` (Ollama) | 384 | Moderate | ~1.5 GB |

## API Reference

### POST /api/v1/vector/embed

Generate embeddings from text.

**Request:**
```json
{
  "text": "Hello world",
  "texts": ["Hello", "World"],
  "model": "text-embedding-3-small"
}
```

**Response:**
```json
{
  "embeddings": [[0.1, 0.2, ...]],
  "model": "text-embedding-3-small",
  "dimensions": 1536,
  "usage": {
    "prompt_tokens": 2,
    "total_tokens": 2
  }
}
```

### POST /api/v1/vector/search

Semantic search with optional auto-embedding.

**Request:**
```json
{
  "table": "documents",
  "column": "embedding",
  "query": "search text",
  "vector": [0.1, 0.2, ...],
  "metric": "cosine",
  "match_threshold": 0.78,
  "match_count": 10,
  "select": "id, title, content",
  "filters": [
    { "column": "status", "operator": "eq", "value": "published" }
  ]
}
```

**Response:**
```json
{
  "data": [
    { "id": "...", "title": "...", "content": "..." }
  ],
  "distances": [0.12, 0.15, ...],
  "model": "text-embedding-3-small"
}
```

### GET /api/v1/capabilities/vector

Check vector search capabilities.

**Response:**
```json
{
  "enabled": true,
  "pgvector_installed": true,
  "pgvector_version": "0.7.0",
  "embedding_enabled": true,
  "embedding_provider": "openai",
  "embedding_model": "text-embedding-3-small"
}
```

## Troubleshooting

### "extension vector is not available"

The pgvector extension is not installed. Contact your database administrator or install it:

```sql
CREATE EXTENSION vector;
```

### "Embedding service not configured"

Enable embedding in your configuration:

```bash
FLUXBASE_AI_EMBEDDING_ENABLED=true
FLUXBASE_AI_EMBEDDING_PROVIDER=openai
FLUXBASE_AI_OPENAI_API_KEY=sk-...
```

### Slow queries

1. Create an appropriate index (HNSW or IVFFlat)
2. Reduce the number of results (`match_count`)
3. Add a `match_threshold` to filter distant results early
4. Use appropriate filters to reduce the search space

### Memory issues with large datasets

1. Use IVFFlat instead of HNSW for very large datasets
2. Consider dimension reduction (smaller embedding model)
3. Increase PostgreSQL `maintenance_work_mem` for index building
