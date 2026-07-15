---
title: "Realtime Subscriptions"
description: Subscribe to real-time database changes via WebSockets in Fluxbase. Get instant notifications for INSERT, UPDATE, and DELETE events with RLS support.
---

Fluxbase provides real-time database change notifications via WebSockets, powered by PostgreSQL's LISTEN/NOTIFY system.

## Installation

```bash
npm install @nimbleflux/fluxbase-sdk
```

## Basic Usage

```typescript
import { createClient } from "@nimbleflux/fluxbase-sdk";

const client = createClient("http://localhost:8080", "your-anon-key");

// Subscribe to table changes
const channel = client.realtime
  .channel("table:public.products")
  .on("INSERT", (payload) => {
    console.log("New product:", payload.new_record);
  })
  .on("UPDATE", (payload) => {
    console.log("Updated:", payload.new_record);
    console.log("Previous:", payload.old_record);
  })
  .on("DELETE", (payload) => {
    console.log("Deleted:", payload.old_record);
  })
  .subscribe();

// Or use wildcard for all events
const channel = client.realtime
  .channel("table:public.products")
  .on("*", (payload) => {
    console.log("Event:", payload.type); // INSERT, UPDATE, or DELETE
  })
  .subscribe();

// Unsubscribe
channel.unsubscribe();
```

## Payload Structure

```typescript
interface RealtimeChangePayload {
  type: "INSERT" | "UPDATE" | "DELETE";
  schema: string;
  table: string;
  new_record?: Record<string, unknown>; // INSERT and UPDATE
  old_record?: Record<string, unknown>; // UPDATE and DELETE
  timestamp: string;
}
```

## React Hook Example

```typescript
import { useEffect, useState } from 'react'
import { createClient } from '@nimbleflux/fluxbase-sdk'

function useRealtimeTable(tableName) {
  const [data, setData] = useState([])
  const client = createClient('http://localhost:8080', 'your-anon-key')

  useEffect(() => {
    const channel = client.realtime
      .channel(`table:public.${tableName}`)
      .on('INSERT', (payload) => {
        setData(prev => [...prev, payload.new_record])
      })
      .on('UPDATE', (payload) => {
        setData(prev =>
          prev.map(item =>
            item.id === payload.new_record.id ? payload.new_record : item
          )
        )
      })
      .on('DELETE', (payload) => {
        setData(prev => prev.filter(item => item.id !== payload.old_record.id))
      })
      .subscribe()

    return () => channel.unsubscribe()
  }, [tableName])

  return data
}

// Usage
function ProductList() {
  const products = useRealtimeTable('products')

  return (
    <div>
      {products.map(product => (
        <div key={product.id}>{product.name}</div>
      ))}
    </div>
  )
}
```

## Filtering Updates

Subscribe to specific rows using RLS policies:

```typescript
// Only receive updates for rows user has access to
// Access control is enforced via Row-Level Security policies
const channel = client.realtime
  .channel("table:public.posts")
  .on("*", (payload) => {
    // Only events matching RLS policies are received
    console.log("Post update:", payload);
  })
  .subscribe();
```

## Multiple Subscriptions

Subscribe to multiple tables:

```typescript
const productsChannel = client.realtime
  .channel("table:public.products")
  .on("*", handleProductChange)
  .subscribe();

const ordersChannel = client.realtime
  .channel("table:public.orders")
  .on("*", handleOrderChange)
  .subscribe();

// Cleanup
productsChannel.unsubscribe();
ordersChannel.unsubscribe();
```

## Connection States

Monitor connection status:

```typescript
const channel = client.realtime
  .channel("table:public.products")
  .on("*", handleChange)
  .subscribe((status) => {
    if (status === "SUBSCRIBED") {
      console.log("Connected and listening");
    } else if (status === "CHANNEL_ERROR") {
      console.error("Subscription error");
    } else if (status === "CLOSED") {
      console.log("Connection closed");
    }
  });
```

## Enabling Realtime on Tables

By default, realtime is disabled on user tables. Use the admin API to enable it:

```typescript
// Enable realtime on a table (all events)
await client.admin.realtime.enableRealtime("products");

// Enable with specific events only
await client.admin.realtime.enableRealtime("orders", {
  events: ["INSERT", "UPDATE"], // Skip DELETE events
});

// Exclude large columns from notifications (helps with 8KB pg_notify limit)
await client.admin.realtime.enableRealtime("posts", {
  exclude: ["content", "raw_html"],
});

// Enable on a custom schema
await client.admin.realtime.enableRealtime("events", {
  schema: "analytics",
});
```

