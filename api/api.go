package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	googleSS "github.com/carlosdimatteo/fintrack-backend-go/adapters/google"
	"github.com/carlosdimatteo/fintrack-backend-go/adapters/postgres"
	"github.com/carlosdimatteo/fintrack-backend-go/adapters/supabase"
	types "github.com/carlosdimatteo/fintrack-backend-go/types"
	"github.com/gorilla/mux"
)

func greet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	res := types.Response{
		Success: true,
		Message: "Fintrack Server up",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func getCategories(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		return
	}
	categories, err := postgres.GetCategories()
	res := map[string][]types.Category{
		"categories": categories,
	}
	if err != nil {
		ServerErrorResponse(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func submitExpenseRow(w http.ResponseWriter, r *http.Request) {
	//Allow CORS here By * or specific origin
	w.Header().Set("Access-Control-Allow-Origin", "*")

	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		return
	}
	var expense types.Expense
	json.NewDecoder(r.Body).Decode(&expense)
	fmt.Println("received: ", expense)
	fmt.Println("submitting row :  description:", expense.Description, " amount:", expense.OriginalAmount, " expense: ", expense.Expense)
	fmt.Println("expense : ", expense.Expense)
	expense.Date = time.Now().Format(time.DateTime)

	// 1. Get config (using postgres)
	config, err := postgres.GetConfigByType("expenses")
	if err != nil {
		fmt.Println("error getting config: ", err)
		ServerErrorResponse(w, r)
		return
	}

	// 2. Append to sheet
	_, err = googleSS.SubmitExpenseRow(expense, config)
	if err != nil {
		ServerErrorResponse(w, r)
		return
	}

	// 3. Insert into database (synchronous, fail on error)
	_, err = postgres.InsertExpense(expense)
	if err != nil {
		log.Printf("Error inserting expense to database: %v", err)
		ServerErrorResponse(w, r)
		return
	}

	res := types.Response{
		Success: true,
		Message: "Expense submitted",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func getExpenses(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		return
	}
	// Get query parameters as strings
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	// Convert limit to int with error handling
	limit := 10 // default value
	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err != nil {
			http.Error(w, "Invalid limit parameter", http.StatusBadRequest)
			return
		}
		limit = parsedLimit
	}

	// Convert offset to int with error handling
	offset := 0 // default value
	if offsetStr != "" {
		parsedOffset, err := strconv.Atoi(offsetStr)
		if err != nil {
			http.Error(w, "Invalid offset parameter", http.StatusBadRequest)
			return
		}
		offset = parsedOffset
	}

	expenses, count, err := postgres.GetExpenses(limit, offset)
	if err != nil {
		ServerErrorResponse(w, r)
		return
	}

	// Create response structure with all fields
	res := map[string]interface{}{
		"expenses": expenses,
		"limit":    limit,
		"offset":   offset,
		"count":    count,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func getBudgets(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		return
	}
	budgets, err := supabase.GetBudgets()
	if err != nil {
		ServerErrorResponse(w, r)
		return
	}
	res := map[string][]types.BudgetByCategory{
		"budgets": budgets,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)

}

func setBudgets(w http.ResponseWriter, r *http.Request) {
	//Allow CORS here By * or specific origin
	w.Header().Set("Access-Control-Allow-Origin", "*")

	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		return
	}

	var arrayOfBudgets []types.Budget
	json.NewDecoder(r.Body).Decode(&arrayOfBudgets)
	config, err := supabase.GetConfigByType(types.ConfigType["budget"])
	if err != nil {
		ServerErrorResponse(w, r)
		return
	}
	_, err = googleSS.SubmitBudget(arrayOfBudgets, config)
	if err != nil {
		ServerErrorResponse(w, r)
		return
	}

	res := types.Response{
		Success: true,
		Message: "Row submitted",
	}
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(res)

	go func() {
		_, err = supabase.InsertBudgetsIntoDatabase(arrayOfBudgets)
		if err != nil {
			log.Fatal(err)
			return
		}
	}()

}
func getConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		return
	}
	config, err := supabase.GetConfig()
	if err != nil {
		ServerErrorResponse(w, r)
		return
	}
	res := map[string][]types.Config{
		"config": config,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}
func setConfig(w http.ResponseWriter, r *http.Request) {
	//Allow CORS here By * or specific origin
	w.Header().Set("Access-Control-Allow-Origin", "*")

	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		return
	}

	var arrayOfConfig []types.Config
	json.NewDecoder(r.Body).Decode(&arrayOfConfig)
	_, err := supabase.InsertConfigIntoDatabase(arrayOfConfig)
	if err != nil {
		ServerErrorResponse(w, r)
		return
	}

	res := types.Response{
		Success: true,
		Message: "Row submitted",
	}
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(res)
}

