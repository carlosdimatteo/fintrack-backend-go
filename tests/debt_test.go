package tests

import (
	"context"
	"testing"
	"time"

	"github.com/carlosdimatteo/fintrack-backend-go/adapters/postgres"
	"github.com/carlosdimatteo/fintrack-backend-go/types"
)

// ========== EXPENSE WITH DEBT (LENDING) ==========

// TestExpenseWithDebtCreation verifies the combined action creates both records
func TestExpenseWithDebtCreation(t *testing.T) {
	CleanupTables(t)
	SeedTestData(t)

	testAccount := GetTestAccount(TestAccountBankID)
	testCategory := GetTestCategory(TestCategoryFoodID)
	testDebtor := GetTestDebtor(TestDebtorJohnID)
	now := time.Now()

	expense := types.Expense{
		Date:           now.Format(time.DateTime),
		Category:       testCategory.Name,
		CategoryId:     testCategory.ID,
		Expense:        100.00,
		Description:    "Lent money to John",
		Method:         "Debit",
		OriginalAmount: 100.00,
		AccountId:      testAccount.ID,
		AccountType:    testAccount.Type,
	}

	debt := types.Debt{
		Description:    "Lent money to John",
		Amount:         100.00,
		DebtorId:       testDebtor.ID,
		DebtorName:     testDebtor.Name,
		Date:           now.Format(time.DateTime),
		OriginalAmount: 100.00,
		Currency:       "USD",
		Outbound:       true,
	}

	expenseResult, debtResult, err := postgres.InsertExpenseWithDebt(expense, debt)
	AssertNoError(t, err, "Insert expense with debt")

	// Verify expense created
	if expenseResult.Id == 0 {
		t.Error("Expense should have an ID")
	}
	AssertFloatEqual(t, 100.00, expenseResult.Expense, 0.01, "Expense amount")

	// Verify debt created
	if debtResult.Id == 0 {
		t.Error("Debt should have an ID")
	}
	AssertFloatEqual(t, 100.00, debtResult.Amount, 0.01, "Debt amount")
	AssertEqual(t, true, debtResult.Outbound, "Debt should be outbound")

	// Verify debt is linked to expense
	if debtResult.ExpenseId == nil || *debtResult.ExpenseId != expenseResult.Id {
		t.Errorf("Debt should be linked to expense ID %d, got %v", expenseResult.Id, debtResult.ExpenseId)
	}
}

// TestExpenseWithDebtAffectsExpectedBalance verifies expected balance decreases
func TestExpenseWithDebtAffectsExpectedBalance(t *testing.T) {
	CleanupTables(t)
	SeedTestData(t)

	testAccount := GetTestAccount(TestAccountBankID)
	testCategory := GetTestCategory(TestCategoryFoodID)
	testDebtor := GetTestDebtor(TestDebtorJohnID)
	now := time.Now()

	// Get initial expected balance
	initialExpected := GetAccountExpectedBalance(t, testAccount.ID)

	expense := types.Expense{
		Date:           now.Format(time.DateTime),
		Category:       testCategory.Name,
		CategoryId:     testCategory.ID,
		Expense:        150.00,
		Description:    "Lent money",
		Method:         "Debit",
		OriginalAmount: 150.00,
		AccountId:      testAccount.ID,
		AccountType:    testAccount.Type,
	}

	debt := types.Debt{
		Description:    "Lent money",
		Amount:         150.00,
		DebtorId:       testDebtor.ID,
		DebtorName:     testDebtor.Name,
		Date:           now.Format(time.DateTime),
		OriginalAmount: 150.00,
		Currency:       "USD",
		Outbound:       true,
	}

	_, _, err := postgres.InsertExpenseWithDebt(expense, debt)
	AssertNoError(t, err, "Insert expense with debt")

	// Check expected balance decreased by expense amount
	newExpected := GetAccountExpectedBalance(t, testAccount.ID)
	expectedChange := initialExpected - 150.00
	AssertFloatEqual(t, expectedChange, newExpected, 0.01, "Expected balance should decrease")
}

