-- Example application schema for a blog application

-- Create posts table
CREATE TABLE IF NOT EXISTS posts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title TEXT NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    content TEXT,
    excerpt TEXT,
    author_id UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    published BOOLEAN DEFAULT false,
    published_at TIMESTAMPTZ,
    view_count INTEGER DEFAULT 0,
    featured_image TEXT,
    tags TEXT[],
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create comments table
CREATE TABLE IF NOT EXISTS comments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id UUID REFERENCES posts(id) ON DELETE CASCADE,
    author_id UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    parent_id UUID REFERENCES comments(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    likes INTEGER DEFAULT 0,
    approved BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create categories table
CREATE TABLE IF NOT EXISTS categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT UNIQUE NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    description TEXT,
    parent_id UUID REFERENCES categories(id) ON DELETE SET NULL,
    position INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create posts_categories junction table
CREATE TABLE IF NOT EXISTS posts_categories (
    post_id UUID REFERENCES posts(id) ON DELETE CASCADE,
    category_id UUID REFERENCES categories(id) ON DELETE CASCADE,
    PRIMARY KEY (post_id, category_id)
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_posts_author_id ON posts(author_id);
CREATE INDEX IF NOT EXISTS idx_posts_published ON posts(published);
CREATE INDEX IF NOT EXISTS idx_posts_slug ON posts(slug);
CREATE INDEX IF NOT EXISTS idx_posts_created_at ON posts(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_posts_tags ON posts USING GIN(tags);
CREATE INDEX IF NOT EXISTS idx_posts_metadata ON posts USING GIN(metadata);

CREATE INDEX IF NOT EXISTS idx_comments_post_id ON comments(post_id);
CREATE INDEX IF NOT EXISTS idx_comments_author_id ON comments(author_id);
CREATE INDEX IF NOT EXISTS idx_comments_parent_id ON comments(parent_id);

CREATE INDEX IF NOT EXISTS idx_categories_slug ON categories(slug);
CREATE INDEX IF NOT EXISTS idx_categories_parent_id ON categories(parent_id);

-- Create update triggers for updated_at
CREATE TRIGGER update_posts_updated_at BEFORE UPDATE ON posts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_comments_updated_at BEFORE UPDATE ON comments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_categories_updated_at BEFORE UPDATE ON categories
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Enable Row Level Security (RLS)
ALTER TABLE posts ENABLE ROW LEVEL SECURITY;
ALTER TABLE comments ENABLE ROW LEVEL SECURITY;
ALTER TABLE categories ENABLE ROW LEVEL SECURITY;

-- Create RLS policies

-- Posts policies
CREATE POLICY "Public posts are viewable by everyone"
    ON posts FOR SELECT
    USING (published = true);

CREATE POLICY "Authors can view their own posts"
    ON posts FOR SELECT
    USING (author_id = current_setting('app.current_user_id', true)::UUID);

CREATE POLICY "Authors can insert their own posts"
    ON posts FOR INSERT
    WITH CHECK (author_id = current_setting('app.current_user_id', true)::UUID);

CREATE POLICY "Authors can update their own posts"
    ON posts FOR UPDATE
    USING (author_id = current_setting('app.current_user_id', true)::UUID);

CREATE POLICY "Authors can delete their own posts"
    ON posts FOR DELETE
    USING (author_id = current_setting('app.current_user_id', true)::UUID);

-- Comments policies
CREATE POLICY "Approved comments are viewable by everyone"
    ON comments FOR SELECT
    USING (approved = true);

CREATE POLICY "Users can insert comments"
    ON comments FOR INSERT
    WITH CHECK (author_id = current_setting('app.current_user_id', true)::UUID);

CREATE POLICY "Users can update their own comments"
    ON comments FOR UPDATE
    USING (author_id = current_setting('app.current_user_id', true)::UUID);

CREATE POLICY "Users can delete their own comments"
    ON comments FOR DELETE
    USING (author_id = current_setting('app.current_user_id', true)::UUID);

-- Categories are publicly readable
CREATE POLICY "Categories are viewable by everyone"
    ON categories FOR SELECT
    USING (true);

-- Register tables for realtime updates
INSERT INTO realtime.schema_registry (schema_name, table_name, realtime_enabled, events)
VALUES
    ('public', 'posts', true, ARRAY['INSERT', 'UPDATE', 'DELETE']),
    ('public', 'comments', true, ARRAY['INSERT', 'UPDATE', 'DELETE']),
    ('public', 'categories', true, ARRAY['INSERT', 'UPDATE', 'DELETE'])
ON CONFLICT (schema_name, table_name) DO UPDATE
SET realtime_enabled = EXCLUDED.realtime_enabled,
    events = EXCLUDED.events;

-- Insert sample data
INSERT INTO categories (name, slug, description) VALUES
    ('Technology', 'technology', 'Articles about technology and programming'),
    ('Design', 'design', 'Articles about design and UX'),
    ('Business', 'business', 'Articles about business and startups')
ON CONFLICT (slug) DO NOTHING;

-- Insert sample storage buckets
INSERT INTO storage.buckets (id, name, public) VALUES
    ('avatars', 'avatars', true),
    ('blog-images', 'blog-images', true),
    ('documents', 'documents', false)
ON CONFLICT (id) DO NOTHING;