### What Happens Under the Hood

When you enable realtime, Fluxbase automatically:

1. Sets `REPLICA IDENTITY FULL` on the table (required for UPDATE/DELETE to include old values)
2. Creates a trigger that calls `pg_notify('fluxbase_changes', ...)` on INSERT/UPDATE/DELETE
3. Registers the table in `realtime.schema_registry` for subscription validation

### Managing Realtime Tables

```typescript
// List all realtime-enabled tables
const { tables, count } = await client.admin.realtime.listTables();
console.log(`${count} tables have realtime enabled`);

// Check status of a specific table
const status = await client.admin.realtime.getStatus("public", "products");
if (status.realtime_enabled) {
  console.log("Events:", status.events.join(", "));
}

// Update configuration (change events or excluded columns)
await client.admin.realtime.updateConfig("public", "products", {
  events: ["INSERT"], // Only track inserts now
  exclude: ["metadata"],
});

// Disable realtime on a table
await client.admin.realtime.disableRealtime("public", "products");
```

### Manual SQL Setup (Alternative)

You can also enable realtime manually with SQL:

```sql
-- Enable realtime for a table
ALTER TABLE products REPLICA IDENTITY FULL;

-- Register in schema registry
INSERT INTO realtime.schema_registry (schema_name, table_name, realtime_enabled, events)
VALUES ('public', 'products', true, ARRAY['INSERT', 'UPDATE', 'DELETE']);

-- Create trigger using the shared notify function
CREATE TRIGGER products_realtime_notify
AFTER INSERT OR UPDATE OR DELETE ON products
FOR EACH ROW EXECUTE FUNCTION public.notify_realtime_change();
```

## Architecture

Fluxbase uses PostgreSQL's LISTEN/NOTIFY system:

1. Database triggers detect changes (INSERT, UPDATE, DELETE)
2. Trigger calls `pg_notify()` to broadcast change
3. Fluxbase server listens to notifications
4. Server forwards changes to subscribed WebSocket clients
5. Clients receive real-time updates

This architecture is lightweight and scales well for moderate traffic.

```mermaid
graph LR
    A[Client App] -->|WebSocket| B[Fluxbase Server]
    B -->|LISTEN| C[(PostgreSQL)]
    C -->|pg_notify| B
    D[Database Trigger] -->|INSERT/UPDATE/DELETE| C
    B -->|Broadcast| A
    B -->|Broadcast| E[Client App 2]
    B -->|Broadcast| F[Client App 3]

    style C fill:#336791,color:#fff
    style B fill:#3178c6,color:#fff
```

### Single-Instance Architecture

In a single-instance deployment, all WebSocket connections are handled by one Fluxbase server. This is the default setup and works well for most use cases.

### Multi-Instance Architecture (Horizontal Scaling)

Fluxbase supports two types of realtime notifications, each handled differently in multi-instance deployments:

#### Database Change Notifications

Database changes (INSERT/UPDATE/DELETE) are automatically broadcast to **all instances** via PostgreSQL's LISTEN/NOTIFY. No extra configuration needed.

```mermaid
graph TB
    A[Client 1] -->|WebSocket| LB[Load Balancer<br/>Session Stickiness]
    B[Client 2] -->|WebSocket| LB
    C[Client 3] -->|WebSocket| LB

    LB -->|Sticky to Instance 1| FB1[Fluxbase Instance 1]
    LB -->|Sticky to Instance 2| FB2[Fluxbase Instance 2]
    LB -->|Sticky to Instance 3| FB3[Fluxbase Instance 3]

    FB1 -->|LISTEN| DB[(PostgreSQL)]
    FB2 -->|LISTEN| DB
    FB3 -->|LISTEN| DB

    DB -->|pg_notify broadcasts<br/>to all instances| FB1
    DB -->|pg_notify broadcasts<br/>to all instances| FB2
    DB -->|pg_notify broadcasts<br/>to all instances| FB3

    style DB fill:#336791,color:#fff
    style FB1 fill:#3178c6,color:#fff
    style FB2 fill:#3178c6,color:#fff
    style FB3 fill:#3178c6,color:#fff
    style LB fill:#ff6b6b,color:#fff
```

