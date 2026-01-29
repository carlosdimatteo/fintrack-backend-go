package tests

import (
	"context"
	"testing"
	"time"

	"github.com/carlosdimatteo/fintrack-backend-go/adapters/postgres"
	"github.com/carlosdimatteo/fintrack-backend-go/types"
)

// ========== MONTHLY SUMMARIES ==========

// TestGetMonthlyExpenseSum verifies expense sum calculation
func TestGetMonthlyExpenseSum(t *testing.T) {
	CleanupTables(t)
	SeedTestData(t)

	testAccount := GetTestAccount(TestAccountBankID)
	testCategory := GetTestCategory(TestCategoryFoodID)
	now := time.Now()

	// Add expenses for current month
	amounts := []float64{100.00, 250.50, 75.25}
	expectedSum := 0.0

	for _, amount := range amounts {
		expense := types.Expense{
			Date:           now.Format(time.DateTime),
			Category:       testCategory.Name,
			CategoryId:     testCategory.ID,
			Expense:        amount,
			Description:    "Monthly expense",
			Method:         "Debit",
			OriginalAmount: amount,
			AccountId:      testAccount.ID,
			AccountType:    testAccount.Type,
		}
		_, err := postgres.InsertExpense(expense)
		AssertNoError(t, err, "Insert expense")
		expectedSum += amount
	}

	// Get monthly sum
	sum, err := postgres.GetMonthlyExpenseSum(now.Year(), int(now.Month()))
	AssertNoError(t, err, "Get monthly expense sum")
	AssertFloatEqual(t, expectedSum, sum, 0.01, "Monthly expense sum")
}

// TestGetMonthlyInvestmentSum verifies investment deposit sum
func TestGetMonthlyInvestmentSum(t *testing.T) {
	CleanupTables(t)
	SeedTestData(t)

	testInvAccount := GetTestInvestmentAccount(TestInvAccountCryptoID)
	testFiatAccount := GetTestAccount(TestAccountBankID)
	now := time.Now()

	// Add deposits for current month
	depositAmounts := []float64{500.00, 300.00, 200.00}
	expectedSum := 0.0

	for _, amount := range depositAmounts {
		investment := types.Investment{
			Date:            now.Format(time.DateTime),
			Description:     "Monthly deposit",
			Amount:          amount,
			AccountId:       testInvAccount.ID,
			AccountName:     testInvAccount.Name,
			Type:            "deposit",
			SourceAccountId: &testFiatAccount.ID,
		}
		_, err := postgres.InsertInvestment(investment)
		AssertNoError(t, err, "Insert investment")
		expectedSum += amount
	}

	// Add a withdrawal (should NOT be included in deposit sum)
	withdrawal := types.Investment{
		Date:            now.Format(time.DateTime),
		Description:     "Withdrawal",
		Amount:          100.00,
		AccountId:       testInvAccount.ID,
		AccountName:     testInvAccount.Name,
		Type:            "withdrawal",
		SourceAccountId: &testFiatAccount.ID,
	}
	_, err := postgres.InsertInvestment(withdrawal)
	AssertNoError(t, err, "Insert withdrawal")

	// Get monthly investment sum (deposits only)
	sum, err := postgres.GetMonthlyInvestmentSum(now.Year(), int(now.Month()))
	AssertNoError(t, err, "Get monthly investment sum")
	AssertFloatEqual(t, expectedSum, sum, 0.01, "Monthly investment sum (deposits only)")
}

// ========== YTD TOTALS ==========

