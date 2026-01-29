package tests

import (
	"context"
	"testing"
	"time"

	"github.com/carlosdimatteo/fintrack-backend-go/adapters/postgres"
	"github.com/carlosdimatteo/fintrack-backend-go/types"
)

// ========== CORE FUNCTIONALITY ==========

// TestExpenseCreation verifies expense is created with correct fields
func TestExpenseCreation(t *testing.T) {
	CleanupTables(t)
	SeedTestData(t)

	testAccount := GetTestAccount(TestAccountBankID)
	testCategory := GetTestCategory(TestCategoryFoodID)

	expense := types.Expense{
		Date:           time.Now().Format(time.DateTime),
		Category:       testCategory.Name,
		CategoryId:     testCategory.ID,
		Expense:        75.50,
		Description:    "Groceries",
		Method:         "Debit",
		OriginalAmount: 75.50,
		AccountId:      testAccount.ID,
		AccountType:    testAccount.Type,
	}

	result, err := postgres.InsertExpense(expense)
	AssertNoError(t, err, "Insert expense")

	// Verify returned data matches input
	if result.Id == 0 {
		t.Error("Expected expense to have an ID assigned")
	}
	AssertFloatEqual(t, expense.Expense, result.Expense, 0.01, "Expense amount")
	AssertEqual(t, expense.Description, result.Description, "Expense description")
	AssertEqual(t, expense.Category, result.Category, "Expense category")
	AssertEqual(t, expense.AccountId, result.AccountId, "Expense account ID")

	// Verify it's actually in the database
	var dbAmount float64
	var dbDescription string
	err = testPool.QueryRow(context.Background(),
		`SELECT expense, description FROM expenses WHERE id = $1`, result.Id,
	).Scan(&dbAmount, &dbDescription)
	AssertNoError(t, err, "Query inserted expense")
	AssertFloatEqual(t, expense.Expense, dbAmount, 0.01, "DB expense amount")
	AssertEqual(t, expense.Description, dbDescription, "DB expense description")
}

// ========== EXPECTED BALANCE INVARIANT ==========
// CRITICAL: Expense MUST decrease expected balance by exact amount

// TestExpenseDecreasesExpectedBalance verifies expense subtracts from expected balance
func TestExpenseDecreasesExpectedBalance(t *testing.T) {
	CleanupTables(t)
	SeedTestData(t)

	testAccount := GetTestAccount(TestAccountBankID)
	testCategory := GetTestCategory(TestCategoryFoodID)

	// Get initial expected balance
	initialExpected := GetAccountExpectedBalance(t, testAccount.ID)
	AssertFloatEqual(t, testAccount.StartingBalance, initialExpected, 0.01,
		"Initial expected equals starting balance")

	// Add expense
	expenseAmount := 150.00
	expense := types.Expense{
		Date:           time.Now().Format(time.DateTime),
		Category:       testCategory.Name,
		CategoryId:     testCategory.ID,
		Expense:        expenseAmount,
		Description:    "Test expense",
		Method:         "Credit",
		OriginalAmount: expenseAmount,
		AccountId:      testAccount.ID,
		AccountType:    testAccount.Type,
	}

	_, err := postgres.InsertExpense(expense)
	AssertNoError(t, err, "Insert expense")

	// Verify expected balance decreased by EXACTLY the expense amount
	newExpected := GetAccountExpectedBalance(t, testAccount.ID)
	expectedValue := initialExpected - expenseAmount
	AssertFloatEqual(t, expectedValue, newExpected, 0.01,
		"Expected balance decreased by expense amount")
}

// TestMultipleExpensesAccumulate verifies expenses stack correctly
func TestMultipleExpensesAccumulate(t *testing.T) {
	CleanupTables(t)
	SeedTestData(t)

	testAccount := GetTestAccount(TestAccountBankID)
	testCategory := GetTestCategory(TestCategoryFoodID)
	initialExpected := GetAccountExpectedBalance(t, testAccount.ID)

	// Add multiple expenses
	amounts := []float64{50.00, 125.75, 30.25, 200.00}
	totalExpenses := 0.0

	for i, amount := range amounts {
		expense := types.Expense{
			Date:           time.Now().Format(time.DateTime),
			Category:       testCategory.Name,
			CategoryId:     testCategory.ID,
			Expense:        amount,
			Description:    "Expense " + string(rune('A'+i)),
			Method:         "Debit",
			OriginalAmount: amount,
			AccountId:      testAccount.ID,
			AccountType:    testAccount.Type,
		}
		_, err := postgres.InsertExpense(expense)
		AssertNoError(t, err, "Insert expense")
		totalExpenses += amount
	}

	// Expected should be starting - sum of all expenses
	finalExpected := GetAccountExpectedBalance(t, testAccount.ID)
	AssertFloatEqual(t, initialExpected-totalExpenses, finalExpected, 0.01,
		"Expected balance equals starting - total expenses")
}

