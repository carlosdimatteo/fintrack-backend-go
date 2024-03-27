package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/carlosdimatteo/fintrack-backend-go/api"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {

	err := godotenv.Load(".env")

	if err != nil {
		log.Fatalf("Error loading .env file")
	}
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
