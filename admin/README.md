# Fluxbase Admin UI

The admin dashboard for Fluxbase, built with React 19, TypeScript, and Vite.

## Features

- **Database Management**: Browse and edit database tables with inline editing
- **Real-time Dashboard**: System health metrics and statistics
- **Authentication**: JWT-based login/logout with token refresh
- **Dark Mode**: Built-in theme switching
- **Responsive Design**: Mobile-first, works on all screen sizes
- **Type-safe**: Full TypeScript support with Fluxbase SDK integration

## Tech Stack

- **React 19** - Latest React with concurrent features
- **TypeScript** - Type-safe development
- **Vite** - Fast build tool and dev server
- **TanStack Router** - Type-safe routing
- **TanStack Query** - Data fetching and caching
- **TanStack Table** - Powerful table component with inline editing
- **Shadcn UI** - Beautiful, accessible component library
- **Tailwind CSS v4** - Utility-first styling
- **Radix UI** - Accessible primitives
- **@fluxbase/sdk-react** - Fluxbase React hooks

## Development

### Prerequisites

- Node.js 18+
- Running Fluxbase backend (on port 8080 by default)

### Setup

```bash
# Install dependencies
npm install

# Start dev server
npm run dev
# or use the Makefile from project root
make admin-dev
```

The admin UI will be available at `http://localhost:5173`

### Environment Variables

Create a `.env` file based on `.env.example`:

```bash
VITE_API_URL=http://localhost:8080
VITE_APP_NAME=Fluxbase
```

## Production Build

### Standalone Build

```bash
npm run build
```

This creates a production build in the `dist/` folder.

### Embedded Build

The admin UI is automatically embedded into the Fluxbase binary during production builds:

```bash
# From project root
make build
```

This will:
1. Build the admin UI (`npm run build`)
2. Copy `dist/` to `internal/adminui/dist/`
3. Embed it into the Go binary
4. Serve it at `/admin` route

## Project Structure

```
admin/
├── src/
│   ├── components/     # Reusable UI components
│   ├── features/       # Feature-specific components
│   ├── hooks/          # Custom React hooks
│   ├── lib/            # Utilities and SDK client
│   ├── routes/         # Route components
│   ├── stores/         # Zustand state stores
│   └── main.tsx        # Application entry point
├── public/             # Static assets
├── index.html          # HTML template
└── vite.config.ts      # Vite configuration
```

## Key Components

- **Dashboard**: Real-time system metrics and quick actions
- **Tables Browser**: Browse and edit database tables with:
  - Pagination, sorting, filtering
  - Inline cell editing (click to edit)
  - CRUD operations (create, edit, delete)
  - Schema grouping
- **User Management**: Manage users via the `auth.users` table
- **Settings**: User profile and preferences

## Development Notes

### Hot Module Replacement (HMR)

Vite provides instant HMR for fast development. Changes to React components, styles, and routes update immediately without full page reload.

### Type Safety

The project uses TypeScript throughout with strict mode enabled. The Fluxbase SDK provides full type safety for all API calls.

### Code Formatting

```bash
# Check formatting
npm run format:check

# Auto-format code
npm run format
```

### Linting

```bash
npm run lint
```

## Architecture

The admin UI uses a modern React architecture:

- **TanStack Router** for file-based, type-safe routing
- **TanStack Query** for server state management with automatic caching
- **Zustand** for client state (auth, UI preferences)
- **Fluxbase SDK** for all API communication
- **Component composition** for reusable, maintainable UI

## Credits

Built on top of [Shadcn Admin](https://github.com/satnaing/shadcn-admin) template.

## License

Part of the Fluxbase project. See root LICENSE file for details.
