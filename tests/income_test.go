package tests

import (
	"context"
	"testing"
	"time"

	"github.com/carlosdimatteo/fintrack-backend-go/adapters/postgres"
	"github.com/carlosdimatteo/fintrack-backend-go/types"
)

// ========== CORE FUNCTIONALITY ==========

// TestIncomeCreation verifies income is created with correct fields
func TestIncomeCreation(t *testing.T) {
	CleanupTables(t)
	SeedTestData(t)

	testAccount := GetTestAccount(TestAccountBankID)
	income := types.Income{
		Date:        time.Now().Format(time.DateTime),
		Amount:      1500.50,
		Description: "Salary payment",
		AccountId:   testAccount.ID,
		AccountName: testAccount.Name,
	}

	result, err := postgres.InsertIncome(income)
	AssertNoError(t, err, "Insert income")

	// Verify returned data matches input
	if result.Id == 0 {
		t.Error("Expected income to have an ID assigned")
	}
	AssertFloatEqual(t, income.Amount, result.Amount, 0.01, "Income amount")
	AssertEqual(t, income.Description, result.Description, "Income description")
	AssertEqual(t, income.AccountId, result.AccountId, "Income account ID")
	AssertEqual(t, income.AccountName, result.AccountName, "Income account name")

	// Verify it's actually in the database
	var dbAmount float64
	var dbDescription string
	err = testPool.QueryRow(context.Background(),
		`SELECT amount, description FROM incomes WHERE id = $1`, result.Id,
	).Scan(&dbAmount, &dbDescription)
	AssertNoError(t, err, "Query inserted income")
	AssertFloatEqual(t, income.Amount, dbAmount, 0.01, "DB income amount")
	AssertEqual(t, income.Description, dbDescription, "DB income description")
}

// ========== EXPECTED BALANCE INVARIANT ==========
// CRITICAL: Income MUST increase expected balance by exact amount

// TestIncomeIncreasesExpectedBalance verifies income adds to expected balance
func TestIncomeIncreasesExpectedBalance(t *testing.T) {
	CleanupTables(t)
	SeedTestData(t)

	testAccount := GetTestAccount(TestAccountBankID)

	// Get initial expected balance
	initialExpected := GetAccountExpectedBalance(t, testAccount.ID)
	AssertFloatEqual(t, testAccount.StartingBalance, initialExpected, 0.01,
		"Initial expected equals starting balance")

	// Add income
	incomeAmount := 500.00
	income := types.Income{
		Date:        time.Now().Format(time.DateTime),
		Amount:      incomeAmount,
		Description: "Test income",
		AccountId:   testAccount.ID,
		AccountName: testAccount.Name,
	}

	_, err := postgres.InsertIncome(income)
	AssertNoError(t, err, "Insert income")

	// Verify expected balance increased by EXACTLY the income amount
	newExpected := GetAccountExpectedBalance(t, testAccount.ID)
	expectedValue := initialExpected + incomeAmount
	AssertFloatEqual(t, expectedValue, newExpected, 0.01,
		"Expected balance increased by income amount")
}

// TestMultipleIncomesAccumulate verifies incomes stack correctly
func TestMultipleIncomesAccumulate(t *testing.T) {
	CleanupTables(t)
	SeedTestData(t)

	testAccount := GetTestAccount(TestAccountBankID)
	initialExpected := GetAccountExpectedBalance(t, testAccount.ID)

	// Add multiple incomes
	amounts := []float64{100.00, 250.50, 75.25, 500.00}
	totalIncome := 0.0

	for i, amount := range amounts {
		income := types.Income{
			Date:        time.Now().Format(time.DateTime),
			Amount:      amount,
			Description: "Income " + string(rune('A'+i)),
			AccountId:   testAccount.ID,
			AccountName: testAccount.Name,
		}
		_, err := postgres.InsertIncome(income)
		AssertNoError(t, err, "Insert income")
		totalIncome += amount
	}

	// Expected should be starting + sum of all incomes
	finalExpected := GetAccountExpectedBalance(t, testAccount.ID)
	AssertFloatEqual(t, initialExpected+totalIncome, finalExpected, 0.01,
		"Expected balance equals starting + total income")
}

