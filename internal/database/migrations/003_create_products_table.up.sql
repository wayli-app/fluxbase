-- Create products table for testing database browser
CREATE TABLE IF NOT EXISTS public.products (
  id SERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  description TEXT,
  price DECIMAL(10,2) NOT NULL,
  stock INTEGER DEFAULT 0,
  active BOOLEAN DEFAULT true,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);

-- Insert sample data
INSERT INTO public.products (name, description, price, stock) VALUES
  ('Laptop', 'High-performance laptop for developers', 1299.99, 15),
  ('Keyboard', 'Mechanical keyboard with RGB lighting', 149.99, 42),
  ('Mouse', 'Ergonomic wireless mouse', 79.99, 68),
  ('Monitor', '27-inch 4K display', 499.99, 23),
  ('Headphones', 'Noise-cancelling wireless headphones', 299.99, 31);
