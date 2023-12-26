package main

import (
	"fmt"
	"net/http"

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
	http.ListenAndServe(":3001", muxRouter)

}