func submitInvestment(w http.ResponseWriter, r *http.Request) {
	// Allow CORS here By * or specific origin
	w.Header().Set("Access-Control-Allow-Origin", "*")

	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		return
	}
	var investment types.Investment
	json.NewDecoder(r.Body).Decode(&investment)
	fmt.Println("received investment: ", investment)
	fmt.Println("submitting row :  description:", investment.Description, " amount:", investment.Amount, " account: ", investment.AccountName, " type: ", investment.Type)

	// Validate type
	if investment.Type != "deposit" && investment.Type != "withdrawal" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(types.Response{
			Success: false,
			Message: "Invalid type: must be 'deposit' or 'withdrawal'",
		})
		return
	}

	investment.Date = time.Now().Format(time.DateTime)

	// 1. Get config for investment row append
	config, err := postgres.GetConfigByType("investments")
	if err != nil {
		fmt.Println("error getting config: ", err)
		ServerErrorResponse(w, r)
		return
	}

	// 2. Append investment row to sheet
	_, err = googleSS.SubmitInvestment(investment, config)
	if err != nil {
		ServerErrorResponse(w, r)
		return
	}

	// 3. Insert investment and update capital (using postgres, fail on error)
	_, err = postgres.InsertInvestment(investment)
	if err != nil {
		log.Printf("Error inserting investment to database: %v", err)
		ServerErrorResponse(w, r)
		return
	}

	// 4. Update capital cell in sheet (async)
	go func() {
		// Get updated capital
		capital, err := postgres.GetInvestmentAccountCapital(investment.AccountId)
		if err != nil {
			log.Printf("Error getting account capital: %v", err)
			return
		}

		// Investment account row in Fintrack Config: L{id+2}
		// id=1 -> L3, id=2 -> L4, id=3 -> L5
		row := int(investment.AccountId) + 2
		cellRange := fmt.Sprintf("Fintrack Config!L%d", row)

		err = googleSS.UpdateSheetCell(cellRange, capital)
		if err != nil {
			log.Printf("Error updating capital cell: %v", err)
			return
		}

		log.Printf("Updated capital for account %d: %.2f in cell %s", investment.AccountId, capital, cellRange)
	}()

	res := types.Response{
		Success: true,
		Message: "Investment submitted",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}
func submitDebt(w http.ResponseWriter, r *http.Request) {
	// Allow CORS here By * or specific origin
	w.Header().Set("Access-Control-Allow-Origin", "*")

	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		return
	}
	var debt types.Debt
	json.NewDecoder(r.Body).Decode(&debt)
	fmt.Println("received debt: ", debt)
	fmt.Println("submitting row :  description:", debt.Description, " amount:", debt.Amount, " debtor: ", debt.DebtorName)
	fmt.Println("amount : ", debt.Amount)
	debt.Date = time.Now().Format(time.DateTime)
	config, err := supabase.GetConfigByType("debt")
	if err != nil {
		fmt.Println("error getting config: ", err)
		ServerErrorResponse(w, r)
		return
	}
	_, err = googleSS.SubmitDebt(debt, config)
	if err != nil {
		ServerErrorResponse(w, r)
		return
	}
	res := types.Response{
		Success: true,
		Message: "Row submitted",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
	go func() {
		_, err = supabase.InsertDebtIntoDatabase(debt)
		if err != nil {
			log.Fatal(err)
			return
		}
	}()
}

