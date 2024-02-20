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
	_, err := googleSS.SubmitExpenseRow(budgetBycategory)
	if err != nil {
		ServerErrorResponse(w, r)
		return
	}

	res := types.Response{
		Success: true,
		Message: "Row submitted",
	}

	json.NewEncoder(w).Encode(res)

	go func() {
		_, err = supabase.InsertIntoDatabase(budgetBycategory)
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
	var rowtoSubmit types.Expense
	json.NewDecoder(r.Body).Decode(&rowtoSubmit)
	fmt.Println("received: ", rowtoSubmit)
	fmt.Println("submitting row :  description:", rowtoSubmit.Description, " amount:", rowtoSubmit.OriginalAmount, " expense: ", rowtoSubmit.Expense)
	fmt.Println("expense : ", rowtoSubmit.Expense)
	rowtoSubmit.Date = time.Now().Format("2006-01-02")
	_, err := googleSS.SubmitExpenseRow(rowtoSubmit)
	if err != nil {
		ServerErrorResponse(w, r)
		return
	}

	res := types.Response{
		Success: true,
		Message: "Row submitted",
	}

	json.NewEncoder(w).Encode(res)

	go func() {
		_, err = supabase.InsertIntoDatabase(rowtoSubmit)
		if err != nil {
			log.Fatal(err)
			return
		}
	}()

}

func LoadRoutes(muxRouter *mux.Router) {
	api := muxRouter.PathPrefix("/api").Subrouter()
	api.HandleFunc("/", greet).Methods("GET")
	api.HandleFunc("/submit", submitExpenseRow).Methods("POST", "OPTIONS")
	api.HandleFunc("/budget", setBudgets).Methods("POST", "OPTIONS")
	api.HandleFunc("/budget", getBudgets).Methods("GET")
	api.HandleFunc("/categories", getCategories).Methods("GET")

}

func NotFoundResponse(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	res := types.Response{
		Success: false,
		Message: "Not Found",
	}
	json.NewEncoder(w).Encode(res)
}

func ServerErrorResponse(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
	res := types.Response{
		Success: false,
		Message: "Internal Server Error",
	}
	json.NewEncoder(w).Encode(res)
}