// TestExpenseWithDebtPartialAmount verifies debt amount can differ from expense
func TestExpenseWithDebtPartialAmount(t *testing.T) {
	CleanupTables(t)
	SeedTestData(t)

	testAccount := GetTestAccount(TestAccountBankID)
	testCategory := GetTestCategory(TestCategoryFoodID)
	testDebtor := GetTestDebtor(TestDebtorJohnID)
	now := time.Now()

	// I paid $100 but they only owe me $60 (maybe I covered the rest)
	expense := types.Expense{
		Date:           now.Format(time.DateTime),
		Category:       testCategory.Name,
		CategoryId:     testCategory.ID,
		Expense:        100.00,
		Description:    "Dinner split",
		Method:         "Debit",
		OriginalAmount: 100.00,
		AccountId:      testAccount.ID,
		AccountType:    testAccount.Type,
	}

	debt := types.Debt{
		Description:    "John's share of dinner",
		Amount:         60.00, // Partial amount
		DebtorId:       testDebtor.ID,
		DebtorName:     testDebtor.Name,
		Date:           now.Format(time.DateTime),
		OriginalAmount: 60.00,
		Currency:       "USD",
		Outbound:       true,
	}

	expenseResult, debtResult, err := postgres.InsertExpenseWithDebt(expense, debt)
	AssertNoError(t, err, "Insert expense with partial debt")

	AssertFloatEqual(t, 100.00, expenseResult.Expense, 0.01, "Expense should be full amount")
	AssertFloatEqual(t, 60.00, debtResult.Amount, 0.01, "Debt should be partial amount")
}

// TestExpenseWithDebtTransaction verifies atomicity (both succeed or both fail)
func TestExpenseWithDebtTransaction(t *testing.T) {
	CleanupTables(t)
	SeedTestData(t)

	testAccount := GetTestAccount(TestAccountBankID)
	testCategory := GetTestCategory(TestCategoryFoodID)
	now := time.Now()

	initialExpenseCount := CountTableRows(t, "expenses")
	initialDebtCount := CountTableRows(t, "debts")

	// Try with invalid debtor (should fail)
	expense := types.Expense{
		Date:           now.Format(time.DateTime),
		Category:       testCategory.Name,
		CategoryId:     testCategory.ID,
		Expense:        100.00,
		Description:    "Test",
		Method:         "Debit",
		OriginalAmount: 100.00,
		AccountId:      testAccount.ID,
		AccountType:    testAccount.Type,
	}

	debt := types.Debt{
		Description:    "Test",
		Amount:         100.00,
		DebtorId:       99999, // Invalid debtor
		DebtorName:     "Ghost",
		Date:           now.Format(time.DateTime),
		OriginalAmount: 100.00,
		Currency:       "USD",
		Outbound:       true,
	}

	_, _, err := postgres.InsertExpenseWithDebt(expense, debt)
	// Should fail due to foreign key constraint
	AssertError(t, err, "Should fail with invalid debtor")

	// Verify neither record was created (transaction rolled back)
	newExpenseCount := CountTableRows(t, "expenses")
	newDebtCount := CountTableRows(t, "debts")

	AssertEqual(t, initialExpenseCount, newExpenseCount, "No expense should be created on failure")
	AssertEqual(t, initialDebtCount, newDebtCount, "No debt should be created on failure")
}

// TestExpenseWithDebtZeroAmount verifies validation
func TestExpenseWithDebtZeroAmount(t *testing.T) {
	CleanupTables(t)
	SeedTestData(t)

	testAccount := GetTestAccount(TestAccountBankID)
	testCategory := GetTestCategory(TestCategoryFoodID)
	testDebtor := GetTestDebtor(TestDebtorJohnID)
	now := time.Now()

	// Zero expense should fail
	expense := types.Expense{
		Date:           now.Format(time.DateTime),
		Category:       testCategory.Name,
		CategoryId:     testCategory.ID,
		Expense:        0,
		Description:    "Zero expense",
		Method:         "Debit",
		OriginalAmount: 0,
		AccountId:      testAccount.ID,
		AccountType:    testAccount.Type,
	}

	debt := types.Debt{
		Description:    "Zero expense",
		Amount:         100.00,
		DebtorId:       testDebtor.ID,
		DebtorName:     testDebtor.Name,
		Date:           now.Format(time.DateTime),
		OriginalAmount: 100.00,
		Currency:       "USD",
		Outbound:       true,
	}

	_, _, err := postgres.InsertExpenseWithDebt(expense, debt)
	AssertError(t, err, "Should reject zero expense amount")

	// Zero debt should fail
	expense.Expense = 100.00
	expense.OriginalAmount = 100.00
	debt.Amount = 0
	debt.OriginalAmount = 0

	_, _, err = postgres.InsertExpenseWithDebt(expense, debt)
	AssertError(t, err, "Should reject zero debt amount")
}

