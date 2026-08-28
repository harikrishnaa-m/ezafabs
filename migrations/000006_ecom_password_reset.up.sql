-- Add password reset support to ecom_customers
ALTER TABLE ecom_customers
    ADD COLUMN IF NOT EXISTS reset_token VARCHAR(255),
    ADD COLUMN IF NOT EXISTS reset_token_expiry TIMESTAMPTZ;
