# Blog Platform - Fluxbase Example

**A full-featured blogging platform built with Next.js 14 and Fluxbase**

![Blog Platform Screenshot](./screenshot.png)

## 🎯 Features

### Core Features
- ✅ Server-Side Rendering (SSR) for SEO
- ✅ Static Site Generation (SSG) for performance
- ✅ User authentication (signup, signin, OAuth)
- ✅ Rich text editor (TipTap)
- ✅ Image upload and optimization
- ✅ Comments system
- ✅ Tags and categories
- ✅ Full-text search
- ✅ Reading time estimation
- ✅ Social sharing

### Advanced Features
- ✅ Admin dashboard
- ✅ Draft posts
- ✅ Post scheduling
- ✅ SEO optimization
- ✅ RSS feed
- ✅ Sitemap generation
- ✅ Analytics integration
- ✅ Dark mode

## 🏗️ Architecture

```
Next.js App Router (RSC) → Fluxbase SDK → Fluxbase Server → PostgreSQL
                                                    ↓
                                             Storage (S3/Local)
```

**Data Flow**:
1. Blog posts fetched server-side for SEO
2. Comments loaded client-side for interactivity
3. Images uploaded to Fluxbase Storage
4. Full-text search uses PostgreSQL ts_vector
5. RLS ensures users can only edit their own posts

## 🚀 Quick Start

### Prerequisites

- Node.js 20+
- Fluxbase instance running
- PostgreSQL database

### 1. Set Up Database

```sql
-- Users table (extends auth.users)
CREATE TABLE profiles (
  id UUID PRIMARY KEY REFERENCES auth.users(id) ON DELETE CASCADE,
  username TEXT UNIQUE NOT NULL,
  full_name TEXT,
  avatar_url TEXT,
  bio TEXT,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Posts table
CREATE TABLE posts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  author_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
  title TEXT NOT NULL,
  slug TEXT UNIQUE NOT NULL,
  content TEXT NOT NULL,
  excerpt TEXT,
  cover_image TEXT,
  published BOOLEAN DEFAULT FALSE,
  published_at TIMESTAMP WITH TIME ZONE,
  views INTEGER DEFAULT 0,
  reading_time INTEGER,  -- Minutes
  search_vector tsvector,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Categories table
CREATE TABLE categories (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT UNIQUE NOT NULL,
  slug TEXT UNIQUE NOT NULL,
  description TEXT,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Tags table
CREATE TABLE tags (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT UNIQUE NOT NULL,
  slug TEXT UNIQUE NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Post-Category relationship (many-to-one)
ALTER TABLE posts ADD COLUMN category_id UUID REFERENCES categories(id);

-- Post-Tag relationship (many-to-many)
CREATE TABLE post_tags (
  post_id UUID REFERENCES posts(id) ON DELETE CASCADE,
  tag_id UUID REFERENCES tags(id) ON DELETE CASCADE,
  PRIMARY KEY (post_id, tag_id)
);

-- Comments table
CREATE TABLE comments (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  post_id UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
  author_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
  content TEXT NOT NULL,
  parent_id UUID REFERENCES comments(id) ON DELETE CASCADE,  -- For nested comments
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Likes table
CREATE TABLE post_likes (
  post_id UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  PRIMARY KEY (post_id, user_id)
);

-- Enable RLS
ALTER TABLE profiles ENABLE ROW LEVEL SECURITY;
ALTER TABLE posts ENABLE ROW LEVEL SECURITY;
ALTER TABLE comments ENABLE ROW LEVEL SECURITY;
ALTER TABLE post_likes ENABLE ROW LEVEL SECURITY;

-- RLS Policies for profiles
CREATE POLICY "Public profiles are viewable by everyone"
  ON profiles FOR SELECT USING (true);

CREATE POLICY "Users can update own profile"
  ON profiles FOR UPDATE
  USING (id::text = current_setting('app.user_id', true))
  WITH CHECK (id::text = current_setting('app.user_id', true));

-- RLS Policies for posts
CREATE POLICY "Published posts are viewable by everyone"
  ON posts FOR SELECT
  USING (published = true OR author_id::text = current_setting('app.user_id', true));

CREATE POLICY "Users can insert own posts"
  ON posts FOR INSERT
  WITH CHECK (author_id::text = current_setting('app.user_id', true));

CREATE POLICY "Users can update own posts"
  ON posts FOR UPDATE
  USING (author_id::text = current_setting('app.user_id', true))
  WITH CHECK (author_id::text = current_setting('app.user_id', true));

CREATE POLICY "Users can delete own posts"
  ON posts FOR DELETE
  USING (author_id::text = current_setting('app.user_id', true));

-- RLS Policies for comments
CREATE POLICY "Comments are viewable by everyone"
  ON comments FOR SELECT USING (true);

CREATE POLICY "Authenticated users can insert comments"
  ON comments FOR INSERT
  WITH CHECK (author_id::text = current_setting('app.user_id', true));

CREATE POLICY "Users can update own comments"
  ON comments FOR UPDATE
  USING (author_id::text = current_setting('app.user_id', true));

CREATE POLICY "Users can delete own comments"
  ON comments FOR DELETE
  USING (author_id::text = current_setting('app.user_id', true));

-- Indexes
CREATE INDEX idx_posts_author ON posts(author_id);
CREATE INDEX idx_posts_published ON posts(published, published_at DESC);
CREATE INDEX idx_posts_slug ON posts(slug);
CREATE INDEX idx_posts_category ON posts(category_id);
CREATE INDEX idx_posts_search ON posts USING gin(search_vector);
CREATE INDEX idx_comments_post ON comments(post_id);
CREATE INDEX idx_comments_author ON comments(author_id);
CREATE INDEX idx_post_tags_post ON post_tags(post_id);
CREATE INDEX idx_post_tags_tag ON post_tags(tag_id);

-- Function to update search vector
CREATE OR REPLACE FUNCTION update_post_search_vector()
RETURNS TRIGGER AS $$
BEGIN
  NEW.search_vector :=
    setweight(to_tsvector('english', COALESCE(NEW.title, '')), 'A') ||
    setweight(to_tsvector('english', COALESCE(NEW.excerpt, '')), 'B') ||
    setweight(to_tsvector('english', COALESCE(NEW.content, '')), 'C');
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_posts_search_vector
  BEFORE INSERT OR UPDATE ON posts
  FOR EACH ROW
  EXECUTE FUNCTION update_post_search_vector();

-- Function to calculate reading time
CREATE OR REPLACE FUNCTION calculate_reading_time()
RETURNS TRIGGER AS $$
DECLARE
  word_count INTEGER;
  words_per_minute INTEGER := 200;
BEGIN
  -- Count words in content
  word_count := array_length(regexp_split_to_array(NEW.content, '\s+'), 1);
  NEW.reading_time := CEIL(word_count::FLOAT / words_per_minute);
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER calculate_post_reading_time
  BEFORE INSERT OR UPDATE ON posts
  FOR EACH ROW
  EXECUTE FUNCTION calculate_reading_time();

-- View for post statistics
CREATE VIEW post_stats AS
SELECT
  p.id,
  p.title,
  p.views,
  COUNT(DISTINCT c.id) AS comment_count,
  COUNT(DISTINCT pl.user_id) AS like_count
FROM posts p
LEFT JOIN comments c ON c.post_id = p.id
LEFT JOIN post_likes pl ON pl.post_id = p.id
GROUP BY p.id;
```

