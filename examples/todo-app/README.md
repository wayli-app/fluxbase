# Todo App - Fluxbase Example

**A simple, production-ready todo list application built with React and Fluxbase**

![Todo App Screenshot](./screenshot.png)

## 🎯 Features

- ✅ User authentication (signup, signin, signout)
- ✅ Create, read, update, delete todos
- ✅ Mark todos as complete/incomplete
- ✅ Filter by status (all, active, completed)
- ✅ Row-Level Security (users only see their own todos)
- ✅ Real-time synchronization across devices
- ✅ Responsive mobile-first design
- ✅ Dark mode support
- ✅ Keyboard shortcuts
- ✅ Offline support (coming soon)

## 🏗️ Architecture

```
Client (React) → Fluxbase SDK → Fluxbase Server → PostgreSQL
                                       ↓
                                   WebSocket (Realtime)
```

**Data Flow**:

1. User authenticates → JWT token stored in SDK
2. User creates todo → INSERT with user_id
3. PostgreSQL RLS ensures user_id matches authenticated user
4. Real-time listener broadcasts change to all connected clients
5. UI updates automatically

## 🚀 Quick Start

### Prerequisites

- Node.js 20+
- Fluxbase instance running (local or deployed)
- PostgreSQL database

### 1. Set Up Database

```sql
-- Create todos table
CREATE TABLE todos (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
  title TEXT NOT NULL,
  completed BOOLEAN DEFAULT FALSE,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Enable RLS
ALTER TABLE todos ENABLE ROW LEVEL SECURITY;

-- Policy: Users can only see their own todos
CREATE POLICY "Users can view own todos"
  ON todos
  FOR SELECT
  USING (user_id::text = current_setting('app.user_id', true));

-- Policy: Users can insert their own todos
CREATE POLICY "Users can insert own todos"
  ON todos
  FOR INSERT
  WITH CHECK (user_id::text = current_setting('app.user_id', true));

-- Policy: Users can update their own todos
CREATE POLICY "Users can update own todos"
  ON todos
  FOR UPDATE
  USING (user_id::text = current_setting('app.user_id', true))
  WITH CHECK (user_id::text = current_setting('app.user_id', true));

-- Policy: Users can delete their own todos
CREATE POLICY "Users can delete own todos"
  ON todos
  FOR DELETE
  USING (user_id::text = current_setting('app.user_id', true));

-- Index for performance
CREATE INDEX idx_todos_user_id ON todos(user_id);
CREATE INDEX idx_todos_completed ON todos(completed);
CREATE INDEX idx_todos_created_at ON todos(created_at DESC);

-- Trigger for updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_todos_updated_at
  BEFORE UPDATE ON todos
  FOR EACH ROW
  EXECUTE FUNCTION update_updated_at_column();
```

### 2. Install Dependencies

```bash
cd examples/todo-app
npm install
```

### 3. Configure Environment

```bash
cp .env.example .env.local
```

Edit `.env.local`:

```env
VITE_FLUXBASE_URL=http://localhost:8080
VITE_FLUXBASE_ANON_KEY=your-anon-key-here
```

Generate keys:

```bash
# Navigate to Fluxbase root
cd ../..

# Generate anon key
./fluxbase generate-key --role anon

# Copy the key to .env.local
```

### 4. Run Development Server

```bash
npm run dev
```