// ========== DEBT REPAYMENT ==========

// TestDebtRepaymentCreation verifies repayment creates income + debt
func TestDebtRepaymentCreation(t *testing.T) {
	CleanupTables(t)
	SeedTestData(t)

	testAccount := GetTestAccount(TestAccountBankID)
	testDebtor := GetTestDebtor(TestDebtorJohnID)
	now := time.Now()

	income := types.Income{
		Date:        now.Format(time.DateTime),
		Amount:      50.00,
		Description: "Repayment from John",
		AccountId:   testAccount.ID,
		AccountName: testAccount.Name,
	}

	accountId := testAccount.ID
	debt := types.Debt{
		Description:    "Repayment from John",
		Amount:         50.00,
		DebtorId:       testDebtor.ID,
		DebtorName:     testDebtor.Name,
		Date:           now.Format(time.DateTime),
		OriginalAmount: 50.00,
		Currency:       "USD",
		Outbound:       false, // Inbound = they paid us
		AccountId:      &accountId,
	}

	incomeResult, debtResult, err := postgres.RecordDebtRepayment(income, debt)
	AssertNoError(t, err, "Record debt repayment")

	// Verify income created
	if incomeResult.Id == 0 {
		t.Error("Income should have an ID")
	}
	AssertFloatEqual(t, 50.00, incomeResult.Amount, 0.01, "Income amount")

	// Verify debt created
	if debtResult.Id == 0 {
		t.Error("Debt should have an ID")
	}
	AssertEqual(t, false, debtResult.Outbound, "Repayment debt should be inbound")

	// Verify debt is linked to income
	if debtResult.IncomeId == nil || *debtResult.IncomeId != incomeResult.Id {
		t.Errorf("Debt should be linked to income ID %d, got %v", incomeResult.Id, debtResult.IncomeId)
	}
}

// TestDebtRepaymentAffectsExpectedBalance verifies expected balance increases
func TestDebtRepaymentAffectsExpectedBalance(t *testing.T) {
	CleanupTables(t)
	SeedTestData(t)

	testAccount := GetTestAccount(TestAccountBankID)
	testDebtor := GetTestDebtor(TestDebtorJohnID)
	now := time.Now()

	// Get initial expected balance
	initialExpected := GetAccountExpectedBalance(t, testAccount.ID)

	income := types.Income{
		Date:        now.Format(time.DateTime),
		Amount:      75.00,
		Description: "Repayment",
		AccountId:   testAccount.ID,
		AccountName: testAccount.Name,
	}

	accountId := testAccount.ID
	debt := types.Debt{
		Description:    "Repayment",
		Amount:         75.00,
		DebtorId:       testDebtor.ID,
		DebtorName:     testDebtor.Name,
		Date:           now.Format(time.DateTime),
		OriginalAmount: 75.00,
		Currency:       "USD",
		Outbound:       false,
		AccountId:      &accountId,
	}

	_, _, err := postgres.RecordDebtRepayment(income, debt)
	AssertNoError(t, err, "Record debt repayment")

	// Check expected balance increased by income amount
	newExpected := GetAccountExpectedBalance(t, testAccount.ID)
	expectedChange := initialExpected + 75.00
	AssertFloatEqual(t, expectedChange, newExpected, 0.01, "Expected balance should increase")
}

// ========== DEBT BY DEBTOR SUMMARY ==========

