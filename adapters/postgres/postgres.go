package postgres

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	types "github.com/carlosdimatteo/fintrack-backend-go/types"
)

var (
	pool     *pgxpool.Pool
	poolOnce sync.Once
)

// GetPool returns a connection pool (singleton)
func GetPool() (*pgxpool.Pool, error) {
	var initErr error
	poolOnce.Do(func() {
		databaseUrl := os.Getenv("DATABASE_URL")
		if databaseUrl == "" {
			initErr = fmt.Errorf("DATABASE_URL environment variable not set")
			return
		}

		var err error
		pool, err = pgxpool.New(context.Background(), databaseUrl)
		if err != nil {
			initErr = fmt.Errorf("unable to create connection pool: %w", err)
			return
		}

		// Test the connection
		if err := pool.Ping(context.Background()); err != nil {
			initErr = fmt.Errorf("unable to ping database: %w", err)
			return
		}

		fmt.Println("[postgres] Connection pool established")
	})

	if initErr != nil {
		return nil, initErr
	}
	return pool, nil
}

// ClosePool closes the connection pool (call on shutdown)
func ClosePool() {
	if pool != nil {
		pool.Close()
		fmt.Println("[postgres] Connection pool closed")
	}
}

// GetConfigByType retrieves a config row by type
func GetConfigByType(configType string) (types.Config, error) {
	pool, err := GetPool()
	if err != nil {
		return types.Config{}, err
	}

	var config types.Config
	err = pool.QueryRow(context.Background(),
		`SELECT type, sheet, range FROM config WHERE type = $1`,
		configType,
	).Scan(&config.Type, &config.Sheet, &config.A1Range)

	if err != nil {
		if err == pgx.ErrNoRows {
			return types.Config{}, fmt.Errorf("config not found for type: %s", configType)
		}
		return types.Config{}, fmt.Errorf("error querying config: %w", err)
	}

	return config, nil
}

// InsertIncome inserts an income record
func InsertIncome(income types.Income) (types.Income, error) {
	// Validate amount
	if income.Amount <= 0 {
		return types.Income{}, fmt.Errorf("income amount must be positive, got: %.2f", income.Amount)
	}

	pool, err := GetPool()
	if err != nil {
		return types.Income{}, err
	}

	var result types.Income
	err = pool.QueryRow(context.Background(),
		`INSERT INTO incomes (date, amount, description, account_id, account_name)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, date, amount, description, account_id, account_name, created_at`,
		income.Date, income.Amount, income.Description, income.AccountId, income.AccountName,
	).Scan(&result.Id, &result.Date, &result.Amount, &result.Description,
		&result.AccountId, &result.AccountName, &result.CreatedAt)

	if err != nil {
		return types.Income{}, fmt.Errorf("error inserting income: %w", err)
	}

	return result, nil
}

// GetMonthlyIncomeSum returns the total income for a given year and month
func GetMonthlyIncomeSum(year int, month int) (float64, error) {
	pool, err := GetPool()
	if err != nil {
		return 0, err
	}

	var total float64
	err = pool.QueryRow(context.Background(),
		`SELECT COALESCE(total_income, 0) FROM monthly_income_summary 
		 WHERE year = $1 AND month = $2`,
		year, month,
	).Scan(&total)

	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, nil // No income for this month yet
		}
		return 0, fmt.Errorf("error querying monthly income: %w", err)
	}

	return total, nil
}

// GetYearlyIncomeSummary returns income totals for all months in a year
func GetYearlyIncomeSummary(year int) ([]types.MonthlyIncomeSummary, error) {
	pool, err := GetPool()
	if err != nil {
		return nil, err
	}

	rows, err := pool.Query(context.Background(),
		`SELECT year, month, total_income FROM monthly_income_summary 
		 WHERE year = $1 ORDER BY month`,
		year,
	)
	if err != nil {
		return nil, fmt.Errorf("error querying yearly income: %w", err)
	}
	defer rows.Close()

	var results []types.MonthlyIncomeSummary
	for rows.Next() {
		var summary types.MonthlyIncomeSummary
		if err := rows.Scan(&summary.Year, &summary.Month, &summary.TotalIncome); err != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}
		results = append(results, summary)
	}

	return results, nil
}

// GetIncomes retrieves incomes with pagination
func GetIncomes(limit int, offset int) ([]types.Income, int, error) {
	pool, err := GetPool()
	if err != nil {
		return nil, 0, err
	}

	// Get total count
	var count int
	err = pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM incomes`,
	).Scan(&count)
	if err != nil {
		return nil, 0, fmt.Errorf("error counting incomes: %w", err)
	}

	// Get paginated results
	rows, err := pool.Query(context.Background(),
		`SELECT id, date, amount, description, account_id, account_name, created_at 
		 FROM incomes ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("error querying incomes: %w", err)
	}
	defer rows.Close()

	var results []types.Income
	for rows.Next() {
		var income types.Income
		if err := rows.Scan(&income.Id, &income.Date, &income.Amount, &income.Description,
			&income.AccountId, &income.AccountName, &income.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("error scanning row: %w", err)
		}
		results = append(results, income)
	}

	return results, count, nil
}

// GetCategories retrieves all categories
func GetCategories() ([]types.Category, error) {
	pool, err := GetPool()
	if err != nil {
		return nil, err
	}

	rows, err := pool.Query(context.Background(),
		`SELECT id, name, description, is_essential, created_at FROM categories ORDER BY name`,
	)
	if err != nil {
		return nil, fmt.Errorf("error querying categories: %w", err)
	}
	defer rows.Close()

	var results []types.Category
	for rows.Next() {
		var cat types.Category
		if err := rows.Scan(&cat.Id, &cat.Name, &cat.Description, &cat.IsEssential, &cat.CreatedAt); err != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}
		results = append(results, cat)
	}

	return results, nil
}