// TestGetYTDTotals verifies year-to-date calculations
func TestGetYTDTotals(t *testing.T) {
	CleanupTables(t)
	SeedTestData(t)

	testAccount := GetTestAccount(TestAccountBankID)
	testCategory := GetTestCategory(TestCategoryFoodID)
	testInvAccount := GetTestInvestmentAccount(TestInvAccountCryptoID)
	now := time.Now()

	// Add income
	totalIncome := 0.0
	incomeAmounts := []float64{5000.00, 3000.00}
	for _, amount := range incomeAmounts {
		income := types.Income{
			Date:        now.Format(time.DateTime),
			Amount:      amount,
			Description: "YTD income",
			AccountId:   testAccount.ID,
			AccountName: testAccount.Name,
		}
		_, err := postgres.InsertIncome(income)
		AssertNoError(t, err, "Insert income")
		totalIncome += amount
	}

	// Add expenses
	totalExpenses := 0.0
	expenseAmounts := []float64{500.00, 300.00, 200.00}
	for _, amount := range expenseAmounts {
		expense := types.Expense{
			Date:           now.Format(time.DateTime),
			Category:       testCategory.Name,
			CategoryId:     testCategory.ID,
			Expense:        amount,
			Description:    "YTD expense",
			Method:         "Debit",
			OriginalAmount: amount,
			AccountId:      testAccount.ID,
			AccountType:    testAccount.Type,
		}
		_, err := postgres.InsertExpense(expense)
		AssertNoError(t, err, "Insert expense")
		totalExpenses += amount
	}

	// Add investment deposits
	totalInvestments := 0.0
	investmentAmounts := []float64{1000.00, 500.00}
	for _, amount := range investmentAmounts {
		investment := types.Investment{
			Date:            now.Format(time.DateTime),
			Description:     "YTD investment",
			Amount:          amount,
			AccountId:       testInvAccount.ID,
			AccountName:     testInvAccount.Name,
			Type:            "deposit",
			SourceAccountId: &testAccount.ID,
		}
		_, err := postgres.InsertInvestment(investment)
		AssertNoError(t, err, "Insert investment")
		totalInvestments += amount
	}

	// Get YTD totals
	ytdIncome, ytdExpenses, ytdInvestments := postgres.GetYTDTotals(now.Year())

	AssertFloatEqual(t, totalIncome, ytdIncome, 0.01, "YTD income")
	AssertFloatEqual(t, totalExpenses, ytdExpenses, 0.01, "YTD expenses")
	AssertFloatEqual(t, totalInvestments, ytdInvestments, 0.01, "YTD investments")
}

// ========== INVESTMENT ACCOUNT SUMMARY ==========

// TestInvestmentAccountSummary verifies PnL calculation
func TestInvestmentAccountSummary(t *testing.T) {
	CleanupTables(t)
	SeedTestData(t)

	// Get summary
	summary, err := postgres.GetInvestmentAccountSummary()
	AssertNoError(t, err, "Get investment account summary")

	if len(summary) < 2 {
		t.Fatalf("Expected at least 2 investment accounts, got %d", len(summary))
	}

	// Find our test crypto account
	var cryptoSummary *types.InvestmentAccountSummary
	for i := range summary {
		if summary[i].Id == TestInvAccountCryptoID {
			cryptoSummary = &summary[i]
			break
		}
	}

	if cryptoSummary == nil {
		t.Fatal("Test crypto account not found in summary")
	}

	testInvAccount := GetTestInvestmentAccount(TestInvAccountCryptoID)

	// PnL = real_balance - capital
	expectedPnL := testInvAccount.Balance - testInvAccount.Capital
	AssertFloatEqual(t, expectedPnL, cryptoSummary.PnL, 0.01, "PnL calculation")

	// PnL percent = (PnL / capital) * 100
	if testInvAccount.Capital > 0 {
		expectedPnLPercent := (expectedPnL / testInvAccount.Capital) * 100
		AssertFloatEqual(t, expectedPnLPercent, cryptoSummary.PnLPercent, 0.1, "PnL percent")
	}
}

// ========== NET WORTH SNAPSHOT ==========

