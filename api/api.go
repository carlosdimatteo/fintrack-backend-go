package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	googleSS "github.com/carlosdimatteo/fintrack-backend-go/adapters"
	"github.com/gorilla/mux"
)

type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func greet(w http.ResponseWriter, r *http.Request) {
	res := Response{
		Success: true,
		Message: "Fintrack Server up",
	}
	json.NewEncoder(w).Encode(res)
}

func submitRow(w http.ResponseWriter, r *http.Request) {
	// if r.Method != "POST" {
	// 	NotFoundResponse(w, r)
	// 	return
	// }
	/** TODO:
	implement google spreadsheet adapter
	*/
	fmt.Println("submitting row")
	rowtoSubmit := googleSS.FintrackRow{
		Date:           "2021-09-01",
		Category:       "Food",
		Expense:        "Groceries",
		Description:    "Walmart",
		Method:         "Credit Card",
		OriginalAmount: "100.00",
	}
	_, err := googleSS.SubmitRow(rowtoSubmit)
	if err != nil {
		ServerErrorResponse(w, r)
		return
	}
	res := Response{
		Success: true,
		Message: "Row submitted",
	}

	json.NewEncoder(w).Encode(res)
}

func LoadRoutes(muxRouter *mux.Router) {
	api := muxRouter.PathPrefix("/api").Subrouter()
	api.HandleFunc("/", greet).Methods("GET")
	api.HandleFunc("/submit", submitRow)
}

func NotFoundResponse(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	res := Response{
		Success: false,
		Message: "Not Found",
	}
	json.NewEncoder(w).Encode(res)
}

func ServerErrorResponse(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
	res := Response{
		Success: false,
		Message: "Internal Server Error",
	}
	json.NewEncoder(w).Encode(res)
}