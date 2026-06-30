---
title: "Knowledge Bases & RAG"
description: Create knowledge bases for RAG-powered chatbots in Fluxbase. Upload documents, automatic chunking and embedding, and semantic search with pgvector.
---

Fluxbase provides built-in support for Retrieval-Augmented Generation (RAG), allowing you to create knowledge bases that provide context to AI chatbots. This enables chatbots to answer questions based on your custom documentation, product information, or any text content.

## Overview

Knowledge bases in Fluxbase enable:

- **RAG-Powered Chatbots**: Chatbots automatically retrieve relevant context from knowledge bases
- **Vector Search**: Semantic similarity search using pgvector
- **Document Management**: Upload, chunk, and embed documents automatically
- **Multiple Chunking Strategies**: Recursive, sentence, paragraph, or fixed-size chunking
- **Flexible Linking**: Connect multiple knowledge bases to a single chatbot

## Architecture

```mermaid
graph LR
    A[User Query] --> B[Embed Query]
    B --> C[Vector Search]
    C --> D[Retrieve Chunks]
    D --> E[Build Context]
    E --> F[LLM with RAG Context]
    F --> G[Response]

    subgraph Knowledge Base
        H[Documents] --> I[Chunking]
        I --> J[Embedding]
        J --> K[(pgvector)]
    end

    C --> K

    style B fill:#10a37f,color:#fff
    style K fill:#336791,color:#fff
    style F fill:#3178c6,color:#fff
```

The RAG pipeline:

1. Documents are chunked into smaller segments
2. Each chunk is embedded using an embedding model (e.g., text-embedding-3-small)
3. Embeddings are stored in PostgreSQL using pgvector
4. When a user asks a question, the query is embedded
5. Similar chunks are retrieved via vector similarity search
6. Retrieved context is injected into the chatbot's system prompt
7. The LLM generates a response using the provided context

## Prerequisites

Before using knowledge bases, ensure:

1. **pgvector Extension**: Install the pgvector extension in PostgreSQL
2. **Embedding Provider**: Configure an embedding provider (OpenAI, Azure, or Ollama)
3. **AI Feature Enabled**: Enable the AI feature in Fluxbase settings

### Installing pgvector

```sql
CREATE EXTENSION IF NOT EXISTS vector;
```

### Configuring Embedding Provider

**Automatic Fallback**: If you already have an AI provider configured for chatbots (e.g., OpenAI API key set), embeddings will automatically work using those same credentials. No additional configuration is needed.

**Explicit Configuration**: For fine-grained control or to use a different provider for embeddings, set these environment variables:

```bash
# OpenAI (explicit configuration)
FLUXBASE_AI_EMBEDDING_ENABLED=true
FLUXBASE_AI_EMBEDDING_PROVIDER=openai
FLUXBASE_AI_EMBEDDING_MODEL=text-embedding-3-small
FLUXBASE_AI_OPENAI_API_KEY=sk-...

# Or Azure OpenAI
FLUXBASE_AI_EMBEDDING_ENABLED=true
FLUXBASE_AI_EMBEDDING_PROVIDER=azure
FLUXBASE_AI_AZURE_API_KEY=...
FLUXBASE_AI_AZURE_ENDPOINT=https://your-resource.openai.azure.com
FLUXBASE_AI_AZURE_EMBEDDING_DEPLOYMENT_NAME=text-embedding-ada-002

# Or Ollama (local)
FLUXBASE_AI_EMBEDDING_ENABLED=true
FLUXBASE_AI_EMBEDDING_PROVIDER=ollama
FLUXBASE_AI_OLLAMA_ENDPOINT=http://localhost:11434
FLUXBASE_AI_EMBEDDING_MODEL=nomic-embed-text
```

**Default Models** (when using AI provider fallback):

- OpenAI: `text-embedding-3-small`
- Azure: `text-embedding-ada-002`
- Ollama: `nomic-embed-text`

## Creating Knowledge Bases

### Using the Admin Dashboard