// InsertExpense inserts an expense record
func InsertExpense(expense types.Expense) (types.Expense, error) {
	// Validate amount
	if expense.Expense <= 0 {
		return types.Expense{}, fmt.Errorf("expense amount must be positive, got: %.2f", expense.Expense)
	}

	pool, err := GetPool()
	if err != nil {
		return types.Expense{}, err
	}

	var result types.Expense
	err = pool.QueryRow(context.Background(),
		`INSERT INTO expenses (date, category, category_id, expense, description, method, "originalAmount", account_id, account_type)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING id, date, category, category_id, expense, description, method, "originalAmount", account_id, account_type`,
		expense.Date, expense.Category, expense.CategoryId, expense.Expense,
		expense.Description, expense.Method, expense.OriginalAmount,
		expense.AccountId, expense.AccountType,
	).Scan(&result.Id, &result.Date, &result.Category, &result.CategoryId,
		&result.Expense, &result.Description, &result.Method, &result.OriginalAmount,
		&result.AccountId, &result.AccountType)

	if err != nil {
		return types.Expense{}, fmt.Errorf("error inserting expense: %w", err)
	}

	return result, nil
}

// InsertInvestment inserts an investment record and updates account capital
func InsertInvestment(investment types.Investment) (types.Investment, error) {
	pool, err := GetPool()
	if err != nil {
		return types.Investment{}, err
	}

	// Validate type
	if investment.Type != "deposit" && investment.Type != "withdrawal" {
		return types.Investment{}, fmt.Errorf("invalid investment type: %s (must be 'deposit' or 'withdrawal')", investment.Type)
	}

	ctx := context.Background()

	// Start transaction
	tx, err := pool.Begin(ctx)
	if err != nil {
		return types.Investment{}, fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Insert investment record
	var result types.Investment
	err = tx.QueryRow(ctx,
		`INSERT INTO investments (date, description, amount, account_id, account_name, type, source_account_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, date, description, amount, account_id, account_name, type, source_account_id`,
		investment.Date, investment.Description, investment.Amount,
		investment.AccountId, investment.AccountName, investment.Type, investment.SourceAccountId,
	).Scan(&result.Id, &result.Date, &result.Description, &result.Amount,
		&result.AccountId, &result.AccountName, &result.Type, &result.SourceAccountId)

	if err != nil {
		return types.Investment{}, fmt.Errorf("error inserting investment: %w", err)
	}

	// Update investment account capital
	// deposit adds to capital, withdrawal subtracts
	capitalChange := investment.Amount
	if investment.Type == "withdrawal" {
		capitalChange = -investment.Amount
	}

	_, err = tx.Exec(ctx,
		`UPDATE investment_accounts SET capital = capital + $1 WHERE id = $2`,
		capitalChange, investment.AccountId,
	)
	if err != nil {
		return types.Investment{}, fmt.Errorf("error updating account capital: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return types.Investment{}, fmt.Errorf("error committing transaction: %w", err)
	}

	return result, nil
}

// GetInvestmentAccountCapital returns the current capital for an investment account
func GetInvestmentAccountCapital(accountId int32) (float64, error) {
	pool, err := GetPool()
	if err != nil {
		return 0, err
	}

	var capital float64
	err = pool.QueryRow(context.Background(),
		`SELECT capital FROM investment_accounts WHERE id = $1`,
		accountId,
	).Scan(&capital)

	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, fmt.Errorf("investment account not found: %d", accountId)
		}
		return 0, fmt.Errorf("error querying capital: %w", err)
	}

	return capital, nil
}

// GetInvestmentAccountSummary returns all investment accounts with PnL
func GetInvestmentAccountSummary() ([]types.InvestmentAccountSummary, error) {
	pool, err := GetPool()
	if err != nil {
		return nil, err
	}

	rows, err := pool.Query(context.Background(),
		`SELECT id, name, type, currency, real_balance, total_capital, starting_capital, pnl, pnl_percent 
		 FROM investment_account_summary`,
	)
	if err != nil {
		return nil, fmt.Errorf("error querying investment summary: %w", err)
	}
	defer rows.Close()

	var results []types.InvestmentAccountSummary
	for rows.Next() {
		var s types.InvestmentAccountSummary
		if err := rows.Scan(&s.Id, &s.Name, &s.Type, &s.Currency, &s.RealBalance,
			&s.TotalCapital, &s.StartingCapital, &s.PnL, &s.PnLPercent); err != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}
		results = append(results, s)
	}

	return results, nil
}

// ========== YEARLY GOALS ==========

// GetYearlyGoals retrieves goals for a specific year
func GetYearlyGoals(year int) (types.YearlyGoals, error) {
	pool, err := GetPool()
	if err != nil {
		return types.YearlyGoals{}, err
	}

	var goals types.YearlyGoals
	err = pool.QueryRow(context.Background(),
		`SELECT id, created_at, year, savings_goal, investment_goal, ideal_investment 
		 FROM yearly_goals WHERE year = $1`,
		year,
	).Scan(&goals.Id, &goals.CreatedAt, &goals.Year, &goals.SavingsGoal,
		&goals.InvestmentGoal, &goals.IdealInvestment)

	if err != nil {
		if err == pgx.ErrNoRows {
			return types.YearlyGoals{Year: year}, nil // Return empty goals for year
		}
		return types.YearlyGoals{}, fmt.Errorf("error querying goals: %w", err)
	}

	return goals, nil
}

// UpsertYearlyGoals creates or updates goals for a year
func UpsertYearlyGoals(goals types.YearlyGoals) (types.YearlyGoals, error) {
	pool, err := GetPool()
	if err != nil {
		return types.YearlyGoals{}, err
	}

	var result types.YearlyGoals
	err = pool.QueryRow(context.Background(),
		`INSERT INTO yearly_goals (year, savings_goal, investment_goal, ideal_investment)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (year) DO UPDATE SET
		   savings_goal = EXCLUDED.savings_goal,
		   investment_goal = EXCLUDED.investment_goal,
		   ideal_investment = EXCLUDED.ideal_investment
		 RETURNING id, created_at, year, savings_goal, investment_goal, ideal_investment`,
		goals.Year, goals.SavingsGoal, goals.InvestmentGoal, goals.IdealInvestment,
	).Scan(&result.Id, &result.CreatedAt, &result.Year, &result.SavingsGoal,
		&result.InvestmentGoal, &result.IdealInvestment)

	if err != nil {
		return types.YearlyGoals{}, fmt.Errorf("error upserting goals: %w", err)
	}

	return result, nil
}

