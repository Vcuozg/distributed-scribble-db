package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/Vcuozg/distributed-scribble-db/scribble"
)

var db *scribble.Driver

type Response struct {
	Message string `json:"message"`
}

func main() {
	var err error

	db, err = scribble.New("./data", nil)
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", homeHandler)

	log.Println("Server running on :8080")

	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(Response{
		Message: "Distributed Scribble API is running",
	})
}