func submitIncome(w http.ResponseWriter, r *http.Request) {
	// Allow CORS here By * or specific origin
	w.Header().Set("Access-Control-Allow-Origin", "*")

	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		return
	}
	var income types.Income
	json.NewDecoder(r.Body).Decode(&income)
	fmt.Println("received income: ", income)
	fmt.Println("submitting row :  description:", income.Description, " amount:", income.Amount, " account: ", income.AccountName)
	fmt.Println("amount : ", income.Amount)
	income.Date = time.Now().Format(time.DateTime)

	// 1. Get config for income row append (using postgres now)
	config, err := postgres.GetConfigByType("income")
	if err != nil {
		fmt.Println("error getting config: ", err)
		ServerErrorResponse(w, r)
		return
	}

	// 2. Append income row to sheet
	_, err = googleSS.SubmitIncome(income, config)
	if err != nil {
		ServerErrorResponse(w, r)
		return
	}

	// 3. Insert income into database (using postgres now, fail on error)
	_, err = postgres.InsertIncome(income)
	if err != nil {
		log.Printf("Error inserting income to database: %v", err)
		ServerErrorResponse(w, r)
		return
	}

	// 4. Update monthly income sum in sheet (async)
	go func() {
		now := time.Now()
		year := now.Year()
		month := int(now.Month())

		// Get monthly config (using postgres)
		monthlyConfig, err := postgres.GetConfigByType("income_monthly")
		if err != nil {
			log.Printf("Error getting income_monthly config: %v", err)
			return
		}

		// Get sum for this month (using postgres)
		sum, err := postgres.GetMonthlyIncomeSum(year, month)
		if err != nil {
			log.Printf("Error getting monthly income sum: %v", err)
			return
		}

		// Calculate the cell for this month
		cellRange := googleSS.CalculateMonthlyCellRange(monthlyConfig.Sheet, monthlyConfig.A1Range, month)

		// Update the cell
		err = googleSS.UpdateSheetCell(cellRange, sum)
		if err != nil {
			log.Printf("Error updating monthly income cell: %v", err)
			return
		}

		log.Printf("Updated monthly income for %d/%d: %.2f in cell %s", month, year, sum, cellRange)
	}()

	res := types.Response{
		Success: true,
		Message: "Row submitted",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func getIncomes(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		return
	}
	// Get query parameters as strings
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	// Convert limit to int with error handling
	limit := 10 // default value
	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err != nil {
			http.Error(w, "Invalid limit parameter", http.StatusBadRequest)
			return
		}
		limit = parsedLimit
	}

	// Convert offset to int with error handling
	offset := 0 // default value
	if offsetStr != "" {
		parsedOffset, err := strconv.Atoi(offsetStr)
		if err != nil {
			http.Error(w, "Invalid offset parameter", http.StatusBadRequest)
			return
		}
		offset = parsedOffset
	}

	incomes, count, err := supabase.GetIncomes(limit, offset)
	if err != nil {
		ServerErrorResponse(w, r)
		return
	}

	// Create response structure with all fields
	res := map[string]interface{}{
		"incomes": incomes,
		"limit":   limit,
		"offset":  offset,
		"count":   count,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func getAccounts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		return
	}
	accounts, err := supabase.GetAccounts()
	if err != nil {
		ServerErrorResponse(w, r)
		return
	}
	res := map[string][]types.Account{
		"accounts": accounts,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func createAccount(w http.ResponseWriter, r *http.Request) {
	// Allow CORS here By * or specific origin
	w.Header().Set("Access-Control-Allow-Origin", "*")

	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		return
	}
	var accountToInsert types.Account
	json.NewDecoder(r.Body).Decode(&accountToInsert)
	fmt.Println("received account: ", accountToInsert)
	config, err := supabase.GetConfigByType("accounts")
	if err != nil {
		ServerErrorResponse(w, r)
		return
	}
	account, err := supabase.InsertAccountIntoDatabase(accountToInsert)
	if err != nil {
		ServerErrorResponse(w, r)
		return
	}
	if err != nil {
		fmt.Println("error getting config: ", err)
		ServerErrorResponse(w, r)
		return
	}
	_, err = googleSS.SubmitAccount(account, config)
	if err != nil {
		ServerErrorResponse(w, r)
		return
	}

	res := types.Response{
		Success: true,
		Message: "Row submitted",
	}
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(res)
}

func getInvestmentAccounts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		return
	}
	accounts, err := supabase.GetInvestmentAccounts()
	if err != nil {
		ServerErrorResponse(w, r)
		return
	}
	res := map[string][]types.InvestmentAccount{
		"accounts": accounts,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func createInvestmentAccount(w http.ResponseWriter, r *http.Request) {
	// Allow CORS here By * or specific origin
	w.Header().Set("Access-Control-Allow-Origin", "*")

	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		return
	}
	var accountToInsert types.InvestmentAccount

	json.NewDecoder(r.Body).Decode(&accountToInsert)
	fmt.Println("received account: ", accountToInsert)
	config, err := supabase.GetConfigByType("investment_accounts")
	if err != nil {
		ServerErrorResponse(w, r)
		return
	}
	accounts, err := supabase.InsertInvestmentAccountIntoDatabase(accountToInsert)
	if err != nil {
		ServerErrorResponse(w, r)
		return
	}
	_, err = googleSS.SubmitInvestmentAccount(accounts[0], config)
	if err != nil {
		ServerErrorResponse(w, r)
		return
	}

	res := types.Response{
		Success: true,
		Message: "Row submitted",
	}
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(res)
}