// TestDebtByDebtorSummary verifies net owed calculation
func TestDebtByDebtorSummary(t *testing.T) {
	CleanupTables(t)
	SeedTestData(t)

	testAccount := GetTestAccount(TestAccountBankID)
	testCategory := GetTestCategory(TestCategoryFoodID)
	testDebtor := GetTestDebtor(TestDebtorJohnID)
	now := time.Now()

	// Lend $100 to John (creates expense + outbound debt)
	expense1 := types.Expense{
		Date:           now.Format(time.DateTime),
		Category:       testCategory.Name,
		CategoryId:     testCategory.ID,
		Expense:        100.00,
		Description:    "Lent to John",
		Method:         "Debit",
		OriginalAmount: 100.00,
		AccountId:      testAccount.ID,
		AccountType:    testAccount.Type,
	}
	debt1 := types.Debt{
		Description:    "Lent to John",
		Amount:         100.00,
		DebtorId:       testDebtor.ID,
		DebtorName:     testDebtor.Name,
		Date:           now.Format(time.DateTime),
		OriginalAmount: 100.00,
		Currency:       "USD",
		Outbound:       true,
	}
	_, _, err := postgres.InsertExpenseWithDebt(expense1, debt1)
	AssertNoError(t, err, "Create first debt")

	// John pays back $40
	income := types.Income{
		Date:        now.Format(time.DateTime),
		Amount:      40.00,
		Description: "Partial repayment",
		AccountId:   testAccount.ID,
		AccountName: testAccount.Name,
	}
	accountId := testAccount.ID
	debt2 := types.Debt{
		Description:    "Partial repayment",
		Amount:         40.00,
		DebtorId:       testDebtor.ID,
		DebtorName:     testDebtor.Name,
		Date:           now.Format(time.DateTime),
		OriginalAmount: 40.00,
		Currency:       "USD",
		Outbound:       false,
		AccountId:      &accountId,
	}
	_, _, err = postgres.RecordDebtRepayment(income, debt2)
	AssertNoError(t, err, "Record repayment")

	// Get summary by debtor
	summary, err := postgres.GetDebtorsWithDebts()
	AssertNoError(t, err, "Get debts by debtor")

	// Find John's summary
	var johnSummary *types.DebtByDebtor
	for i := range summary {
		if summary[i].DebtorId == testDebtor.ID {
			johnSummary = &summary[i]
			break
		}
	}

	if johnSummary == nil {
		t.Fatal("John's debt summary not found")
	}

	// Verify: lent $100, received $40, net owed = $60
	AssertFloatEqual(t, 100.00, johnSummary.TotalLent, 0.01, "Total lent")
	AssertFloatEqual(t, 40.00, johnSummary.TotalReceived, 0.01, "Total received")
	AssertFloatEqual(t, 60.00, johnSummary.NetOwed, 0.01, "Net owed")
	AssertEqual(t, 2, int(johnSummary.TransactionCount), "Transaction count")
}

// TestDebtDoesNotAffectExpectedBalance verifies standalone debt doesn't affect balance
func TestDebtDoesNotAffectExpectedBalance(t *testing.T) {
	CleanupTables(t)
	SeedTestData(t)

	testAccount := GetTestAccount(TestAccountBankID)
	testDebtor := GetTestDebtor(TestDebtorJohnID)
	now := time.Now()

	// Get initial expected balance
	initialExpected := GetAccountExpectedBalance(t, testAccount.ID)

	// Create standalone debt (no expense linked)
	accountId := testAccount.ID
	debt := types.Debt{
		Description:    "Standalone tracking",
		Amount:         200.00,
		DebtorId:       testDebtor.ID,
		DebtorName:     testDebtor.Name,
		Date:           now.Format(time.DateTime),
		OriginalAmount: 200.00,
		Currency:       "USD",
		Outbound:       true,
		AccountId:      &accountId,
	}

	_, err := postgres.InsertDebt(debt)
	AssertNoError(t, err, "Insert standalone debt")

	// Expected balance should be unchanged
	newExpected := GetAccountExpectedBalance(t, testAccount.ID)
	AssertFloatEqual(t, initialExpected, newExpected, 0.01, "Standalone debt should not affect expected balance")
}

