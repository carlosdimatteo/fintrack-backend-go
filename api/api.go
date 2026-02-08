package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	googleSS "github.com/carlosdimatteo/fintrack-backend-go/adapters/google"
	"github.com/carlosdimatteo/fintrack-backend-go/adapters/postgres"
	"github.com/carlosdimatteo/fintrack-backend-go/helpers"
	types "github.com/carlosdimatteo/fintrack-backend-go/types"
	"github.com/gorilla/mux"
)

const exchangerateAPIHost = "https://v6.exchangerate-api.com/v6"

var (
	warnSkipAPIKeyCheckOnce       sync.Once
	warnExchangeRateKeyDevOnce    sync.Once
	warnExchangeRateKeyNotDevOnce sync.Once
)

// corsMiddleware handles CORS headers and preflight requests
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowed := false
		allowedOrigins := helpers.GetAllowedOrigins()

		// Check if origin is allowed
		for _, o := range allowedOrigins {
			if o == "*" || o == origin {
				allowed = true
				w.Header().Set("Access-Control-Allow-Origin", origin)
				break
			}
		}

		if !allowed && origin != "" {
			// Origin not allowed
			http.Error(w, "Origin not allowed", http.StatusForbidden)
			return
		}

		// If no origin (same-origin request), allow it
		if origin == "" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key, Authorization")
		w.Header().Set("Access-Control-Max-Age", "86400")

		// Handle preflight
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// apiKeyMiddleware validates the API key
func apiKeyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for OPTIONS (preflight)
		if r.Method == "OPTIONS" {
			next.ServeHTTP(w, r)
			return
		}

		apiKey := os.Getenv("API_KEY")
		if apiKey == "" {
			// Skip auth only in dev to avoid unprotected API in production
			if helpers.IsDevMode() {
				warnSkipAPIKeyCheckOnce.Do(func() {
					log.Println("WARNING: API_KEY not set and GO_ENV is dev - skipping API key check (set API_KEY in production)")
				})
				next.ServeHTTP(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(types.Response{
				Success: false,
				Message: "API_KEY not configured (set API_KEY or use GO_ENV=dev for local dev)",
			})
			return
		}

		// Check X-API-Key header
		providedKey := r.Header.Get("X-API-Key")
		if providedKey == "" {
			// Also check Authorization: Bearer <key>
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				providedKey = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if providedKey != apiKey {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(types.Response{
				Success: false,
				Message: "Invalid or missing API key",
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}

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
	if err := json.NewDecoder(r.Body).Decode(&expense); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(types.Response{Success: false, Message: "Invalid JSON"})
		return
	}
	expense.Date = time.Now().Format(time.DateTime)

	// 1. Get config (using postgres)
	config, err := postgres.GetConfigByType("expenses")
	if err != nil {
		log.Printf("Error getting config: %v", err)
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
	budgets, err := postgres.GetBudgets()
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
	if err := json.NewDecoder(r.Body).Decode(&arrayOfBudgets); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(types.Response{Success: false, Message: "Invalid JSON"})
		return
	}
	config, err := postgres.GetConfigByType(types.ConfigType["budget"])
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
		_, err = postgres.InsertBudgetsIntoDatabase(arrayOfBudgets)
		if err != nil {
			log.Printf("Error inserting budgets: %v", err)
			ServerErrorResponse(w, r)
			return
		}
	}()

}

func getBudgetHistory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		return
	}

	// Get year from query param, default to current year
	year := time.Now().Year()
	yearStr := r.URL.Query().Get("year")
	if yearStr != "" {
		if y, err := strconv.Atoi(yearStr); err == nil && y >= 2000 && y <= 2100 {
			year = y
		}
	}

	history, err := postgres.GetBudgetHistory(year)
	if err != nil {
		log.Printf("Error getting budget history: %v", err)
		ServerErrorResponse(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

func getConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		return
	}
	config, err := postgres.GetConfig()
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
	if err := json.NewDecoder(r.Body).Decode(&arrayOfConfig); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(types.Response{Success: false, Message: "Invalid JSON"})
		return
	}
	_, err := postgres.InsertConfigIntoDatabase(arrayOfConfig)
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
	if err := json.NewDecoder(r.Body).Decode(&investment); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(types.Response{Success: false, Message: "Invalid JSON"})
		return
	}

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
		log.Printf("Error getting config: %v", err)
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
	if err := json.NewDecoder(r.Body).Decode(&debt); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(types.Response{Success: false, Message: "Invalid JSON"})
		return
	}
	debt.Date = time.Now().Format(time.DateTime)
	config, err := postgres.GetConfigByType("debt")
	if err != nil {
		log.Printf("Error getting config: %v", err)
		ServerErrorResponse(w, r)
		return
	}
	_, err = googleSS.SubmitDebt(debt, config)
	if err != nil {
		ServerErrorResponse(w, r)
		return
	}

	// Insert into database (fail on error)
	_, err = postgres.InsertDebt(debt)
	if err != nil {
		log.Printf("Error inserting debt to database: %v", err)
		ServerErrorResponse(w, r)
		return
	}

	res := types.Response{
		Success: true,
		Message: "Debt submitted",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func submitIncome(w http.ResponseWriter, r *http.Request) {
	// Allow CORS here By * or specific origin
	w.Header().Set("Access-Control-Allow-Origin", "*")

	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		return
	}
	var income types.Income
	if err := json.NewDecoder(r.Body).Decode(&income); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(types.Response{Success: false, Message: "Invalid JSON"})
		return
	}
	income.Date = time.Now().Format(time.DateTime)
	if income.OriginalAmount == 0 {
		income.OriginalAmount = income.Amount
	}
	if income.Currency == "" {
		income.Currency = "USD"
	}

	// 1. Get config for income row append (using postgres now)
	config, err := postgres.GetConfigByType("income")
	if err != nil {
		log.Printf("Error getting config: %v", err)
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

	incomes, count, err := postgres.GetIncomes(limit, offset)
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
	accounts, err := postgres.GetAccounts()
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
	if err := json.NewDecoder(r.Body).Decode(&accountToInsert); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(types.Response{Success: false, Message: "Invalid JSON"})
		return
	}
	config, err := postgres.GetConfigByType("accounts")
	if err != nil {
		log.Printf("Error getting config: %v", err)
		ServerErrorResponse(w, r)
		return
	}
	account, err := postgres.InsertAccountIntoDatabase(accountToInsert)
	if err != nil {
		log.Printf("Error inserting account: %v", err)
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

func getInvestments(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		return
	}

	// Parse query params
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	accountIdStr := r.URL.Query().Get("account_id")

	limit := 50
	offset := 0
	var accountId *int32

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}
	if accountIdStr != "" {
		if id, err := strconv.Atoi(accountIdStr); err == nil {
			id32 := int32(id)
			accountId = &id32
		}
	}

	investments, count, err := postgres.GetInvestments(limit, offset, accountId)
	if err != nil {
		log.Printf("Error getting investments: %v", err)
		ServerErrorResponse(w, r)
		return
	}

	res := map[string]interface{}{
		"investments": investments,
		"total":       count,
		"limit":       limit,
		"offset":      offset,
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
	accounts, err := postgres.GetInvestmentAccounts()
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
	if err := json.NewDecoder(r.Body).Decode(&accountToInsert); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(types.Response{Success: false, Message: "Invalid JSON"})
		return
	}
	config, err := postgres.GetConfigByType("investment_accounts")
	if err != nil {
		ServerErrorResponse(w, r)
		return
	}
	account, err := postgres.InsertInvestmentAccountIntoDatabase(accountToInsert)
	if err != nil {
		ServerErrorResponse(w, r)
		return
	}
	_, err = googleSS.SubmitInvestmentAccount(account, config)
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
	debtors, err := postgres.GetDebtors()
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
	result, err := postgres.GetDebtorsWithDebts()
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
	if err := json.NewDecoder(r.Body).Decode(&debtorToInsert); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(types.Response{Success: false, Message: "Invalid JSON"})
		return
	}
	config, err := postgres.GetConfigByType("debtors")
	if err != nil {
		ServerErrorResponse(w, r)
		return
	}
	debtor, err := postgres.InsertDebtorIntoDatabase(debtorToInsert)
	if err != nil {
		ServerErrorResponse(w, r)
		return
	}
	_, err = googleSS.SubmitDebtor(debtor, config)
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
	if err := json.NewDecoder(r.Body).Decode(&accountToInsert); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(types.Response{Success: false, Message: "Invalid JSON"})
		return
	}
	res := types.RealBalanceByAccounts{Accounts: []types.Account{}, InvestmentAccounts: []types.InvestmentAccount{}}

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

		log.Printf("Created net worth snapshot for %d/%d: Real $%.2f, Expected $%.2f, Discrepancy $%.2f", now.Month(), now.Year(), snapshot.TotalRealNetWorth, snapshot.ExpectedNetWorth, snapshot.TotalDiscrepancy)
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

