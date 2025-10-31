-- Sample RPC functions for testing Fluxbase RPC endpoints

-- Simple function that returns a scalar value
CREATE OR REPLACE FUNCTION public.hello(name TEXT DEFAULT 'World')
RETURNS TEXT AS $$
BEGIN
    RETURN 'Hello, ' || name || '!';
END;
$$ LANGUAGE plpgsql VOLATILE;

COMMENT ON FUNCTION public.hello IS 'Returns a greeting message';

-- Function that returns a set of rows
CREATE OR REPLACE FUNCTION public.get_user_stats()
RETURNS TABLE(
    total_users BIGINT,
    active_users BIGINT,
    created_today BIGINT
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        COUNT(*)::BIGINT as total_users,
        COUNT(*) FILTER (WHERE email_confirmed = TRUE)::BIGINT as active_users,
        COUNT(*) FILTER (WHERE created_at::DATE = CURRENT_DATE)::BIGINT as created_today
    FROM auth.users;
END;
$$ LANGUAGE plpgsql STABLE;

COMMENT ON FUNCTION public.get_user_stats IS 'Returns user statistics';

-- Function with multiple parameters
CREATE OR REPLACE FUNCTION public.search_users(
    search_term TEXT,
    limit_count INTEGER DEFAULT 10
)
RETURNS TABLE(
    id UUID,
    email TEXT,
    created_at TIMESTAMPTZ
) AS $$
BEGIN
    RETURN QUERY
    SELECT u.id, u.email, u.created_at
    FROM auth.users u
    WHERE u.email ILIKE '%' || search_term || '%'
    LIMIT limit_count;
END;
$$ LANGUAGE plpgsql STABLE;

COMMENT ON FUNCTION public.search_users IS 'Search users by email pattern';

-- Function that returns JSON
CREATE OR REPLACE FUNCTION public.get_user_info(user_id UUID)
RETURNS JSON AS $$
DECLARE
    result JSON;
BEGIN
    SELECT json_build_object(
        'id', u.id,
        'email', u.email,
        'email_confirmed', u.email_confirmed,
        'created_at', u.created_at,
        'updated_at', u.updated_at
    ) INTO result
    FROM auth.users u
    WHERE u.id = user_id;

    RETURN result;
END;
$$ LANGUAGE plpgsql STABLE;

COMMENT ON FUNCTION public.get_user_info IS 'Get detailed user information as JSON';

-- Function for aggregation (count)
CREATE OR REPLACE FUNCTION public.count_users_by_domain(email_domain TEXT)
RETURNS INTEGER AS $$
BEGIN
    RETURN (
        SELECT COUNT(*)::INTEGER
        FROM auth.users
        WHERE email LIKE '%@' || email_domain
    );
END;
$$ LANGUAGE plpgsql STABLE;

COMMENT ON FUNCTION public.count_users_by_domain IS 'Count users by email domain';

-- Void function (performs action, returns nothing meaningful)
CREATE OR REPLACE FUNCTION public.cleanup_old_tokens()
RETURNS VOID AS $$
BEGIN
    DELETE FROM auth.refresh_tokens
    WHERE expires_at < NOW() - INTERVAL '30 days';
END;
$$ LANGUAGE plpgsql VOLATILE;

COMMENT ON FUNCTION public.cleanup_old_tokens IS 'Cleanup expired refresh tokens';
