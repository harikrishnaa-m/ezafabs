-- Revert Google OAuth support.
ALTER TABLE ecom_customers
    DROP COLUMN IF EXISTS google_id,
    ALTER COLUMN password_hash SET NOT NULL;

-- Restore unique constraint on variant_code.
DROP INDEX IF EXISTS idx_variants_variant_code;
ALTER TABLE variants ADD CONSTRAINT variants_variant_code_key UNIQUE (variant_code);