// ========== INCOME SUMMARY ==========

func getIncomeSummary(w http.ResponseWriter, r *http.Request) {
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

	summary, err := postgres.GetYearlyIncomeSummary(year)
	if err != nil {
		log.Printf("Error getting income summary: %v", err)
		ServerErrorResponse(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// ========== DASHBOARD ==========

type DashboardResponse struct {
	CurrentMonth struct {
		Year               int     `json:"year"`
		Month              int     `json:"month"`
		Income             float64 `json:"income"`
		Expenses           float64 `json:"expenses"`
		InvestmentDeposits float64 `json:"investment_deposits"`
		Savings            float64 `json:"savings"`
		SavingsRate        float64 `json:"savings_rate"`
	} `json:"current_month"`
	YTD struct {
		Income             float64 `json:"income"`
		Expenses           float64 `json:"expenses"`
		InvestmentDeposits float64 `json:"investment_deposits"`
		Savings            float64 `json:"savings"`
	} `json:"ytd"`
	Goals           types.YearlyGoals                `json:"goals"`
	NetWorth        types.NetWorthSnapshot           `json:"net_worth"`
	Investments     []types.InvestmentAccountSummary `json:"investments"`
	SavingsProgress types.SavingsProgress            `json:"savings_progress"`
}

func getDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		return
	}

	now := time.Now()
	year := now.Year()
	month := int(now.Month())

	// Allow querying for specific year/month via query params
	yearStr := r.URL.Query().Get("year")
	monthStr := r.URL.Query().Get("month")

	if yearStr != "" {
		if y, err := strconv.Atoi(yearStr); err == nil && y >= 2000 && y <= 2100 {
			year = y
		}
	}
	if monthStr != "" {
		if m, err := strconv.Atoi(monthStr); err == nil && m >= 1 && m <= 12 {
			month = m
		}
	}

	var dashboard DashboardResponse
	dashboard.CurrentMonth.Year = year
	dashboard.CurrentMonth.Month = month

	// Get current month income
	monthIncome, _ := postgres.GetMonthlyIncomeSum(year, month)
	dashboard.CurrentMonth.Income = monthIncome

	// Get current month expenses
	monthExpenses, _ := postgres.GetMonthlyExpenseSum(year, month)
	dashboard.CurrentMonth.Expenses = monthExpenses

	// Get current month investment deposits
	monthInvestments, _ := postgres.GetMonthlyInvestmentSum(year, month)
	dashboard.CurrentMonth.InvestmentDeposits = monthInvestments

	// Calculate savings
	dashboard.CurrentMonth.Savings = monthIncome - monthExpenses - monthInvestments
	if monthIncome > 0 {
		dashboard.CurrentMonth.SavingsRate = (dashboard.CurrentMonth.Savings / monthIncome) * 100
	}

	// Get YTD totals (for the specified year)
	ytdIncome, ytdExpenses, ytdInvestments := postgres.GetYTDTotals(year)
	dashboard.YTD.Income = ytdIncome
	dashboard.YTD.Expenses = ytdExpenses
	dashboard.YTD.InvestmentDeposits = ytdInvestments
	dashboard.YTD.Savings = ytdIncome - ytdExpenses - ytdInvestments

	// Get goals for the specified year
	goals, _ := postgres.GetYearlyGoals(year)
	dashboard.Goals = goals

	// Get net worth snapshot for the specified month
	// For historical months, use stored snapshot; for current month, calculate live
	currentYear := time.Now().Year()
	currentMonth := int(time.Now().Month())

	if year == currentYear && month == currentMonth {
		// Current month - calculate live
		snapshot, _ := postgres.CalculateNetWorthSnapshot(year, month)
		dashboard.NetWorth = snapshot
	} else {
		// Historical - try to get stored snapshot
		storedSnapshot, found, err := postgres.GetNetWorthSnapshot(year, month)
		if err == nil && found {
			dashboard.NetWorth = storedSnapshot
		} else {
			// No stored snapshot - return empty with year/month set
			dashboard.NetWorth = types.NetWorthSnapshot{Year: year, Month: month}
		}
	}

	// Get investment summary (current state - not historical)
	investments, _ := postgres.GetInvestmentAccountSummary()
	dashboard.Investments = investments

	// Calculate savings progress
	// For current year: use live calculated net worth
	// For past years: use December snapshot of that year (end of year state)
	var progressNetWorth types.NetWorthSnapshot
	if year == currentYear {
		// Current year - calculate live
		progressNetWorth, _ = postgres.CalculateNetWorthSnapshot(year, int(time.Now().Month()))
	} else {
		// Historical year - use December snapshot (year-end state)
		storedSnapshot, found, err := postgres.GetNetWorthSnapshot(year, 12)
		if err == nil && found {
			progressNetWorth = storedSnapshot
		} else {
			// No December snapshot - try to calculate (won't be accurate for historical)
			progressNetWorth, _ = postgres.CalculateNetWorthSnapshot(year, 12)
		}
	}
	dashboard.SavingsProgress = postgres.CalculateSavingsProgress(dashboard.Goals, progressNetWorth)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dashboard)
}

