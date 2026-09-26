package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type Response struct {
	Message string `json:"message"`
}

func basicHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	resp := Response{
		Message: "basic response",
	}
	json.NewEncoder(w).Encode(resp)
}

func main() {
	fmt.Print("it runs yya\n")
	router := setupRoutes()
	http.HandleFunc("/api/test", basicHandler)
	port := ":8000"
	fmt.Println("Server is listening on: http://localhost" + port)
	err := http.ListenAndServe(port, router)
	if err != nil {
		log.Fatalf("Server failed to start %v", err)
	}
}
