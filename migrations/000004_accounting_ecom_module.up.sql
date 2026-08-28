-- ============================================================
-- Accounting Module — Tally-Style Double-Entry Bookkeeping
-- ============================================================

-- 1. Account Groups (hierarchical, like Tally groups)
CREATE TABLE account_groups (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name        VARCHAR(100) NOT NULL,
    parent_id   UUID REFERENCES account_groups(id),
    nature      VARCHAR(20) NOT NULL,  -- ASSET, LIABILITY, INCOME, EXPENSE, EQUITY
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

-- 2. Ledger Accounts (Chart of Accounts)
CREATE TABLE ledger_accounts (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    code            VARCHAR(20) UNIQUE NOT NULL,
    name            VARCHAR(150) NOT NULL,
    account_group_id UUID NOT NULL REFERENCES account_groups(id),
    nature          VARCHAR(10) NOT NULL,   -- DEBIT or CREDIT (normal balance side)
    is_system       BOOLEAN DEFAULT FALSE,  -- system-managed accounts can't be deleted
    is_active       BOOLEAN DEFAULT TRUE,
    description     TEXT,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

-- 3. Financial Years
CREATE TABLE financial_years (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name        VARCHAR(50) NOT NULL,       -- e.g. "2025-26"
    start_date  DATE NOT NULL,
    end_date    DATE NOT NULL,
    is_active   BOOLEAN DEFAULT TRUE,
    is_closed   BOOLEAN DEFAULT FALSE,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

-- 4. Vouchers (Tally-style journal entries)
CREATE TABLE vouchers (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    voucher_number    VARCHAR(30) UNIQUE NOT NULL,
    voucher_type      VARCHAR(30) NOT NULL,  -- SALES, PURCHASE, RECEIPT, PAYMENT, JOURNAL, CONTRA
    voucher_date      DATE NOT NULL,
    narration         TEXT,
    ref_type          VARCHAR(30),            -- sales_invoice, purchase_invoice, supplier_payment, sales_payment
    ref_id            UUID,                   -- ID of the source record
    financial_year_id UUID REFERENCES financial_years(id),
    branch_id         UUID REFERENCES branches(id),
    is_cancelled      BOOLEAN DEFAULT FALSE,
    created_by        UUID REFERENCES users(id),
    created_at        TIMESTAMPTZ DEFAULT NOW(),
    updated_at        TIMESTAMPTZ DEFAULT NOW()
);

-- 5. Voucher Lines (debit/credit entries)
CREATE TABLE voucher_lines (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    voucher_id        UUID NOT NULL REFERENCES vouchers(id) ON DELETE CASCADE,
    ledger_account_id UUID NOT NULL REFERENCES ledger_accounts(id),
    debit             DECIMAL(15,2) NOT NULL DEFAULT 0,
    credit            DECIMAL(15,2) NOT NULL DEFAULT 0,
    narration         TEXT,
    created_at        TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_vouchers_date ON vouchers(voucher_date);
CREATE INDEX idx_vouchers_type ON vouchers(voucher_type);
CREATE INDEX idx_vouchers_ref ON vouchers(ref_type, ref_id);
CREATE INDEX idx_vouchers_branch ON vouchers(branch_id);
CREATE INDEX idx_vouchers_fy ON vouchers(financial_year_id);
CREATE INDEX idx_voucher_lines_account ON voucher_lines(ledger_account_id);
CREATE INDEX idx_voucher_lines_voucher ON voucher_lines(voucher_id);

-- Seed: Default Account Groups
INSERT INTO account_groups (id, name, parent_id, nature) VALUES
    ('a0000000-0000-0000-0000-000000000001', 'Assets',              NULL, 'ASSET'),
    ('a0000000-0000-0000-0000-000000000002', 'Current Assets',      'a0000000-0000-0000-0000-000000000001', 'ASSET'),
    ('a0000000-0000-0000-0000-000000000003', 'Fixed Assets',        'a0000000-0000-0000-0000-000000000001', 'ASSET'),
    ('a0000000-0000-0000-0000-000000000004', 'Liabilities',         NULL, 'LIABILITY'),
    ('a0000000-0000-0000-0000-000000000005', 'Current Liabilities', 'a0000000-0000-0000-0000-000000000004', 'LIABILITY'),
    ('a0000000-0000-0000-0000-000000000006', 'Income',              NULL, 'INCOME'),
    ('a0000000-0000-0000-0000-000000000007', 'Direct Income',       'a0000000-0000-0000-0000-000000000006', 'INCOME'),
    ('a0000000-0000-0000-0000-000000000008', 'Expenses',            NULL, 'EXPENSE'),
    ('a0000000-0000-0000-0000-000000000009', 'Direct Expenses',     'a0000000-0000-0000-0000-000000000008', 'EXPENSE'),
    ('a0000000-0000-0000-0000-000000000010', 'Indirect Expenses',   'a0000000-0000-0000-0000-000000000008', 'EXPENSE'),
    ('a0000000-0000-0000-0000-000000000011', 'Equity',              NULL, 'EQUITY');

-- Seed: Default Ledger Accounts
INSERT INTO ledger_accounts (id, code, name, account_group_id, nature, is_system, description) VALUES
    ('b0000000-0000-0000-0000-000000000001', '1001', 'Cash',                 'a0000000-0000-0000-0000-000000000002', 'DEBIT', TRUE, 'Cash in hand'),
    ('b0000000-0000-0000-0000-000000000002', '1002', 'Bank Account',         'a0000000-0000-0000-0000-000000000002', 'DEBIT', TRUE, 'Primary bank account'),
    ('b0000000-0000-0000-0000-000000000003', '1003', 'Accounts Receivable',  'a0000000-0000-0000-0000-000000000002', 'DEBIT', TRUE, 'Trade debtors — customers who owe us'),
    ('b0000000-0000-0000-0000-000000000004', '1004', 'Inventory',            'a0000000-0000-0000-0000-000000000002', 'DEBIT', TRUE, 'Stock-in-hand value'),
    ('b0000000-0000-0000-0000-000000000005', '1005', 'UPI Receivable',       'a0000000-0000-0000-0000-000000000002', 'DEBIT', TRUE, 'UPI payments received'),
    ('b0000000-0000-0000-0000-000000000006', '1006', 'Card Receivable',      'a0000000-0000-0000-0000-000000000002', 'DEBIT', TRUE, 'Card payments received'),
    ('b0000000-0000-0000-0000-000000000010', '2001', 'Accounts Payable',     'a0000000-0000-0000-0000-000000000005', 'CREDIT', TRUE, 'Trade creditors — money we owe suppliers'),
    ('b0000000-0000-0000-0000-000000000011', '2002', 'GST Payable',          'a0000000-0000-0000-0000-000000000005', 'CREDIT', TRUE, 'Output GST collected on sales'),
    ('b0000000-0000-0000-0000-000000000012', '2003', 'GST Receivable',       'a0000000-0000-0000-0000-000000000005', 'DEBIT', TRUE, 'Input tax credit on purchases'),
    ('b0000000-0000-0000-0000-000000000020', '4001', 'Sales Revenue',        'a0000000-0000-0000-0000-000000000007', 'CREDIT', TRUE, 'Revenue from sales'),
    ('b0000000-0000-0000-0000-000000000030', '5001', 'Cost of Goods Sold',   'a0000000-0000-0000-0000-000000000009', 'DEBIT', TRUE, 'Cost of goods sold'),
    ('b0000000-0000-0000-0000-000000000031', '5002', 'Purchase Expense',     'a0000000-0000-0000-0000-000000000009', 'DEBIT', TRUE, 'Direct purchase costs'),
    ('b0000000-0000-0000-0000-000000000040', '5101', 'Discount Allowed',     'a0000000-0000-0000-0000-000000000010', 'DEBIT', TRUE, 'Discounts given to customers'),
    ('b0000000-0000-0000-0000-000000000041', '5102', 'Discount Received',    'a0000000-0000-0000-0000-000000000010', 'CREDIT', TRUE, 'Discounts received from suppliers'),
    ('b0000000-0000-0000-0000-000000000050', '3001', 'Owner Capital',        'a0000000-0000-0000-0000-000000000011', 'CREDIT', TRUE, 'Owner equity');

-- ============================================================
-- E-Commerce Module Tables
-- ============================================================

CREATE TABLE ecom_customers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(200) NOT NULL,
    email VARCHAR(150) UNIQUE NOT NULL,
    phone VARCHAR(30),
    password_hash VARCHAR(255) NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE ecom_addresses (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    customer_id UUID NOT NULL REFERENCES ecom_customers(id) ON DELETE CASCADE,
    label VARCHAR(50) DEFAULT 'Home',
    full_name VARCHAR(200) NOT NULL,
    phone VARCHAR(30) NOT NULL,
    address_line1 VARCHAR(300) NOT NULL,
    address_line2 VARCHAR(300),
    city VARCHAR(100) NOT NULL,
    state VARCHAR(100) NOT NULL,
    pincode VARCHAR(10) NOT NULL,
    is_default BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_ecom_addresses_customer ON ecom_addresses(customer_id);

CREATE TABLE ecom_carts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    customer_id UUID UNIQUE NOT NULL REFERENCES ecom_customers(id) ON DELETE CASCADE,
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE ecom_cart_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    cart_id UUID NOT NULL REFERENCES ecom_carts(id) ON DELETE CASCADE,
    variant_id UUID NOT NULL REFERENCES variants(id),
    quantity INT NOT NULL DEFAULT 1 CHECK (quantity > 0),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(cart_id, variant_id)
);
CREATE INDEX idx_ecom_cart_items_cart ON ecom_cart_items(cart_id);

CREATE TABLE ecom_orders (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_number VARCHAR(50) UNIQUE NOT NULL,
    customer_id UUID NOT NULL REFERENCES ecom_customers(id),
    address_id UUID REFERENCES ecom_addresses(id),
    shipping_name VARCHAR(200),
    shipping_phone VARCHAR(30),
    shipping_address TEXT,
    shipping_city VARCHAR(100),
    shipping_state VARCHAR(100),
    shipping_pincode VARCHAR(10),
    item_count INT NOT NULL DEFAULT 0,
    sub_total DECIMAL(12,2) NOT NULL DEFAULT 0,
    discount_amount DECIMAL(12,2) NOT NULL DEFAULT 0,
    tax_amount DECIMAL(12,2) NOT NULL DEFAULT 0,
    shipping_charge DECIMAL(10,2) NOT NULL DEFAULT 0,
    grand_total DECIMAL(12,2) NOT NULL DEFAULT 0,
    status VARCHAR(30) NOT NULL DEFAULT 'PENDING',
    payment_status VARCHAR(30) NOT NULL DEFAULT 'UNPAID',
    payment_method VARCHAR(50),
    payment_ref VARCHAR(200),
    notes TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_ecom_orders_customer ON ecom_orders(customer_id);
CREATE INDEX idx_ecom_orders_status ON ecom_orders(status);

CREATE TABLE ecom_order_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id UUID NOT NULL REFERENCES ecom_orders(id) ON DELETE CASCADE,
    variant_id UUID NOT NULL REFERENCES variants(id),
    product_name VARCHAR(200) NOT NULL,
    variant_name VARCHAR(150) NOT NULL,
    sku VARCHAR(100) NOT NULL,
    variant_code INT NOT NULL,
    quantity INT NOT NULL DEFAULT 1,
    unit_price DECIMAL(10,2) NOT NULL,
    total_price DECIMAL(12,2) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_ecom_order_items_order ON ecom_order_items(order_id);

CREATE SEQUENCE ecom_order_seq START 1;

ALTER TABLE ecom_customers
    ADD COLUMN google_id VARCHAR(255) UNIQUE,
    ALTER COLUMN password_hash DROP NOT NULL;

ALTER TABLE ecom_customers
    ADD COLUMN reset_token VARCHAR(255),
    ADD COLUMN reset_token_expiry TIMESTAMPTZ;