### 2. Install Dependencies

```bash
cd examples/blog-platform
npm install
```

### 3. Configure Environment

```bash
cp .env.example .env.local
```

Edit `.env.local`:

```env
# Fluxbase
NEXT_PUBLIC_FLUXBASE_URL=http://localhost:8080
NEXT_PUBLIC_FLUXBASE_ANON_KEY=your-anon-key
FLUXBASE_SERVICE_ROLE_KEY=your-service-key

# Site Config
NEXT_PUBLIC_SITE_URL=http://localhost:3000
NEXT_PUBLIC_SITE_NAME=My Blog
```

### 4. Run Development Server

```bash
npm run dev
```

Open [http://localhost:3000](http://localhost:3000)

## 📁 Project Structure

```
blog-platform/
├── app/                      # Next.js 14 App Router
│   ├── (auth)/              # Auth pages (grouped)
│   │   ├── signin/
│   │   └── signup/
│   ├── (blog)/              # Blog pages (grouped)
│   │   ├── page.tsx         # Home page (SSG)
│   │   ├── post/
│   │   │   └── [slug]/
│   │   │       └── page.tsx # Post page (SSG)
│   │   ├── category/
│   │   │   └── [slug]/
│   │   │       └── page.tsx # Category page
│   │   ├── tag/
│   │   │   └── [slug]/
│   │   │       └── page.tsx # Tag page
│   │   └── search/
│   │       └── page.tsx     # Search page
│   ├── dashboard/           # Admin dashboard
│   │   ├── page.tsx
│   │   ├── posts/
│   │   │   ├── page.tsx     # My posts
│   │   │   ├── new/         # Create post
│   │   │   └── [id]/edit/   # Edit post
│   │   └── profile/
│   │       └── page.tsx
│   ├── api/                 # API routes
│   │   ├── posts/
│   │   └── upload/
│   ├── layout.tsx           # Root layout
│   └── globals.css          # Global styles
├── components/
│   ├── blog/
│   │   ├── PostCard.tsx
│   │   ├── PostList.tsx
│   │   ├── PostContent.tsx
│   │   └── Comments.tsx
│   ├── editor/
│   │   ├── RichTextEditor.tsx
│   │   ├── ImageUpload.tsx
│   │   └── TagInput.tsx
│   ├── layout/
│   │   ├── Header.tsx
│   │   ├── Footer.tsx
│   │   └── Sidebar.tsx
│   └── ui/                  # Reusable UI components
├── lib/
│   ├── fluxbase.ts          # Fluxbase client
│   ├── fluxbase-server.ts   # Server-side client
│   └── utils.ts             # Utilities
├── types/
│   └── index.ts             # TypeScript types
└── public/
    └── images/
```

## 💻 Code Examples

### Server Component (SSG)

```typescript
// app/(blog)/post/[slug]/page.tsx
import { fluxbaseServer } from '@/lib/fluxbase-server'
import PostContent from '@/components/blog/PostContent'
import Comments from '@/components/blog/Comments'
import type { Metadata } from 'next'

// Generate static params for all posts
export async function generateStaticParams() {
  const { data: posts } = await fluxbaseServer
    .from('posts')
    .select('slug')
    .eq('published', true)

  return posts?.map((post) => ({
    slug: post.slug,
  })) || []
}

// Generate metadata for SEO
export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { data: post } = await fluxbaseServer
    .from('posts')
    .select('*, profiles(*), categories(*)')
    .eq('slug', params.slug)
    .single()

  if (!post) return { title: 'Post not found' }

  return {
    title: post.title,
    description: post.excerpt,
    authors: [{ name: post.profiles.full_name }],
    openGraph: {
      title: post.title,
      description: post.excerpt,
      images: post.cover_image ? [post.cover_image] : [],
    },
  }
}

export default async function PostPage({ params }: Props) {
  const { data: post } = await fluxbaseServer
    .from('posts')
    .select(`
      *,
      profiles:author_id (*),
      categories (*),
      tags:post_tags(tags(*))
    `)
    .eq('slug', params.slug)
    .eq('published', true)
    .single()

  if (!post) {
    return <div>Post not found</div>
  }

  // Increment view count (async, don't await)
  fluxbaseServer
    .from('posts')
    .update({ views: post.views + 1 })
    .eq('id', post.id)
    .then()

  return (
    <article className="max-w-4xl mx-auto px-4 py-8">
      <PostContent post={post} />
      <Comments postId={post.id} />
    </article>
  )
}
```

### Rich Text Editor

```typescript
// components/editor/RichTextEditor.tsx
'use client'

import { useEditor, EditorContent } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import Image from '@tiptap/extension-image'
import Link from '@tiptap/extension-link'
import { useState } from 'react'
import ImageUpload from './ImageUpload'

export default function RichTextEditor({
  content,
  onChange
}: {
  content: string
  onChange: (content: string) => void
}) {
  const [showImageUpload, setShowImageUpload] = useState(false)

  const editor = useEditor({
    extensions: [
      StarterKit,
      Image,
      Link.configure({
        openOnClick: false,
      }),
    ],
    content,
    onUpdate: ({ editor }) => {
      onChange(editor.getHTML())
    },
  })

  if (!editor) return null

  const addImage = (url: string) => {
    editor.chain().focus().setImage({ src: url }).run()
    setShowImageUpload(false)
  }

  return (
    <div className="border rounded-lg overflow-hidden">
      {/* Toolbar */}
      <div className="flex gap-2 p-2 border-b bg-gray-50">
        <button
          onClick={() => editor.chain().focus().toggleBold().run()}
          className={editor.isActive('bold') ? 'font-bold' : ''}
        >
          Bold
        </button>
        <button
          onClick={() => editor.chain().focus().toggleItalic().run()}
          className={editor.isActive('italic') ? 'italic' : ''}
        >
          Italic
        </button>
        <button
          onClick={() => editor.chain().focus().toggleHeading({ level: 2 }).run()}
          className={editor.isActive('heading', { level: 2 }) ? 'font-bold' : ''}
        >
          H2
        </button>
        <button
          onClick={() => editor.chain().focus().toggleBulletList().run()}
          className={editor.isActive('bulletList') ? 'font-bold' : ''}
        >
          List
        </button>
        <button onClick={() => setShowImageUpload(true)}>
          Image
        </button>
      </div>

      {/* Editor */}
      <EditorContent
        editor={editor}
        className="prose max-w-none p-4 min-h-[300px]"
      />

      {/* Image Upload Modal */}
      {showImageUpload && (
        <ImageUpload
          onUpload={addImage}
          onClose={() => setShowImageUpload(false)}
        />
      )}
    </div>
  )
}
```

### Image Upload

```typescript
// components/editor/ImageUpload.tsx
'use client'

import { useState } from 'react'
import { fluxbase } from '@/lib/fluxbase'

export default function ImageUpload({
  onUpload,
  onClose
}: {
  onUpload: (url: string) => void
  onClose: () => void
}) {
  const [uploading, setUploading] = useState(false)
  const [error, setError] = useState('')

  const handleUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return

    setUploading(true)
    setError('')

    try {
      // Upload to Fluxbase Storage
      const fileName = `${Date.now()}-${file.name}`
      const { data, error } = await fluxbase.storage
        .from('blog-images')
        .upload(fileName, file)

      if (error) throw error

      // Get public URL
      const { publicURL } = fluxbase.storage
        .from('blog-images')
        .getPublicUrl(fileName)

      onUpload(publicURL)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Upload failed')
    } finally {
      setUploading(false)
    }
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg p-6 max-w-md w-full">
        <h2 className="text-xl font-bold mb-4">Upload Image</h2>

        {error && (
          <div className="mb-4 p-3 bg-red-100 text-red-700 rounded">
            {error}
          </div>
        )}

        <input
          type="file"
          accept="image/*"
          onChange={handleUpload}
          disabled={uploading}
          className="w-full"
        />

        <div className="mt-4 flex gap-2">
          <button
            onClick={onClose}
            className="flex-1 px-4 py-2 border rounded hover:bg-gray-50"
          >
            Cancel
          </button>
        </div>
      </div>
    </div>
  )
}
```

### Search Page

```typescript
// app/(blog)/search/page.tsx
'use client'

import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { fluxbase } from '@/lib/fluxbase'
import PostCard from '@/components/blog/PostCard'

export default function SearchPage() {
  const [query, setQuery] = useState('')

  const { data: results, isLoading } = useQuery({
    queryKey: ['search', query],
    queryFn: async () => {
      if (!query) return []

      const { data, error } = await fluxbase
        .from('posts')
        .select('*, profiles:author_id(*)')
        .textSearch('search_vector', query)
        .eq('published', true)
        .order('published_at', { ascending: false })

      if (error) throw error
      return data || []
    },
    enabled: query.length > 0,
  })

  return (
    <div className="max-w-6xl mx-auto px-4 py-8">
      <h1 className="text-3xl font-bold mb-6">Search Posts</h1>

      <input
        type="search"
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        placeholder="Search for posts..."
        className="w-full px-4 py-3 text-lg border rounded-lg mb-8"
      />

      {isLoading && <div>Searching...</div>}

      {results && results.length === 0 && query && (
        <div className="text-center text-gray-600 py-12">
          No posts found for "{query}"
        </div>
      )}

      <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-6">
        {results?.map((post) => (
          <PostCard key={post.id} post={post} />
        ))}
      </div>
    </div>
  )
}
```

## 🎨 Features Deep Dive

### SEO Optimization

- Server-side rendering for search engines
- Semantic HTML with proper heading hierarchy
- Open Graph meta tags
- JSON-LD structured data
- Dynamic sitemap generation
- Canonical URLs

### Performance

- Static site generation for published posts
- Image optimization with Next.js Image
- Code splitting by route
- Lazy loading for comments
- Incremental Static Regeneration

### Security

- Row-Level Security for data isolation
- XSS protection in rich text editor
- CSRF protection for forms
- Rate limiting on API routes
- Sanitized user input

## 🚀 Deployment

See [deployment guide](./DEPLOYMENT.md) for detailed instructions.

## 📚 Related Documentation

- [Next.js 14 App Router](https://nextjs.org/docs)
- [TipTap Editor](https://tiptap.dev/)
- [API Cookbook](../../docs/API_COOKBOOK.md)

---

**Status**: Complete ✅
**Difficulty**: Intermediate
**Time to Complete**: 2-3 hours
**Lines of Code**: ~2,500