// ========== NET WORTH SNAPSHOTS ==========

// UpsertNetWorthSnapshot creates or updates a snapshot for a year/month
func UpsertNetWorthSnapshot(snapshot types.NetWorthSnapshot) (types.NetWorthSnapshot, error) {
	pool, err := GetPool()
	if err != nil {
		return types.NetWorthSnapshot{}, err
	}

	var result types.NetWorthSnapshot
	err = pool.QueryRow(context.Background(),
		`INSERT INTO net_worth_snapshots (
			date, year, month, total_fiat_balance,
			crypto_balance, crypto_capital, broker_balance, broker_capital,
			total_investment_balance, total_investment_capital,
			total_real_net_worth, total_pnl,
			expected_fiat_balance, expected_net_worth, fiat_discrepancy, total_discrepancy,
			fiat_percent, crypto_percent, broker_percent
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
		ON CONFLICT (year, month) DO UPDATE SET
			date = EXCLUDED.date,
			total_fiat_balance = EXCLUDED.total_fiat_balance,
			crypto_balance = EXCLUDED.crypto_balance,
			crypto_capital = EXCLUDED.crypto_capital,
			broker_balance = EXCLUDED.broker_balance,
			broker_capital = EXCLUDED.broker_capital,
			total_investment_balance = EXCLUDED.total_investment_balance,
			total_investment_capital = EXCLUDED.total_investment_capital,
			total_real_net_worth = EXCLUDED.total_real_net_worth,
			total_pnl = EXCLUDED.total_pnl,
			expected_fiat_balance = EXCLUDED.expected_fiat_balance,
			expected_net_worth = EXCLUDED.expected_net_worth,
			fiat_discrepancy = EXCLUDED.fiat_discrepancy,
			total_discrepancy = EXCLUDED.total_discrepancy,
			fiat_percent = EXCLUDED.fiat_percent,
			crypto_percent = EXCLUDED.crypto_percent,
			broker_percent = EXCLUDED.broker_percent
		RETURNING id, created_at, date, year, month, total_fiat_balance,
			crypto_balance, crypto_capital, broker_balance, broker_capital,
			total_investment_balance, total_investment_capital,
			total_real_net_worth, total_pnl,
			COALESCE(expected_fiat_balance, 0), COALESCE(expected_net_worth, 0),
			COALESCE(fiat_discrepancy, 0), COALESCE(total_discrepancy, 0),
			fiat_percent, crypto_percent, broker_percent`,
		snapshot.Date, snapshot.Year, snapshot.Month, snapshot.TotalFiatBalance,
		snapshot.CryptoBalance, snapshot.CryptoCapital, snapshot.BrokerBalance, snapshot.BrokerCapital,
		snapshot.TotalInvestmentBalance, snapshot.TotalInvestmentCapital,
		snapshot.TotalRealNetWorth, snapshot.TotalPnL,
		snapshot.ExpectedFiatBalance, snapshot.ExpectedNetWorth, snapshot.FiatDiscrepancy, snapshot.TotalDiscrepancy,
		snapshot.FiatPercent, snapshot.CryptoPercent, snapshot.BrokerPercent,
	).Scan(&result.Id, &result.CreatedAt, &result.Date, &result.Year, &result.Month,
		&result.TotalFiatBalance, &result.CryptoBalance, &result.CryptoCapital,
		&result.BrokerBalance, &result.BrokerCapital, &result.TotalInvestmentBalance,
		&result.TotalInvestmentCapital, &result.TotalRealNetWorth, &result.TotalPnL,
		&result.ExpectedFiatBalance, &result.ExpectedNetWorth, &result.FiatDiscrepancy, &result.TotalDiscrepancy,
		&result.FiatPercent, &result.CryptoPercent, &result.BrokerPercent)

	if err != nil {
		return types.NetWorthSnapshot{}, fmt.Errorf("error upserting snapshot: %w", err)
	}

	return result, nil
}

// GetNetWorthHistory retrieves all snapshots ordered by date
func GetNetWorthHistory() ([]types.NetWorthSnapshot, error) {
	pool, err := GetPool()
	if err != nil {
		return nil, err
	}

	rows, err := pool.Query(context.Background(),
		`SELECT id, created_at, date, year, month, total_fiat_balance,
			crypto_balance, crypto_capital, broker_balance, broker_capital,
			total_investment_balance, total_investment_capital,
			total_real_net_worth, total_pnl,
			COALESCE(expected_fiat_balance, 0), COALESCE(expected_net_worth, 0),
			COALESCE(fiat_discrepancy, 0), COALESCE(total_discrepancy, 0),
			fiat_percent, crypto_percent, broker_percent
		 FROM net_worth_snapshots ORDER BY year, month`,
	)
	if err != nil {
		return nil, fmt.Errorf("error querying snapshots: %w", err)
	}
	defer rows.Close()

	var results []types.NetWorthSnapshot
	for rows.Next() {
		var s types.NetWorthSnapshot
		if err := rows.Scan(&s.Id, &s.CreatedAt, &s.Date, &s.Year, &s.Month,
			&s.TotalFiatBalance, &s.CryptoBalance, &s.CryptoCapital,
			&s.BrokerBalance, &s.BrokerCapital, &s.TotalInvestmentBalance,
			&s.TotalInvestmentCapital, &s.TotalRealNetWorth, &s.TotalPnL,
			&s.ExpectedFiatBalance, &s.ExpectedNetWorth, &s.FiatDiscrepancy, &s.TotalDiscrepancy,
			&s.FiatPercent, &s.CryptoPercent, &s.BrokerPercent); err != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}
		results = append(results, s)
	}

	return results, nil
}

