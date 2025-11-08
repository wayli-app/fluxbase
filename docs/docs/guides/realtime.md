# Realtime Subscriptions

Fluxbase provides real-time database change notifications via WebSockets, powered by PostgreSQL's LISTEN/NOTIFY system.

## Using the SDK (Recommended)

The easiest way to use realtime subscriptions is with the Fluxbase SDK:

### Installation

```bash
npm install @fluxbase/sdk
```

### Basic Usage

```typescript
import { createClient } from "@fluxbase/sdk";

// Create client
const client = createClient("http://localhost:8080", "your-api-key");

// Subscribe to table changes
const channel = client.realtime
  .channel("table:public.products")
  .on("INSERT", (payload) => {
    console.log("New product:", payload.new_record);
  })
  .on("UPDATE", (payload) => {
    console.log("Updated product:", payload.new_record);
    console.log("Previous data:", payload.old_record);
  })
  .on("DELETE", (payload) => {
    console.log("Deleted product:", payload.old_record);
  })
  .subscribe();

// Or use wildcard to listen to all events
const channel = client.realtime
  .channel("table:public.products")
  .on("*", (payload) => {
    console.log("Change type:", payload.type); // INSERT, UPDATE, or DELETE
    console.log("Payload:", payload);
  })
  .subscribe();

// Later: Unsubscribe
channel.unsubscribe();
```

### React Example

```typescript
import { useEffect, useState } from "react";
import { createClient } from "@fluxbase/sdk";

function ProductList() {
  const [products, setProducts] = useState([]);
  const [client] = useState(() =>
    createClient("http://localhost:8080", "your-api-key")
  );

  useEffect(() => {
    // Subscribe to realtime changes
    const channel = client.realtime
      .channel("table:public.products")
      .on("INSERT", (payload) => {
        setProducts((prev) => [...prev, payload.new_record]);
      })
      .on("UPDATE", (payload) => {
        setProducts((prev) =>
          prev.map((p) => (p.id === payload.new_record.id ? payload.new_record : p))
        );
      })
      .on("DELETE", (payload) => {
        setProducts((prev) => prev.filter((p) => p.id !== payload.old_record.id));
      })
      .subscribe();

    // Cleanup on unmount
    return () => {
      channel.unsubscribe();
    };
  }, [client]);

  return (
    <div>
      <h2>Products</h2>
      {products.map((product) => (
        <div key={product.id}>{product.name}</div>
      ))}
    </div>
  );
}
```

### Payload Structure

```typescript
interface RealtimeChangePayload {
  type: "INSERT" | "UPDATE" | "DELETE";
  schema: string;
  table: string;
  new_record?: Record<string, unknown>; // Present for INSERT and UPDATE
  old_record?: Record<string, unknown>; // Present for UPDATE and DELETE
  timestamp: string;
}
```

---

## Advanced: Raw WebSocket Protocol

For advanced use cases or non-JavaScript environments, you can use the raw WebSocket protocol directly.

### 1. Connect to WebSocket

```javascript
// Connect to the realtime endpoint
const ws = new WebSocket("ws://localhost:8080/realtime");

// Optional: Include JWT token for authenticated subscriptions
const ws = new WebSocket("ws://localhost:8080/realtime?token=YOUR_JWT_TOKEN");

ws.onopen = () => {
  console.log("Connected to Fluxbase realtime");
};
```

### 2. Subscribe to a Table

```javascript
// Subscribe to changes on the products table
ws.send(
  JSON.stringify({
    type: "subscribe",
    channel: "table:public.products",
  })
);
```

### 3. Receive Changes

```javascript
ws.onmessage = (event) => {
  const message = JSON.parse(event.data);

  switch (message.type) {
    case "broadcast":
      console.log("Database change:", message.payload);
      // Handle INSERT, UPDATE, or DELETE
      break;
    case "ack":
      console.log("Subscription confirmed");
      break;
    case "heartbeat":
      // Connection is alive
      break;
  }
};
```

### Message Protocol

#### Client → Server Messages

##### Subscribe

```json
{
  "type": "subscribe",
  "channel": "table:public.products"
}
```

##### Unsubscribe

```json
{
  "type": "unsubscribe",
  "channel": "table:public.products"
}
```

#### Server → Client Messages

##### Broadcast (Database Change)

```json
{
  "type": "broadcast",
  "channel": "table:public.products",
  "payload": {
    "type": "INSERT",
    "table": "products",
    "schema": "public",
    "record": {
      "id": 123,
      "name": "New Product",
      "price": 99.99
    }
  }
}
```

##### Update Event (includes old record)

```json
{
  "type": "broadcast",
  "channel": "table:public.products",
  "payload": {
    "type": "UPDATE",
    "table": "products",
    "schema": "public",
    "record": {
      "id": 123,
      "name": "Updated Product",
      "price": 149.99
    },
    "old_record": {
      "id": 123,
      "name": "Old Product",
      "price": 99.99
    }
  }
}
```

