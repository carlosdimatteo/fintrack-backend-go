package tests

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

var testPool *pgxpool.Pool

// TestMain runs before all tests - sets up DB connection
func TestMain(m *testing.M) {
	// Get database URL - fallback to local postgres
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		// Default local postgres
		databaseURL = "postgres://postgres:postgres@localhost:5432/fintrack_test?sslmode=disable"
	}

	// Set DATABASE_URL so the postgres adapter uses the test database
	os.Setenv("DATABASE_URL", databaseURL)

	// Connect to database
	var err error
	testPool, err = pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}

	// Verify connection
	if err := testPool.Ping(context.Background()); err != nil {
		log.Fatalf("Unable to ping database: %v", err)
	}

	fmt.Println("[test] Connected to test database")

	// Run tests
	code := m.Run()

	// Cleanup
	testPool.Close()
	fmt.Println("[test] Database connection closed")

	os.Exit(code)
}

// ========== HELPER FUNCTIONS ==========

// CleanupTables truncates all tables for a fresh test state
func CleanupTables(t *testing.T) {
	t.Helper()

	tables := []string{
		"debts",
		"transfers",
		"investments",
		"incomes",
		"expenses",
		"net_worth_snapshots",
		"yearly_goals",
	}

	ctx := context.Background()
	for _, table := range tables {
		_, err := testPool.Exec(ctx, fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		if err != nil {
			t.Fatalf("Failed to truncate %s: %v", table, err)
		}
	}

	// Reset account balances to starting_balance
	_, err := testPool.Exec(ctx, `UPDATE accounts SET balance = starting_balance`)
	if err != nil {
		t.Fatalf("Failed to reset account balances: %v", err)
	}

	// Reset investment account balances and capital from fixtures
	for _, acc := range TestInvestmentAccounts {
		_, err = testPool.Exec(ctx,
			`UPDATE investment_accounts SET balance = $1, capital = $2 WHERE id = $3`,
			acc.Balance, acc.StartingCapital, acc.ID)
		if err != nil {
			t.Fatalf("Failed to reset investment account %d: %v", acc.ID, err)
		}
	}
}

// SeedTestData loads the test fixtures from fixtures.go
func SeedTestData(t *testing.T) {
	t.Helper()
	ctx := context.Background()

	// Seed accounts from fixtures
	for _, acc := range TestAccounts {
		_, err := testPool.Exec(ctx, `
			INSERT INTO accounts (id, name, description, type, currency, balance, starting_balance, starting_date)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (id) DO UPDATE SET 
				name = EXCLUDED.name,
				balance = EXCLUDED.balance,
				starting_balance = EXCLUDED.starting_balance`,
			acc.ID, acc.Name, acc.Description, acc.Type, acc.Currency,
			acc.Balance, acc.StartingBalance, acc.StartingDate)
		if err != nil {
			t.Fatalf("Failed to seed account %s: %v", acc.Name, err)
		}
	}

	// Seed investment accounts from fixtures
	for _, acc := range TestInvestmentAccounts {
		_, err := testPool.Exec(ctx, `
			INSERT INTO investment_accounts (id, name, description, type, currency, balance, capital, starting_capital, starting_date)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (id) DO UPDATE SET 
				name = EXCLUDED.name,
				balance = EXCLUDED.balance,
				capital = EXCLUDED.capital,
				starting_capital = EXCLUDED.starting_capital`,
			acc.ID, acc.Name, acc.Description, acc.Type, acc.Currency,
			acc.Balance, acc.Capital, acc.StartingCapital, acc.StartingDate)
		if err != nil {
			t.Fatalf("Failed to seed investment account %s: %v", acc.Name, err)
		}
	}

	// Seed categories from fixtures
	for _, cat := range TestCategories {
		_, err := testPool.Exec(ctx, `
			INSERT INTO categories (id, name, description, is_essential)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (id) DO NOTHING`,
			cat.ID, cat.Name, cat.Description, cat.IsEssential)
		if err != nil {
			t.Fatalf("Failed to seed category %s: %v", cat.Name, err)
		}
	}

	// Seed debtors from fixtures
	for _, d := range TestDebtors {
		_, err := testPool.Exec(ctx, `
			INSERT INTO debtors (id, name, first_name, last_name)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (id) DO NOTHING`,
			d.ID, d.Name, d.FirstName, d.LastName)
		if err != nil {
			t.Fatalf("Failed to seed debtor %s: %v", d.Name, err)
		}
	}

	// Ensure config entries exist for tests
	_, err := testPool.Exec(ctx, `
		INSERT INTO config (type, sheet, range)
		VALUES 
			('test_expenses', 'TestSheet', 'A1'),
			('test_income', 'TestSheet', 'B1'),
			('test_investments', 'TestSheet', 'C1')
		ON CONFLICT (type) DO NOTHING
	`)
	if err != nil {
		t.Fatalf("Failed to seed config: %v", err)
	}
}

// GetPool returns the test database pool
func GetPool() *pgxpool.Pool {
	return testPool
}

// ========== ASSERTION HELPERS ==========

// AssertEqual fails the test if expected != actual
func AssertEqual[T comparable](t *testing.T, expected, actual T, msg string) {
	t.Helper()
	if expected != actual {
		t.Errorf("%s: expected %v, got %v", msg, expected, actual)
	}
}

// AssertFloatEqual fails the test if floats differ by more than tolerance
func AssertFloatEqual(t *testing.T, expected, actual, tolerance float64, msg string) {
	t.Helper()
	diff := expected - actual
	if diff < 0 {
		diff = -diff
	}
	if diff > tolerance {
		t.Errorf("%s: expected %.2f, got %.2f (diff: %.2f)", msg, expected, actual, diff)
	}
}

// AssertNoError fails the test if err is not nil
func AssertNoError(t *testing.T, err error, msg string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: unexpected error: %v", msg, err)
	}
}