// ========== TRANSFERS ==========

func submitTransfer(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		return
	}

	var transfer types.Transfer
	if err := json.NewDecoder(r.Body).Decode(&transfer); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(types.Response{Success: false, Message: "Invalid JSON"})
		return
	}

	transfer.Date = time.Now().Format(time.DateTime)

	result, err := postgres.InsertTransfer(transfer)
	if err != nil {
		log.Printf("Error inserting transfer: %v", err)
		ServerErrorResponse(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func getTransfers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		return
	}

	limit := 50
	offset := 0

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil {
			limit = parsed
		}
	}
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if parsed, err := strconv.Atoi(offsetStr); err == nil {
			offset = parsed
		}
	}

	transfers, count, err := postgres.GetTransfers(limit, offset)
	if err != nil {
		log.Printf("Error getting transfers: %v", err)
		ServerErrorResponse(w, r)
		return
	}

	res := map[string]interface{}{
		"transfers": transfers,
		"count":     count,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

// ========== EXPECTED BALANCE ==========

func getExpectedBalances(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		return
	}

	balances, err := postgres.GetAccountExpectedBalances()
	if err != nil {
		log.Printf("Error getting expected balances: %v", err)
		ServerErrorResponse(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(balances)
}

func getInvestmentExpectedCapital(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		return
	}

	capital, err := postgres.GetInvestmentAccountExpectedCapital()
	if err != nil {
		log.Printf("Error getting investment expected capital: %v", err)
		ServerErrorResponse(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(capital)
}

// ========== PHASE 7: DEBT MODULE ENHANCEMENTS ==========

func getDebts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		return
	}

	limit := 50
	offset := 0
	var debtorId *int32

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil {
			limit = parsed
		}
	}
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if parsed, err := strconv.Atoi(offsetStr); err == nil {
			offset = parsed
		}
	}
	if debtorIdStr := r.URL.Query().Get("debtor_id"); debtorIdStr != "" {
		if parsed, err := strconv.Atoi(debtorIdStr); err == nil {
			id := int32(parsed)
			debtorId = &id
		}
	}

	debts, count, err := postgres.GetDebts(limit, offset, debtorId)
	if err != nil {
		log.Printf("Error getting debts: %v", err)
		ServerErrorResponse(w, r)
		return
	}

	res := map[string]interface{}{
		"debts": debts,
		"count": count,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func getDebtsByDebtor(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		return
	}

	summary, err := postgres.GetDebtorsWithDebts()
	if err != nil {
		log.Printf("Error getting debts by debtor: %v", err)
		ServerErrorResponse(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

func getRecentExpenses(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		return
	}

	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil {
			limit = parsed
		}
	}

	expenses, err := postgres.GetRecentExpenses(limit)
	if err != nil {
		log.Printf("Error getting recent expenses: %v", err)
		ServerErrorResponse(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(expenses)
}

type RepaymentRequest struct {
	DebtorId       int32   `json:"debtor_id"`
	DebtorName     string  `json:"debtor_name"`
	Amount         float64 `json:"amount"`
	Description    string  `json:"description"`
	AccountId      int32   `json:"account_id"`
	Account        string  `json:"account"`
	Currency       string  `json:"currency"`
	OriginalAmount float64 `json:"originalAmount"` // Informative (e.g. non-USD); defaulted to amount if omitted
}

// ExpenseDebtRequest is for creating an expense	 that also creates a linked debt
// Use case: "I lent $100 to John from my BOFA account"
// DebtEntry represents a single debt in the expense-debt request
type DebtEntry struct {
	DebtorId       int32   `json:"debtor_id"`
	DebtorName     string  `json:"debtor_name"`
	Amount         float64 `json:"amount"`
	Currency       string  `json:"currency"`
	AccountId      int32   `json:"account_id,omitempty"`
	OriginalAmount float64 `json:"original_amount,omitempty"`
}

type ExpenseDebtRequest struct {
	// Expense fields
	Date           string  `json:"date"`
	Category       string  `json:"category"`
	CategoryId     int32   `json:"category_id"`
	Expense        float64 `json:"expense"` // The expense amount (money that left your account)
	Description    string  `json:"description"`
	Method         string  `json:"method"`
	OriginalAmount float64 `json:"originalAmount"`
	AccountId      int32   `json:"account_id"`
	AccountType    string  `json:"account_type"`
	// Multiple debts (preferred)
	Debts []DebtEntry `json:"debts"`
	// Single debt fields (backward compatible)
	DebtorId   int32   `json:"debtor_id"`
	DebtorName string  `json:"debtor_name"`
	DebtAmount float64 `json:"debt_amount"`
	Currency   string  `json:"currency"`
}

func submitExpenseWithDebt(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		return
	}

	var req ExpenseDebtRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(types.Response{Success: false, Message: "Invalid JSON"})
		return
	}

	// Default date to now if not provided
	date := req.Date
	if date == "" {
		date = time.Now().Format(time.DateTime)
	}

	// Create expense record
	expense := types.Expense{
		Date:           date,
		Category:       req.Category,
		CategoryId:     req.CategoryId,
		Expense:        req.Expense,
		Description:    req.Description,
		Method:         req.Method,
		OriginalAmount: req.OriginalAmount,
		AccountId:      req.AccountId,
		AccountType:    req.AccountType,
	}

	// Build debts array - support both new format (debts array) and old format (single debt fields)
	var debts []types.Debt
	accountId := req.AccountId
	currency := req.Currency
	if currency == "" {
		currency = "USD"
	}
	if len(req.Debts) > 0 {
		// New format: multiple debts
		for _, d := range req.Debts {
			debt := types.Debt{
				Description:    req.Description,
				Amount:         d.Amount,
				DebtorId:       d.DebtorId,
				DebtorName:     d.DebtorName,
				Date:           date,
				OriginalAmount: req.OriginalAmount,
				Currency:       currency,
				Outbound:       true,
				AccountId:      &accountId,
			}
			debts = append(debts, debt)
		}
	} else if req.DebtorId != 0 {
		// Old format: single debt (backward compatible)
		debtAmount := req.DebtAmount
		if debtAmount == 0 {
			debtAmount = req.Expense
		}
		debt := types.Debt{
			Description:    req.Description,
			Amount:         debtAmount,
			DebtorId:       req.DebtorId,
			DebtorName:     req.DebtorName,
			Date:           date,
			OriginalAmount: debtAmount,
			Currency:       req.Currency,
			Outbound:       true,
			AccountId:      &accountId,
		}
		debts = append(debts, debt)
	} else {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(types.Response{Success: false, Message: "At least one debt is required (use 'debts' array or single debt fields)"})
		return
	}

	expenseResult, debtResults, err := postgres.InsertExpenseWithDebts(expense, debts)
	if err != nil {
		log.Printf("Error creating expense with debts: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(types.Response{Success: false, Message: err.Error()})
		return
	}

	// Update expense sheet asynchronously
	go func() {
		config, err := postgres.GetConfigByType("expenses")
		if err != nil {
			log.Printf("Error getting expense config: %v", err)
			return
		}
		googleSS.SubmitExpenseRow(expenseResult, config)
	}()

	// Return expense and all debts
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"expense": expenseResult,
		"debts":   debtResults,
	})
}