// CalculateNetWorthSnapshot calculates current net worth from accounts
func CalculateNetWorthSnapshot(year int, month int) (types.NetWorthSnapshot, error) {
	pool, err := GetPool()
	if err != nil {
		return types.NetWorthSnapshot{}, err
	}

	snapshot := types.NetWorthSnapshot{
		Date:  time.Now(),
		Year:  year,
		Month: month,
	}

	ctx := context.Background()

	// Get real fiat balance (from accounting)
	err = pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(balance), 0) FROM accounts`,
	).Scan(&snapshot.TotalFiatBalance)
	if err != nil {
		return types.NetWorthSnapshot{}, fmt.Errorf("error getting fiat balance: %w", err)
	}

	// Get expected fiat balance (from transactions via account_expected_balance view)
	err = pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(expected_balance), 0) FROM account_expected_balance`,
	).Scan(&snapshot.ExpectedFiatBalance)
	if err != nil {
		// View might not exist yet, default to 0
		snapshot.ExpectedFiatBalance = 0
	}

	// Get crypto accounts (type = 'Crypto')
	err = pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(balance), 0), COALESCE(SUM(capital), 0) 
		 FROM investment_accounts WHERE type = 'Crypto'`,
	).Scan(&snapshot.CryptoBalance, &snapshot.CryptoCapital)
	if err != nil {
		return types.NetWorthSnapshot{}, fmt.Errorf("error getting crypto: %w", err)
	}

	// Get broker accounts (type = 'Broker')
	err = pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(balance), 0), COALESCE(SUM(capital), 0) 
		 FROM investment_accounts WHERE type = 'Broker'`,
	).Scan(&snapshot.BrokerBalance, &snapshot.BrokerCapital)
	if err != nil {
		return types.NetWorthSnapshot{}, fmt.Errorf("error getting broker: %w", err)
	}

	// Calculate totals
	snapshot.TotalInvestmentBalance = snapshot.CryptoBalance + snapshot.BrokerBalance
	snapshot.TotalInvestmentCapital = snapshot.CryptoCapital + snapshot.BrokerCapital
	snapshot.TotalRealNetWorth = snapshot.TotalFiatBalance + snapshot.TotalInvestmentBalance
	snapshot.TotalPnL = snapshot.TotalInvestmentBalance - snapshot.TotalInvestmentCapital

	// Expected net worth = expected fiat + real investment balances (investments are always real from reconciliation)
	snapshot.ExpectedNetWorth = snapshot.ExpectedFiatBalance + snapshot.TotalInvestmentBalance

	// Calculate discrepancies
	snapshot.FiatDiscrepancy = snapshot.TotalFiatBalance - snapshot.ExpectedFiatBalance
	snapshot.TotalDiscrepancy = snapshot.TotalRealNetWorth - snapshot.ExpectedNetWorth

	// Calculate percentages (based on real net worth)
	if snapshot.TotalRealNetWorth > 0 {
		snapshot.FiatPercent = (snapshot.TotalFiatBalance / snapshot.TotalRealNetWorth) * 100
		snapshot.CryptoPercent = (snapshot.CryptoBalance / snapshot.TotalRealNetWorth) * 100
		snapshot.BrokerPercent = (snapshot.BrokerBalance / snapshot.TotalRealNetWorth) * 100
	}

	return snapshot, nil
}

// ========== ACCOUNTS ==========

// GetAccounts retrieves all fiat accounts
func GetAccounts() ([]types.Account, error) {
	pool, err := GetPool()
	if err != nil {
		return nil, err
	}

	rows, err := pool.Query(context.Background(),
		`SELECT id, name, COALESCE(description, ''), COALESCE(type, ''), COALESCE(currency, 'USD'), balance,
			COALESCE(starting_balance, 0), COALESCE(starting_date, NOW())
		 FROM accounts ORDER BY name`,
	)
	if err != nil {
		return nil, fmt.Errorf("error querying accounts: %w", err)
	}
	defer rows.Close()

	var results []types.Account
	for rows.Next() {
		var a types.Account
		if err := rows.Scan(&a.Id, &a.Name, &a.Description, &a.Type, &a.Currency, &a.Balance,
			&a.StartingBalance, &a.StartingDate); err != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}
		results = append(results, a)
	}

	return results, nil
}

// GetInvestmentAccounts retrieves all investment accounts
func GetInvestmentAccounts() ([]types.InvestmentAccount, error) {
	pool, err := GetPool()
	if err != nil {
		return nil, err
	}

	rows, err := pool.Query(context.Background(),
		`SELECT id, name, COALESCE(description, ''), COALESCE(type, ''), COALESCE(currency, 'USD'), 
			balance, COALESCE(capital, 0), COALESCE(starting_capital, 0), COALESCE(starting_date, NOW())
		 FROM investment_accounts ORDER BY name`,
	)
	if err != nil {
		return nil, fmt.Errorf("error querying investment accounts: %w", err)
	}
	defer rows.Close()

	var results []types.InvestmentAccount
	for rows.Next() {
		var a types.InvestmentAccount
		if err := rows.Scan(&a.Id, &a.Name, &a.Description, &a.Type, &a.Currency, &a.Balance,
			&a.Capital, &a.StartingCapital, &a.StartingDate); err != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}
		results = append(results, a)
	}

	return results, nil
}

// UpdateAccountBalance updates the balance for a fiat account
func UpdateAccountBalance(accountId int32, balance float64) error {
	pool, err := GetPool()
	if err != nil {
		return err
	}

	_, err = pool.Exec(context.Background(),
		`UPDATE accounts SET balance = $1 WHERE id = $2`,
		balance, accountId,
	)
	if err != nil {
		return fmt.Errorf("error updating account balance: %w", err)
	}

	return nil
}