// AssertError fails the test if err is nil
func AssertError(t *testing.T, err error, msg string) {
	t.Helper()
	if err == nil {
		t.Errorf("%s: expected error but got nil", msg)
	}
}

// ========== DATABASE QUERY HELPERS ==========

// GetAccountBalance returns the current balance for an account
func GetAccountBalance(t *testing.T, accountId int32) float64 {
	t.Helper()
	var balance float64
	err := testPool.QueryRow(context.Background(),
		`SELECT balance FROM accounts WHERE id = $1`, accountId,
	).Scan(&balance)
	if err != nil {
		t.Fatalf("Failed to get account balance: %v", err)
	}
	return balance
}

// GetAccountExpectedBalance returns the expected balance from the view
func GetAccountExpectedBalance(t *testing.T, accountId int32) float64 {
	t.Helper()
	var expected float64
	err := testPool.QueryRow(context.Background(),
		`SELECT expected_balance FROM account_expected_balance WHERE id = $1`, accountId,
	).Scan(&expected)
	if err != nil {
		t.Fatalf("Failed to get expected balance: %v", err)
	}
	return expected
}

// GetInvestmentAccountCapital returns the current capital for an investment account
func GetInvestmentAccountCapital(t *testing.T, accountId int32) float64 {
	t.Helper()
	var capital float64
	err := testPool.QueryRow(context.Background(),
		`SELECT capital FROM investment_accounts WHERE id = $1`, accountId,
	).Scan(&capital)
	if err != nil {
		t.Fatalf("Failed to get investment capital: %v", err)
	}
	return capital
}

// CountTableRows returns the number of rows in a table
func CountTableRows(t *testing.T, table string) int {
	t.Helper()
	var count int
	err := testPool.QueryRow(context.Background(),
		fmt.Sprintf(`SELECT COUNT(*) FROM %s`, table),
	).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count rows in %s: %v", table, err)
	}
	return count
}

// ========== MOCK GOOGLE SHEETS ==========

// MockSheetUpdates tracks sheet update calls for verification
type MockSheetUpdates struct {
	Calls []SheetUpdateCall
}

type SheetUpdateCall struct {
	Range string
	Value interface{}
}

var mockSheet = &MockSheetUpdates{}

// ResetMockSheet clears all recorded calls
func ResetMockSheet() {
	mockSheet.Calls = nil
}

// GetMockSheetCalls returns all recorded sheet update calls
func GetMockSheetCalls() []SheetUpdateCall {
	return mockSheet.Calls
}

// RecordSheetUpdate records a sheet update (call this from mocked sheet functions)
func RecordSheetUpdate(sheetRange string, value interface{}) {
	mockSheet.Calls = append(mockSheet.Calls, SheetUpdateCall{
		Range: sheetRange,
		Value: value,
	})
}