func getDebtors(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		return
	}
	debtors, err := supabase.GetDebtors()
	if err != nil {
		ServerErrorResponse(w, r)
		return
	}
	res := map[string][]types.Debtor{
		"debtors": debtors,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func getDebtorsWithDebts(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		return
	}
	result, err := supabase.GetDebtorsWithDebts()
	if err != nil {
		ServerErrorResponse(w, r)
		return
	}
	res := map[string][]types.DebtByDebtor{
		"result": result,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func createDebtor(w http.ResponseWriter, r *http.Request) {
	// Allow CORS here By * or specific origin
	w.Header().Set("Access-Control-Allow-Origin", "*")

	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		return
	}
	var debtorToInsert types.Debtor
	json.NewDecoder(r.Body).Decode(&debtorToInsert)
	fmt.Println("received account: ", debtorToInsert)
	config, err := supabase.GetConfigByType("debtors")
	if err != nil {
		ServerErrorResponse(w, r)
		return
	}
	debtors, err := supabase.InsertDebtorIntoDatabase(debtorToInsert)
	if err != nil {
		ServerErrorResponse(w, r)
		return
	}
	_, err = googleSS.SubmitDebtor(debtors[0], config)
	if err != nil {
		ServerErrorResponse(w, r)
		return
	}
	res := types.Response{
		Success: true,
		Message: "Row submitted",
	}
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(res)
}

func setAccountingForCurrentMonth(w http.ResponseWriter, r *http.Request) {
	// get account array for accounts and for investment_accounts , then do update for each on balance
	w.Header().Set("Access-Control-Allow-Origin", "*")

	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		return
	}
	var accountToInsert types.RealBalanceByAccounts
	res := types.RealBalanceByAccounts{Accounts: []types.Account{}, InvestmentAccounts: []types.InvestmentAccount{}}
	json.NewDecoder(r.Body).Decode(&accountToInsert)

	if len(accountToInsert.Accounts) > 0 {
		accounts, err := postgres.UpdateAccountBalances(accountToInsert.Accounts)
		if err != nil {
			log.Printf("Error updating account balances: %v", err)
			ServerErrorResponse(w, r)
			return
		}
		accountConfig, err := postgres.GetConfigByType(types.ConfigType["accounting_accounts"])
		if err != nil {
			log.Printf("Error getting accounting_accounts config: %v", err)
			ServerErrorResponse(w, r)
			return
		}

		_, err = googleSS.UpdateAccountBalances(accounts, accountConfig)
		if err != nil {
			log.Printf("Error updating sheet account balances: %v", err)
			ServerErrorResponse(w, r)
			return
		}

		res.Accounts = accounts
	}

	if len(accountToInsert.InvestmentAccounts) > 0 {
		investmentAccounts, err := postgres.UpdateInvestmentAccountBalances(accountToInsert.InvestmentAccounts)
		if err != nil {
			log.Printf("Error updating investment account balances: %v", err)
			ServerErrorResponse(w, r)
			return
		}
		investmentAccountConfig, err := postgres.GetConfigByType(types.ConfigType["accounting_investment_accounts"])
		if err != nil {
			log.Printf("Error getting accounting_investment_accounts config: %v", err)
			ServerErrorResponse(w, r)
			return
		}

		_, err = googleSS.UpdateInvestmentAccountBalances(investmentAccounts, investmentAccountConfig)
		if err != nil {
			log.Printf("Error updating sheet investment balances: %v", err)
			ServerErrorResponse(w, r)
			return
		}

		res.InvestmentAccounts = investmentAccounts
	}

	// Create net worth snapshot after updating balances
	go func() {
		now := time.Now()
		snapshot, err := postgres.CalculateNetWorthSnapshot(now.Year(), int(now.Month()))
		if err != nil {
			log.Printf("Error calculating net worth snapshot: %v", err)
			return
		}

		_, err = postgres.UpsertNetWorthSnapshot(snapshot)
		if err != nil {
			log.Printf("Error saving net worth snapshot: %v", err)
			return
		}

		log.Printf("Created net worth snapshot for %d/%d: Total $%.2f", now.Month(), now.Year(), snapshot.TotalNetWorth)
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)

}

