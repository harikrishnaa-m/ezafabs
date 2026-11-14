-- Map an ERP branch to its corresponding CoFee branch.
ALTER TABLE branches
    ADD COLUMN IF NOT EXISTS cofee_branch_id VARCHAR(100);

CREATE INDEX IF NOT EXISTS idx_branches_cofee_branch_id
    ON branches(cofee_branch_id);
