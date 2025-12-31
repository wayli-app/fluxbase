---
title: "Integration Guide"
description: "Step-by-step guide to connecting AI assistants to Fluxbase"
---

This guide walks you through connecting AI assistants like Claude to your Fluxbase instance using the Model Context Protocol (MCP).

## What is MCP?

The Model Context Protocol (MCP) is a standard for AI assistants to interact with external services. With Fluxbase's MCP server, Claude can:

- Query your database tables
- Create, update, and delete records
- Upload and download files
- Invoke edge functions
- Run background jobs
- Search vector embeddings

## Prerequisites

- Fluxbase running and accessible
- Claude Desktop installed (or another MCP-compatible client)
- Admin access to create client keys

## Step 1: Enable MCP in Fluxbase

Ensure the MCP server is enabled in your configuration:

```yaml
# fluxbase.yaml
mcp:
  enabled: true
  base_path: /mcp
  rate_limit_per_min: 100
```

Restart Fluxbase if you made changes:

```bash
docker compose restart fluxbase
```

Verify the MCP server is running:

```bash
curl http://localhost:8080/mcp/health
```

Expected response:
```json
{"status":"healthy","version":"1.0"}
```

## Step 2: Create an MCP Client Key

Create a client key with the appropriate scopes for your use case.

### Via Dashboard

1. Go to **Settings** > **Client Keys**
2. Click **Create Key**
3. Set the name (e.g., "Claude Desktop")
4. Select the key type: **Anon** (public) or **Service** (admin)
5. Add the required scopes:
   - `read:tables` - Query data
   - `write:tables` - Modify data
   - `execute:functions` - Call functions
   - `execute:rpc` - Run SQL procedures
   - `read:storage` - Access files
   - `write:storage` - Upload files
   - `execute:jobs` - Run background jobs
   - `read:vectors` - Vector search
   - `read:schema` - View schema

6. Copy the generated key

### Via CLI

```bash
fluxbase clientkeys create \
  --name "Claude Desktop" \
  --type anon \
  --scopes "read:tables,write:tables,execute:functions,read:storage,read:schema"
```

:::caution[Key Security]
- **Anon keys** respect Row Level Security - users only see their own data
- **Service keys** bypass RLS and have full database access
- For personal assistants, service keys are fine
- For shared/production use, prefer anon keys with proper RLS
:::

## Step 3: Configure Claude Desktop

Locate your Claude Desktop configuration file:

- **macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Windows**: `%APPDATA%\Claude\claude_desktop_config.json`
- **Linux**: `~/.config/Claude/claude_desktop_config.json`

Add the Fluxbase MCP server:

```json
{
  "mcpServers": {
    "fluxbase": {
      "command": "curl",
      "args": [
        "-X", "POST",
        "http://localhost:8080/mcp",
        "-H", "Content-Type: application/json",
        "-H", "X-Client-Key: your-client-key-here"
      ]
    }
  }
}
```

### Alternative: Use a Local Proxy

For better performance, you can use an MCP proxy:

```json
{
  "mcpServers": {
    "fluxbase": {
      "command": "npx",
      "args": [
        "@anthropic/mcp-proxy",
        "http://localhost:8080/mcp",
        "--header", "X-Client-Key: your-client-key-here"
      ]
    }
  }
}
```

### For Remote Servers

If your Fluxbase instance is remote (not localhost):

```json
{
  "mcpServers": {
    "fluxbase": {
      "command": "npx",
      "args": [
        "@anthropic/mcp-proxy",
        "https://api.your-domain.com/mcp",
        "--header", "X-Client-Key: your-client-key-here"
      ]
    }
  }
}
```

## Step 4: Restart Claude Desktop

After updating the configuration:

1. Quit Claude Desktop completely
2. Reopen Claude Desktop
3. Look for the MCP indicator showing "fluxbase" is connected

## Step 5: Test the Connection

Start a new conversation with Claude and try:

> "What tables are available in my Fluxbase database?"

Claude should respond with a list of your tables.

Try some operations:

> "Show me the first 5 users from the users table"

> "Create a new task with the title 'Test from Claude'"

> "What functions are deployed in Fluxbase?"

## Example Interactions

### Querying Data

**You**: Show me all tasks that are not completed, ordered by creation date.

**Claude**: I'll query the tasks table for incomplete tasks.

*Uses `query_table` tool*

Here are the incomplete tasks:
| Title | Created At |
|-------|-----------|
| Review PR #42 | 2024-01-15 |
| Update docs | 2024-01-14 |