1. Navigate to **Knowledge Bases** in the sidebar
2. Click **Create Knowledge Base**
3. Configure:
   - **Name**: Unique identifier (e.g., `product-docs`)
   - **Description**: What content this KB contains
   - **Chunk Size**: Characters per chunk (default: 512)
   - **Chunk Overlap**: Overlap between chunks (default: 50)
4. Click **Create**

### Using the REST API

```bash
curl -X POST http://localhost:8080/api/v1/admin/ai/knowledge-bases \
  -H "Authorization: Bearer YOUR_SERVICE_ROLE_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "product-docs",
    "description": "Product documentation",
    "chunk_size": 512,
    "chunk_overlap": 50
  }'
```

## Adding Documents

Once you have a knowledge base, add documents to it. Documents are automatically chunked and embedded.

### Using the REST API

```bash
curl -X POST http://localhost:8080/api/v1/admin/ai/knowledge-bases/KB_ID/documents \
  -H "Authorization: Bearer YOUR_SERVICE_ROLE_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Getting Started Guide",
    "content": "# Getting Started\n\nWelcome to our product! This guide will help you get started.",
    "metadata": {
      "category": "guides",
      "version": "1.0"
    }
  }'
```

Documents are processed asynchronously. Status values: `pending`, `processing`, `indexed`, `failed`.

## Uploading Document Files

In addition to pasting text content, you can upload document files directly. Fluxbase automatically extracts text from various file formats.

### Supported File Types

| Format            | Extension       | MIME Type                                                                 |
| ----------------- | --------------- | ------------------------------------------------------------------------- |
| PDF               | `.pdf`          | `application/pdf`                                                         |
| Plain Text        | `.txt`          | `text/plain`                                                              |
| Markdown          | `.md`           | `text/markdown`                                                           |
| HTML              | `.html`, `.htm` | `text/html`                                                               |
| CSV               | `.csv`          | `text/csv`                                                                |
| Word Document     | `.docx`         | `application/vnd.openxmlformats-officedocument.wordprocessingml.document` |
| Excel Spreadsheet | `.xlsx`         | `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`       |
| Rich Text         | `.rtf`          | `application/rtf`                                                         |
| EPUB              | `.epub`         | `application/epub+zip`                                                    |
| JSON              | `.json`         | `application/json`                                                        |

**Maximum file size:** 50MB

### Upload via Admin Dashboard

1. Navigate to **Knowledge Bases** and select your knowledge base
2. Click **Add Document**
3. Select the **Upload File** tab
4. Drag and drop a file or click **Browse Files**
5. Optionally provide a custom title
6. Click **Upload Document**

### Upload via REST API

```bash
curl -X POST http://localhost:8080/api/v1/admin/ai/knowledge-bases/KB_ID/documents/upload \
  -H "Authorization: Bearer YOUR_SERVICE_ROLE_KEY" \
  -F "file=@document.pdf" \
  -F "title=My Document"
```

### Best Practices for File Uploads

1. **Clean PDFs**: Ensure PDFs are text-based, not scanned images (OCR not supported)
2. **Simple formatting**: Documents with simpler formatting extract more cleanly
3. **File size**: Smaller files process faster; split very large documents if needed
4. **Text density**: Avoid uploading files with mostly images or charts

## Chunking Strategies

Choose the chunking strategy that best fits your content:

| Strategy    | Description                                           | Best For                         |
| ----------- | ----------------------------------------------------- | -------------------------------- |
| `recursive` | Splits by paragraphs, then sentences, then characters | General text, documentation      |
| `sentence`  | Splits by sentence boundaries                         | Q&A content, conversational text |
| `paragraph` | Splits by paragraph (double newlines)                 | Well-structured documents        |
| `fixed`     | Fixed character count splits                          | Code, logs, structured data      |

### Configuring Chunk Size

- **Smaller chunks (256-512)**: More precise retrieval, better for specific facts
- **Larger chunks (1024-2048)**: More context per chunk, better for complex topics
- **Overlap (10-20% of chunk size)**: Prevents losing context at chunk boundaries

## Linking Knowledge Bases to Chatbots

Connect knowledge bases to chatbots to enable RAG.

