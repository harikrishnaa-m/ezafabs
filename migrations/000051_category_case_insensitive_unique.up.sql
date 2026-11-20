CREATE UNIQUE INDEX IF NOT EXISTS categories_name_lower_unique
ON categories (LOWER(name));
