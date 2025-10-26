#!/bin/bash

# Test Database Setup Script
# This script sets up the test database for E2E testing

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Setting up test database...${NC}"

# Database configuration
DB_HOST="${DB_HOST:-postgres}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-postgres}"
DB_PASSWORD="${DB_PASSWORD:-postgres}"
DB_NAME="fluxbase_test"

# Check if PostgreSQL is ready
echo -e "${YELLOW}Checking PostgreSQL connection...${NC}"
until PGPASSWORD=$DB_PASSWORD psql -h "$DB_HOST" -U "$DB_USER" -p "$DB_PORT" -c '\q' 2>/dev/null; do
  echo "Waiting for PostgreSQL to be ready..."
  sleep 2
done

echo -e "${GREEN}PostgreSQL is ready!${NC}"

# Drop and recreate test database
echo -e "${YELLOW}Dropping existing test database (if exists)...${NC}"
PGPASSWORD=$DB_PASSWORD psql -h "$DB_HOST" -U "$DB_USER" -p "$DB_PORT" -c "DROP DATABASE IF EXISTS $DB_NAME;" 2>/dev/null || true

echo -e "${YELLOW}Creating test database...${NC}"
PGPASSWORD=$DB_PASSWORD psql -h "$DB_HOST" -U "$DB_USER" -p "$DB_PORT" -c "CREATE DATABASE $DB_NAME;"

echo -e "${YELLOW}Setting up database extensions...${NC}"
PGPASSWORD=$DB_PASSWORD psql -h "$DB_HOST" -U "$DB_USER" -p "$DB_PORT" -d "$DB_NAME" <<-EOSQL
	-- Enable required extensions
	CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
	CREATE EXTENSION IF NOT EXISTS "pgcrypto";
	CREATE EXTENSION IF NOT EXISTS "pg_trgm";

	-- Create schemas for testing
	CREATE SCHEMA IF NOT EXISTS auth;
	CREATE SCHEMA IF NOT EXISTS storage;
	CREATE SCHEMA IF NOT EXISTS realtime;
	CREATE SCHEMA IF NOT EXISTS functions;

	-- Set search path
	ALTER DATABASE $DB_NAME SET search_path TO public, auth, storage;
EOSQL

echo -e "${YELLOW}Creating test tables...${NC}"
PGPASSWORD=$DB_PASSWORD psql -h "$DB_HOST" -U "$DB_USER" -p "$DB_PORT" -d "$DB_NAME" <<-EOSQL
	-- Auth schema tables (for authentication tests)
	CREATE TABLE IF NOT EXISTS auth.users (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		email TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		email_verified BOOLEAN DEFAULT false,
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS auth.sessions (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
		access_token TEXT NOT NULL,
		refresh_token TEXT NOT NULL,
		expires_at TIMESTAMPTZ NOT NULL,
		created_at TIMESTAMPTZ DEFAULT NOW(),
		UNIQUE(access_token),
		UNIQUE(refresh_token)
	);

	-- Storage schema tables (for file storage tests)
	CREATE TABLE IF NOT EXISTS storage.buckets (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name TEXT UNIQUE NOT NULL,
		public BOOLEAN DEFAULT false,
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS storage.objects (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		bucket_id UUID NOT NULL REFERENCES storage.buckets(id) ON DELETE CASCADE,
		name TEXT NOT NULL,
		size BIGINT NOT NULL,
		mime_type TEXT,
		metadata JSONB DEFAULT '{}',
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW(),
		UNIQUE(bucket_id, name)
	);

	-- Public schema tables (for REST API tests)
	CREATE TABLE IF NOT EXISTS items (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name TEXT NOT NULL,
		description TEXT,
		quantity INTEGER DEFAULT 0,
		active BOOLEAN DEFAULT true,
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS products (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name TEXT NOT NULL,
		price DECIMAL(10,2) NOT NULL,
		category TEXT,
		in_stock BOOLEAN DEFAULT true,
		tags TEXT[] DEFAULT '{}',
		metadata JSONB DEFAULT '{}',
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS categories (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name TEXT UNIQUE NOT NULL,
		description TEXT,
		parent_id UUID REFERENCES categories(id),
		created_at TIMESTAMPTZ DEFAULT NOW()
	);

	-- Create indexes
	CREATE INDEX IF NOT EXISTS idx_users_email ON auth.users(email);
	CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON auth.sessions(user_id);
	CREATE INDEX IF NOT EXISTS idx_sessions_access_token ON auth.sessions(access_token);
	CREATE INDEX IF NOT EXISTS idx_objects_bucket_id ON storage.objects(bucket_id);
	CREATE INDEX IF NOT EXISTS idx_products_category ON products(category);
	CREATE INDEX IF NOT EXISTS idx_products_in_stock ON products(in_stock);

	-- Create triggers for updated_at
	CREATE OR REPLACE FUNCTION update_updated_at_column()
	RETURNS TRIGGER AS \$\$
	BEGIN
		NEW.updated_at = NOW();
		RETURN NEW;
	END;
	\$\$ language 'plpgsql';

	CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON auth.users
		FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

	CREATE TRIGGER update_buckets_updated_at BEFORE UPDATE ON storage.buckets
		FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

	CREATE TRIGGER update_objects_updated_at BEFORE UPDATE ON storage.objects
		FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

	CREATE TRIGGER update_items_updated_at BEFORE UPDATE ON items
		FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

	CREATE TRIGGER update_products_updated_at BEFORE UPDATE ON products
		FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
EOSQL

echo -e "${YELLOW}Inserting seed data for testing...${NC}"
PGPASSWORD=$DB_PASSWORD psql -h "$DB_HOST" -U "$DB_USER" -p "$DB_PORT" -d "$DB_NAME" <<-EOSQL
	-- Insert test products
	INSERT INTO products (name, price, category, in_stock, tags) VALUES
		('Product A', 10.99, 'electronics', true, ARRAY['featured', 'new']),
		('Product B', 25.50, 'electronics', false, ARRAY['sale']),
		('Product C', 5.99, 'books', true, ARRAY['bestseller']),
		('Product D', 15.00, 'books', true, ARRAY['new']),
		('Product E', 99.99, 'electronics', true, ARRAY['premium', 'featured'])
	ON CONFLICT DO NOTHING;

	-- Insert test categories
	INSERT INTO categories (name, description) VALUES
		('Electronics', 'Electronic devices and accessories'),
		('Books', 'Physical and digital books'),
		('Clothing', 'Apparel and fashion items')
	ON CONFLICT (name) DO NOTHING;
EOSQL

echo -e "${GREEN}Test database setup complete!${NC}"
echo -e "${GREEN}Database: ${DB_NAME}${NC}"
echo -e "${GREEN}Host: ${DB_HOST}:${DB_PORT}${NC}"
echo -e "${GREEN}User: ${DB_USER}${NC}"