Open [http://localhost:5173](http://localhost:5173)

## 📁 Project Structure

```
todo-app/
├── public/               # Static assets
├── src/
│   ├── components/       # React components
│   │   ├── Auth/        # Auth components
│   │   │   ├── SignIn.tsx
│   │   │   ├── SignUp.tsx
│   │   │   └── AuthGuard.tsx
│   │   ├── Todo/        # Todo components
│   │   │   ├── TodoList.tsx
│   │   │   ├── TodoItem.tsx
│   │   │   ├── TodoForm.tsx
│   │   │   └── TodoFilters.tsx
│   │   └── Layout/      # Layout components
│   │       ├── Header.tsx
│   │       └── Footer.tsx
│   ├── hooks/           # Custom React hooks
│   │   ├── useTodos.ts
│   │   ├── useAuth.ts
│   │   └── useRealtime.ts
│   ├── lib/             # Utilities
│   │   ├── fluxbase.ts  # Fluxbase client
│   │   └── types.ts     # TypeScript types
│   ├── App.tsx          # Main app component
│   ├── main.tsx         # Entry point
│   └── index.css        # Global styles
├── .env.example         # Environment template
├── package.json
├── tsconfig.json
├── vite.config.ts
└── README.md
```

## 💻 Code Examples

### Fluxbase Client Setup

```typescript
// src/lib/fluxbase.ts
import { createClient } from "@fluxbase/client";

export const fluxbase = createClient({
  url: import.meta.env.VITE_FLUXBASE_URL,
  anonKey: import.meta.env.VITE_FLUXBASE_ANON_KEY,
});

export type Todo = {
  id: string;
  user_id: string;
  title: string;
  completed: boolean;
  created_at: string;
  updated_at: string;
};
```

### Custom Hook: useTodos

```typescript
// src/hooks/useTodos.ts
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { fluxbase, type Todo } from "../lib/fluxbase";

export function useTodos(filter: "all" | "active" | "completed" = "all") {
  const queryClient = useQueryClient();

  // Fetch todos
  const { data: todos, isLoading } = useQuery({
    queryKey: ["todos", filter],
    queryFn: async () => {
      let query = fluxbase
        .from<Todo>("todos")
        .select("*")
        .order("created_at", { ascending: false });

      if (filter === "active") {
        query = query.eq("completed", false);
      } else if (filter === "completed") {
        query = query.eq("completed", true);
      }

      const { data, error } = await query;

      if (error) throw error;
      return data || [];
    },
  });

  // Create todo
  const createTodo = useMutation({
    mutationFn: async (title: string) => {
      const { data, error } = await fluxbase
        .from<Todo>("todos")
        .insert({ title, user_id: fluxbase.auth.user()!.id })
        .select()
        .single();

      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["todos"] });
    },
  });

  // Update todo
  const updateTodo = useMutation({
    mutationFn: async ({ id, ...updates }: Partial<Todo> & { id: string }) => {
      const { data, error } = await fluxbase
        .from<Todo>("todos")
        .update(updates)
        .eq("id", id)
        .select()
        .single();

      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["todos"] });
    },
  });

  // Delete todo
  const deleteTodo = useMutation({
    mutationFn: async (id: string) => {
      const { error } = await fluxbase
        .from<Todo>("todos")
        .delete()
        .eq("id", id);

      if (error) throw error;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["todos"] });
    },
  });

  return {
    todos: todos || [],
    isLoading,
    createTodo: createTodo.mutate,
    updateTodo: updateTodo.mutate,
    deleteTodo: deleteTodo.mutate,
  };
}
```

### Custom Hook: useRealtime

```typescript
// src/hooks/useRealtime.ts
import { useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { fluxbase } from "../lib/fluxbase";

export function useRealtimeTodos() {
  const queryClient = useQueryClient();

  useEffect(() => {
    // Subscribe to todo changes
    const subscription = fluxbase
      .from("todos")
      .on("*", () => {
        // Invalidate todos query when any change occurs
        queryClient.invalidateQueries({ queryKey: ["todos"] });
      })
      .subscribe();

    return () => {
      subscription.unsubscribe();
    };
  }, [queryClient]);
}
```

### Todo List Component

```typescript
// src/components/Todo/TodoList.tsx
import { useState } from "react";
import { useTodos } from "../../hooks/useTodos";
import { useRealtimeTodos } from "../../hooks/useRealtime";
import TodoItem from "./TodoItem";
import TodoForm from "./TodoForm";
import TodoFilters from "./TodoFilters";

export default function TodoList() {
  const [filter, setFilter] = useState<"all" | "active" | "completed">("all");
  const { todos, isLoading, createTodo, updateTodo, deleteTodo } =
    useTodos(filter);

  // Enable realtime updates
  useRealtimeTodos();

  if (isLoading) {
    return <div className="text-center p-8">Loading...</div>;
  }

  const activeCount = todos.filter((t) => !t.completed).length;

  return (
    <div className="max-w-2xl mx-auto p-4">
      <h1 className="text-3xl font-bold mb-6">My Todos</h1>

      <TodoForm onSubmit={createTodo} />

      <TodoFilters
        filter={filter}
        onFilterChange={setFilter}
        activeCount={activeCount}
      />

      <div className="space-y-2">
        {todos.length === 0 ? (
          <p className="text-center text-gray-500 p-8">
            No todos yet. Add one above!
          </p>
        ) : (
          todos.map((todo) => (
            <TodoItem
              key={todo.id}
              todo={todo}
              onUpdate={updateTodo}
              onDelete={deleteTodo}
            />
          ))
        )}
      </div>

      {todos.length > 0 && (
        <div className="mt-4 text-sm text-gray-600 text-center">
          {activeCount} {activeCount === 1 ? "item" : "items"} left
        </div>
      )}
    </div>
  );
}
```

### Authentication Component

```typescript
// src/components/Auth/SignIn.tsx
import { useState } from "react";
import { fluxbase } from "../../lib/fluxbase";

export default function SignIn({
  onSwitchToSignUp,
}: {
  onSwitchToSignUp: () => void;
}) {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);

    const { error } = await fluxbase.auth.signIn({
      email,
      password,
    });

    if (error) {
      setError(error.message);
    }

    setLoading(false);
  };

  return (
    <div className="max-w-md mx-auto p-6 bg-white rounded-lg shadow">
      <h2 className="text-2xl font-bold mb-4">Sign In</h2>

      {error && (
        <div className="mb-4 p-3 bg-red-100 text-red-700 rounded">{error}</div>
      )}

      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label htmlFor="email" className="block text-sm font-medium mb-1">
            Email
          </label>
          <input
            id="email"
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
            className="w-full px-3 py-2 border rounded focus:ring-2 focus:ring-blue-500"
          />
        </div>

        <div>
          <label htmlFor="password" className="block text-sm font-medium mb-1">
            Password
          </label>
          <input
            id="password"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
            className="w-full px-3 py-2 border rounded focus:ring-2 focus:ring-blue-500"
          />
        </div>

        <button
          type="submit"
          disabled={loading}
          className="w-full py-2 px-4 bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50"
        >
          {loading ? "Signing in..." : "Sign In"}
        </button>
      </form>

      <div className="mt-4 text-center">
        <button
          onClick={onSwitchToSignUp}
          className="text-blue-600 hover:underline"
        >
          Don't have an account? Sign up
        </button>
      </div>
    </div>
  );
}
```

## 🎨 Styling

The app uses Tailwind CSS for styling. Key design decisions:

- **Mobile-first**: Responsive design works on all screen sizes
- **Accessible**: WCAG 2.1 AA compliant
- **Dark mode**: Respects system preference
- **Animations**: Smooth transitions for better UX

## 🧪 Testing

```bash
# Run unit tests
npm test

# Run E2E tests
npm run test:e2e

# Run with coverage
npm run test:coverage
```

## 🚀 Deployment

### Vercel

```bash
# Install Vercel CLI
npm install -g vercel

# Deploy
vercel

# Add environment variables in Vercel dashboard
```

### Netlify

```bash
# Install Netlify CLI
npm install -g netlify-cli

# Deploy
netlify deploy --prod

# Configure environment variables
netlify env:set VITE_FLUXBASE_URL https://your-fluxbase.com
netlify env:set VITE_FLUXBASE_ANON_KEY your-key
```

### Docker

```dockerfile
# Dockerfile
FROM node:25-alpine AS builder

WORKDIR /app
COPY package*.json ./
RUN npm ci

COPY . .
RUN npm run build

FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
```

```bash
# Build and run
docker build -t todo-app .
docker run -p 80:80 todo-app
```

## 🎓 Learning Objectives

By studying this example, you'll learn:

1. ✅ Setting up Fluxbase client in React
2. ✅ User authentication flow
3. ✅ CRUD operations with TypeScript
4. ✅ Row-Level Security implementation
5. ✅ Real-time data synchronization
6. ✅ Custom React hooks for data fetching
7. ✅ Error handling best practices
8. ✅ Loading states and UX patterns

## 🔧 Customization Ideas

- Add todo categories/tags
- Implement todo sharing between users
- Add due dates and reminders
- Create recurring todos
- Add file attachments
- Implement drag-and-drop reordering
- Add keyboard shortcuts (already has some!)
- Create mobile app with React Native

## 📚 Related Documentation

- [API Cookbook](../../docs/API_COOKBOOK.md)
- [Authentication Guide](../../docs/guides/authentication.md)
- [Row-Level Security](../../docs/guides/rls.md)
- [Realtime Guide](../../docs/guides/realtime.md)

## 🐛 Troubleshooting

See [main examples README](../README.md#troubleshooting) for common issues.

---

**Status**: Complete ✅
**Difficulty**: Beginner
**Time to Complete**: 30 minutes
**Lines of Code**: ~500