### Method 1: Using Annotations (Recommended)

Add RAG annotations to your chatbot definition:

```typescript
/**
 * Product Support Bot
 *
 * @fluxbase:description Product support chatbot with RAG
 * @fluxbase:knowledge-base product-docs
 * @fluxbase:knowledge-base faq
 * @fluxbase:rag-max-chunks 5
 * @fluxbase:rag-similarity-threshold 0.7
 */

export default `You are a helpful product support assistant.

Use the provided context to answer questions about our product.
If you don't find relevant information in the context, say so honestly.

Current user ID: {{user_id}}
`;
```

### RAG Annotations Reference

| Annotation                           | Description                                          | Default |
| ------------------------------------ | ---------------------------------------------------- | ------- |
| `@fluxbase:knowledge-base`           | Name of knowledge base to use (can specify multiple) | -       |
| `@fluxbase:rag-max-chunks`           | Maximum chunks to retrieve                           | `5`     |
| `@fluxbase:rag-similarity-threshold` | Minimum similarity score (0.0-1.0)                   | `0.7`   |

### Method 2: Using the SDK

The SDK provides methods to manage knowledge base links on chatbots:

```typescript
import { createClient } from "@nimbleflux/fluxbase-sdk";

const client = createClient("http://localhost:8080", "service-role-key");

// Link a knowledge base to a chatbot
const { data, error } = await client.admin.ai.linkKnowledgeBase("chatbot-id", {
  knowledge_base_id: "kb-id",
  priority: 1,
  max_chunks: 5,
  similarity_threshold: 0.7,
});

// Update link settings
await client.admin.ai.updateChatbotKnowledgeBase("chatbot-id", "kb-id", {
  max_chunks: 10,
  enabled: true,
});

// List linked knowledge bases
const { data: links } =
  await client.admin.ai.listChatbotKnowledgeBases("chatbot-id");

// Unlink a knowledge base
await client.admin.ai.unlinkKnowledgeBase("chatbot-id", "kb-id");
```

## How RAG Works in Chat

When a user sends a message to a RAG-enabled chatbot:

1. **Query Embedding**: The user's message is embedded using the same model as documents
2. **Similarity Search**: pgvector finds the most similar chunks across linked knowledge bases
3. **Context Building**: Retrieved chunks are formatted into a context section
4. **Prompt Injection**: Context is added to the system prompt before the LLM call
5. **Response Generation**: The LLM uses the context to generate an informed response

### System Prompt with RAG Context

The chatbot receives a system prompt like:

```
[Original System Prompt]

## Relevant Context

The following information was retrieved from the knowledge base and may be relevant to the user's question:

### From: Getting Started Guide
Welcome to our product! This guide will help you get started...

### From: FAQ
Q: How do I reset my password?
A: Navigate to Settings > Account > Reset Password...

---

Use this context to answer the user's question. If the context doesn't contain relevant information, say so.
```

## Testing Knowledge Base Search

### Using the Admin Dashboard

1. Navigate to **Knowledge Bases**
2. Click on your knowledge base
3. Use the **Search** tab to test queries
4. Review similarity scores and retrieved content

### Using the REST API

```bash
curl -X POST http://localhost:8080/api/v1/admin/ai/knowledge-bases/KB_ID/search \
  -H "Authorization: Bearer YOUR_SERVICE_ROLE_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "how do I reset my password",
    "max_chunks": 5,
    "threshold": 0.5
  }'
```

## Best Practices

### Document Quality

1. **Clean Content**: Remove unnecessary formatting, headers, footers
2. **Consistent Structure**: Use consistent heading styles and formatting
3. **Complete Information**: Ensure documents contain full context
4. **Regular Updates**: Keep knowledge bases current with product changes

### Chunking Configuration

1. **Match Content Type**: Use appropriate chunking for your content
2. **Test Different Sizes**: Experiment to find optimal chunk size
3. **Monitor Retrieval**: Check if retrieved chunks are relevant

### Performance Optimization

1. **Index Size**: Keep knowledge bases focused on relevant content
2. **Similarity Threshold**: Higher thresholds (0.7-0.8) reduce noise
3. **Chunk Limit**: Limit retrieved chunks to avoid context overflow