// TestIncomeOnlyAffectsTargetAccount verifies isolation between accounts
func TestIncomeOnlyAffectsTargetAccount(t *testing.T) {
	CleanupTables(t)
	SeedTestData(t)

	accountA := GetTestAccount(TestAccountBankID)
	accountB := GetTestAccount(TestAccountSavingsID)

	// Get initial balances for both
	initialA := GetAccountExpectedBalance(t, accountA.ID)
	initialB := GetAccountExpectedBalance(t, accountB.ID)

	// Add income ONLY to account A
	income := types.Income{
		Date:        time.Now().Format(time.DateTime),
		Amount:      1000.00,
		Description: "Income for A only",
		AccountId:   accountA.ID,
		AccountName: accountA.Name,
	}
	_, err := postgres.InsertIncome(income)
	AssertNoError(t, err, "Insert income")

	// Account A should increase
	newA := GetAccountExpectedBalance(t, accountA.ID)
	AssertFloatEqual(t, initialA+1000.00, newA, 0.01,
		"Account A expected increased")

	// Account B should be UNCHANGED
	newB := GetAccountExpectedBalance(t, accountB.ID)
	AssertFloatEqual(t, initialB, newB, 0.01,
		"Account B expected unchanged")
}

// ========== MONTHLY AGGREGATION ==========

// TestMonthlyIncomeSum verifies sum calculation is correct
func TestMonthlyIncomeSum(t *testing.T) {
	CleanupTables(t)
	SeedTestData(t)

	testAccount := GetTestAccount(TestAccountBankID)
	now := time.Now()

	// Add incomes for current month
	amounts := []float64{1000.00, 500.00, 250.00}
	expectedSum := 1750.00

	for _, amount := range amounts {
		income := types.Income{
			Date:        now.Format(time.DateTime),
			Amount:      amount,
			Description: "Monthly income",
			AccountId:   testAccount.ID,
			AccountName: testAccount.Name,
		}
		_, err := postgres.InsertIncome(income)
		AssertNoError(t, err, "Insert income")

	}

	// Get monthly sum
	sum, err := postgres.GetMonthlyIncomeSum(now.Year(), int(now.Month()))
	AssertNoError(t, err, "Get monthly sum")
	AssertFloatEqual(t, expectedSum, sum, 0.01, "Monthly income sum")
}

// TestMonthlyIncomeSumEmptyMonth verifies empty month returns 0
func TestMonthlyIncomeSumEmptyMonth(t *testing.T) {
	CleanupTables(t)
	SeedTestData(t)

	// Query for a month with no data (use future date)
	sum, err := postgres.GetMonthlyIncomeSum(2099, 12)
	AssertNoError(t, err, "Get empty month sum")
	AssertFloatEqual(t, 0, sum, 0.01, "Empty month should return 0")
}

// TestYearlyIncomeSummary verifies monthly breakdown with multiple incomes
func TestYearlyIncomeSummary(t *testing.T) {
	CleanupTables(t)
	SeedTestData(t)

	testAccount := GetTestAccount(TestAccountBankID)
	now := time.Now()

	// Add multiple incomes for current month from different sources
	incomeAmounts := []float64{3500.00, 1200.00, 800.50, 250.00}
	expectedTotal := 0.0

	for i, amount := range incomeAmounts {
		income := types.Income{
			Date:        now.Format(time.DateTime),
			Amount:      amount,
			Description: []string{"Salary", "Freelance", "Dividends", "Refund"}[i],
			AccountId:   testAccount.ID,
			AccountName: testAccount.Name,
		}
		_, err := postgres.InsertIncome(income)
		AssertNoError(t, err, "Insert income")
		expectedTotal += amount
	}

	// Get yearly summary
	summary, err := postgres.GetYearlyIncomeSummary(now.Year())
	AssertNoError(t, err, "Get yearly summary")

	// Should have at least 1 month
	if len(summary) == 0 {
		t.Fatal("Expected at least 1 month in summary")
	}

	// Find current month and verify total
	found := false
	for _, s := range summary {
		if s.Month == int(now.Month()) {
			found = true
			AssertFloatEqual(t, expectedTotal, s.TotalIncome, 0.01,
				"Current month total should sum all incomes")
		}
	}
	if !found {
		t.Error("Current month not found in yearly summary")
	}
}

// ========== EDGE CASES ==========

