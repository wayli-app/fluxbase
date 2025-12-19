---
title: "Knowledge Bases & RAG"
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
-- Enable pgvector extension
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

### Using the SDK

```typescript
import { createClient } from '@fluxbase/sdk'

const client = createClient('http://localhost:8080', 'service-role-key')

// Create a knowledge base
const { data: kb, error } = await client.admin.ai.createKnowledgeBase({
  name: 'product-docs',
  description: 'Product documentation and FAQs',
  chunk_size: 512,
  chunk_overlap: 50,
  chunk_strategy: 'recursive', // or 'sentence', 'paragraph', 'fixed'
})

console.log('Created KB:', kb.id)
```

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

### Using the SDK

```typescript
// Add a document
const { data, error } = await client.admin.ai.addDocument('kb-id', {
  title: 'Getting Started Guide',
  content: `
    # Getting Started with Our Product

    Welcome to our product! This guide will help you get started.

    ## Installation

    1. Download the installer from our website
    2. Run the installer
    3. Follow the on-screen instructions

    ## First Steps

    After installation, launch the application and create an account...
  `,
  metadata: {
    category: 'guides',
    version: '1.0',
  },
})

console.log('Document ID:', data.document_id)
console.log('Status:', data.status) // 'processing'
```

### Bulk Document Upload

```typescript
const documents = [
  { title: 'FAQ', content: faqContent },
  { title: 'API Reference', content: apiContent },
  { title: 'Troubleshooting', content: troubleshootingContent },
]

for (const doc of documents) {
  const { data, error } = await client.admin.ai.addDocument('kb-id', doc)
  if (error) {
    console.error(`Failed to add ${doc.title}:`, error)
  } else {
    console.log(`Added ${doc.title}: ${data.document_id}`)
  }
}
```

### Document Status

Documents are processed asynchronously. Check status:

```typescript
const { data: docs } = await client.admin.ai.listDocuments('kb-id')

for (const doc of docs) {
  console.log(`${doc.title}: ${doc.status}`)
  // Status: 'pending' | 'processing' | 'indexed' | 'failed'
  if (doc.status === 'failed') {
    console.error(`Error: ${doc.error_message}`)
  }
}
```

## Uploading Document Files

In addition to pasting text content, you can upload document files directly. Fluxbase automatically extracts text from various file formats.

### Supported File Types

| Format | Extension | MIME Type |
|--------|-----------|-----------|
| PDF | `.pdf` | `application/pdf` |
| Plain Text | `.txt` | `text/plain` |
| Markdown | `.md` | `text/markdown` |
| HTML | `.html`, `.htm` | `text/html` |
| CSV | `.csv` | `text/csv` |
| Word Document | `.docx` | `application/vnd.openxmlformats-officedocument.wordprocessingml.document` |
| Excel Spreadsheet | `.xlsx` | `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet` |
| Rich Text | `.rtf` | `application/rtf` |
| EPUB | `.epub` | `application/epub+zip` |
| JSON | `.json` | `application/json` |

**Maximum file size:** 50MB

### Upload via Admin Dashboard

1. Navigate to **Knowledge Bases** and select your knowledge base
2. Click **Add Document**
3. Select the **Upload File** tab
4. Drag and drop a file or click **Browse Files**
5. Optionally provide a custom title
6. Click **Upload Document**

### Upload via SDK

```typescript
// Browser environment
const fileInput = document.getElementById('file') as HTMLInputElement
const file = fileInput.files?.[0]

if (file) {
  const { data, error } = await client.admin.ai.uploadDocument(
    'kb-id',
    file,
    'Custom Document Title' // Optional
  )

  if (data) {
    console.log('Document ID:', data.document_id)
    console.log('Filename:', data.filename)
    console.log('Extracted text length:', data.extracted_length)
    console.log('MIME type:', data.mime_type)
  }
}
```

```typescript
// Node.js environment
import { readFile } from 'fs/promises'
import { Blob } from 'buffer'

const content = await readFile('document.pdf')
const blob = new Blob([content], { type: 'application/pdf' })

const { data, error } = await client.admin.ai.uploadDocument(
  'kb-id',
  blob,
  'My PDF Document'
)
```

