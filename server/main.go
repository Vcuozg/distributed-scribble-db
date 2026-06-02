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

type WriteRequest struct {
	Collection string                 `json:"collection"`
	Resource   string                 `json:"resource"`
	Data       map[string]interface{} `json:"data"`
}
type ReadResponse struct {
	Data interface{} `json:"data"`
}

func main() {
	var err error

	db, err = scribble.New("./data", nil)
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/write", writeHandler)
	http.HandleFunc("/read", readHandler)
	http.HandleFunc("/delete", deleteHandler)

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

func writeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(Response{
			Message: "Only POST method is allowed",
		})
		return
	}

	var req WriteRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{
			Message: "Invalid JSON request",
		})
		return
	}

	err = db.Write(req.Collection, req.Resource, req.Data)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Response{
			Message: "Failed to write data",
		})
		return
	}

	json.NewEncoder(w).Encode(Response{
		Message: "Data written successfully",
	})
}
func readHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	collection := r.URL.Query().Get("collection")
	resource := r.URL.Query().Get("resource")

	if collection == "" || resource == "" {
		w.WriteHeader(http.StatusBadRequest)

		json.NewEncoder(w).Encode(Response{
			Message: "collection and resource are required",
		})
		return
	}

	var result interface{}

	err := db.Read(collection, resource, &result)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)

		json.NewEncoder(w).Encode(Response{
			Message: "Data not found",
		})
		return
	}

	json.NewEncoder(w).Encode(ReadResponse{
		Data: result,
	})
}
func deleteHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	collection := r.URL.Query().Get("collection")
	resource := r.URL.Query().Get("resource")

	if collection == "" || resource == "" {
		w.WriteHeader(http.StatusBadRequest)

		json.NewEncoder(w).Encode(Response{
			Message: "collection and resource are required",
		})
		return
	}

	err := db.Delete(collection, resource)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)

		json.NewEncoder(w).Encode(Response{
			Message: "Data not found",
		})
		return
	}

	json.NewEncoder(w).Encode(Response{
		Message: "Data deleted successfully",
	})
}