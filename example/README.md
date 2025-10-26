# Fluxbase SDK Examples

This directory contains example applications demonstrating how to use the Fluxbase SDK.

## Examples

### 1. Vanilla JavaScript (`vanilla-js/`)

A simple HTML file demonstrating the SDK without any framework.

**Features:**
- Authentication (sign up, sign in, sign out)
- Database queries (fetch, insert)
- Realtime subscriptions

**To run:**
```bash
cd vanilla-js
# Open index.html in your browser
# Or serve with any static file server:
python3 -m http.server 3000
# Then open http://localhost:3000
```

### 2. React Application (`react-app/`)

Coming soon - A React application using `@fluxbase/sdk-react` hooks.

**Features:**
- Full authentication flow
- CRUD operations
- Realtime updates
- File uploads
- TypeScript support

## Prerequisites

Before running the examples, make sure you have:

1. **Fluxbase server running**
   ```bash
   # From project root
   make dev
   # Or
   go run cmd/fluxbase/main.go
   ```

2. **PostgreSQL database** with the `products` table:
   ```sql
   CREATE TABLE products (
     id SERIAL PRIMARY KEY,
     name VARCHAR(255) NOT NULL,
     price DECIMAL(10, 2) NOT NULL,
     category VARCHAR(100),
     created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
   );
   ```

3. **Enable realtime** for the products table (optional, for realtime examples):
   ```sql
   SELECT enable_realtime('public', 'products');
   ```

## API Configuration

All examples default to `http://localhost:8080` for the Fluxbase API. To use a different URL, update the `url` parameter when creating the client:

```javascript
const client = createClient({
  url: 'https://your-fluxbase-instance.com',
})
```

## Common Issues

### CORS Errors

If you see CORS errors, make sure your Fluxbase server is configured to allow requests from your origin:

```yaml
# config.yaml
server:
  cors:
    allowed_origins:
      - "http://localhost:3000"
      - "http://localhost:5173"
```

### Authentication Errors

Make sure you have the auth tables set up. Run migrations:

```bash
make migrate-up
```

### Realtime Not Working

1. Check that the Fluxbase server has realtime enabled
2. Make sure the table has the realtime trigger:
   ```sql
   SELECT enable_realtime('public', 'products');
   ```
3. Check the browser console for WebSocket connection errors

## Learn More

- [Fluxbase Documentation](../docs/)
- [SDK Documentation](../sdk/README.md)
- [React Hooks Documentation](../sdk-react/README.md)
- [API Reference](../docs/docs/api/)

## Contributing

Feel free to add more examples! See [Contributing Guide](../CONTRIBUTING.md).
