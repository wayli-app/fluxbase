-- Rollback RLS Example - Tasks Table

DROP TRIGGER IF EXISTS tasks_updated_at ON public.tasks;
DROP FUNCTION IF EXISTS public.update_updated_at();
DROP TABLE IF EXISTS public.tasks CASCADE;
