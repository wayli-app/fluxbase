# Realtime Subscriptions

Fluxbase provides real-time database change notifications via WebSockets, powered by PostgreSQL's LISTEN/NOTIFY system.

## Installation

```bash
npm install @fluxbase/sdk
```

## Basic Usage

```typescript
import { createClient } from '@fluxbase/sdk'

const client = createClient('http://localhost:8080', 'your-api-key')

// Subscribe to table changes
const channel = client.realtime
  .channel('table:public.products')
  .on('INSERT', (payload) => {
    console.log('New product:', payload.new_record)
  })
  .on('UPDATE', (payload) => {
    console.log('Updated:', payload.new_record)
    console.log('Previous:', payload.old_record)
  })
  .on('DELETE', (payload) => {
    console.log('Deleted:', payload.old_record)
  })
  .subscribe()

// Or use wildcard for all events
const channel = client.realtime
  .channel('table:public.products')
  .on('*', (payload) => {
    console.log('Event:', payload.type) // INSERT, UPDATE, or DELETE
  })
  .subscribe()

// Unsubscribe
channel.unsubscribe()
```

## Payload Structure

```typescript
interface RealtimeChangePayload {
  type: 'INSERT' | 'UPDATE' | 'DELETE'
  schema: string
  table: string
  new_record?: Record<string, unknown> // INSERT and UPDATE
  old_record?: Record<string, unknown> // UPDATE and DELETE
  timestamp: string
}
```

## React Hook Example

```typescript
import { useEffect, useState } from 'react'
import { createClient } from '@fluxbase/sdk'

function useRealtimeTable(tableName) {
  const [data, setData] = useState([])
  const client = createClient('http://localhost:8080', 'your-api-key')

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
  .channel('table:public.posts')
  .on('*', (payload) => {
    // Only events matching RLS policies are received
    console.log('Post update:', payload)
  })
  .subscribe()
```

## Multiple Subscriptions

Subscribe to multiple tables:

```typescript
const productsChannel = client.realtime
  .channel('table:public.products')
  .on('*', handleProductChange)
  .subscribe()

const ordersChannel = client.realtime
  .channel('table:public.orders')
  .on('*', handleOrderChange)
  .subscribe()

// Cleanup
productsChannel.unsubscribe()
ordersChannel.unsubscribe()
```

## Connection States

Monitor connection status:

```typescript
const channel = client.realtime
  .channel('table:public.products')
  .on('*', handleChange)
  .subscribe((status) => {
    if (status === 'SUBSCRIBED') {
      console.log('Connected and listening')
    } else if (status === 'CHANNEL_ERROR') {
      console.error('Subscription error')
    } else if (status === 'CLOSED') {
      console.log('Connection closed')
    }
  })
```

## Enabling Realtime on Tables

By default, realtime is disabled on tables. Enable it with:

```sql
-- Enable realtime for a table
ALTER TABLE products REPLICA IDENTITY FULL;

-- Create trigger to publish changes
CREATE OR REPLACE FUNCTION notify_table_change()
RETURNS TRIGGER AS $$
BEGIN
  PERFORM pg_notify(
    'table_changes',
    json_build_object(
      'schema', TG_TABLE_SCHEMA,
      'table', TG_TABLE_NAME,
      'type', TG_OP,
      'new_record', CASE WHEN TG_OP != 'DELETE' THEN row_to_json(NEW) END,
      'old_record', CASE WHEN TG_OP != 'INSERT' THEN row_to_json(OLD) END
    )::text
  );
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Attach trigger
CREATE TRIGGER products_notify_change
AFTER INSERT OR UPDATE OR DELETE ON products
FOR EACH ROW EXECUTE FUNCTION notify_table_change();
```

Or use the admin API:

```typescript
await client.admin.enableRealtime('products')
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

For horizontal scaling with multiple Fluxbase instances, you need session stickiness at the load balancer level to ensure WebSocket connections remain on the same instance:

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

**Key points for horizontal scaling:**
- Each Fluxbase instance maintains its own PostgreSQL LISTEN connection
- PostgreSQL broadcasts `pg_notify()` to **all** listening instances simultaneously
- Load balancer uses session stickiness (source IP or cookie-based) to route each client to the same Fluxbase instance
- Requires external PostgreSQL (which also stores authentication sessions shared across instances)
- Sessions work across instances; rate limiting and CSRF are per-instance

See [Deployment: Scaling](/docs/deployment/scaling#horizontal-scaling) for configuration details.

## Connection Management

**Auto-reconnect:** SDK automatically reconnects on connection loss

**Heartbeat:** Periodic ping/pong to detect stale connections

**Cleanup:** Always unsubscribe when done to prevent memory leaks

```typescript
// Good: cleanup in effect
useEffect(() => {
  const channel = client.realtime
    .channel('table:public.products')
    .on('*', handleChange)
    .subscribe()

  return () => channel.unsubscribe() // Cleanup
}, [])
```

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

For non-JavaScript environments, see the [Realtime API Reference](/docs/api/realtime) for WebSocket protocol details.

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

- [Row-Level Security](/docs/guides/row-level-security) - Control data access
- [Authentication](/docs/guides/authentication) - Secure subscriptions
- [Monitoring](/docs/guides/monitoring-observability) - Track realtime performance