// UpdateInvestmentAccountBalance updates the balance for an investment account
func UpdateInvestmentAccountBalance(accountId int32, balance float64) error {
	pool, err := GetPool()
	if err != nil {
		return err
	}

	_, err = pool.Exec(context.Background(),
		`UPDATE investment_accounts SET balance = $1 WHERE id = $2`,
		balance, accountId,
	)
	if err != nil {
		return fmt.Errorf("error updating investment account balance: %w", err)
	}

	return nil
}

// UpdateAccountBalances updates balances for multiple accounts (for accounting)
func UpdateAccountBalances(accounts []types.Account) ([]types.Account, error) {
	pool, err := GetPool()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	var updated []types.Account

	for _, account := range accounts {
		var result types.Account
		err := pool.QueryRow(ctx,
			`UPDATE accounts SET balance = $1 WHERE id = $2
			 RETURNING id, name, COALESCE(description, ''), COALESCE(type, ''), COALESCE(currency, 'USD'), balance`,
			account.Balance, account.Id,
		).Scan(&result.Id, &result.Name, &result.Description, &result.Type, &result.Currency, &result.Balance)

		if err != nil {
			return nil, fmt.Errorf("error updating account %d: %w", account.Id, err)
		}
		updated = append(updated, result)
	}

	return updated, nil
}

// UpdateInvestmentAccountBalances updates balances for multiple investment accounts (for accounting)
func UpdateInvestmentAccountBalances(accounts []types.InvestmentAccount) ([]types.InvestmentAccount, error) {
	pool, err := GetPool()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	var updated []types.InvestmentAccount

	for _, account := range accounts {
		var result types.InvestmentAccount
		err := pool.QueryRow(ctx,
			`UPDATE investment_accounts SET balance = $1 WHERE id = $2
			 RETURNING id, name, COALESCE(description, ''), COALESCE(type, ''), COALESCE(currency, 'USD'), balance, COALESCE(capital, 0)`,
			account.Balance, account.Id,
		).Scan(&result.Id, &result.Name, &result.Description, &result.Type, &result.Currency, &result.Balance, &result.Capital)

		if err != nil {
			return nil, fmt.Errorf("error updating investment account %d: %w", account.Id, err)
		}
		updated = append(updated, result)
	}

	return updated, nil
}

// ========== DEBTS ==========

// InsertDebt inserts a debt record
func InsertDebt(debt types.Debt) (types.Debt, error) {
	pool, err := GetPool()
	if err != nil {
		return types.Debt{}, err
	}

	var result types.Debt
	err = pool.QueryRow(context.Background(),
		`INSERT INTO debts (description, amount, debtor_id, debtor_name, date, original_amount, currency, outbound, account_id, expense_id, income_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 RETURNING id, description, amount, debtor_id, debtor_name, date, created_at, original_amount, currency, outbound`,
		debt.Description, debt.Amount, debt.DebtorId, debt.DebtorName, debt.Date,
		debt.OriginalAmount, debt.Currency, debt.Outbound, debt.AccountId, debt.ExpenseId, debt.IncomeId,
	).Scan(&result.Id, &result.Description, &result.Amount, &result.DebtorId, &result.DebtorName,
		&result.Date, &result.CreatedAt, &result.OriginalAmount, &result.Currency, &result.Outbound)

	if err != nil {
		return types.Debt{}, fmt.Errorf("error inserting debt: %w", err)
	}

	return result, nil
}

// GetDebts retrieves debts with optional filters
func GetDebts(limit int, offset int, debtorId *int32) ([]types.Debt, int, error) {
	pool, err := GetPool()
	if err != nil {
		return nil, 0, err
	}

	ctx := context.Background()

	// Build query based on filters
	countQuery := `SELECT COUNT(*) FROM debts`
	selectQuery := `SELECT id, description, amount, debtor_id, debtor_name, date, created_at, 
		original_amount, currency, outbound, account_id, expense_id, income_id FROM debts`

	var args []interface{}
	argIndex := 1

	if debtorId != nil {
		countQuery += fmt.Sprintf(" WHERE debtor_id = $%d", argIndex)
		selectQuery += fmt.Sprintf(" WHERE debtor_id = $%d", argIndex)
		args = append(args, *debtorId)
		argIndex++
	}

	selectQuery += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)

	// Get total count
	var count int
	if len(args) > 0 {
		err = pool.QueryRow(ctx, countQuery, args...).Scan(&count)
	} else {
		err = pool.QueryRow(ctx, countQuery).Scan(&count)
	}
	if err != nil {
		return nil, 0, fmt.Errorf("error counting debts: %w", err)
	}

	// Add pagination args
	args = append(args, limit, offset)

	// Get paginated results
	rows, err := pool.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("error querying debts: %w", err)
	}
	defer rows.Close()

	var results []types.Debt
	for rows.Next() {
		var d types.Debt
		if err := rows.Scan(&d.Id, &d.Description, &d.Amount, &d.DebtorId, &d.DebtorName,
			&d.Date, &d.CreatedAt, &d.OriginalAmount, &d.Currency, &d.Outbound,
			&d.AccountId, &d.ExpenseId, &d.IncomeId); err != nil {
			return nil, 0, fmt.Errorf("error scanning row: %w", err)
		}
		results = append(results, d)
	}

	return results, count, nil
}

// GetRecentExpenses retrieves recent expenses for linking to debts
func GetRecentExpenses(limit int) ([]types.Expense, error) {
	pool, err := GetPool()
	if err != nil {
		return nil, err
	}

	rows, err := pool.Query(context.Background(),
		`SELECT id, date, category, category_id, expense, description, method, "originalAmount", account_id, account_type 
		 FROM expenses ORDER BY created_at DESC LIMIT $1`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("error querying recent expenses: %w", err)
	}
	defer rows.Close()

	var results []types.Expense
	for rows.Next() {
		var e types.Expense
		if err := rows.Scan(&e.Id, &e.Date, &e.Category, &e.CategoryId, &e.Expense,
			&e.Description, &e.Method, &e.OriginalAmount, &e.AccountId, &e.AccountType); err != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}
		results = append(results, e)
	}

	return results, nil
}