// TestExpenseOnlyAffectsTargetAccount verifies isolation between accounts
func TestExpenseOnlyAffectsTargetAccount(t *testing.T) {
	CleanupTables(t)
	SeedTestData(t)

	accountA := GetTestAccount(TestAccountBankID)
	accountB := GetTestAccount(TestAccountSavingsID)
	testCategory := GetTestCategory(TestCategoryFoodID)

	// Get initial balances for both
	initialA := GetAccountExpectedBalance(t, accountA.ID)
	initialB := GetAccountExpectedBalance(t, accountB.ID)

	// Add expense ONLY to account A
	expense := types.Expense{
		Date:           time.Now().Format(time.DateTime),
		Category:       testCategory.Name,
		CategoryId:     testCategory.ID,
		Expense:        200.00,
		Description:    "Expense for A only",
		Method:         "Debit",
		OriginalAmount: 200.00,
		AccountId:      accountA.ID,
		AccountType:    accountA.Type,
	}
	_, err := postgres.InsertExpense(expense)
	AssertNoError(t, err, "Insert expense")

	// Account A should decrease
	newA := GetAccountExpectedBalance(t, accountA.ID)
	AssertFloatEqual(t, initialA-200.00, newA, 0.01,
		"Account A expected decreased")

	// Account B should be UNCHANGED
	newB := GetAccountExpectedBalance(t, accountB.ID)
	AssertFloatEqual(t, initialB, newB, 0.01,
		"Account B expected unchanged")
}

// ========== INCOME + EXPENSE INTERACTION ==========

// TestIncomeAndExpenseNetEffect verifies combined effect on expected balance
func TestIncomeAndExpenseNetEffect(t *testing.T) {
	CleanupTables(t)
	SeedTestData(t)

	testAccount := GetTestAccount(TestAccountBankID)
	testCategory := GetTestCategory(TestCategoryFoodID)
	initialExpected := GetAccountExpectedBalance(t, testAccount.ID)

	// Add income
	income := types.Income{
		Date:        time.Now().Format(time.DateTime),
		Amount:      1000.00,
		Description: "Paycheck",
		AccountId:   testAccount.ID,
		AccountName: testAccount.Name,
	}
	_, err := postgres.InsertIncome(income)
	AssertNoError(t, err, "Insert income")

	// Add expenses
	expenses := []float64{200.00, 150.00, 100.00}
	totalExpenses := 0.0
	for _, amount := range expenses {
		expense := types.Expense{
			Date:           time.Now().Format(time.DateTime),
			Category:       testCategory.Name,
			CategoryId:     testCategory.ID,
			Expense:        amount,
			Description:    "Spending",
			Method:         "Debit",
			OriginalAmount: amount,
			AccountId:      testAccount.ID,
			AccountType:    testAccount.Type,
		}
		_, err := postgres.InsertExpense(expense)
		AssertNoError(t, err, "Insert expense")
		totalExpenses += amount
	}

	// Net effect: +1000 - 450 = +550
	finalExpected := GetAccountExpectedBalance(t, testAccount.ID)
	expectedNet := initialExpected + 1000.00 - totalExpenses
	AssertFloatEqual(t, expectedNet, finalExpected, 0.01,
		"Net effect of income and expenses correct")
}

// ========== EDGE CASES ==========

// TestExpenseWithZeroAmount verifies zero amount is rejected
func TestExpenseWithZeroAmount(t *testing.T) {
	CleanupTables(t)
	SeedTestData(t)

	testAccount := GetTestAccount(TestAccountBankID)
	testCategory := GetTestCategory(TestCategoryFoodID)
	initialExpected := GetAccountExpectedBalance(t, testAccount.ID)

	expense := types.Expense{
		Date:           time.Now().Format(time.DateTime),
		Category:       testCategory.Name,
		CategoryId:     testCategory.ID,
		Expense:        0,
		Description:    "Zero expense",
		Method:         "Debit",
		OriginalAmount: 0,
		AccountId:      testAccount.ID,
		AccountType:    testAccount.Type,
	}

	_, err := postgres.InsertExpense(expense)
	AssertError(t, err, "Zero expense should be rejected")

	// Expected balance unchanged
	newExpected := GetAccountExpectedBalance(t, testAccount.ID)
	AssertFloatEqual(t, initialExpected, newExpected, 0.01,
		"Expected balance unchanged after rejected expense")
}