// TestCalculateNetWorthSnapshot verifies net worth calculation
func TestCalculateNetWorthSnapshot(t *testing.T) {
	CleanupTables(t)
	SeedTestData(t)

	now := time.Now()

	// Calculate snapshot
	snapshot, err := postgres.CalculateNetWorthSnapshot(now.Year(), int(now.Month()))
	AssertNoError(t, err, "Calculate net worth snapshot")

	// Verify year/month
	AssertEqual(t, now.Year(), snapshot.Year, "Snapshot year")
	AssertEqual(t, int(now.Month()), snapshot.Month, "Snapshot month")

	// Total fiat should be sum of all test account balances
	expectedFiat := 0.0
	for _, acc := range TestAccounts {
		expectedFiat += acc.Balance
	}

	// Note: There might be other accounts in DB, so we check >= expected
	if snapshot.TotalFiatBalance < expectedFiat {
		t.Errorf("Total fiat balance %.2f should be >= %.2f", snapshot.TotalFiatBalance, expectedFiat)
	}

	// Total real net worth = fiat + investments
	if snapshot.TotalRealNetWorth <= 0 {
		t.Error("Total real net worth should be positive")
	}

	// Percentages should sum to ~100
	totalPercent := snapshot.FiatPercent + snapshot.CryptoPercent + snapshot.BrokerPercent
	if totalPercent < 99 || totalPercent > 101 {
		t.Errorf("Percentages should sum to ~100, got %.2f", totalPercent)
	}
}

// TestNetWorthSnapshotExpectedVsReal verifies expected/real tracking
func TestNetWorthSnapshotExpectedVsReal(t *testing.T) {
	CleanupTables(t)
	SeedTestData(t)

	testAccount := GetTestAccount(TestAccountBankID)
	testCategory := GetTestCategory(TestCategoryFoodID)
	now := time.Now()

	// Initially, expected = real (no transactions)
	snapshot1, err := postgres.CalculateNetWorthSnapshot(now.Year(), int(now.Month()))
	AssertNoError(t, err, "Calculate initial snapshot")

	// Add expense (creates discrepancy: expected decreases, real stays same)
	expense := types.Expense{
		Date:           now.Format(time.DateTime),
		Category:       testCategory.Name,
		CategoryId:     testCategory.ID,
		Expense:        500.00,
		Description:    "Create discrepancy",
		Method:         "Debit",
		OriginalAmount: 500.00,
		AccountId:      testAccount.ID,
		AccountType:    testAccount.Type,
	}
	_, err = postgres.InsertExpense(expense)
	AssertNoError(t, err, "Insert expense")

	// Calculate again
	snapshot2, err := postgres.CalculateNetWorthSnapshot(now.Year(), int(now.Month()))
	AssertNoError(t, err, "Calculate snapshot after expense")

	// Expected fiat should be less than before
	if snapshot2.ExpectedFiatBalance >= snapshot1.ExpectedFiatBalance {
		t.Error("Expected fiat should decrease after expense")
	}

	// Real fiat should be unchanged
	AssertFloatEqual(t, snapshot1.TotalFiatBalance, snapshot2.TotalFiatBalance, 0.01,
		"Real fiat unchanged by expense")

	// Fiat discrepancy should be positive (real > expected)
	if snapshot2.FiatDiscrepancy <= 0 {
		t.Errorf("Fiat discrepancy should be positive, got %.2f", snapshot2.FiatDiscrepancy)
	}
}