### Upload via REST API

```bash
curl -X POST http://localhost:8080/api/v1/admin/ai/knowledge-bases/KB_ID/documents/upload \
  -H "Authorization: Bearer YOUR_SERVICE_ROLE_KEY" \
  -F "file=@document.pdf" \
  -F "title=My Document"
```

### Text Extraction Details

Fluxbase uses specialized libraries to extract text from each file type:

- **PDF**: Extracts text content from all pages, preserving paragraph structure
- **DOCX**: Extracts text from Word documents including paragraphs and tables
- **XLSX**: Extracts content from all sheets, preserving cell structure with tab delimiters
- **HTML**: Strips tags and scripts, extracts visible text content
- **CSV**: Preserves tabular structure with tab delimiters
- **RTF**: Removes RTF formatting codes, extracts plain text
- **EPUB**: Extracts text from all chapters in reading order
- **Plain text/Markdown/JSON**: Used as-is without transformation

### Best Practices for File Uploads

1. **Clean PDFs**: Ensure PDFs are text-based, not scanned images (OCR not supported)
2. **Simple formatting**: Documents with simpler formatting extract more cleanly
3. **File size**: Smaller files process faster; split very large documents if needed
4. **Text density**: Avoid uploading files with mostly images or charts

## Chunking Strategies

Choose the chunking strategy that best fits your content:

| Strategy | Description | Best For |
|----------|-------------|----------|
| `recursive` | Splits by paragraphs, then sentences, then characters | General text, documentation |
| `sentence` | Splits by sentence boundaries | Q&A content, conversational text |
| `paragraph` | Splits by paragraph (double newlines) | Well-structured documents |
| `fixed` | Fixed character count splits | Code, logs, structured data |

### Configuring Chunk Size

- **Smaller chunks (256-512)**: More precise retrieval, better for specific facts
- **Larger chunks (1024-2048)**: More context per chunk, better for complex topics
- **Overlap (10-20% of chunk size)**: Prevents losing context at chunk boundaries

```typescript
// For FAQ-style content (shorter, precise chunks)
await client.admin.ai.createKnowledgeBase({
  name: 'faq',
  chunk_size: 256,
  chunk_overlap: 25,
  chunk_strategy: 'sentence',
})

// For technical documentation (larger chunks with context)
await client.admin.ai.createKnowledgeBase({
  name: 'tech-docs',
  chunk_size: 1024,
  chunk_overlap: 100,
  chunk_strategy: 'recursive',
})
```

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
`
```

### RAG Annotations Reference

| Annotation | Description | Default |
|------------|-------------|---------|
| `@fluxbase:knowledge-base` | Name of knowledge base to use (can specify multiple) | - |
| `@fluxbase:rag-max-chunks` | Maximum chunks to retrieve | `5` |
| `@fluxbase:rag-similarity-threshold` | Minimum similarity score (0.0-1.0) | `0.7` |

### Method 2: Using the Admin API

```typescript
// Link a knowledge base to a chatbot
const { data, error } = await client.admin.ai.linkKnowledgeBase('chatbot-id', {
  knowledge_base_id: 'kb-id',
  priority: 1,         // Higher priority = searched first
  max_chunks: 5,       // Max chunks from this KB
  similarity_threshold: 0.7,
})

// Update link settings
await client.admin.ai.updateChatbotKnowledgeBase('chatbot-id', 'kb-id', {
  max_chunks: 10,
  enabled: true,
})

// List linked knowledge bases
const { data: links } = await client.admin.ai.listChatbotKnowledgeBases('chatbot-id')

// Unlink a knowledge base
await client.admin.ai.unlinkKnowledgeBase('chatbot-id', 'kb-id')
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

Test your knowledge base setup before deploying:

### Using the SDK

```typescript
// Search a knowledge base directly
const { data, error } = await client.admin.ai.searchKnowledgeBase(
  'kb-id',
  'how do I reset my password',
  {
    max_chunks: 5,
    threshold: 0.5, // Lower threshold for testing
  }
)