// TestExpenseWithNegativeAmount verifies negative amounts are rejected
func TestExpenseWithNegativeAmount(t *testing.T) {
	CleanupTables(t)
	SeedTestData(t)

	testAccount := GetTestAccount(TestAccountBankID)
	testCategory := GetTestCategory(TestCategoryFoodID)
	initialExpected := GetAccountExpectedBalance(t, testAccount.ID)

	expense := types.Expense{
		Date:           time.Now().Format(time.DateTime),
		Category:       testCategory.Name,
		CategoryId:     testCategory.ID,
		Expense:        -50.00,
		Description:    "Negative expense",
		Method:         "Debit",
		OriginalAmount: -50.00,
		AccountId:      testAccount.ID,
		AccountType:    testAccount.Type,
	}

	_, err := postgres.InsertExpense(expense)
	AssertError(t, err, "Negative expense should be rejected")

	// Expected balance unchanged
	newExpected := GetAccountExpectedBalance(t, testAccount.ID)
	AssertFloatEqual(t, initialExpected, newExpected, 0.01,
		"Expected balance unchanged after rejected expense")
}

// TestExpenseWithDifferentCategories verifies category is tracked correctly
func TestExpenseWithDifferentCategories(t *testing.T) {
	CleanupTables(t)
	SeedTestData(t)

	testAccount := GetTestAccount(TestAccountBankID)

	// Create expenses in different categories
	categories := []int32{TestCategoryFoodID, TestCategoryTransportID, TestCategoryUtilitiesID}

	for _, catId := range categories {
		cat := GetTestCategory(catId)
		expense := types.Expense{
			Date:           time.Now().Format(time.DateTime),
			Category:       cat.Name,
			CategoryId:     cat.ID,
			Expense:        100.00,
			Description:    "Expense in " + cat.Name,
			Method:         "Debit",
			OriginalAmount: 100.00,
			AccountId:      testAccount.ID,
			AccountType:    testAccount.Type,
		}
		result, err := postgres.InsertExpense(expense)
		AssertNoError(t, err, "Insert expense in "+cat.Name)
		AssertEqual(t, cat.ID, result.CategoryId, "Category ID preserved")
		AssertEqual(t, cat.Name, result.Category, "Category name preserved")
	}

	// Verify all expenses created
	count := CountTableRows(t, "expenses")
	AssertEqual(t, len(categories), count, "All expenses created")
}

// TestExpenseWithLargeAmount verifies handling of large numbers
func TestExpenseWithLargeAmount(t *testing.T) {
	CleanupTables(t)
	SeedTestData(t)

	testAccount := GetTestAccount(TestAccountBankID)
	testCategory := GetTestCategory(TestCategoryFoodID)
	initialExpected := GetAccountExpectedBalance(t, testAccount.ID)

	// Large expense
	largeAmount := 50000.00
	expense := types.Expense{
		Date:           time.Now().Format(time.DateTime),
		Category:       testCategory.Name,
		CategoryId:     testCategory.ID,
		Expense:        largeAmount,
		Description:    "Large purchase",
		Method:         "Credit",
		OriginalAmount: largeAmount,
		AccountId:      testAccount.ID,
		AccountType:    testAccount.Type,
	}

	result, err := postgres.InsertExpense(expense)
	AssertNoError(t, err, "Insert large expense")
	AssertFloatEqual(t, largeAmount, result.Expense, 0.01, "Large amount preserved")

	// Verify expected balance (will go negative, which is valid)
	newExpected := GetAccountExpectedBalance(t, testAccount.ID)
	AssertFloatEqual(t, initialExpected-largeAmount, newExpected, 0.01,
		"Large expense subtracted from expected balance")
}