### Security Considerations

1. **Access Control**: Use RLS policies on knowledge base tables if needed
2. **Sensitive Content**: Avoid storing sensitive data in knowledge bases
3. **User Context**: Consider user-specific knowledge bases for personalized content

## Example: Building a Support Chatbot

### Step 1: Create Knowledge Base

```bash
curl -X POST http://localhost:8080/api/v1/admin/ai/knowledge-bases \
  -H "Authorization: Bearer YOUR_SERVICE_ROLE_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "support-kb",
    "description": "Customer support documentation",
    "chunk_size": 512,
    "chunk_overlap": 50
  }'
```

### Step 2: Add Support Documentation

```bash
KB_ID="<knowledge-base-id-from-step-1>"

curl -X POST "http://localhost:8080/api/v1/admin/ai/knowledge-bases/${KB_ID}/documents" \
  -H "Authorization: Bearer YOUR_SERVICE_ROLE_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Frequently Asked Questions",
    "content": "## Account Questions\n\n### How do I create an account?\nVisit signup.example.com and fill out the registration form...\n\n### How do I reset my password?\nClick \"Forgot Password\" on the login page..."
  }'

curl -X POST "http://localhost:8080/api/v1/admin/ai/knowledge-bases/${KB_ID}/documents" \
  -H "Authorization: Bearer YOUR_SERVICE_ROLE_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Troubleshooting Guide",
    "content": "## Common Issues\n\n### Error: Connection Failed\n1. Check your internet connection\n2. Verify the service is running\n3. Clear browser cache..."
  }'
```

### Step 3: Create RAG-Enabled Chatbot

Create `chatbots/support-bot/index.ts`:

```typescript
/**
 * Customer Support Bot
 *
 * @fluxbase:description AI-powered customer support with knowledge base
 * @fluxbase:knowledge-base support-kb
 * @fluxbase:rag-max-chunks 5
 * @fluxbase:rag-similarity-threshold 0.7
 * @fluxbase:allowed-tables support_tickets,users
 * @fluxbase:allowed-operations SELECT
 * @fluxbase:rate-limit 30
 * @fluxbase:public true
 */

export default `You are a friendly customer support assistant.

## Your Role

- Answer questions using the provided knowledge base context
- Help users troubleshoot common issues
- Look up support ticket status when asked

## Guidelines

1. Always check the provided context first
2. If you can't find an answer in the context, say so politely
3. Offer to escalate to human support for complex issues
4. Be friendly and professional

## Available Actions

- Answer questions from knowledge base
- Look up user's support tickets (use execute_sql)

Current user ID: {{user_id}}
`;
```

### Step 4: Deploy and Test

```bash
# Sync chatbot
curl -X POST http://localhost:8080/api/v1/admin/ai/chatbots/sync \
  -H "Authorization: Bearer YOUR_SERVICE_ROLE_KEY"
```

Or using the SDK:

```typescript
import { createClient } from "@nimbleflux/fluxbase-sdk";

const client = createClient("http://localhost:8080", "service-role-key");

await client.admin.ai.sync();
```

## Monitoring & Analytics

### View Knowledge Base Stats

Use the REST API to check knowledge base statistics:

```bash
curl http://localhost:8080/api/v1/admin/ai/knowledge-bases/KB_ID \
  -H "Authorization: Bearer YOUR_SERVICE_ROLE_KEY"
```

### Track Document Processing

```bash
curl http://localhost:8080/api/v1/admin/ai/knowledge-bases/KB_ID/documents \
  -H "Authorization: Bearer YOUR_SERVICE_ROLE_KEY"
```

## Troubleshooting

### Documents Not Being Embedded

- Verify `FLUXBASE_AI_EMBEDDING_ENABLED=true`
- Confirm API keys are valid
- Check provider endpoint is accessible

### Poor Search Results

- Lower the similarity threshold for testing (e.g., 0.3)
- Ensure documents are properly chunked
- Verify content is relevant to expected queries
- Try different chunking strategies