if (data) {
  console.log(`Found ${data.count} results:`)
  for (const result of data.results) {
    console.log(`\n--- ${result.document_title} (${result.similarity.toFixed(3)}) ---`)
    console.log(result.content)
  }
}
```

### Using the Admin Dashboard

1. Navigate to **Knowledge Bases**
2. Click on your knowledge base
3. Use the **Search** tab to test queries
4. Review similarity scores and retrieved content

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

```typescript
const { data: kb } = await client.admin.ai.createKnowledgeBase({
  name: 'support-kb',
  description: 'Customer support documentation',
  chunk_size: 512,
  chunk_overlap: 50,
})
```

### Step 2: Add Support Documentation

```typescript
// Add FAQ
await client.admin.ai.addDocument(kb.id, {
  title: 'Frequently Asked Questions',
  content: `
    ## Account Questions

    ### How do I create an account?
    Visit signup.example.com and fill out the registration form...

    ### How do I reset my password?
    Click "Forgot Password" on the login page...

    ## Billing Questions

    ### How do I update my payment method?
    Go to Settings > Billing > Payment Methods...
  `,
})

// Add troubleshooting guide
await client.admin.ai.addDocument(kb.id, {
  title: 'Troubleshooting Guide',
  content: `
    ## Common Issues

    ### Error: Connection Failed
    1. Check your internet connection
    2. Verify the service is running
    3. Clear browser cache...

    ### Error: Authentication Failed
    1. Verify your credentials
    2. Check if your account is active...
  `,
})
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
`
```

### Step 4: Deploy and Test

```bash
# Sync chatbot
curl -X POST http://localhost:8080/api/v1/admin/ai/chatbots/sync \
  -H "Authorization: Bearer SERVICE_ROLE_KEY"
```

```typescript
// Test the chatbot
const chat = client.ai.createChat({
  token: userJWT,
  onContent: (delta) => process.stdout.write(delta),
})

await chat.connect()
const convId = await chat.startChat('support-bot')
chat.sendMessage(convId, 'How do I reset my password?')
```

## Monitoring & Analytics

### View Knowledge Base Stats

```typescript
const { data: kb } = await client.admin.ai.getKnowledgeBase('kb-id')

console.log('Documents:', kb.document_count)
console.log('Total Chunks:', kb.total_chunks)
console.log('Embedding Model:', kb.embedding_model)
```

### Track Document Processing

```typescript
const { data: docs } = await client.admin.ai.listDocuments('kb-id')

const stats = {
  total: docs.length,
  indexed: docs.filter(d => d.status === 'indexed').length,
  processing: docs.filter(d => d.status === 'processing').length,
  failed: docs.filter(d => d.status === 'failed').length,
}

console.log('Document Stats:', stats)
```

## Troubleshooting

### Documents Not Being Embedded

**Check embedding provider configuration:**
- Verify `FLUXBASE_AI_EMBEDDING_ENABLED=true`
- Confirm API keys are valid
- Check provider endpoint is accessible

**Check document status:**
```typescript
const { data: doc } = await client.admin.ai.getDocument('kb-id', 'doc-id')
console.log('Status:', doc.status)
console.log('Error:', doc.error_message)
```

### Poor Search Results

**Lower similarity threshold for testing:**
```typescript
const { data } = await client.admin.ai.searchKnowledgeBase('kb-id', query, {
  threshold: 0.3, // Lower threshold to see more results
})
```

**Check chunk content:**
- Ensure documents are properly chunked
- Verify content is relevant to expected queries
- Try different chunking strategies

### RAG Context Not Appearing

**Verify chatbot has linked knowledge bases:**
```typescript
const { data: links } = await client.admin.ai.listChatbotKnowledgeBases('chatbot-id')
console.log('Linked KBs:', links)
```

**Check annotation syntax:**
```typescript
// Correct
* @fluxbase:knowledge-base my-kb-name

// Wrong (no asterisk in multi-line comment)
@fluxbase:knowledge-base my-kb-name
```

## Next Steps

- [AI Chatbots](/docs/guides/ai-chatbots) - Chatbot configuration and usage
- [Vector Search](/docs/guides/vector-search) - Direct vector search operations
- [TypeScript SDK](/docs/guides/typescript-sdk) - Full SDK documentation