### Creating Records

**You**: Add a new task called "Prepare demo" with high priority.

**Claude**: I'll create that task for you.

*Uses `insert_record` tool*

Done! Created task with ID `abc123`:
- Title: Prepare demo
- Priority: high
- Created: 2024-01-15

### Invoking Functions

**You**: Call the send-welcome-email function for user@example.com

**Claude**: I'll invoke that function.

*Uses `invoke_function` tool*

Function executed successfully. The welcome email has been sent to user@example.com.

### Working with Files

**You**: What files are in the uploads bucket?

**Claude**: Let me check the storage.

*Uses `list_files` tool*

Found 3 files in the uploads bucket:
- report-2024.pdf (2.3 MB)
- logo.png (45 KB)
- data.csv (1.1 MB)

### Vector Search

**You**: Find documents similar to "user authentication best practices"

**Claude**: I'll search your knowledge base.

*Uses `vector_search` tool*

Found 3 relevant documents:
1. "Authentication Patterns" (similarity: 0.92)
2. "Security Guidelines" (similarity: 0.87)
3. "OAuth Implementation" (similarity: 0.84)

## Available Tools

| Tool | Scope Required | Description |
|------|----------------|-------------|
| `query_table` | `read:tables` | Query database tables |
| `insert_record` | `write:tables` | Create new records |
| `update_record` | `write:tables` | Update existing records |
| `delete_record` | `write:tables` | Delete records |
| `invoke_function` | `execute:functions` | Call edge functions |
| `invoke_rpc` | `execute:rpc` | Execute SQL procedures |
| `list_files` | `read:storage` | List storage files |
| `download_file` | `read:storage` | Get file contents |
| `upload_file` | `write:storage` | Upload files |
| `delete_file` | `write:storage` | Remove files |
| `submit_job` | `execute:jobs` | Start background jobs |
| `get_job_status` | `execute:jobs` | Check job status |
| `vector_search` | `read:vectors` | Similarity search |

## Available Resources

Resources provide context information to the AI:

| Resource | Description |
|----------|-------------|
| `schema://tables` | Database schema information |
| `schema://tables/{table}` | Specific table structure |
| `functions://list` | Deployed edge functions |
| `functions://{name}` | Function details |
| `storage://buckets` | Storage bucket list |
| `rpc://list` | Available RPC procedures |

## Troubleshooting

### "MCP server not found"

- Verify Fluxbase is running
- Check the URL is correct in your config
- Test the health endpoint: `curl http://localhost:8080/mcp/health`

### "Authentication failed"

- Verify your client key is correct
- Check the key hasn't expired
- Ensure the key has the required scopes

### "Permission denied"

- The operation requires additional scopes
- Check RLS policies if using an anon key
- Try with a service key for admin operations

### "Rate limited"

- You've exceeded the requests per minute limit
- Wait a moment and try again
- Consider increasing `rate_limit_per_min` in config

### Claude doesn't see the MCP server

- Ensure the config file is valid JSON
- Restart Claude Desktop completely
- Check Claude Desktop logs for errors

## Security Best Practices

### Use Minimal Scopes

Only grant the scopes your AI assistant needs:

```bash
# Read-only access
fluxbase clientkeys create \
  --name "Claude Read-Only" \
  --scopes "read:tables,read:schema"

# Full access
fluxbase clientkeys create \
  --name "Claude Full Access" \
  --scopes "read:tables,write:tables,execute:functions,execute:rpc,read:storage,write:storage"
```

### Set Up RLS

If using anon keys, ensure Row Level Security is configured:

```sql
-- Users can only see their own data
CREATE POLICY "Users see own data"
ON public.tasks FOR SELECT
USING (auth.uid() = user_id);
```

### Rotate Keys Regularly

Create a new key and update your config periodically:

```bash
# Create new key
fluxbase clientkeys create --name "Claude Desktop v2"

# Revoke old key
fluxbase clientkeys revoke <old-key-id>
```

### Monitor Usage

Check MCP activity:

```bash
fluxbase logs list --category mcp --since 24h
```

## Next Steps

- [MCP Tools Reference](/guides/mcp/tools/) - Detailed tool documentation
- [MCP Resources Reference](/guides/mcp/resources/) - Resource documentation
- [MCP Security](/security/mcp-security/) - Security guidelines
- [Row Level Security](/guides/row-level-security/) - Secure your data
