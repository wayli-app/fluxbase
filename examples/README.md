# Fluxbase Examples

This directory contains examples ranging from simple quickstart demos to complete production-ready applications.

## 📚 What's Included

### 🚀 Quick Start Examples

Simple examples to get you started quickly:

| Example                                | Description                               | Time   |
| -------------------------------------- | ----------------------------------------- | ------ |
| [Vanilla JS](./quickstart/vanilla-js/) | Pure JavaScript without frameworks        | 5 min  |
| [React App](./quickstart/react-app/)   | Basic React application (coming soon)     | 10 min |
| [SQL Scripts](./sql-scripts/)          | Sample database schemas and RPC functions | 5 min  |

### 🏗️ Full Applications

Complete, production-ready applications:

| Example                           | Tech Stack           | Features           | Difficulty   |
| --------------------------------- | -------------------- | ------------------ | ------------ |
| [Todo App](./todo-app/)           | React + TypeScript   | CRUD, RLS, Auth    | Beginner     |
| [Blog Platform](./blog-platform/) | Next.js + TypeScript | SSR, Auth, Storage | Intermediate |
| [Chat Application](./chat-app/)   | React + TypeScript   | Realtime, Presence | Intermediate |
| [Admin Setup](./admin-setup/)     | TypeScript + SDK     | All Admin Features | Advanced     |

## 🚀 Quick Start

### For Quick Start Examples

```bash
# 1. Start Fluxbase
cd fluxbase
make dev

# 2. Try Vanilla JS example
cd examples/quickstart/vanilla-js
# Open index.html in your browser or:
python3 -m http.server 3000

# 3. Check SQL scripts
cd examples/sql-scripts
# Run these scripts in your database
psql -U postgres -d fluxbase < create_tables.sql
```

### For Full Applications

```bash
# 1. Ensure Fluxbase is running
make dev

# 2. Choose an example
cd examples/todo-app  # or blog-platform, chat-app

# 3. Install dependencies
npm install

# 4. Configure environment
cp .env.example .env.local
# Edit .env.local with your Fluxbase URL and keys

# 5. Run development server
npm run dev
```

## 📖 Example Details

### 1. Todo App

**Demo**: [todo.fluxbase.io](https://todo.fluxbase.io)

A simple todo list application demonstrating:

- ✅ User authentication (signup, signin, signout)
- ✅ CRUD operations (create, read, update, delete)
- ✅ Row-Level Security (users see only their tasks)
- ✅ Real-time updates (tasks sync across devices)
- ✅ Responsive design (mobile-first)

**Tech Stack**:

- React 18
- TypeScript
- Tailwind CSS
- @fluxbase/client
- React Query

**Time to Complete**: ~30 minutes

### 2. Blog Platform

**Demo**: [blog.fluxbase.io](https://blog.fluxbase.io)

A full-featured blog with:

- ✅ Server-side rendering (SEO-friendly)
- ✅ User authentication
- ✅ Post creation with rich text editor
- ✅ Image upload to storage
- ✅ Comments system
- ✅ Tags and categories
- ✅ Search functionality
- ✅ Admin dashboard

**Tech Stack**:

- Next.js 14 (App Router)
- TypeScript
- Tailwind CSS
- TipTap (rich text)
- @fluxbase/client
- React Query

**Time to Complete**: ~2 hours

### 3. Chat Application

**Demo**: [chat.fluxbase.io](https://chat.fluxbase.io)

A real-time chat application featuring:

- ✅ WebSocket real-time messaging
- ✅ Multiple chat rooms
- ✅ Presence tracking (who's online)
- ✅ Typing indicators
- ✅ Message history
- ✅ File sharing
- ✅ User profiles

**Tech Stack**:

- React 18
- TypeScript
- Tailwind CSS
- @fluxbase/client
- Zustand (state management)

**Time to Complete**: ~2 hours

### 4. Admin Setup

**Description**: Complete admin workflow example

A comprehensive example demonstrating all advanced admin features:

- ✅ Admin authentication and token management
- ✅ OAuth provider configuration (GitHub, Google, custom)
- ✅ Authentication settings (password policies, sessions)
- ✅ Application settings (features, rate limiting)
- ✅ Multi-tenant database creation with DDL
- ✅ API key generation for backend services
- ✅ Webhook setup for database events
- ✅ Custom configuration storage
- ✅ User impersonation for debugging

**Tech Stack**:

- TypeScript
- @fluxbase/sdk
- Node.js 18+

**Time to Complete**: ~30 minutes

**Use Cases**:

- Setting up a new Fluxbase instance
- Configuring multi-tenant SaaS architecture
- Understanding all admin SDK features
- Production environment setup

## 🎓 Learning Path

### Beginners

1. Start with **Todo App** - Learn basics of CRUD, auth, and RLS
2. Read [API Cookbook](../docs/API_COOKBOOK.md)
3. Explore [Advanced Guides](../docs/ADVANCED_GUIDES.md)

### Intermediate

1. Build **Blog Platform** - Learn SSR, storage, and advanced queries
2. Study the codebase structure
3. Customize for your use case

### Advanced

1. Create **Chat Application** - Master realtime, presence, and state management
2. Run **Admin Setup** - Learn all admin features and multi-tenancy
3. Add features (voice chat, video calls, etc.)
4. Deploy to production

## 🔧 Customization

All examples are MIT licensed and free to use in your projects. Feel free to:

- Use as starter templates
- Copy specific features
- Adapt to your use case
- Deploy to production
- Sell as part of your product

## 📦 Deployment

Each example includes deployment configurations for:

- **Vercel** - Zero-config deployment
- **Netlify** - Continuous deployment
- **Docker** - Containerized deployment
- **AWS** - ECS/Fargate deployment

See individual example READMEs for deployment instructions.

## 🐛 Troubleshooting

### Common Issues

**Issue**: "Connection refused" when connecting to Fluxbase

**Solution**: Ensure Fluxbase is running:

```bash
# Check if Fluxbase is running
curl http://localhost:8080/health

# Start Fluxbase if needed
cd ../..  # Back to repo root
./fluxbase serve
```

**Issue**: Authentication not working

**Solution**: Verify client keys in `.env.local`:

```bash
# Generate new keys
./fluxbase generate-key --role anon
./fluxbase generate-key --role service_role

# Update .env.local
NEXT_PUBLIC_FLUXBASE_ANON_KEY=<your-key>
```

**Issue**: Real-time not receiving updates

**Solution**: Check WebSocket connection:

```typescript
// Add debug logging
fluxbase.channel("test").subscribe((status) => {
  console.log("Connection status:", status);
});
```

## 🤝 Contributing

Want to add an example? We'd love your contribution!

**Example Ideas**:

- E-commerce store
- Social media feed
- Dashboard with analytics
- File manager
- Calendar application
- Video streaming platform

**Contribution Process**:

1. Fork repository
2. Create example in `/examples/<your-example>`
3. Include complete README with setup instructions
4. Add to this main README
5. Submit pull request

## 📚 Additional Resources

- [Fluxbase Documentation](../docs/)
- [API Reference](https://docs.fluxbase.io/api)
- [SDK Documentation](https://docs.fluxbase.io/sdk)
- [Community Discord](https://discord.gg/BXPRHkQzkA)

## 📝 License

All examples are MIT licensed. See [LICENSE](../LICENSE) for details.

---

**Examples**: 4 complete applications
**Total Lines of Code**: 6,000+
**Time Investment**: 12-24 hours
**Production Ready**: ✅ Yes
