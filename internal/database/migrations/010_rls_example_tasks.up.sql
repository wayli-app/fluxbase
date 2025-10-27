-- Migration: RLS Example - Tasks Table
-- This migration demonstrates how to use RLS for a multi-tenant tasks table

-- Create example tasks table
CREATE TABLE IF NOT EXISTS public.tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    description TEXT,
    is_public BOOLEAN DEFAULT FALSE,
    completed BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_tasks_user_id ON public.tasks(user_id);
CREATE INDEX idx_tasks_is_public ON public.tasks(is_public) WHERE is_public = TRUE;

COMMENT ON TABLE public.tasks IS
'Example table with RLS enabled - users can only access their own tasks unless is_public is TRUE';

-- Enable RLS on the tasks table (using direct SQL instead of helper function due to transaction limitations)
ALTER TABLE public.tasks ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.tasks FORCE ROW LEVEL SECURITY;

-- Policy 1: Users can select their own tasks
CREATE POLICY tasks_select_own ON public.tasks
    FOR SELECT
    USING (user_id = auth.current_user_id());

-- Policy 2: Admins can select all tasks
CREATE POLICY tasks_select_admin ON public.tasks
    FOR SELECT
    USING (auth.is_admin());

-- Policy 3: Anyone can select public tasks (including anonymous users)
CREATE POLICY tasks_select_public ON public.tasks
    FOR SELECT
    USING (is_public = TRUE);

-- Policy 4: Authenticated users can insert their own tasks
CREATE POLICY tasks_insert_own ON public.tasks
    FOR INSERT
    WITH CHECK (
        auth.is_authenticated()
        AND user_id = auth.current_user_id()
    );

-- Policy 5: Users can update their own tasks
CREATE POLICY tasks_update_own ON public.tasks
    FOR UPDATE
    USING (user_id = auth.current_user_id())
    WITH CHECK (user_id = auth.current_user_id());

-- Policy 6: Admins can update any task
CREATE POLICY tasks_update_admin ON public.tasks
    FOR UPDATE
    USING (auth.is_admin());

-- Policy 7: Users can delete their own tasks
CREATE POLICY tasks_delete_own ON public.tasks
    FOR DELETE
    USING (user_id = auth.current_user_id());

-- Policy 8: Admins can delete any task
CREATE POLICY tasks_delete_admin ON public.tasks
    FOR DELETE
    USING (auth.is_admin());

-- Create updated_at trigger
CREATE OR REPLACE FUNCTION public.update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER tasks_updated_at
    BEFORE UPDATE ON public.tasks
    FOR EACH ROW
    EXECUTE FUNCTION public.update_updated_at();

-- Grant permissions
GRANT SELECT, INSERT, UPDATE, DELETE ON public.tasks TO authenticated;
GRANT SELECT ON public.tasks TO anon;
