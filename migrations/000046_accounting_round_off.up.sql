INSERT INTO ledger_accounts (id, code, name, account_group_id, nature, is_system, description)
VALUES (
    'b0000000-0000-0000-0000-000000000042',
    '5103',
    'Round-Off',
    'a0000000-0000-0000-0000-000000000010',
    'DEBIT',
    TRUE,
    'Rounding differences on invoices and vouchers'
)
ON CONFLICT (id) DO NOTHING;