// RecordDebtRepayment creates an income record and a corresponding debt record in one transaction
func RecordDebtRepayment(income types.Income, debt types.Debt) (types.Income, types.Debt, error) {
	pool, err := GetPool()
	if err != nil {
		return types.Income{}, types.Debt{}, err
	}

	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		return types.Income{}, types.Debt{}, fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Insert income
	var incomeResult types.Income
	err = tx.QueryRow(ctx,
		`INSERT INTO incomes (date, amount, description, account_id, account_name)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, date, amount, description, account_id, account_name, created_at`,
		income.Date, income.Amount, income.Description, income.AccountId, income.AccountName,
	).Scan(&incomeResult.Id, &incomeResult.Date, &incomeResult.Amount, &incomeResult.Description,
		&incomeResult.AccountId, &incomeResult.AccountName, &incomeResult.CreatedAt)
	if err != nil {
		return types.Income{}, types.Debt{}, fmt.Errorf("error inserting income: %w", err)
	}

	// Insert debt with income_id reference
	debt.IncomeId = &incomeResult.Id
	var debtResult types.Debt
	err = tx.QueryRow(ctx,
		`INSERT INTO debts (description, amount, debtor_id, debtor_name, date, original_amount, currency, outbound, account_id, expense_id, income_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 RETURNING id, description, amount, debtor_id, debtor_name, date, created_at, original_amount, currency, outbound`,
		debt.Description, debt.Amount, debt.DebtorId, debt.DebtorName, debt.Date,
		debt.OriginalAmount, debt.Currency, debt.Outbound, debt.AccountId, debt.ExpenseId, debt.IncomeId,
	).Scan(&debtResult.Id, &debtResult.Description, &debtResult.Amount, &debtResult.DebtorId, &debtResult.DebtorName,
		&debtResult.Date, &debtResult.CreatedAt, &debtResult.OriginalAmount, &debtResult.Currency, &debtResult.Outbound)
	if err != nil {
		return types.Income{}, types.Debt{}, fmt.Errorf("error inserting debt: %w", err)
	}
	debtResult.IncomeId = debt.IncomeId

	if err := tx.Commit(ctx); err != nil {
		return types.Income{}, types.Debt{}, fmt.Errorf("error committing transaction: %w", err)
	}

	return incomeResult, debtResult, nil
}

// ========== EXPENSES (READ) ==========

// GetExpenses retrieves expenses with pagination
func GetExpenses(limit int, offset int) ([]types.Expense, int, error) {
	pool, err := GetPool()
	if err != nil {
		return nil, 0, err
	}

	// Get total count
	var count int
	err = pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM expenses`,
	).Scan(&count)
	if err != nil {
		return nil, 0, fmt.Errorf("error counting expenses: %w", err)
	}

	// Get paginated results
	rows, err := pool.Query(context.Background(),
		`SELECT id, date, category, category_id, expense, description, method, "originalAmount", account_id, account_type 
		 FROM expenses ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("error querying expenses: %w", err)
	}
	defer rows.Close()

	var results []types.Expense
	for rows.Next() {
		var e types.Expense
		if err := rows.Scan(&e.Id, &e.Date, &e.Category, &e.CategoryId, &e.Expense,
			&e.Description, &e.Method, &e.OriginalAmount, &e.AccountId, &e.AccountType); err != nil {
			return nil, 0, fmt.Errorf("error scanning row: %w", err)
		}
		results = append(results, e)
	}

	return results, count, nil
}

// ========== BUDGETS ==========

// GetBudgets retrieves budget by category from the view
func GetBudgets() ([]types.BudgetByCategory, error) {
	pool, err := GetPool()
	if err != nil {
		return nil, err
	}

	rows, err := pool.Query(context.Background(),
		`SELECT amount, spent, category_name, category_id FROM budget_by_category_current_month ORDER BY category_name`,
	)
	if err != nil {
		return nil, fmt.Errorf("error querying budgets: %w", err)
	}
	defer rows.Close()

	var results []types.BudgetByCategory
	for rows.Next() {
		var b types.BudgetByCategory
		if err := rows.Scan(&b.Amount, &b.Spent, &b.CategoryName, &b.CategoryId); err != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}
		results = append(results, b)
	}

	return results, nil
}

// ========== DEBTORS ==========

// GetDebtors retrieves all debtors
func GetDebtors() ([]types.Debtor, error) {
	pool, err := GetPool()
	if err != nil {
		return nil, err
	}

	rows, err := pool.Query(context.Background(),
		`SELECT id, name, COALESCE(first_name, ''), COALESCE(last_name, ''), COALESCE(description, '') 
		 FROM debtors ORDER BY name`,
	)
	if err != nil {
		return nil, fmt.Errorf("error querying debtors: %w", err)
	}
	defer rows.Close()

	var results []types.Debtor
	for rows.Next() {
		var d types.Debtor
		if err := rows.Scan(&d.Id, &d.Name, &d.FirstName, &d.LastName, &d.Description); err != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}
		results = append(results, d)
	}

	return results, nil
}

// GetDebtorsWithDebts retrieves debt summary by debtor
func GetDebtorsWithDebts() ([]types.DebtByDebtor, error) {
	pool, err := GetPool()
	if err != nil {
		return nil, err
	}

	rows, err := pool.Query(context.Background(),
		`SELECT debtor_id, debtor_name, total_lent, total_received, net_owed, transaction_count 
		 FROM debt_by_debtor`,
	)
	if err != nil {
		return nil, fmt.Errorf("error querying debt_by_debtor: %w", err)
	}
	defer rows.Close()

	var results []types.DebtByDebtor
	for rows.Next() {
		var d types.DebtByDebtor
		if err := rows.Scan(&d.DebtorId, &d.DebtorName, &d.TotalLent, &d.TotalReceived, &d.NetOwed, &d.TransactionCount); err != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}
		results = append(results, d)
	}

	return results, nil
}

