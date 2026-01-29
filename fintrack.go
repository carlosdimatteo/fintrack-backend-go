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
	// Load .env file if it exists (not required in production)
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("No .env file found, using environment variables")
	}

	muxRouter := mux.NewRouter()
	api.LoadRoutes(muxRouter)
	fmt.Println("API routes loaded")

	// Wrap router with security middleware
	handler := api.WithMiddleware(muxRouter)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
		log.Printf("Defaulting to port %s", port)
	}

	log.Printf("Starting server on port %s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal(err)
	}
}
