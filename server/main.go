package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/Vcuozg/distributed-scribble-db/scribble"
)

var db *scribble.Driver
var (
	nodeRole  string
	serverPort string
	replicaURLs []string
)

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

		nodeRole = os.Getenv("NODE_ROLE")
	serverPort = os.Getenv("PORT")
	replicaEnv := os.Getenv("REPLICA_URLS")
if replicaEnv != "" {
	replicaURLs = strings.Split(replicaEnv, ",")
}

	if nodeRole == "" {
		nodeRole = "master"
	}

	if serverPort == "" {
		serverPort = "8080"
	}
	var err error

	db, err = scribble.New("./data", nil)
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/write", writeHandler)
	http.HandleFunc("/replicate", replicateHandler)
	http.HandleFunc("/read", readHandler)
	http.HandleFunc("/delete", deleteHandler)

	log.Printf("Node: %s running on :%s\n", nodeRole, serverPort)

	err = http.ListenAndServe(":"+serverPort, nil)
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

	if nodeRole == "master" && len(replicaURLs) > 0 {
	for _, replicaURL := range replicaURLs {
		err = replicateToReplica(req, replicaURL)
		if err != nil {
			log.Println("Replication failed to", replicaURL, ":", err)
			continue
		}

		log.Println("Data replicated successfully to", replicaURL)
	}
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
func replicateHandler(w http.ResponseWriter, r *http.Request) {
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
			Message: "Invalid replication request",
		})
		return
	}

	err = db.Write(req.Collection, req.Resource, req.Data)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Response{
			Message: "Failed to replicate data",
		})
		return
	}

	log.Printf("Replica received data: collection=%s resource=%s\n", req.Collection, req.Resource)

	json.NewEncoder(w).Encode(Response{
		Message: "Data replicated successfully",
	})
}
func replicateToReplica(req WriteRequest, replicaURL string) error {
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	resp, err := http.Post(
		replicaURL+"/replicate",
		"application/json",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return http.ErrHandlerTimeout
	}

	return nil
}
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "healthy",
		"role":   nodeRole,
		"port":   serverPort,
	})
}