// GetConfig retrieves all config entries
func GetConfig() ([]types.Config, error) {
	pool, err := GetPool()
	if err != nil {
		return nil, err
	}

	rows, err := pool.Query(context.Background(),
		`SELECT type, sheet, range FROM config`,
	)
	if err != nil {
		return nil, fmt.Errorf("error querying config: %w", err)
	}
	defer rows.Close()

	var results []types.Config
	for rows.Next() {
		var c types.Config
		if err := rows.Scan(&c.Type, &c.Sheet, &c.A1Range); err != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}
		results = append(results, c)
	}

	return results, nil
}

// ========== DASHBOARD HELPERS ==========

// GetMonthlyExpenseSum returns total expenses for a given year and month
func GetMonthlyExpenseSum(year int, month int) (float64, error) {
	pool, err := GetPool()
	if err != nil {
		return 0, err
	}

	var total float64
	err = pool.QueryRow(context.Background(),
		`SELECT COALESCE(SUM(expense), 0) FROM expenses 
		 WHERE EXTRACT(YEAR FROM created_at) = $1 AND EXTRACT(MONTH FROM created_at) = $2`,
		year, month,
	).Scan(&total)

	if err != nil {
		return 0, fmt.Errorf("error querying monthly expenses: %w", err)
	}

	return total, nil
}

// GetMonthlyInvestmentSum returns total investment deposits for a given year and month
func GetMonthlyInvestmentSum(year int, month int) (float64, error) {
	pool, err := GetPool()
	if err != nil {
		return 0, err
	}

	var total float64
	err = pool.QueryRow(context.Background(),
		`SELECT COALESCE(SUM(CASE WHEN type = 'deposit' THEN amount ELSE 0 END), 0) FROM investments 
		 WHERE EXTRACT(YEAR FROM created_at) = $1 AND EXTRACT(MONTH FROM created_at) = $2`,
		year, month,
	).Scan(&total)

	if err != nil {
		return 0, fmt.Errorf("error querying monthly investments: %w", err)
	}

	return total, nil
}