// TestExpensePagination verifies pagination works correctly
func TestExpensePagination(t *testing.T) {
	CleanupTables(t)
	SeedTestData(t)

	testAccount := GetTestAccount(TestAccountBankID)
	testCategory := GetTestCategory(TestCategoryFoodID)

	// Insert 20 expenses
	for i := 0; i < 20; i++ {
		expense := types.Expense{
			Date:           time.Now().Format(time.DateTime),
			Category:       testCategory.Name,
			CategoryId:     testCategory.ID,
			Expense:        float64(i+1) * 10,
			Description:    "Paginated expense",
			Method:         "Debit",
			OriginalAmount: float64(i+1) * 10,
			AccountId:      testAccount.ID,
			AccountType:    testAccount.Type,
		}
		_, err := postgres.InsertExpense(expense)
		AssertNoError(t, err, "Insert expense for pagination")
	}

	// Get first page
	page1, count, err := postgres.GetExpenses(10, 0)
	AssertNoError(t, err, "Get first page")
	AssertEqual(t, 20, count, "Total count")
	AssertEqual(t, 10, len(page1), "First page size")

	// Get second page
	page2, count2, err := postgres.GetExpenses(10, 10)
	AssertNoError(t, err, "Get second page")
	AssertEqual(t, 20, count2, "Total count unchanged")
	AssertEqual(t, 10, len(page2), "Second page size")

	// Ensure no overlap
	page1Ids := make(map[int32]bool)
	for _, exp := range page1 {
		page1Ids[exp.Id] = true
	}
	for _, exp := range page2 {
		if page1Ids[exp.Id] {
			t.Errorf("Expense ID %d appears in both pages", exp.Id)
		}
	}
}

// TestRecentExpenses verifies recent expenses endpoint
func TestRecentExpenses(t *testing.T) {
	CleanupTables(t)
	SeedTestData(t)

	testAccount := GetTestAccount(TestAccountBankID)
	testCategory := GetTestCategory(TestCategoryFoodID)

	// Insert 15 expenses
	for i := 0; i < 15; i++ {
		expense := types.Expense{
			Date:           time.Now().Format(time.DateTime),
			Category:       testCategory.Name,
			CategoryId:     testCategory.ID,
			Expense:        float64(i+1) * 5,
			Description:    "Recent expense test",
			Method:         "Debit",
			OriginalAmount: float64(i+1) * 5,
			AccountId:      testAccount.ID,
			AccountType:    testAccount.Type,
		}
		_, err := postgres.InsertExpense(expense)
		AssertNoError(t, err, "Insert expense")
	}

	// Get recent 5
	recent, err := postgres.GetRecentExpenses(5)
	AssertNoError(t, err, "Get recent expenses")
	AssertEqual(t, 5, len(recent), "Recent expenses count")
}

// ========== DISCREPANCY TRACKING ==========

// TestExpenseCreatesDiscrepancy verifies discrepancy tracking
func TestExpenseCreatesDiscrepancy(t *testing.T) {
	CleanupTables(t)
	SeedTestData(t)

	testAccount := GetTestAccount(TestAccountBankID)
	testCategory := GetTestCategory(TestCategoryFoodID)

	// Initially: real = expected = starting balance, discrepancy = 0
	var initialDiscrepancy float64
	err := testPool.QueryRow(context.Background(),
		`SELECT discrepancy FROM account_expected_balance WHERE id = $1`,
		testAccount.ID,
	).Scan(&initialDiscrepancy)
	AssertNoError(t, err, "Get initial discrepancy")
	AssertFloatEqual(t, 0, initialDiscrepancy, 0.01, "Initial discrepancy is 0")

	// Add expense - this decreases expected but not real
	expense := types.Expense{
		Date:           time.Now().Format(time.DateTime),
		Category:       testCategory.Name,
		CategoryId:     testCategory.ID,
		Expense:        300.00,
		Description:    "Test expense",
		Method:         "Debit",
		OriginalAmount: 300.00,
		AccountId:      testAccount.ID,
		AccountType:    testAccount.Type,
	}
	_, err = postgres.InsertExpense(expense)
	AssertNoError(t, err, "Insert expense")

	// Now discrepancy should be positive (real > expected)
	// discrepancy = real - expected = 1000 - 700 = 300
	var newDiscrepancy float64
	err = testPool.QueryRow(context.Background(),
		`SELECT discrepancy FROM account_expected_balance WHERE id = $1`,
		testAccount.ID,
	).Scan(&newDiscrepancy)
	AssertNoError(t, err, "Get new discrepancy")
	AssertFloatEqual(t, 300.00, newDiscrepancy, 0.01,
		"Discrepancy equals real - expected (positive when real > expected)")
}
