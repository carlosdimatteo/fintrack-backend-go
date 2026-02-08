-- Migration 003: Add expected vs real tracking to net_worth_snapshots
-- This adds columns to track expected balances (from transactions) vs real balances (from accounting)

-- Rename total_net_worth to total_real_net_worth (idempotent: only if column exists)
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = 'public' AND table_name = 'net_worth_snapshots' AND column_name = 'total_net_worth'
  ) THEN
    ALTER TABLE net_worth_snapshots RENAME COLUMN total_net_worth TO total_real_net_worth;
  END IF;
END $$;

-- Add expected balance columns
ALTER TABLE net_worth_snapshots ADD COLUMN IF NOT EXISTS expected_fiat_balance NUMERIC(12,2) DEFAULT 0;
ALTER TABLE net_worth_snapshots ADD COLUMN IF NOT EXISTS expected_net_worth NUMERIC(12,2) DEFAULT 0;

-- Add discrepancy columns
ALTER TABLE net_worth_snapshots ADD COLUMN IF NOT EXISTS fiat_discrepancy NUMERIC(12,2) DEFAULT 0;
ALTER TABLE net_worth_snapshots ADD COLUMN IF NOT EXISTS total_discrepancy NUMERIC(12,2) DEFAULT 0;