// GetYTDTotals returns year-to-date totals for income, expenses, and investments
func GetYTDTotals(year int) (income float64, expenses float64, investments float64) {
	pool, err := GetPool()
	if err != nil {
		return 0, 0, 0
	}

	ctx := context.Background()

	// Income YTD
	pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(amount), 0) FROM incomes WHERE EXTRACT(YEAR FROM created_at) = $1`,
		year,
	).Scan(&income)

	// Expenses YTD
	pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(expense), 0) FROM expenses WHERE EXTRACT(YEAR FROM created_at) = $1`,
		year,
	).Scan(&expenses)

	// Investments YTD (deposits only)
	pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(amount), 0) FROM investments WHERE type = 'deposit' AND EXTRACT(YEAR FROM created_at) = $1`,
		year,
	).Scan(&investments)

	return income, expenses, investments
}

// ========== TRANSFERS ==========

// InsertTransfer inserts a transfer record
func InsertTransfer(transfer types.Transfer) (types.Transfer, error) {
	pool, err := GetPool()
	if err != nil {
		return types.Transfer{}, err
	}

	// Calculate exchange rate if not provided
	if transfer.ExchangeRate == 0 && transfer.SourceAmount > 0 {
		transfer.ExchangeRate = transfer.DestAmount / transfer.SourceAmount
	}

	var result types.Transfer
	err = pool.QueryRow(context.Background(),
		`INSERT INTO transfers (date, description, source_account_id, source_amount, dest_account_id, dest_amount, exchange_rate)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, created_at, date, description, source_account_id, source_amount, dest_account_id, dest_amount, exchange_rate`,
		transfer.Date, transfer.Description, transfer.SourceAccountId, transfer.SourceAmount,
		transfer.DestAccountId, transfer.DestAmount, transfer.ExchangeRate,
	).Scan(&result.Id, &result.CreatedAt, &result.Date, &result.Description,
		&result.SourceAccountId, &result.SourceAmount, &result.DestAccountId, &result.DestAmount, &result.ExchangeRate)

	if err != nil {
		return types.Transfer{}, fmt.Errorf("error inserting transfer: %w", err)
	}

	return result, nil
}

// GetTransfers retrieves transfers with pagination
func GetTransfers(limit int, offset int) ([]types.Transfer, int, error) {
	pool, err := GetPool()
	if err != nil {
		return nil, 0, err
	}

	// Get total count
	var count int
	err = pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM transfers`,
	).Scan(&count)
	if err != nil {
		return nil, 0, fmt.Errorf("error counting transfers: %w", err)
	}

	// Get paginated results
	rows, err := pool.Query(context.Background(),
		`SELECT id, created_at, date, COALESCE(description, ''), source_account_id, source_amount, 
			dest_account_id, dest_amount, COALESCE(exchange_rate, 0)
		 FROM transfers ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("error querying transfers: %w", err)
	}
	defer rows.Close()

	var results []types.Transfer
	for rows.Next() {
		var t types.Transfer
		if err := rows.Scan(&t.Id, &t.CreatedAt, &t.Date, &t.Description,
			&t.SourceAccountId, &t.SourceAmount, &t.DestAccountId, &t.DestAmount, &t.ExchangeRate); err != nil {
			return nil, 0, fmt.Errorf("error scanning row: %w", err)
		}
		results = append(results, t)
	}

	return results, count, nil
}

// ========== EXPECTED BALANCE ==========

// GetAccountExpectedBalances retrieves expected balance view for all accounts
func GetAccountExpectedBalances() ([]types.AccountExpectedBalance, error) {
	pool, err := GetPool()
	if err != nil {
		return nil, err
	}

	rows, err := pool.Query(context.Background(),
		`SELECT id, name, currency, starting_balance, starting_date,
			total_income, total_expenses, total_investment_deposits, total_investment_withdrawals,
			total_transfers_out, total_transfers_in, expected_balance, real_balance, discrepancy
		 FROM account_expected_balance`,
	)
	if err != nil {
		return nil, fmt.Errorf("error querying expected balances: %w", err)
	}
	defer rows.Close()

	var results []types.AccountExpectedBalance
	for rows.Next() {
		var a types.AccountExpectedBalance
		if err := rows.Scan(&a.Id, &a.Name, &a.Currency, &a.StartingBalance, &a.StartingDate,
			&a.TotalIncome, &a.TotalExpenses, &a.TotalInvestmentDeposits, &a.TotalInvestmentWithdrawals,
			&a.TotalTransfersOut, &a.TotalTransfersIn, &a.ExpectedBalance, &a.RealBalance, &a.Discrepancy); err != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}
		results = append(results, a)
	}

	return results, nil
}

// ========== INSERT FUNCTIONS (migrated from supabase) ==========

// InsertBudgetsIntoDatabase upserts budget records
func InsertBudgetsIntoDatabase(budgets []types.Budget) ([]types.Budget, error) {
	pool, err := GetPool()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	var results []types.Budget

	for _, b := range budgets {
		var result types.Budget
		err := pool.QueryRow(ctx,
			`INSERT INTO budgets (category_id, budget)
			 VALUES ($1, $2)
			 ON CONFLICT (category_id) DO UPDATE SET budget = EXCLUDED.budget
			 RETURNING id, category_id, budget`,
			b.CategoryId, b.Amount,
		).Scan(&result.Id, &result.CategoryId, &result.Amount)

		if err != nil {
			return nil, fmt.Errorf("error upserting budget: %w", err)
		}
		results = append(results, result)
	}

	return results, nil
}

// InsertConfigIntoDatabase upserts config records
func InsertConfigIntoDatabase(configs []types.Config) ([]types.Config, error) {
	pool, err := GetPool()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	var results []types.Config

	for _, c := range configs {
		var result types.Config
		err := pool.QueryRow(ctx,
			`INSERT INTO config (type, sheet, range)
			 VALUES ($1, $2, $3)
			 ON CONFLICT (type) DO UPDATE SET sheet = EXCLUDED.sheet, range = EXCLUDED.range
			 RETURNING type, sheet, range`,
			c.Type, c.Sheet, c.A1Range,
		).Scan(&result.Type, &result.Sheet, &result.A1Range)

		if err != nil {
			return nil, fmt.Errorf("error upserting config: %w", err)
		}
		results = append(results, result)
	}

	return results, nil
}

// InsertAccountIntoDatabase inserts a new account
func InsertAccountIntoDatabase(account types.Account) (types.Account, error) {
	pool, err := GetPool()
	if err != nil {
		return types.Account{}, err
	}

	var result types.Account
	err = pool.QueryRow(context.Background(),
		`INSERT INTO accounts (name, description, type, currency, balance, starting_balance, starting_date)
		 VALUES ($1, $2, $3, $4, $5, $6, COALESCE($7, NOW()))
		 RETURNING id, name, COALESCE(description, ''), COALESCE(type, ''), COALESCE(currency, 'USD'), balance,
			COALESCE(starting_balance, 0), COALESCE(starting_date, NOW())`,
		account.Name, account.Description, account.Type, account.Currency, account.Balance,
		account.StartingBalance, account.StartingDate,
	).Scan(&result.Id, &result.Name, &result.Description, &result.Type, &result.Currency, &result.Balance,
		&result.StartingBalance, &result.StartingDate)

	if err != nil {
		return types.Account{}, fmt.Errorf("error inserting account: %w", err)
	}

	return result, nil
}

// InsertInvestmentAccountIntoDatabase inserts a new investment account
func InsertInvestmentAccountIntoDatabase(account types.InvestmentAccount) (types.InvestmentAccount, error) {
	pool, err := GetPool()
	if err != nil {
		return types.InvestmentAccount{}, err
	}

	var result types.InvestmentAccount
	err = pool.QueryRow(context.Background(),
		`INSERT INTO investment_accounts (name, description, type, currency, balance, capital, starting_capital, starting_date)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, COALESCE($8, NOW()))
		 RETURNING id, name, COALESCE(description, ''), COALESCE(type, ''), COALESCE(currency, 'USD'), balance,
			COALESCE(capital, 0), COALESCE(starting_capital, 0), COALESCE(starting_date, NOW())`,
		account.Name, account.Description, account.Type, account.Currency, account.Balance,
		account.Capital, account.StartingCapital, account.StartingDate,
	).Scan(&result.Id, &result.Name, &result.Description, &result.Type, &result.Currency, &result.Balance,
		&result.Capital, &result.StartingCapital, &result.StartingDate)

	if err != nil {
		return types.InvestmentAccount{}, fmt.Errorf("error inserting investment account: %w", err)
	}

	return result, nil
}

// InsertDebtorIntoDatabase inserts a new debtor
func InsertDebtorIntoDatabase(debtor types.Debtor) (types.Debtor, error) {
	pool, err := GetPool()
	if err != nil {
		return types.Debtor{}, err
	}

	var result types.Debtor
	err = pool.QueryRow(context.Background(),
		`INSERT INTO debtors (name, first_name, last_name, description)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, name, COALESCE(first_name, ''), COALESCE(last_name, ''), COALESCE(description, '')`,
		debtor.Name, debtor.FirstName, debtor.LastName, debtor.Description,
	).Scan(&result.Id, &result.Name, &result.FirstName, &result.LastName, &result.Description)

	if err != nil {
		return types.Debtor{}, fmt.Errorf("error inserting debtor: %w", err)
	}

	return result, nil
}