// TestUpsertNetWorthSnapshot verifies upsert behavior
func TestUpsertNetWorthSnapshot(t *testing.T) {
	CleanupTables(t)
	SeedTestData(t)

	now := time.Now()

	// Calculate and save snapshot
	snapshot1, err := postgres.CalculateNetWorthSnapshot(now.Year(), int(now.Month()))
	AssertNoError(t, err, "Calculate snapshot")

	saved1, err := postgres.UpsertNetWorthSnapshot(snapshot1)
	AssertNoError(t, err, "Upsert snapshot 1")

	if saved1.Id == 0 {
		t.Error("Saved snapshot should have an ID")
	}

	// Upsert again for same year/month - should update, not create new
	snapshot2, err := postgres.CalculateNetWorthSnapshot(now.Year(), int(now.Month()))
	AssertNoError(t, err, "Calculate snapshot 2")

	saved2, err := postgres.UpsertNetWorthSnapshot(snapshot2)
	AssertNoError(t, err, "Upsert snapshot 2")

	// Should have same ID (updated, not inserted)
	AssertEqual(t, saved1.Id, saved2.Id, "Upsert should update same record")

	// Verify only 1 snapshot for this month
	var count int
	err = testPool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM net_worth_snapshots WHERE year = $1 AND month = $2`,
		now.Year(), now.Month(),
	).Scan(&count)
	AssertNoError(t, err, "Count snapshots")
	AssertEqual(t, 1, count, "Only one snapshot per month")
}

// TestGetNetWorthHistory verifies history retrieval
func TestGetNetWorthHistory(t *testing.T) {
	CleanupTables(t)
	SeedTestData(t)

	now := time.Now()

	// Create and save a snapshot
	snapshot, err := postgres.CalculateNetWorthSnapshot(now.Year(), int(now.Month()))
	AssertNoError(t, err, "Calculate snapshot")

	_, err = postgres.UpsertNetWorthSnapshot(snapshot)
	AssertNoError(t, err, "Upsert snapshot")

	// Get history
	history, err := postgres.GetNetWorthHistory()
	AssertNoError(t, err, "Get net worth history")

	if len(history) == 0 {
		t.Fatal("Expected at least 1 snapshot in history")
	}

	// Find our snapshot
	found := false
	for _, h := range history {
		if h.Year == now.Year() && h.Month == int(now.Month()) {
			found = true
			// Verify all fields are populated
			if h.TotalRealNetWorth <= 0 {
				t.Error("Snapshot in history should have positive net worth")
			}
		}
	}
	if !found {
		t.Error("Current month snapshot not found in history")
	}
}

// ========== YEARLY GOALS ==========

// TestYearlyGoals verifies goals CRUD
func TestYearlyGoals(t *testing.T) {
	CleanupTables(t)
	SeedTestData(t)

	year := time.Now().Year()

	// Get goals (might be empty)
	goals1, err := postgres.GetYearlyGoals(year)
	AssertNoError(t, err, "Get initial goals")
	// Empty goals should have the year set
	AssertEqual(t, year, goals1.Year, "Goals year")

	// Upsert new goals
	newGoals := types.YearlyGoals{
		Year:            year,
		SavingsGoal:     15000.00,
		InvestmentGoal:  8000.00,
		IdealInvestment: 12000.00,
	}

	saved, err := postgres.UpsertYearlyGoals(newGoals)
	AssertNoError(t, err, "Upsert goals")

	AssertFloatEqual(t, 15000.00, saved.SavingsGoal, 0.01, "Savings goal")
	AssertFloatEqual(t, 8000.00, saved.InvestmentGoal, 0.01, "Investment goal")
	AssertFloatEqual(t, 12000.00, saved.IdealInvestment, 0.01, "Ideal investment")

	// Update goals
	updatedGoals := types.YearlyGoals{
		Year:            year,
		SavingsGoal:     20000.00,
		InvestmentGoal:  10000.00,
		IdealInvestment: 15000.00,
	}

	saved2, err := postgres.UpsertYearlyGoals(updatedGoals)
	AssertNoError(t, err, "Update goals")

	AssertFloatEqual(t, 20000.00, saved2.SavingsGoal, 0.01, "Updated savings goal")

	// Verify only 1 record for this year
	var count int
	err = testPool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM yearly_goals WHERE year = $1`, year,
	).Scan(&count)
	AssertNoError(t, err, "Count goals")
	AssertEqual(t, 1, count, "Only one goals record per year")
}

// ========== EXPECTED BALANCE VIEW ==========

// TestAccountExpectedBalanceView verifies the view returns correct data
func TestAccountExpectedBalanceView(t *testing.T) {
	CleanupTables(t)
	SeedTestData(t)

	// Get expected balances
	balances, err := postgres.GetAccountExpectedBalances()
	AssertNoError(t, err, "Get expected balances")

	if len(balances) == 0 {
		t.Fatal("Expected at least 1 account in expected balance view")
	}

	// Find our test account
	var testBalance *types.AccountExpectedBalance
	for i := range balances {
		if balances[i].Id == TestAccountBankID {
			testBalance = &balances[i]
			break
		}
	}

	if testBalance == nil {
		t.Fatal("Test account not found in expected balance view")
	}

	testAccount := GetTestAccount(TestAccountBankID)

	// With no transactions, expected = starting = real
	AssertFloatEqual(t, testAccount.StartingBalance, testBalance.ExpectedBalance, 0.01,
		"Expected equals starting with no transactions")
	AssertFloatEqual(t, testAccount.Balance, testBalance.RealBalance, 0.01,
		"Real balance matches account balance")
	AssertFloatEqual(t, 0, testBalance.Discrepancy, 0.01,
		"No discrepancy with no transactions")
}

