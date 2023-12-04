package main

import (
	"log"
	"net/http"
)

func main() {
	setupAPI()

	// Serve on port :8080, fudge yeah hardcoded port
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// setupAPI will start all Routes and their Handlers
func setupAPI() {
	// Serve the ./frontend directory at Route /
	http.Handle("/", http.FileServer(http.Dir("./frontend")))
}