##### Delete Event

```json
{
  "type": "broadcast",
  "channel": "table:public.products",
  "payload": {
    "type": "DELETE",
    "table": "products",
    "schema": "public",
    "old_record": {
      "id": 123,
      "name": "Deleted Product",
      "price": 99.99
    }
  }
}
```

##### Acknowledgment

```json
{
  "type": "ack",
  "channel": "table:public.products"
}
```

##### Heartbeat

```json
{
  "type": "heartbeat"
}
```

Sent every 30 seconds to keep the connection alive.

##### Error

```json
{
  "type": "error",
  "error": "Error message here"
}
```

### Channel Naming

Channels follow the format: `table:{schema}.{table_name}`

Examples:

- `table:public.products`
- `table:public.orders`
- `table:auth.users`
- `table:inventory.items`

### Authentication

#### Unauthenticated Connections

```javascript
const ws = new WebSocket("ws://localhost:8080/realtime");
// Can subscribe to tables without RLS
```

#### Authenticated Connections

```javascript
const token = "your.jwt.token.here";
const ws = new WebSocket(`ws://localhost:8080/realtime?token=${token}`);
// User ID is attached to the connection
```

## Enabling Realtime on Tables

Realtime is automatically enabled on tables with the `notify_table_change()` trigger.

### Enable Realtime on a Table

```sql
SELECT enable_realtime('public', 'your_table_name');
```

### Disable Realtime on a Table

```sql
SELECT disable_realtime('public', 'your_table_name');
```

### Check Which Tables Have Realtime

```sql
SELECT
  trigger_schema,
  event_object_table as table_name,
  trigger_name
FROM information_schema.triggers
WHERE trigger_name LIKE '%_notify_change';
```

## Architecture

### PostgreSQL LISTEN/NOTIFY

Fluxbase uses PostgreSQL's built-in LISTEN/NOTIFY system:

1. Database triggers fire on INSERT/UPDATE/DELETE
2. Triggers send notifications via `pg_notify('fluxbase_changes', ...)`
3. Dedicated connection listens on the `fluxbase_changes` channel
4. Notifications are parsed and routed to WebSocket subscribers

### Benefits

- **Lightweight**: No polling or external message queue needed
- **Native**: Built into PostgreSQL
- **Reliable**: Guaranteed delivery within the database
- **Low Latency**: Notifications arrive in milliseconds

### Limitations

- Notifications are lost if no one is listening
- No message history/replay (consider adding if needed)
- No cross-database notifications (single database only)

## Connection Management

- **Heartbeat**: 30-second interval to detect disconnections
- **Auto-cleanup**: Dead connections are automatically removed
- **Reconnection**: Clients should implement exponential backoff
- **Stats Endpoint**: `GET /api/v1/realtime/stats` shows connection count

## Security

### Current Implementation

- JWT authentication supported via query parameter
- User ID attached to authenticated connections
- Basic structure for RLS enforcement in place

### Future Enhancement (TODO)

- Full RLS policy enforcement per user
- Only broadcast changes the user has access to
- Session-based user context in queries

## Monitoring

### Stats Endpoint

```bash
curl http://localhost:8080/api/v1/realtime/stats
```

Response:

```json
{
  "connections": 5,
  "channels": 3
}
```

## Troubleshooting

### Connection Refused

- Ensure the server is running
- Check that WebSocket endpoint is accessible
- Verify firewall rules allow WebSocket connections

### Not Receiving Updates

1. Check if realtime is enabled on the table:

   ```sql
   SELECT * FROM information_schema.triggers
   WHERE event_object_table = 'your_table'
   AND trigger_name LIKE '%_notify_change';
   ```

2. Verify subscription is active (check client message logs)

3. Test direct database changes:
   ```sql
   INSERT INTO products (name, price) VALUES ('Test', 99.99);
   ```

### Authentication Issues

- Verify JWT token is valid
- Check token hasn't expired
- Ensure token is passed in query parameter: `?token=xxx`

## Best Practices

1. **Reconnect on Disconnect**: Implement exponential backoff
2. **Subscribe Selectively**: Only subscribe to tables you need
3. **Handle All Event Types**: INSERT, UPDATE, DELETE
4. **Validate Messages**: Always parse and validate incoming messages
5. **Cleanup Subscriptions**: Unsubscribe when component unmounts
6. **Error Handling**: Handle connection errors gracefully

## Next Steps

- [ ] Add presence tracking for online users
- [ ] Implement message history/replay
- [ ] Add broadcast-only channels (not tied to tables)
- [ ] Full RLS policy enforcement
- [ ] TypeScript SDK with automatic reconnection
- [ ] React hooks package
