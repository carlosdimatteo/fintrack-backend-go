-- Migration 005: Add baseline fields to yearly_goals for savings progress tracking
-- These fields store the starting net worth values at the beginning of the year
-- to calculate savings progress: current - baseline = saved this year
-- Seed data (goals rows, historical incomes, net worth snapshots) → seeds/seed_005_baseline_data.sql

ALTER TABLE yearly_goals ADD COLUMN IF NOT EXISTS baseline_fiat_balance DOUBLE PRECISION DEFAULT 0;
ALTER TABLE yearly_goals ADD COLUMN IF NOT EXISTS baseline_crypto_balance DOUBLE PRECISION DEFAULT 0;
ALTER TABLE yearly_goals ADD COLUMN IF NOT EXISTS baseline_crypto_capital DOUBLE PRECISION DEFAULT 0;
ALTER TABLE yearly_goals ADD COLUMN IF NOT EXISTS baseline_broker_balance DOUBLE PRECISION DEFAULT 0;
ALTER TABLE yearly_goals ADD COLUMN IF NOT EXISTS baseline_broker_capital DOUBLE PRECISION DEFAULT 0;

COMMENT ON COLUMN yearly_goals.baseline_fiat_balance IS 'Fiat balance at start of year (from Dec previous year accounting)';
COMMENT ON COLUMN yearly_goals.baseline_crypto_balance IS 'Crypto real balance at start of year';
COMMENT ON COLUMN yearly_goals.baseline_crypto_capital IS 'Crypto capital at start of year';
COMMENT ON COLUMN yearly_goals.baseline_broker_balance IS 'Broker real balance at start of year';
COMMENT ON COLUMN yearly_goals.baseline_broker_capital IS 'Broker capital at start of year';