// TestIncomeWithZeroAmount verifies zero amount is rejected
func TestIncomeWithZeroAmount(t *testing.T) {
	CleanupTables(t)
	SeedTestData(t)

	testAccount := GetTestAccount(TestAccountBankID)
	initialExpected := GetAccountExpectedBalance(t, testAccount.ID)

	income := types.Income{
		Date:        time.Now().Format(time.DateTime),
		Amount:      0,
		Description: "Zero amount income",
		AccountId:   testAccount.ID,
		AccountName: testAccount.Name,
	}

	_, err := postgres.InsertIncome(income)
	AssertError(t, err, "Zero income should be rejected")

	// Expected balance should be unchanged (no income was created)
	newExpected := GetAccountExpectedBalance(t, testAccount.ID)
	AssertFloatEqual(t, initialExpected, newExpected, 0.01,
		"Expected balance unchanged after rejected income")
}

// TestIncomeWithNegativeAmount verifies negative amounts are rejected
func TestIncomeWithNegativeAmount(t *testing.T) {
	CleanupTables(t)
	SeedTestData(t)

	testAccount := GetTestAccount(TestAccountBankID)
	initialExpected := GetAccountExpectedBalance(t, testAccount.ID)

	income := types.Income{
		Date:        time.Now().Format(time.DateTime),
		Amount:      -100.00,
		Description: "Negative income",
		AccountId:   testAccount.ID,
		AccountName: testAccount.Name,
	}

	_, err := postgres.InsertIncome(income)
	AssertError(t, err, "Negative income should be rejected")

	// Expected balance should be unchanged (no income was created)
	newExpected := GetAccountExpectedBalance(t, testAccount.ID)
	AssertFloatEqual(t, initialExpected, newExpected, 0.01,
		"Expected balance unchanged after rejected income")
}

// TestIncomeWithEmptyDescription verifies empty description is allowed
func TestIncomeWithEmptyDescription(t *testing.T) {
	CleanupTables(t)
	SeedTestData(t)

	testAccount := GetTestAccount(TestAccountBankID)

	income := types.Income{
		Date:        time.Now().Format(time.DateTime),
		Amount:      100.00,
		Description: "", // Empty description
		AccountId:   testAccount.ID,
		AccountName: testAccount.Name,
	}

	result, err := postgres.InsertIncome(income)
	AssertNoError(t, err, "Insert income with empty description")
	AssertEqual(t, "", result.Description, "Empty description preserved")
}

// TestIncomeWithLargeAmount verifies handling of large numbers
func TestIncomeWithLargeAmount(t *testing.T) {
	CleanupTables(t)
	SeedTestData(t)

	testAccount := GetTestAccount(TestAccountBankID)
	initialExpected := GetAccountExpectedBalance(t, testAccount.ID)

	// Large amount (1 million)
	largeAmount := 1000000.00
	income := types.Income{
		Date:        time.Now().Format(time.DateTime),
		Amount:      largeAmount,
		Description: "Large income",
		AccountId:   testAccount.ID,
		AccountName: testAccount.Name,
	}

	result, err := postgres.InsertIncome(income)
	AssertNoError(t, err, "Insert large income")
	AssertFloatEqual(t, largeAmount, result.Amount, 0.01, "Large amount preserved")

	// Verify expected balance
	newExpected := GetAccountExpectedBalance(t, testAccount.ID)
	AssertFloatEqual(t, initialExpected+largeAmount, newExpected, 0.01,
		"Large income added to expected balance")
}

// TestIncomeWithPreciseDecimal verifies decimal precision
func TestIncomeWithPreciseDecimal(t *testing.T) {
	CleanupTables(t)
	SeedTestData(t)

	testAccount := GetTestAccount(TestAccountBankID)

	// Precise decimal amount
	preciseAmount := 1234.56
	income := types.Income{
		Date:        time.Now().Format(time.DateTime),
		Amount:      preciseAmount,
		Description: "Precise decimal income",
		AccountId:   testAccount.ID,
		AccountName: testAccount.Name,
	}

	result, err := postgres.InsertIncome(income)
	AssertNoError(t, err, "Insert precise decimal income")
	AssertFloatEqual(t, preciseAmount, result.Amount, 0.001, "Precise amount preserved")
}

