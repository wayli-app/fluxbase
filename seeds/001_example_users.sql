-- Example seed file: Test users
-- This file demonstrates how to create seed data for development/testing

-- Insert test users with deterministic UUIDs
INSERT INTO auth.users (id, email, email_confirmed_at, role)
VALUES
  ('00000000-0000-0000-0000-000000000001', 'admin@test.local', NOW(), 'admin'),
  ('00000000-0000-0000-0000-000000000002', 'user@test.local', NOW(), 'authenticated'),
  ('00000000-0000-0000-0000-000000000003', 'demo@test.local', NOW(), 'authenticated')
ON CONFLICT (email) DO NOTHING;
