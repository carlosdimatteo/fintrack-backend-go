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
	var budgetBycategory types.Expense
	json.NewDecoder(r.Body).Decode(&budgetBycategory)
	fmt.Println("received: ", budgetBycategory)
	fmt.Println("submitting row :  description:", budgetBycategory.Description, " amount:", budgetBycategory.OriginalAmount, " expense: ", budgetBycategory.Expense)
	fmt.Println("expense : ", budgetBycategory.Expense)
	budgetBycategory.Date = time.Now().Format("2006-01-02")
	config, err := supabase.GetConfigByType("expenses")
	if err != nil {
		fmt.Println("error getting config: ", err)
		ServerErrorResponse(w, r)
		return
	}
	_, err = googleSS.SubmitExpenseRow(budgetBycategory, config)
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
		_, err = supabase.InsertExpenseIntoDatabase(budgetBycategory)
		if err != nil {
			log.Fatal(err)
			return
		}
	}()
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
func LoadRoutes(muxRouter *mux.Router) {
	api := muxRouter.PathPrefix("/api").Subrouter()
	api.HandleFunc("/", greet).Methods("GET")
	api.HandleFunc("/submit", submitExpenseRow).Methods("POST", "OPTIONS")
	api.HandleFunc("/budget", setBudgets).Methods("POST", "OPTIONS")
	api.HandleFunc("/budget", getBudgets).Methods("GET")
	api.HandleFunc("/categories", getCategories).Methods("GET")
	api.HandleFunc("/config", getConfig).Methods("GET")
	api.HandleFunc("/config", setConfig).Methods("POST", "OPTIONS")

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
