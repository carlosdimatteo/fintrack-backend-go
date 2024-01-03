package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	googleSS "github.com/carlosdimatteo/fintrack-backend-go/adapters/google"
	"github.com/carlosdimatteo/fintrack-backend-go/adapters/supabase"
	"github.com/gorilla/mux"
)

type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func greet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	res := Response{
		Success: true,
		Message: "Fintrack Server up",
	}
	json.NewEncoder(w).Encode(res)
}

func submitRow(w http.ResponseWriter, r *http.Request) {
	//Allow CORS here By * or specific origin
	w.Header().Set("Access-Control-Allow-Origin", "*")

	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	/** TODO:
	implement google spreadsheet adapter
	*/
	if r.Method == "OPTIONS" {
		return
	}
	var rowtoSubmit googleSS.FintrackRow
	json.NewDecoder(r.Body).Decode(&rowtoSubmit)
	fmt.Println("submitting row :  description:", rowtoSubmit.Description, " amount:", rowtoSubmit.OriginalAmount)
	rowtoSubmit.Date = time.Now().Format("2006-01-02")
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
	api.HandleFunc("/submit", submitRow).Methods("POST", "OPTIONS")
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
