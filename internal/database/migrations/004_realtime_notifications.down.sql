-- Drop triggers
DROP TRIGGER IF EXISTS products_notify_change ON public.products;

-- Drop helper functions
DROP FUNCTION IF EXISTS enable_realtime(TEXT, TEXT);
DROP FUNCTION IF EXISTS disable_realtime(TEXT, TEXT);

-- Drop notification function
DROP FUNCTION IF EXISTS notify_table_change();
