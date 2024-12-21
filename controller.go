package main

import (
	"fmt"
	"net/http"
)

func handleWithdraw(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20) // 10 MB max memory
	if err != nil {
		http.Error(w, "Error parsing multipart form data", http.StatusBadRequest)
		return
	}

	// Retrieve specific form values
	name := r.FormValue("Name")
	email := r.FormValue("Message")

	// Print the values to the server logs (for debugging)
	fmt.Printf("Name: %s, Email: %s, Message: %s\n", name, email)
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	email := r.URL.Query().Get("email")
	fmt.Printf("Name: %s, Email: %s, Message: %s\n", name, email)
}
