package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/carlosdimatteo/fintrack-backend-go/api"
	"github.com/gorilla/mux"
)

func main() {

	/**
	TODO:
		. Load database connection (supabase) from a separate file
		. add google adapter package for handling google spreadsheet api calls

	*/

	muxRouter := mux.NewRouter()
	api.LoadRoutes(muxRouter)
	fmt.Println("API routes loaded")
	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
		log.Printf("Defaulting to port %s", port)
	}
	if err := http.ListenAndServe(":"+port, muxRouter); err != nil {
		log.Fatal(err)
	}

}
