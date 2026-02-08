-- Add original_amount and currency to incomes (informative; ledger uses amount in USD).
-- Backfill existing rows so we can set NOT NULL.

ALTER TABLE incomes ADD COLUMN IF NOT EXISTS original_amount DOUBLE PRECISION;
ALTER TABLE incomes ADD COLUMN IF NOT EXISTS currency TEXT;

UPDATE incomes SET original_amount = amount WHERE original_amount IS NULL;
UPDATE incomes SET currency = 'USD' WHERE currency IS NULL;

ALTER TABLE incomes ALTER COLUMN original_amount SET NOT NULL;
ALTER TABLE incomes ALTER COLUMN currency SET NOT NULL;

COMMENT ON COLUMN incomes.original_amount IS 'Original amount in source currency (informative); amount is the USD value used for ledger.';
COMMENT ON COLUMN incomes.currency IS 'Currency of original_amount (e.g. USD, EUR); ledger amount is always USD.';