func submitDebtRepayment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		return
	}

	var req RepaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(types.Response{Success: false, Message: "Invalid JSON"})
		return
	}

	// Create income record (originalAmount and currency in request body; default if not provided)
	origAmt := req.OriginalAmount
	if origAmt == 0 {
		origAmt = req.Amount
	}
	currency := req.Currency
	if currency == "" {
		currency = "USD"
	}
	income := types.Income{
		Date:           time.Now().Format(time.DateTime),
		Amount:         req.Amount,
		Description:    fmt.Sprintf("Debt repayment from %s: %s", req.DebtorName, req.Description),
		AccountId:      req.AccountId,
		AccountName:    req.Account,
		OriginalAmount: origAmt,
		Currency:       currency,
	}

	// Create debt record (negative outbound = they paid us back)
	accountId := req.AccountId
	debt := types.Debt{
		Description:    req.Description,
		Amount:         req.Amount,
		DebtorId:       req.DebtorId,
		DebtorName:     req.DebtorName,
		Date:           time.Now().Format(time.DateTime),
		OriginalAmount: origAmt,
		Currency:       req.Currency,
		Outbound:       false, // Inbound = they paid us
		AccountId:      &accountId,
	}

	incomeResult, debtResult, err := postgres.RecordDebtRepayment(income, debt)
	if err != nil {
		log.Printf("Error recording repayment: %v", err)
		ServerErrorResponse(w, r)
		return
	}

	// Update income sheet asynchronously
	go func() {
		config, err := postgres.GetConfigByType("income_monthly")
		if err != nil {
			log.Printf("Error getting income config: %v", err)
			return
		}

		// Get updated monthly sum and update sheet
		now := time.Now()
		sum, err := postgres.GetMonthlyIncomeSum(now.Year(), int(now.Month()))
		if err != nil {
			log.Printf("Error getting monthly sum: %v", err)
			return
		}

		cellRange := googleSS.CalculateMonthlyCellRange(config.Sheet, config.A1Range, int(now.Month()))
		if err := googleSS.UpdateSheetCell(cellRange, sum); err != nil {
			log.Printf("Error updating sheet: %v", err)
		}
	}()

	res := map[string]interface{}{
		"income": incomeResult,
		"debt":   debtResult,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

// ========== EXCHANGE RATE (ExchangeRate-API pair endpoint, key server-side only) ==========

func getExchangeRate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		return
	}

	if os.Getenv("EXCHANGERATE_API_KEY") == "" {
		if helpers.IsDevMode() {
			warnExchangeRateKeyDevOnce.Do(func() {
				log.Println("WARNING: EXCHANGERATE_API_KEY not set and GO_ENV is dev - exchange rate endpoint returns 503 (set EXCHANGERATE_API_KEY for production)")
			})
		} else {
			warnExchangeRateKeyNotDevOnce.Do(func() {
				log.Println("WARNING: EXCHANGERATE_API_KEY not set - exchange rate endpoint returns 503 (set EXCHANGERATE_API_KEY or use GO_ENV=dev for local dev)")
			})
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(types.Response{
			Success: false,
			Message: "Exchange rate service not configured (EXCHANGERATE_API_KEY missing)",
		})
		return
	}

	from := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("from")))
	to := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("to")))
	amount := strings.TrimSpace(r.URL.Query().Get("amount"))

	if from == "" || to == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(types.Response{
			Success: false,
			Message: "Query params 'from' and 'to' (ISO 4217 codes, e.g. EUR, GBP) are required",
		})
		return
	}

	cacheKey := from + "|" + to

	// Serve from cache if fresh (12h TTL)
	if body, cachedAt, ok := helpers.ExchangeRateCacheGet(cacheKey); ok {
		lastFetched := cachedAt.Format(time.RFC3339)
		if amount == "" {
			var cached map[string]interface{}
			if err := json.Unmarshal(body, &cached); err == nil {
				cached["last_fetched"] = lastFetched
				out, _ := json.Marshal(cached)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write(out)
				return
			}
		}
		// Cached rate: add conversion_result for requested amount and last_fetched
		var cached map[string]interface{}
		if err := json.Unmarshal(body, &cached); err == nil {
			if rate, _ := cached["conversion_rate"].(float64); rate != 0 {
				if amt, err := strconv.ParseFloat(amount, 64); err == nil {
					cached["conversion_result"] = rate * amt
					cached["last_fetched"] = lastFetched
					out, _ := json.Marshal(cached)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					w.Write(out)
					return
				}
			}
		}
		// Fall through to API on parse/amount error
	}

	url := fmt.Sprintf("%s/%s/pair/%s/%s", exchangerateAPIHost, os.Getenv("EXCHANGERATE_API_KEY"), from, to)
	if amount != "" {
		url = fmt.Sprintf("%s/%s", url, amount)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		log.Printf("ExchangeRate API request failed: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(types.Response{
			Success: false,
			Message: "Failed to reach exchange rate service",
		})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("ExchangeRate API read body: %v", err)
		ServerErrorResponse(w, r)
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		log.Printf("ExchangeRate API decode: %v", err)
		ServerErrorResponse(w, r)
		return
	}

	if result["result"] == "error" {
		code := http.StatusBadRequest
		if errType, _ := result["error-type"].(string); errType == "invalid-key" || errType == "inactive-account" || errType == "quota-reached" {
			code = http.StatusServiceUnavailable
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		json.NewEncoder(w).Encode(result)
		return
	}

	// Add last_fetched so clients know when we hit the API and can verify cache behavior
	now := time.Now()
	result["last_fetched"] = now.Format(time.RFC3339)
	body, _ = json.Marshal(result)

	// Cache successful no-amount response only (12h TTL) to avoid re-hitting the API
	if amount == "" {
		helpers.ExchangeRateCacheSet(cacheKey, body)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(body)
}

func LoadRoutes(muxRouter *mux.Router) {
	api := muxRouter.PathPrefix("/api").Subrouter()
	api.HandleFunc("/", greet).Methods("GET")
	api.HandleFunc("/submit", submitExpenseRow).Methods("POST", "OPTIONS")
	api.HandleFunc("/expenses", getExpenses).Methods("GET", "OPTIONS")
	api.HandleFunc("/budget", setBudgets).Methods("POST", "OPTIONS")
	api.HandleFunc("/budget", getBudgets).Methods("GET")
	api.HandleFunc("/budget/history", getBudgetHistory).Methods("GET")
	api.HandleFunc("/categories", getCategories).Methods("GET")
	api.HandleFunc("/config", getConfig).Methods("GET")
	api.HandleFunc("/config", setConfig).Methods("POST", "OPTIONS")
	api.HandleFunc("/investment", submitInvestment).Methods("POST", "OPTIONS")
	api.HandleFunc("/investments", getInvestments).Methods("GET")
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

	// Phase 5: Investment Account Summary & Dashboard
	api.HandleFunc("/investment-accounts/summary", getInvestmentAccountsSummary).Methods("GET")
	api.HandleFunc("/income/summary", getIncomeSummary).Methods("GET")
	api.HandleFunc("/dashboard", getDashboard).Methods("GET")

	// Phase 6: Transfers
	api.HandleFunc("/transfer", submitTransfer).Methods("POST", "OPTIONS")
	api.HandleFunc("/transfers", getTransfers).Methods("GET")

	// Expected Balance (Phase 1B view)
	api.HandleFunc("/accounts/expected-balance", getExpectedBalances).Methods("GET")
	api.HandleFunc("/investment-accounts/expected-capital", getInvestmentExpectedCapital).Methods("GET")

	// Phase 7: Debt Module Enhancements
	api.HandleFunc("/debts", getDebts).Methods("GET")
	api.HandleFunc("/debts/by-debtor", getDebtsByDebtor).Methods("GET")
	api.HandleFunc("/debt/repayment", submitDebtRepayment).Methods("POST", "OPTIONS")
	api.HandleFunc("/expense-debt", submitExpenseWithDebt).Methods("POST", "OPTIONS")
	api.HandleFunc("/expenses/recent", getRecentExpenses).Methods("GET")

	// Exchange rate (ExchangeRate-API pair; key in EXCHANGERATE_API_KEY, not exposed to FE)
	api.HandleFunc("/exchange-rate", getExchangeRate).Methods("GET", "OPTIONS")
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

// WithMiddleware wraps the router with CORS and API key middleware
func WithMiddleware(handler http.Handler) http.Handler {
	// Apply middleware in order: CORS first, then API key
	// This ensures preflight requests get proper CORS headers before auth check
	return corsMiddleware(apiKeyMiddleware(handler))
}