// TestExpectedBalanceFullFormula verifies the complete expected balance formula
func TestExpectedBalanceFullFormula(t *testing.T) {
	CleanupTables(t)
	SeedTestData(t)

	testAccount := GetTestAccount(TestAccountBankID)
	testCategory := GetTestCategory(TestCategoryFoodID)
	testInvAccount := GetTestInvestmentAccount(TestInvAccountCryptoID)
	otherAccount := GetTestAccount(TestAccountSavingsID)
	now := time.Now()

	// Starting balance
	starting := testAccount.StartingBalance // 1000

	// Add income: +500
	income := types.Income{
		Date:        now.Format(time.DateTime),
		Amount:      500.00,
		Description: "Test income",
		AccountId:   testAccount.ID,
		AccountName: testAccount.Name,
	}
	_, err := postgres.InsertIncome(income)
	AssertNoError(t, err, "Insert income")

	// Add expense: -200
	expense := types.Expense{
		Date:           now.Format(time.DateTime),
		Category:       testCategory.Name,
		CategoryId:     testCategory.ID,
		Expense:        200.00,
		Description:    "Test expense",
		Method:         "Debit",
		OriginalAmount: 200.00,
		AccountId:      testAccount.ID,
		AccountType:    testAccount.Type,
	}
	_, err = postgres.InsertExpense(expense)
	AssertNoError(t, err, "Insert expense")

	// Add investment deposit (from this account): -300
	investment := types.Investment{
		Date:            now.Format(time.DateTime),
		Description:     "Test deposit",
		Amount:          300.00,
		AccountId:       testInvAccount.ID,
		AccountName:     testInvAccount.Name,
		Type:            "deposit",
		SourceAccountId: &testAccount.ID,
	}
	_, err = postgres.InsertInvestment(investment)
	AssertNoError(t, err, "Insert investment deposit")

	// Add investment withdrawal (to this account): +100
	withdrawal := types.Investment{
		Date:            now.Format(time.DateTime),
		Description:     "Test withdrawal",
		Amount:          100.00,
		AccountId:       testInvAccount.ID,
		AccountName:     testInvAccount.Name,
		Type:            "withdrawal",
		SourceAccountId: &testAccount.ID,
	}
	_, err = postgres.InsertInvestment(withdrawal)
	AssertNoError(t, err, "Insert investment withdrawal")

	// Add transfer out: -150
	transferOut := types.Transfer{
		Date:            now.Format(time.DateTime),
		Description:     "Transfer out",
		SourceAccountId: testAccount.ID,
		SourceAmount:    150.00,
		DestAccountId:   otherAccount.ID,
		DestAmount:      150.00,
	}
	_, err = postgres.InsertTransfer(transferOut)
	AssertNoError(t, err, "Insert transfer out")

	// Add transfer in: +80
	transferIn := types.Transfer{
		Date:            now.Format(time.DateTime),
		Description:     "Transfer in",
		SourceAccountId: otherAccount.ID,
		SourceAmount:    80.00,
		DestAccountId:   testAccount.ID,
		DestAmount:      80.00,
	}
	_, err = postgres.InsertTransfer(transferIn)
	AssertNoError(t, err, "Insert transfer in")

	// Expected = starting + income - expense - inv_deposit + inv_withdrawal - transfer_out + transfer_in
	// Expected = 1000 + 500 - 200 - 300 + 100 - 150 + 80 = 1030
	expectedBalance := starting + 500 - 200 - 300 + 100 - 150 + 80

	actualExpected := GetAccountExpectedBalance(t, testAccount.ID)
	AssertFloatEqual(t, expectedBalance, actualExpected, 0.01,
		"Expected balance follows full formula")
}