#### Application Broadcasts (Cross-Instance)

For application-level broadcasts (chat messages, custom events, presence), configure a distributed pub/sub backend:

```bash
# Option 1: PostgreSQL LISTEN/NOTIFY (default, no extra dependencies)
FLUXBASE_SCALING_BACKEND=postgres

# Option 2: Redis/Dragonfly (for high-scale deployments)
FLUXBASE_SCALING_BACKEND=redis
FLUXBASE_SCALING_REDIS_URL=redis://dragonfly:6379
```

When configured, broadcasts sent via `BroadcastGlobal()` are delivered to clients on **all instances**, not just the originating one.

**Key points for horizontal scaling:**

- Each Fluxbase instance maintains its own PostgreSQL LISTEN connection
- PostgreSQL broadcasts `pg_notify()` to **all** listening instances simultaneously
- Load balancer uses session stickiness (source IP or cookie-based) to route each client to the same Fluxbase instance
- Requires external PostgreSQL (which also stores authentication sessions shared across instances)
- With `postgres` or `redis` scaling backend, rate limiting is shared across instances

See [Deployment: Scaling](/deployment/scaling#horizontal-scaling) for configuration details.

## Connection Management

**Auto-reconnect:** SDK automatically reconnects on connection loss

**Heartbeat:** Periodic ping/pong to detect stale connections

**Cleanup:** Always unsubscribe when done to prevent memory leaks

```typescript
// Good: cleanup in effect
useEffect(() => {
  const channel = client.realtime
    .channel("table:public.products")
    .on("*", handleChange)
    .subscribe();

  return () => channel.unsubscribe(); // Cleanup
}, []);
```

## Presence Tracking

Presence tracking enables real-time user online/offline status and custom state sharing. This is useful for collaborative applications, chat systems, and multiplayer features.

### Tracking User Presence

Track when users join and leave channels:

```typescript
import { createClient } from '@nimbleflux/fluxbase-sdk'

const client = createClient('http://localhost:8080', 'your-anon-key')

// Track presence with custom state
const channel = client.realtime
  .channel('presence:online_users')
  .on('presence', { event: 'sync' }, (payload) => {
    // Initial presence state (all users currently online)
    console.log('Current users:', payload.presences)
  })
  .on('presence', { event: 'join' }, (payload) => {
    // User came online
    console.log('User joined:', payload.key)
    console.log('User state:', payload.presence)
  })
  .on('presence', { event: 'leave' }, (payload) => {
    // User went offline
    console.log('User left:', payload.key)
  })
  .subscribe()

// Track your own presence with custom state
channel.track({
  online: true,
  last_seen: new Date().toISOString(),
  status: 'working',
})
```

### Presence State Structure

Presence data is sent as JSON objects:

```typescript
interface PresenceState {
  online: boolean
  last_seen: string
  status?: 'online' | 'away' | 'busy' | 'offline'
  custom_field?: string  // Any custom data
}

interface PresencePayload {
  event: 'sync' | 'join' | 'leave'
  key: string          // Unique identifier (user ID, session ID, etc.)
  presence: PresenceState
  presences: Record<string, PresenceState[]>  // All presences for 'sync' event
}
```

### Presence Events

| Event | Description | When It Fires |
|-------|-------------|--------------|
| `sync` | Initial presence state | When you first subscribe |
| `join` | User came online | When a user tracks their presence |
| `leave` | User went offline | When a user disconnects or untracks |

### Custom Presence State

Update your presence state in real-time:

```typescript
// Update presence state
channel.track({
  online: true,
  status: 'busy',
  inMeeting: true,
  meetingId: 'abc-123',
})

// Update state later
channel.track({
  online: true,
  status: 'available',
  inMeeting: false,
})
```

### Untracking Presence

Remove your presence when done:

```typescript
// Stop being tracked
channel.untrack()

// Or unsubscribe to automatically untrack
channel.unsubscribe()
```

### Channel Patterns

Presence works with any channel name:

```typescript
// Global presence across app
const globalPresence = client.realtime
  .channel('presence:global')

// Room-specific presence
const roomPresence = client.realtime
  .channel('presence:room:123')

// Document collaboration presence
const docPresence = client.realtime
  .channel('presence:document:abc')
```

### React Example

Build a user list with online indicators:

```typescript
import { useEffect, useState } from 'react'
import { createClient } from '@nimbleflux/fluxbase-sdk'

interface UserPresence {
  key: string
  state: {
    online: boolean
    status: string
    last_seen: string
  }
}

function OnlineUsers({ documentId }: { documentId: string }) {
  const [users, setUsers] = useState<UserPresence[]>([])
  const client = createClient('http://localhost:8080', 'your-anon-key')

  useEffect(() => {
    const channel = client.realtime
      .channel(`presence:document:${documentId}`)
      .on('presence', { event: 'sync' }, (payload) => {
        // Convert presences object to array
        const userArray = Object.entries(payload.presences).map(([key, states]) => ({
          key,
          state: states[0],
        }))
        setUsers(userArray)
      })
      .on('presence', { event: 'join' }, (payload) => {
        setUsers(prev => [...prev, {
          key: payload.key,
          state: payload.presence,
        }])
      })
      .on('presence', { event: 'leave' }, (payload) => {
        setUsers(prev => prev.filter(u => u.key !== payload.key))
      })
      .subscribe()

    // Track your own presence
    channel.track({
      online: true,
      last_seen: new Date().toISOString(),
    })

    return () => {
      channel.untrack()
      channel.unsubscribe()
    }
  }, [documentId])

  return (
    <div>
      <h3>Online Users ({users.filter(u => u.state.online).length})</h3>
      <ul>
        {users.map(user => (
          <li key={user.key}>
            <span style={{
              display: 'inline-block',
              width: 10,
              height: 10,
              borderRadius: '50%',
              backgroundColor: user.state.online ? 'green' : 'gray',
              marginRight: 8,
            }} />
            {user.key} ({user.state.status})
          </li>
        ))}
      </ul>
    </div>
  )
}
```

### Presence with RLS

Presence channels respect authentication but not RLS policies:

```typescript
// Authenticated presence
const authChannel = client.realtime
  .channel('presence:room:private')
  .subscribe()  // Requires authentication

// Anonymous presence (if allowed)
const anonChannel = client.realtime
  .channel('presence:public')
  .subscribe()
```

### Rate Limiting

Presence tracking is subject to the connection limits in [Configuration](/reference/configuration/#realtime) (`max_connections`, `max_connections_per_user`, `max_connections_per_ip`). There are no `realtime.presence.*` config keys.

## Security

Realtime subscriptions respect Row-Level Security policies. Users only receive updates for rows they have permission to view.

```sql
-- Example: Users only see their own posts
CREATE POLICY "Users see own posts"
ON posts
FOR SELECT
USING (current_setting('app.user_id', true)::uuid = user_id);
```

When authenticated, users receive realtime updates only for their own posts.

## Best Practices

**Performance:**

- Limit number of active subscriptions per client
- Unsubscribe from unused channels
- Use wildcard (`*`) when listening to all event types

**Security:**

- Always use RLS policies to control data access
- Validate JWT tokens for authenticated subscriptions
- Never expose sensitive data in realtime payloads

**Reliability:**

- Handle connection errors gracefully
- Implement reconnection logic for long-running apps
- Cache local state to handle brief disconnections

**Debugging:**

- Monitor connection status
- Log payload structures during development
- Use browser DevTools to inspect WebSocket traffic

## Raw WebSocket Protocol

For non-JavaScript environments, see the [Realtime SDK Documentation](/api/sdk/classes/fluxbaserealtime/) for WebSocket protocol details.

## Troubleshooting

**No updates received:**

- Verify realtime is enabled on table (triggers exist)
- Check RLS policies allow access to rows
- Confirm WebSocket connection is established
- Verify channel name matches table: `table:schema.table_name`

**Connection drops:**

- Check network stability
- Verify Fluxbase server is running
- Review firewall/proxy WebSocket support
- Ensure JWT token is valid (not expired)

**Performance issues:**

- Reduce number of subscriptions
- Optimize RLS policies (avoid slow queries)
- Consider aggregating rapid changes client-side
- Monitor PostgreSQL NOTIFY queue size

## Next Steps

- [Row-Level Security](/guides/row-level-security) - Control data access
- [Authentication](/guides/authentication) - Secure subscriptions
- [Monitoring](/guides/monitoring-observability) - Track realtime performance
- [Scaling Guide](/deployment/scaling#horizontal-scaling) - Configure realtime for horizontal scaling
