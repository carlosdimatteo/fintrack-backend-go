package tests

import (
	"context"
	"testing"
)

// TestDatabaseConnection verifies the test database is accessible
func TestDatabaseConnection(t *testing.T) {
	pool := GetPool()
	if pool == nil {
		t.Fatal("Database pool is nil")
	}

	// Verify we can query
	var result int
	err := pool.QueryRow(context.Background(), "SELECT 1").Scan(&result)
	AssertNoError(t, err, "Simple query")
	AssertEqual(t, 1, result, "Query result")
}

// TestSeedData verifies seed data loads correctly
func TestSeedData(t *testing.T) {
	CleanupTables(t)
	SeedTestData(t)

	// Verify test accounts exist (count should match fixtures)
	var accountCount int
	accountIDs := make([]int32, len(TestAccounts))
	for i, acc := range TestAccounts {
		accountIDs[i] = acc.ID
	}

	err := testPool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM accounts WHERE id = ANY($1)`,
		accountIDs,
	).Scan(&accountCount)
	AssertNoError(t, err, "Count test accounts")
	AssertEqual(t, len(TestAccounts), accountCount, "Test accounts count")

	// Verify test investment accounts exist
	var invAccountCount int
	invAccountIDs := make([]int32, len(TestInvestmentAccounts))
	for i, acc := range TestInvestmentAccounts {
		invAccountIDs[i] = acc.ID
	}

	err = testPool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM investment_accounts WHERE id = ANY($1)`,
		invAccountIDs,
	).Scan(&invAccountCount)
	AssertNoError(t, err, "Count test investment accounts")
	AssertEqual(t, len(TestInvestmentAccounts), invAccountCount, "Test investment accounts count")

	// Verify test debtors exist
	var debtorCount int
	debtorIDs := make([]int32, len(TestDebtors))
	for i, d := range TestDebtors {
		debtorIDs[i] = d.ID
	}

	err = testPool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM debtors WHERE id = ANY($1)`,
		debtorIDs,
	).Scan(&debtorCount)
	AssertNoError(t, err, "Count test debtors")
	AssertEqual(t, len(TestDebtors), debtorCount, "Test debtors count")
}

// TestCleanup verifies cleanup works correctly
func TestCleanup(t *testing.T) {
	SeedTestData(t)

	// Use first test account from fixtures
	testAccount := TestAccounts[0]

	// Add some test data
	_, err := testPool.Exec(context.Background(),
		`INSERT INTO incomes (amount, description, account_id, account_name, date)
		 VALUES ($1, $2, $3, $4, '2026-01-15')`,
		100.0, "Test income", testAccount.ID, testAccount.Name)
	AssertNoError(t, err, "Insert test income")

	// Verify it exists
	incomeCount := CountTableRows(t, "incomes")
	if incomeCount == 0 {
		t.Fatal("Expected at least 1 income after insert")
	}

	// Cleanup
	CleanupTables(t)

	// Verify incomes are gone
	incomeCountAfter := CountTableRows(t, "incomes")
	AssertEqual(t, 0, incomeCountAfter, "Incomes after cleanup")

	// Verify account balance reset to starting
	balance := GetAccountBalance(t, testAccount.ID)
	AssertFloatEqual(t, testAccount.StartingBalance, balance, 0.01, "Account balance reset")
}

// TestExpectedBalanceView verifies the view exists and works
func TestExpectedBalanceView(t *testing.T) {
	CleanupTables(t)
	SeedTestData(t)

	// Get expected balance for first test account
	testAccount := TestAccounts[0]
	expected := GetAccountExpectedBalance(t, testAccount.ID)

	// With no transactions, expected should equal starting balance
	AssertFloatEqual(t, testAccount.StartingBalance, expected, 0.01, "Expected balance equals starting")
}

// TestMockSheet verifies the mock sheet tracking works
func TestMockSheet(t *testing.T) {
	ResetMockSheet()

	// Record some mock calls
	testValue := 100.50
	RecordSheetUpdate("Sheet1!A1", testValue)
	RecordSheetUpdate("Sheet1!B2", "test")

	calls := GetMockSheetCalls()
	AssertEqual(t, 2, len(calls), "Mock sheet call count")
	AssertEqual(t, "Sheet1!A1", calls[0].Range, "First call range")
	AssertFloatEqual(t, testValue, calls[0].Value.(float64), 0.01, "First call value")
}

// TestInvestmentAccountReset verifies investment account reset uses correct values
func TestInvestmentAccountReset(t *testing.T) {
	SeedTestData(t)
	CleanupTables(t)

	// After cleanup, investment accounts should have fixture values
	for _, acc := range TestInvestmentAccounts {
		var actualBalance, actualCapital float64
		err := testPool.QueryRow(context.Background(),
			`SELECT balance, capital FROM investment_accounts WHERE id = $1`, acc.ID,
		).Scan(&actualBalance, &actualCapital)
		AssertNoError(t, err, "Get investment account")

		AssertFloatEqual(t, acc.Balance, actualBalance, 0.01,
			"Investment account balance after reset")
		AssertFloatEqual(t, acc.StartingCapital, actualCapital, 0.01,
			"Investment account capital after reset")
	}
}
