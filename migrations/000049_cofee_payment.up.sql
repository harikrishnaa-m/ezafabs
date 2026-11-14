-- CoFee payment tracking and manual refund records.
ALTER TABLE ecom_orders
    ADD COLUMN IF NOT EXISTS cofee_order_id VARCHAR(100),
    ADD COLUMN IF NOT EXISTS cofee_payment_link VARCHAR(500),
    ADD COLUMN IF NOT EXISTS refunded_amount DECIMAL(12,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS refund_reference VARCHAR(200),
    ADD COLUMN IF NOT EXISTS refunded_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS refund_note TEXT;

ALTER TABLE ecom_returns
    ADD COLUMN IF NOT EXISTS refunded_amount DECIMAL(12,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS refund_reference VARCHAR(200),
    ADD COLUMN IF NOT EXISTS refunded_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS refund_note TEXT;

CREATE INDEX IF NOT EXISTS idx_ecom_orders_cofee_order_id
    ON ecom_orders(cofee_order_id);
