-- Add bill_discount to sales_orders and sales_invoices
ALTER TABLE sales_orders ADD COLUMN IF NOT EXISTS bill_discount DECIMAL(12,2) NOT NULL DEFAULT 0;
ALTER TABLE sales_invoices ADD COLUMN IF NOT EXISTS bill_discount DECIMAL(12,2) NOT NULL DEFAULT 0;
