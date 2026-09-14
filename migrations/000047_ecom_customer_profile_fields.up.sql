ALTER TABLE ecom_customers
    ADD COLUMN IF NOT EXISTS first_name VARCHAR(100),
    ADD COLUMN IF NOT EXISTS last_name VARCHAR(100),
    ADD COLUMN IF NOT EXISTS date_of_birth DATE;

UPDATE ecom_customers
SET first_name = COALESCE(NULLIF(first_name, ''), name)
WHERE first_name IS NULL;
