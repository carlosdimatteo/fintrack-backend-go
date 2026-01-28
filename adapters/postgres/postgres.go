package postgres

import (
	"context"
	"fmt"
	"os"
	"sync"

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
