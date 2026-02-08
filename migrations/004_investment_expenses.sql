-- Migration 004: Allow expenses from investment accounts
-- This updates the investment_account_summary view to include expenses in capital calculation

-- Create a view for investment account expected capital (similar to fiat expected balance)
CREATE OR REPLACE VIEW investment_account_expected_capital AS
SELECT 
    ia.id,
    ia.name,
    ia.type,
    ia.currency,
    ia.starting_capital,
    -- Deposits increase capital
    COALESCE((
        SELECT SUM(amount) FROM investments 
        WHERE account_id = ia.id AND type = 'deposit'
    ), 0) as total_deposits,
    -- Withdrawals decrease capital
    COALESCE((
        SELECT SUM(amount) FROM investments 
        WHERE account_id = ia.id AND type = 'withdrawal'
    ), 0) as total_withdrawals,
    -- Expenses from this investment account decrease capital
    COALESCE((
        SELECT SUM(expense) FROM expenses 
        WHERE account_id = ia.id AND account_type IN ('Investment', 'Crypto', 'Broker')
    ), 0) as total_expenses,
    -- Expected capital = starting + deposits - withdrawals - expenses
    ia.starting_capital 
        + COALESCE((SELECT SUM(amount) FROM investments WHERE account_id = ia.id AND type = 'deposit'), 0)
        - COALESCE((SELECT SUM(amount) FROM investments WHERE account_id = ia.id AND type = 'withdrawal'), 0)
        - COALESCE((SELECT SUM(expense) FROM expenses WHERE account_id = ia.id AND account_type IN ('Investment', 'Crypto', 'Broker')), 0)
    as expected_capital
FROM investment_accounts ia;

GRANT ALL ON investment_account_expected_capital TO anon, authenticated, service_role;

-- Update investment_account_summary to use expected capital
CREATE OR REPLACE VIEW investment_account_summary AS
SELECT 
    ia.id,
    ia.name,
    ia.type,
    ia.currency,
    ia.balance as real_balance,
    -- Use dynamically calculated capital instead of stored value
    ia.starting_capital 
        + COALESCE((SELECT SUM(amount) FROM investments WHERE account_id = ia.id AND type = 'deposit'), 0)
        - COALESCE((SELECT SUM(amount) FROM investments WHERE account_id = ia.id AND type = 'withdrawal'), 0)
        - COALESCE((SELECT SUM(expense) FROM expenses WHERE account_id = ia.id AND account_type IN ('Investment', 'Crypto', 'Broker')), 0)
    as total_capital,
    ia.starting_capital,
    -- PnL = real_balance - expected_capital
    ia.balance - (
        ia.starting_capital 
        + COALESCE((SELECT SUM(amount) FROM investments WHERE account_id = ia.id AND type = 'deposit'), 0)
        - COALESCE((SELECT SUM(amount) FROM investments WHERE account_id = ia.id AND type = 'withdrawal'), 0)
        - COALESCE((SELECT SUM(expense) FROM expenses WHERE account_id = ia.id AND account_type IN ('Investment', 'Crypto', 'Broker')), 0)
    ) as pnl,
    -- PnL percent
    CASE 
        WHEN (
            ia.starting_capital 
            + COALESCE((SELECT SUM(amount) FROM investments WHERE account_id = ia.id AND type = 'deposit'), 0)
            - COALESCE((SELECT SUM(amount) FROM investments WHERE account_id = ia.id AND type = 'withdrawal'), 0)
            - COALESCE((SELECT SUM(expense) FROM expenses WHERE account_id = ia.id AND account_type IN ('Investment', 'Crypto', 'Broker')), 0)
        ) > 0 
        THEN (
            (ia.balance - (
                ia.starting_capital 
                + COALESCE((SELECT SUM(amount) FROM investments WHERE account_id = ia.id AND type = 'deposit'), 0)
                - COALESCE((SELECT SUM(amount) FROM investments WHERE account_id = ia.id AND type = 'withdrawal'), 0)
                - COALESCE((SELECT SUM(expense) FROM expenses WHERE account_id = ia.id AND account_type IN ('Investment', 'Crypto', 'Broker')), 0)
            )) / (
                ia.starting_capital 
                + COALESCE((SELECT SUM(amount) FROM investments WHERE account_id = ia.id AND type = 'deposit'), 0)
                - COALESCE((SELECT SUM(amount) FROM investments WHERE account_id = ia.id AND type = 'withdrawal'), 0)
                - COALESCE((SELECT SUM(expense) FROM expenses WHERE account_id = ia.id AND account_type IN ('Investment', 'Crypto', 'Broker')), 0)
            )
        ) * 100 
        ELSE 0 
    END as pnl_percent
FROM investment_accounts ia;
