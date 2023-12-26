package main

import (
	"github.com/carlosdimatteo/fintrack-backend-go/api"
	"github.com/gorilla/mux"
)

func main() {

	/**
	TODO:
		1. Load api routes from a separate file
		2. Load database connection (supabase) from a separate file
		3. add google adapter package for handling google spreadsheet api calls

	*/

	muxRouter := mux.NewRouter()

	api.LoadRoutes(muxRouter)

}
