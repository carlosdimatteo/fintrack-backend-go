package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	googleSS "github.com/carlosdimatteo/fintrack-backend-go/adapters/google"
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
	categories, err := supabase.GetCategories()
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
	expense.Date = time.Now().Format("2006-01-02")
	config, err := supabase.GetConfigByType("expenses")
	if err != nil {
		fmt.Println("error getting config: ", err)
		ServerErrorResponse(w, r)
		return
	}
	_, err = googleSS.SubmitExpenseRow(expense, config)
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
		_, err = supabase.InsertExpenseIntoDatabase(expense)
		if err != nil {
			log.Fatal(err)
			return
		}
	}()
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
	fmt.Println("submitting row :  description:", investment.Description, " amount:", investment.Amount, " account: ", investment.AccountName)
	fmt.Println("amount : ", investment.Amount)
	investment.Date = time.Now().Format("2006-01-02")
	config, err := supabase.GetConfigByType("investments")
	if err != nil {
		fmt.Println("error getting config: ", err)
		ServerErrorResponse(w, r)
		return
	}
	_, err = googleSS.SubmitInvestment(investment, config)
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
		_, err = supabase.InsertInvestmentIntoDatabase(investment)
		if err != nil {
			log.Fatal(err)
			return
		}
	}()
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
	debt.Date = time.Now().Format("2006-01-02")
	config, err := supabase.GetConfigByType("debts")
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
	income.Date = time.Now().Format("2006-01-02")
	config, err := supabase.GetConfigByType("incomes")
	if err != nil {
		fmt.Println("error getting config: ", err)
		ServerErrorResponse(w, r)
		return
	}
	_, err = googleSS.SubmitIncome(income, config)
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
		_, err = supabase.InsertIncomeIntoDatabase(income)
		if err != nil {
			log.Fatal(err)
			return
		}
	}()
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
		accounts, err := supabase.UpdateAccountBalances(accountToInsert.Accounts)
		if err != nil {
			ServerErrorResponse(w, r)
			return
		}
		accountConfig, err := supabase.GetConfigByType(types.ConfigType["accounting_accounts"])

		if err != nil {
			ServerErrorResponse(w, r)
			return
		}

		_, err = googleSS.UpdateAccountBalances(accounts, accountConfig)

		if err != nil {
			ServerErrorResponse(w, r)
			return
		}

		res.Accounts = accounts
	}

	if len(accountToInsert.InvestmentAccounts) > 0 {
		investmentAccounts, err := supabase.UpdateInvestmentAccountBalances(accountToInsert.InvestmentAccounts)
		if err != nil {
			ServerErrorResponse(w, r)
			return
		}
		investmentAccountConfig, err := supabase.GetConfigByType(types.ConfigType["accounting_investment_accounts"])

		if err != nil {
			ServerErrorResponse(w, r)
			return
		}

		_, err = googleSS.UpdateInvestmentAccountBalances(investmentAccounts, investmentAccountConfig)

		if err != nil {
			ServerErrorResponse(w, r)
			return
		}

		res.InvestmentAccounts = investmentAccounts
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)

}

func LoadRoutes(muxRouter *mux.Router) {
	api := muxRouter.PathPrefix("/api").Subrouter()
	api.HandleFunc("/", greet).Methods("GET")
	api.HandleFunc("/submit", submitExpenseRow).Methods("POST", "OPTIONS")
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
	// api.HandleFunc("/income", getIncome).Methods("GET")
	api.HandleFunc("/accounts", getAccounts).Methods("GET")
	api.HandleFunc("/accounts", createAccount).Methods("POST", "OPTIONS")
	api.HandleFunc("/investment-accounts", getInvestmentAccounts).Methods("GET")
	api.HandleFunc("/investment-accounts", createInvestmentAccount).Methods("POST", "OPTIONS")
	api.HandleFunc("/debtors", getDebtors).Methods("GET")
	api.HandleFunc("/debtors", createDebtor).Methods("POST", "OPTIONS")
	api.HandleFunc("/accounting", setAccountingForCurrentMonth).Methods("POST", "OPTIONS")
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