// TestFullDebtLifecycle verifies the complete lend -> repay workflow
func TestFullDebtLifecycle(t *testing.T) {
	CleanupTables(t)
	SeedTestData(t)

	testAccount := GetTestAccount(TestAccountBankID)
	testCategory := GetTestCategory(TestCategoryFoodID)
	testDebtor := GetTestDebtor(TestDebtorJohnID)
	now := time.Now()

	initialExpected := GetAccountExpectedBalance(t, testAccount.ID)

	// Step 1: Lend $100 to John
	expense := types.Expense{
		Date:           now.Format(time.DateTime),
		Category:       testCategory.Name,
		CategoryId:     testCategory.ID,
		Expense:        100.00,
		Description:    "Lent to John",
		Method:         "Debit",
		OriginalAmount: 100.00,
		AccountId:      testAccount.ID,
		AccountType:    testAccount.Type,
	}
	debt := types.Debt{
		Description:    "Lent to John",
		Amount:         100.00,
		DebtorId:       testDebtor.ID,
		DebtorName:     testDebtor.Name,
		Date:           now.Format(time.DateTime),
		OriginalAmount: 100.00,
		Currency:       "USD",
		Outbound:       true,
	}
	_, _, err := postgres.InsertExpenseWithDebt(expense, debt)
	AssertNoError(t, err, "Lend money")

	// Expected balance should be -$100
	afterLend := GetAccountExpectedBalance(t, testAccount.ID)
	AssertFloatEqual(t, initialExpected-100.00, afterLend, 0.01, "After lending")

	// Step 2: John pays back in full
	income := types.Income{
		Date:        now.Format(time.DateTime),
		Amount:      100.00,
		Description: "Full repayment from John",
		AccountId:   testAccount.ID,
		AccountName: testAccount.Name,
	}
	accountId := testAccount.ID
	repaymentDebt := types.Debt{
		Description:    "Full repayment",
		Amount:         100.00,
		DebtorId:       testDebtor.ID,
		DebtorName:     testDebtor.Name,
		Date:           now.Format(time.DateTime),
		OriginalAmount: 100.00,
		Currency:       "USD",
		Outbound:       false,
		AccountId:      &accountId,
	}
	_, _, err = postgres.RecordDebtRepayment(income, repaymentDebt)
	AssertNoError(t, err, "Full repayment")

	// Expected balance should be back to initial (lent $100, got $100 back)
	afterRepay := GetAccountExpectedBalance(t, testAccount.ID)
	AssertFloatEqual(t, initialExpected, afterRepay, 0.01, "After full repayment, balance restored")

	// Verify net owed is $0
	summary, err := postgres.GetDebtorsWithDebts()
	AssertNoError(t, err, "Get summary")

	var johnSummary *types.DebtByDebtor
	for i := range summary {
		if summary[i].DebtorId == testDebtor.ID {
			johnSummary = &summary[i]
			break
		}
	}

	if johnSummary == nil {
		t.Fatal("John's summary not found")
	}

	AssertFloatEqual(t, 0, johnSummary.NetOwed, 0.01, "Net owed should be $0 after full repayment")
}

// ========== DEBTOR CRUD ==========

// TestDebtorCreation verifies debtor creation
func TestDebtorCreation(t *testing.T) {
	CleanupTables(t)
	SeedTestData(t)

	debtor := types.Debtor{
		Name:        "TestDebtor",
		FirstName:   "Test",
		LastName:    "Debtor",
		Description: "Created in test",
	}

	result, err := postgres.InsertDebtorIntoDatabase(debtor)
	AssertNoError(t, err, "Create debtor")

	if result.Id == 0 {
		t.Error("Debtor should have an ID")
	}
	AssertEqual(t, "TestDebtor", result.Name, "Debtor name")
}

// TestGetDebtors verifies debtor listing
func TestGetDebtors(t *testing.T) {
	CleanupTables(t)
	SeedTestData(t)

	// Should have test debtors from seed
	var count int
	err := testPool.QueryRow(context.Background(), "SELECT COUNT(*) FROM debtors").Scan(&count)
	AssertNoError(t, err, "Count debtors")

	if count < 2 {
		t.Errorf("Expected at least 2 debtors from seed, got %d", count)
	}
}