// TestIncomePagination verifies pagination works correctly
func TestIncomePagination(t *testing.T) {
	CleanupTables(t)
	SeedTestData(t)

	testAccount := GetTestAccount(TestAccountBankID)

	// Insert 15 incomes
	for i := 0; i < 15; i++ {
		income := types.Income{
			Date:        time.Now().Format(time.DateTime),
			Amount:      float64(i+1) * 100,
			Description: "Paginated income",
			AccountId:   testAccount.ID,
			AccountName: testAccount.Name,
		}
		_, err := postgres.InsertIncome(income)
		AssertNoError(t, err, "Insert income for pagination")
	}

	// Get first page
	page1, count, err := postgres.GetIncomes(10, 0)
	AssertNoError(t, err, "Get first page")
	AssertEqual(t, 15, count, "Total count")
	AssertEqual(t, 10, len(page1), "First page size")

	// Get second page
	page2, count2, err := postgres.GetIncomes(10, 10)
	AssertNoError(t, err, "Get second page")
	AssertEqual(t, 15, count2, "Total count unchanged")
	AssertEqual(t, 5, len(page2), "Second page size")

	// Ensure no overlap (incomes should be different)
	page1Ids := make(map[int32]bool)
	for _, inc := range page1 {
		page1Ids[inc.Id] = true
	}
	for _, inc := range page2 {
		if page1Ids[inc.Id] {
			t.Errorf("Income ID %d appears in both pages", inc.Id)
		}
	}
}

// ========== DISCREPANCY TRACKING ==========

// TestIncomeDoesNotAffectRealBalance verifies income only affects expected, not real
func TestIncomeDoesNotAffectRealBalance(t *testing.T) {
	CleanupTables(t)
	SeedTestData(t)

	testAccount := GetTestAccount(TestAccountBankID)

	// Get initial real balance
	initialReal := GetAccountBalance(t, testAccount.ID)

	// Add income
	income := types.Income{
		Date:        time.Now().Format(time.DateTime),
		Amount:      1000.00,
		Description: "Test income",
		AccountId:   testAccount.ID,
		AccountName: testAccount.Name,
	}
	_, err := postgres.InsertIncome(income)
	AssertNoError(t, err, "Insert income")

	// Real balance should be UNCHANGED (only expected changes)
	newReal := GetAccountBalance(t, testAccount.ID)
	AssertFloatEqual(t, initialReal, newReal, 0.01,
		"Real balance unchanged by income")

	// Expected should have increased
	expected := GetAccountExpectedBalance(t, testAccount.ID)
	if expected <= initialReal {
		t.Errorf("Expected (%f) should be greater than initial (%f) after income",
			expected, initialReal)
	}
}

// TestIncomeCreatesDiscrepancy verifies discrepancy is tracked
func TestIncomeCreatesDiscrepancy(t *testing.T) {
	CleanupTables(t)
	SeedTestData(t)

	testAccount := GetTestAccount(TestAccountBankID)

	// Initially: real = expected = starting balance, discrepancy = 0
	var initialDiscrepancy float64
	err := testPool.QueryRow(context.Background(),
		`SELECT discrepancy FROM account_expected_balance WHERE id = $1`,
		testAccount.ID,
	).Scan(&initialDiscrepancy)
	AssertNoError(t, err, "Get initial discrepancy")
	AssertFloatEqual(t, 0, initialDiscrepancy, 0.01, "Initial discrepancy is 0")

	// Add income - this increases expected but not real
	income := types.Income{
		Date:        time.Now().Format(time.DateTime),
		Amount:      500.00,
		Description: "Test income",
		AccountId:   testAccount.ID,
		AccountName: testAccount.Name,
	}
	_, err = postgres.InsertIncome(income)
	AssertNoError(t, err, "Insert income")

	// Now discrepancy should be negative (real < expected)
	// discrepancy = real - expected = 1000 - 1500 = -500
	var newDiscrepancy float64
	err = testPool.QueryRow(context.Background(),
		`SELECT discrepancy FROM account_expected_balance WHERE id = $1`,
		testAccount.ID,
	).Scan(&newDiscrepancy)
	AssertNoError(t, err, "Get new discrepancy")
	AssertFloatEqual(t, -500.00, newDiscrepancy, 0.01,
		"Discrepancy equals real - expected (negative when expected > real)")
}