// ========== GOALS ENDPOINTS ==========

func getGoals(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		return
	}

	// Get year from query param, default to current year
	yearStr := r.URL.Query().Get("year")
	year := time.Now().Year()
	if yearStr != "" {
		if parsed, err := strconv.Atoi(yearStr); err == nil {
			year = parsed
		}
	}

	goals, err := postgres.GetYearlyGoals(year)
	if err != nil {
		log.Printf("Error getting goals: %v", err)
		ServerErrorResponse(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(goals)
}

func setGoals(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		return
	}

	var goals types.YearlyGoals
	if err := json.NewDecoder(r.Body).Decode(&goals); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(types.Response{Success: false, Message: "Invalid JSON"})
		return
	}

	if goals.Year == 0 {
		goals.Year = time.Now().Year()
	}

	result, err := postgres.UpsertYearlyGoals(goals)
	if err != nil {
		log.Printf("Error saving goals: %v", err)
		ServerErrorResponse(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// ========== NET WORTH ENDPOINTS ==========

func getNetWorthHistory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		return
	}

	history, err := postgres.GetNetWorthHistory()
	if err != nil {
		log.Printf("Error getting net worth history: %v", err)
		ServerErrorResponse(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

// ========== INVESTMENT ACCOUNT SUMMARY ==========

func getInvestmentAccountsSummary(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		return
	}

	summary, err := postgres.GetInvestmentAccountSummary()
	if err != nil {
		log.Printf("Error getting investment summary: %v", err)
		ServerErrorResponse(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

func LoadRoutes(muxRouter *mux.Router) {
	api := muxRouter.PathPrefix("/api").Subrouter()
	api.HandleFunc("/", greet).Methods("GET")
	api.HandleFunc("/submit", submitExpenseRow).Methods("POST", "OPTIONS")
	api.HandleFunc("/expenses", getExpenses).Methods("GET", "OPTIONS")
	api.HandleFunc("/budget", setBudgets).Methods("POST", "OPTIONS")
	api.HandleFunc("/budget", getBudgets).Methods("GET")
	api.HandleFunc("/categories", getCategories).Methods("GET")
	api.HandleFunc("/config", getConfig).Methods("GET")
	api.HandleFunc("/config", setConfig).Methods("POST", "OPTIONS")
	api.HandleFunc("/investment", submitInvestment).Methods("POST", "OPTIONS")
	// api.HandleFunc("/investment", getInvestments).Methods("GET")
	api.HandleFunc("/debt", submitDebt).Methods("POST", "OPTIONS")
	// api.HandleFunc("/debt", getDebts).Methods("GET")
	api.HandleFunc("/income", submitIncome).Methods("POST", "OPTIONS")
	api.HandleFunc("/income", getIncomes).Methods("GET")
	api.HandleFunc("/accounts", getAccounts).Methods("GET")
	api.HandleFunc("/accounts", createAccount).Methods("POST", "OPTIONS")
	api.HandleFunc("/investment-accounts", getInvestmentAccounts).Methods("GET")
	api.HandleFunc("/investment-accounts", createInvestmentAccount).Methods("POST", "OPTIONS")
	api.HandleFunc("/debtors", getDebtors).Methods("GET")
	api.HandleFunc("/debtors", createDebtor).Methods("POST", "OPTIONS")
	api.HandleFunc("/debtors/debt", getDebtorsWithDebts).Methods("GET")
	api.HandleFunc("/accounting", setAccountingForCurrentMonth).Methods("POST", "OPTIONS")

	// Phase 4: Goals & Net Worth
	api.HandleFunc("/goals", getGoals).Methods("GET")
	api.HandleFunc("/goals", setGoals).Methods("POST", "OPTIONS")
	api.HandleFunc("/net-worth/history", getNetWorthHistory).Methods("GET")

	// Phase 5: Investment Account Summary
	api.HandleFunc("/investment-accounts/summary", getInvestmentAccountsSummary).Methods("GET")
}

func NotFoundResponse(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	res := types.Response{
		Success: false,
		Message: "Not Found",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func ServerErrorResponse(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
	res := types.Response{
		Success: false,
		Message: "Internal Server Error",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}