### RAG Context Not Appearing

Verify the chatbot has linked knowledge bases:

```typescript
const client = createClient("http://localhost:8080", "service-role-key");
const { data: links } =
  await client.admin.ai.listChatbotKnowledgeBases("chatbot-id");
console.log("Linked KBs:", links);
```

Check annotation syntax:

```typescript
// Correct
* @fluxbase:knowledge-base my-kb-name

// Wrong (no asterisk in multi-line comment)
@fluxbase:knowledge-base my-kb-name
```

## Entity Extraction

### Overview

Fluxbase automatically extracts entities and relationships from documents when they are processed. This enables knowledge graph capabilities and improves search relevance.

### Entity Types

The following entity types are extracted by the built-in rule-based extractor. The extractor is regex-driven and works best on developer/technical content; it is US/English-centric. Set `entity_extraction_enabled=false` on a KB to skip extraction entirely (see [Disabling extraction](#disabling-extraction)).

| Type               | What the rule matches                                   | Examples                                       |
| ------------------ | ------------------------------------------------------- | ---------------------------------------------- |
| **person**         | Title-prefixed names, 2+ capitalized words              | John Smith, Dr. Jane Doe, CEO Bob              |
| **organization**   | Company suffixes (Inc/Corp/LLC/…) + a built-in list     | Google, Microsoft, Acme Corp                   |
| **location**       | Built-in list of US/CA/EU cities and US states          | New York, London, Paris, California            |
| **product**        | Built-in list of ~30 tech products                      | iPhone, AWS, Docker, Kubernetes                |
| **url**            | URLs and links                                          | `https://example.com`, `www.fluxbase.com/docs` |
| **api_endpoint**   | HTTP-method + `/api/...`, `/api/vN/...`, `/graphql`     | `POST /api/v1/users`, `GET /api/v1/auth/...`   |
| **datetime**       | ISO 8601, `Month DD, YYYY`, relative phrases            | `2025-02-18`, `2h 30m`, `3 days ago`           |
| **code_reference** | File paths, `github.com/user/repo`                      | `internal/ai/kb.go`, `github.com/user/repo`    |
| **error**          | `Err[A-Z]…`, HTTP codes, quoted error phrases           | `ErrNotFound`, `404 Unauthorized`              |
| **table**          | Created by table export, not the rule extractor         | `auth.users`, `public.orders`                  |
| **other**          | Reserved; not produced by the extractor                 | —                                              |

:::note
The `person` rule (capitalized bigrams) is aggressive. Expect false positives on prose ("San Francisco" can match as both `person` and `location`). For non-technical KBs, consider disabling extraction.
:::

### Relationship Types

The schema defines 19 relationship types. Only three are produced automatically; the rest are available for manual insertion (e.g. via SQL or future tools).

**Automatically inferred:**

| Type            | What the rule looks for                              | Example                          |
| --------------- | ---------------------------------------------------- | -------------------------------- |
| **works_at**    | `(?:works at\|is employed by\|joined\|works for)`     | "John works at Google"           |
| **founded_by**  | `(?:founded\|co-founded\|started\|created)`          | "Apple founded by Steve Jobs"    |
| **foreign_key** | Created by table export, not the rule extractor      | `orders.user_id → users.id`      |

**Schema-supported (manual only — not produced by any extractor):**

`located_in`, `owns`, `part_of`, `related_to`, `knows`, `customer_of`, `supplier_of`, `invested_in`, `acquired`, `merged_with`, `competitor_of`, `parent_of`, `child_of`, `spouse_of`, `sibling_of`, `depends_on`.

### Automatic Extraction

Entities and relationships are extracted automatically when documents are processed:

1. Documents are uploaded to a knowledge base
2. Document content is chunked for embedding
3. Rule-based entity extraction identifies entities and relationships
4. Entities are stored in the knowledge graph
5. Document-entity mentions are tracked

Re-running extraction (e.g. via `ReprocessDocument`) replaces prior document-entity mentions for that document, so editing a document to remove an entity mention no longer leaves stale links. Entities themselves are de-duplicated by canonical name; entities referenced only by a deleted document are garbage-collected.

### Disabling extraction

Extraction is enabled by default for every KB. To skip it (e.g. for a prose-heavy KB where the regex `person` rule would create noise), set `entity_extraction_enabled=false`:

```bash
# CLI
fluxbase kb update <kb-id> --entity-extraction=false

# REST
curl -X PUT http://localhost:8080/api/v1/admin/ai/knowledge-bases/<kb-id> \
  -H "Authorization: Bearer YOUR_SERVICE_ROLE_KEY" \
  -H "Content-Type: application/json" \
  -d '{"entity_extraction_enabled": false}'
```

Disabling extraction on an existing KB does not remove entities already stored; delete and recreate the KB if you need a clean slate, or remove entities via SQL.

## Graph-Boosted Retrieval

:::note
Graph-boosted retrieval is **off by default**. Without enabling it, the knowledge graph has no effect on chatbot responses.
:::

When enabled, RAG retrieval over-fetches candidate chunks and re-ranks them using entity salience: chunks from documents that mention entities found in the user's query are boosted. The re-rank formula is:

```
finalScore = similarity * (1 - weight) + entityBoost * weight
```

where `entityBoost` is the normalized salience of query-extracted entities on the chunk's parent document (0 to 1). Because the rule extractor currently assigns a flat salience of `0.5` to every entity, the boost acts as an entity-presence signal; per-entity salience tuning is a future improvement.

Enable graph boost via either of two knobs (per-chatbot annotation takes precedence over the global default):

```ts
// Chatbot annotation (0.0–1.0, 0 = off)
// @fluxbase:rag-graph-boost-weight 0.3
```

```bash
# Global default for all chatbots (env var)
FLUXBASE_AI_RAG_GRAPH_BOOST_WEIGHT=0.3
```

User isolation is preserved: graph-boosted search honors the same `MetadataFilter` (per-user `user_id`, include-global) as the standard path.

## Database Table Export

### Overview

Export database tables as knowledge base documents with embedded schema information. This enables AI assistants to understand your database structure and answer schema-related questions.

### Using the SDK

```typescript
import { createClient } from "@nimbleflux/fluxbase-sdk";

const client = createClient("http://localhost:8080", "service-role-key");

const { data, error } = await client.admin.ai.getTableDetails("auth", "users");
if (data) {
  console.log("Columns:", data.columns.map((c) => c.name));
  console.log("Primary key:", data.primary_key);
}
```

### Using the REST API

```bash
# List exportable tables
curl "http://localhost:8080/api/v1/admin/ai/tables?schema=auth" \
  -H "Authorization: Bearer YOUR_SERVICE_ROLE_KEY"

# Export table to knowledge base
curl -X POST http://localhost:8080/api/v1/admin/ai/tables/auth/users/export \
  -H "Authorization: Bearer YOUR_SERVICE_ROLE_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "knowledge_base_id": "kb-uuid",
    "include_foreign_keys": true,
    "include_indexes": true,
    "include_sample_rows": false
  }'
```

### What Gets Created

For each exported table, three artifacts are created:

1. **Document**: Markdown schema documentation with columns, types, primary keys
2. **Entity**: Graph representation of the table
3. **Relationships**: Foreign key connections to other tables

### Use Cases

- **Schema Documentation**: Automatically document your database structure
- **AI Assistant Context**: Help AI understand your database schema
- **Relationship Discovery**: Explore table relationships through the knowledge graph
- **Onboarding**: Quickly share database structure with team members

## MCP Tools for Knowledge Graph

### Overview

Fluxbase provides Model Context Protocol (MCP) tools for interacting with the knowledge graph from AI assistants.

### Available Tools

#### 1. query_knowledge_graph

Query entities in the knowledge graph with filtering:

```json
{
  "name": "query_knowledge_graph",
  "arguments": {
    "knowledge_base_id": "kb-uuid",
    "entity_type": "location",
    "search_query": "San Francisco",
    "limit": 50,
    "include_relationships": true
  }
}
```

#### 2. find_related_entities

Find entities related to a starting entity using graph traversal:

```json
{
  "name": "find_related_entities",
  "arguments": {
    "knowledge_base_id": "kb-uuid",
    "entity_id": "entity-uuid",
    "max_depth": 2,
    "relationship_types": ["works_at", "located_in"],
    "limit": 100
  }
}
```

#### 3. browse_knowledge_graph

Browse the knowledge graph from a starting entity:

```json
{
  "name": "browse_knowledge_graph",
  "arguments": {
    "knowledge_base_id": "kb-uuid",
    "start_entity": "entity-id-or-name",
    "direction": "both",
    "limit": 50
  }
}
```

### Tool Scopes

Knowledge graph tools require the `read:vectors` scope for authorization.

## User-Scoped RAG Retrieval

### Overview

For multi-tenant applications, knowledge bases support user-scoped document isolation. When a chatbot retrieves context from a knowledge base, it can filter documents by user ID, ensuring users only see their own documents.

### How It Works

1. **Adding Documents with User Context**: When adding documents, include `user_id` in the metadata
2. **Filter Expression**: Configure chatbot-knowledge base links with filter expressions
3. **Automatic Filtering**: RAG retrieval automatically applies user context during search

### Adding Documents with User Context

```bash
curl -X POST http://localhost:8080/api/v1/admin/ai/knowledge-bases/KB_ID/documents \
  -H "Authorization: Bearer YOUR_SERVICE_ROLE_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "User Travel Notes",
    "content": "My favorite restaurants in Tokyo...",
    "metadata": {
      "user_id": "user-123",
      "category": "travel"
    }
  }'
```

### Configuring Filtered Access

Link a knowledge base with filtered access to enable user-scoped retrieval:

```typescript
import { createClient } from "@nimbleflux/fluxbase-sdk";

const client = createClient("http://localhost:8080", "service-role-key");

await client.admin.ai.linkKnowledgeBase("chatbot-id", {
  knowledge_base_id: "kb-id",
  max_chunks: 5,
  similarity_threshold: 0.7,
});
```

### Automatic User Isolation

When a user chats with a RAG-enabled chatbot:

1. The chat handler extracts the user ID from the authentication context
2. RAG retrieval passes the user ID to the search function
3. Documents with matching `user_id` in metadata are returned
4. Documents without `user_id` are excluded unless explicitly allowed

### Bulk Delete by User

Delete all documents for a user (e.g., account deletion):

```bash
curl -X DELETE "http://localhost:8080/api/v1/admin/ai/knowledge-bases/KB_ID/documents?user_id=user-to-delete" \
  -H "Authorization: Bearer YOUR_SERVICE_ROLE_KEY"
```

## CLI Commands

The Fluxbase CLI provides commands for managing knowledge bases.

### Basic Commands

```bash
# List knowledge bases
fluxbase kb list

# Get knowledge base details
fluxbase kb get <kb-id>

# Create knowledge base
fluxbase kb create my-kb --description "My knowledge base"

# Update knowledge base (e.g. disable entity extraction)
fluxbase kb update <kb-id> --description "Updated description"
fluxbase kb update <kb-id> --entity-extraction=false

# Delete knowledge base
fluxbase kb delete <kb-id>
```

### Document Management

```bash
# List documents
fluxbase kb documents <kb-id>           # alias: kb docs

# Upload a file as a document
fluxbase kb upload <kb-id> ./document.pdf --title "My PDF"
fluxbase kb upload <kb-id> report.md --tag finance --tag q4
```

### Knowledge Graph

```bash
# View knowledge graph data
fluxbase kb graph <kb-id>

# List entities
fluxbase kb entities <kb-id>
fluxbase kb entities <kb-id> --type person
fluxbase kb entities <kb-id> --search "John"

# List chatbots using KB
fluxbase kb chatbots <kb-id>
```

## Next Steps

- [AI Chatbots](/guides/ai-chatbots) - Chatbot configuration and usage
- [Vector Search](/guides/vector-search) - Direct vector search operations
- [TypeScript SDK Reference](/api/sdk/) - Full SDK documentation
