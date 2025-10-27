---
sidebar_position: 4
---

# Realtime API Reference

WebSocket protocol reference for real-time database subscriptions in Fluxbase.

## WebSocket Connection

### Connect

```
ws://localhost:8080/realtime
```

### Authenticated Connection

```
ws://localhost:8080/realtime?token=YOUR_JWT_TOKEN
```

### JavaScript Example

```javascript
const ws = new WebSocket('ws://localhost:8080/realtime?token=YOUR_TOKEN')

ws.onopen = () => {
  console.log('Connected')
}

ws.onmessage = (event) => {
  const message = JSON.parse(event.data)
  console.log('Received:', message)
}

ws.onerror = (error) => {
  console.error('Error:', error)
}

ws.onclose = () => {
  console.log('Disconnected')
}
```

## Message Protocol

All messages are JSON-encoded.

### Client → Server Messages

#### Subscribe to Channel

```json
{
  "type": "subscribe",
  "channel": "table:public.users"
}
```

**Channel Formats:**

| Format | Description | Example |
|--------|-------------|---------|
| `table:{schema}.{table}` | Table changes | `table:public.users` |
| `presence:{room}` | Presence tracking | `presence:chat-room-1` |
| `broadcast:{topic}` | Custom broadcasts | `broadcast:notifications` |

#### Unsubscribe from Channel

```json
{
  "type": "unsubscribe",
  "channel": "table:public.users"
}
```

#### Send Message (Broadcast)

```json
{
  "type": "broadcast",
  "channel": "broadcast:chat",
  "payload": {
    "message": "Hello World",
    "user": "john"
  }
}
```

#### Heartbeat (Ping)

```json
{
  "type": "ping"
}
```

### Server → Client Messages

#### Acknowledgment

```json
{
  "type": "ack",
  "channel": "table:public.users",
  "status": "subscribed"
}
```

#### Database Change Event

```json
{
  "type": "broadcast",
  "channel": "table:public.users",
  "payload": {
    "type": "INSERT",
    "schema": "public",
    "table": "users",
    "record": {
      "id": 123,
      "name": "John Doe",
      "email": "john@example.com"
    }
  }
}
```

#### Update Event

```json
{
  "type": "broadcast",
  "channel": "table:public.users",
  "payload": {
    "type": "UPDATE",
    "schema": "public",
    "table": "users",
    "old_record": {
      "id": 123,
      "name": "John"
    },
    "record": {
      "id": 123,
      "name": "John Doe"
    }
  }
}
```

#### Delete Event

```json
{
  "type": "broadcast",
  "channel": "table:public.users",
  "payload": {
    "type": "DELETE",
    "schema": "public",
    "table": "users",
    "old_record": {
      "id": 123,
      "name": "John Doe"
    }
  }
}
```

#### Heartbeat Response

```json
{
  "type": "pong"
}
```

#### Error

```json
{
  "type": "error",
  "code": "UNAUTHORIZED",
  "message": "Invalid or expired token"
}
```

## Enable Realtime on Tables

```sql
-- Enable realtime for a table
SELECT enable_realtime('users');

-- Disable realtime
SELECT disable_realtime('users');

-- Check if realtime is enabled
SELECT * FROM realtime_enabled_tables;
```

## Complete Example

### HTML + JavaScript

```html
<!DOCTYPE html>
<html>
<head>
  <title>Fluxbase Realtime</title>
</head>
<body>
  <h1>Live User Updates</h1>
  <div id="updates"></div>

  <script>
    const token = 'YOUR_JWT_TOKEN'
    const ws = new WebSocket(`ws://localhost:8080/realtime?token=${token}`)

    ws.onopen = () => {
      console.log('Connected to Fluxbase')

      // Subscribe to users table
      ws.send(JSON.stringify({
        type: 'subscribe',
        channel: 'table:public.users'
      }))
    }

    ws.onmessage = (event) => {
      const message = JSON.parse(event.data)

      if (message.type === 'ack') {
        console.log('Subscribed to', message.channel)
      }

      if (message.type === 'broadcast') {
        const { type, record, old_record } = message.payload

        const div = document.getElementById('updates')
        const p = document.createElement('p')

        switch (type) {
          case 'INSERT':
            p.textContent = `New user: ${record.name}`
            p.style.color = 'green'
            break
          case 'UPDATE':
            p.textContent = `Updated: ${old_record.name} → ${record.name}`
            p.style.color = 'blue'
            break
          case 'DELETE':
            p.textContent = `Deleted: ${old_record.name}`
            p.style.color = 'red'
            break
        }

        div.insertBefore(p, div.firstChild)
      }
    }

    ws.onerror = (error) => {
      console.error('WebSocket error:', error)
    }

    ws.onclose = () => {
      console.log('Disconnected')
    }

    // Heartbeat to keep connection alive
    setInterval(() => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'ping' }))
      }
    }, 30000)
  </script>
</body>
</html>
```

### TypeScript SDK

```typescript
import { createClient } from '@fluxbase/sdk'

const client = createClient({
  url: 'http://localhost:8080',
  auth: { token: 'YOUR_TOKEN' }
})

// Subscribe to table changes
client.realtime
  .channel('table:public.users')
  .on('INSERT', (payload) => {
    console.log('New user:', payload.record)
  })
  .on('UPDATE', (payload) => {
    console.log('Updated:', payload.old_record, '→', payload.record)
  })
  .on('DELETE', (payload) => {
    console.log('Deleted:', payload.old_record)
  })
  .subscribe()

// Unsubscribe
channel.unsubscribe()
```

## Connection States

| State | Description |
|-------|-------------|
| `CONNECTING` | Attempting to connect |
| `OPEN` | Connected and ready |
| `CLOSING` | Closing connection |
| `CLOSED` | Connection closed |

## Error Codes

| Code | Description |
|------|-------------|
| `UNAUTHORIZED` | Invalid or missing token |
| `FORBIDDEN` | Insufficient permissions |
| `INVALID_CHANNEL` | Channel format invalid |
| `SUBSCRIPTION_FAILED` | Could not subscribe |
| `RATE_LIMITED` | Too many connections |

## Limits

| Limit | Default Value |
|-------|---------------|
| Max connections per user | 10 |
| Max subscriptions per connection | 100 |
| Message rate limit | 100 msg/min |
| Heartbeat interval | 30 seconds |
| Connection timeout | 120 seconds |

## Configuration

```yaml
realtime:
  enabled: true
  heartbeat_interval: 30s
  max_connections: 1000
  read_buffer_size: 1024
  write_buffer_size: 1024
```

## Security

### Row-Level Security

Realtime respects PostgreSQL RLS policies:

```sql
-- Only send updates for user's own records
ALTER TABLE todos ENABLE ROW LEVEL SECURITY;

CREATE POLICY todos_realtime_policy ON todos
  FOR SELECT
  USING (user_id = auth.uid());
```

## Troubleshooting

### Connection Refused

- Check Fluxbase is running
- Verify WebSocket port (8080)
- Check firewall rules

### Not Receiving Updates

- Ensure table has realtime enabled
- Check JWT token is valid
- Verify subscription channel format

### Frequent Disconnects

- Implement heartbeat (ping/pong)
- Check network stability
- Increase timeout values

## See Also

- [Realtime Guide](../guides/realtime.md) - Complete realtime documentation
- [SDK Realtime Hooks](../guides/typescript-sdk/react-hooks.md#real-time-hooks) - React integration
- [Configuration](../reference/configuration.md#realtime) - Realtime configuration
