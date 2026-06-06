package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Vcuozg/distributed-scribble-db/scribble"
)

var db *scribble.Driver
var (
	nodeRole    string
	serverPort  string
	replicaURLs []string
)

// ============================================================
// Tính năng 3: Pending Queue
// Lưu lại các request chưa replicate được khi replica bị down
// ============================================================
type PendingEntry struct {
	Req         WriteRequest
	ReplicaURL  string
	FailedAt    time.Time
}

var (
	pendingQueue []PendingEntry
	pendingMu    sync.Mutex
)

// ============================================================
// Tính năng 4: Health Check tự động
// Master theo dõi trạng thái sống/chết của từng replica
// ============================================================
type ReplicaStatus struct {
	URL     string
	Alive   bool
	LastCheck time.Time
}

var (
	replicaStatusMap = make(map[string]*ReplicaStatus)
	replicaStatusMu  sync.RWMutex
)

// ============================================================
// Structs dùng chung
// ============================================================
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

// ============================================================
// main
// ============================================================
func main() {
	nodeRole   = os.Getenv("NODE_ROLE")
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

	// Khởi tạo trạng thái ban đầu cho từng replica
	for _, url := range replicaURLs {
		replicaStatusMap[url] = &ReplicaStatus{
			URL:   url,
			Alive: true,
		}
	}

	// Tính năng 4: Goroutine tự động health check mỗi 10 giây
	if nodeRole == "master" {
		go runHealthChecker()
		// Tính năng 3: Goroutine tự động retry pending queue mỗi 15 giây
		go runPendingRetry()
	}

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/status", statusHandler)
	http.HandleFunc("/write", writeHandler)
	http.HandleFunc("/replicate", replicateHandler)
	http.HandleFunc("/read", readHandler)
	http.HandleFunc("/delete", deleteHandler)
	// Tính năng 3: endpoint xem pending queue (debug)
	http.HandleFunc("/pending", pendingHandler)
	// Tính năng 4: endpoint xem trạng thái các replica
	http.HandleFunc("/replicas", replicasStatusHandler)

	log.Printf("[%s] Node running on :%s\n", nodeRole, serverPort)
	if err = http.ListenAndServe(":"+serverPort, nil); err != nil {
		log.Fatal(err)
	}
}

// ============================================================
// Tính năng 1: REST API handlers (Write / Read / Delete)
// ============================================================
func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{Message: "Distributed Scribble API is running"})
}

func writeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(Response{Message: "Only POST method is allowed"})
		return
	}

	var req WriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{Message: "Invalid JSON request"})
		return
	}

	if err := db.Write(req.Collection, req.Resource, req.Data); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Response{Message: "Failed to write data"})
		return
	}

	// Tính năng 2: Replication sang tất cả replica (với retry)
	if nodeRole == "master" && len(replicaURLs) > 0 {
		for _, replicaURL := range replicaURLs {
			// Tính năng 1 (retry): thử tối đa 3 lần
			err := replicateWithRetry(req, replicaURL, 3)
			if err != nil {
				log.Printf("[master] Replication failed to %s after retries: %v", replicaURL, err)
				// Tính năng 3: đưa vào pending queue để retry sau
				enqueuePending(req, replicaURL)
			} else {
				log.Printf("[master] Replicated successfully to %s", replicaURL)
			}
		}
	}

	json.NewEncoder(w).Encode(Response{Message: "Data written successfully"})
}

func readHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	collection := r.URL.Query().Get("collection")
	resource   := r.URL.Query().Get("resource")

	if collection == "" || resource == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{Message: "collection and resource are required"})
		return
	}

	// Tính năng 2: Nếu là replica node → đọc từ local DB của chính nó
	// Tính năng 2: Nếu client truyền ?prefer_replica=true → master chuyển hướng sang replica còn sống
	if nodeRole == "master" {
		preferReplica := r.URL.Query().Get("prefer_replica")
		if preferReplica == "true" {
			aliveReplica := getAliveReplica()
			if aliveReplica != "" {
				// Chuyển hướng request sang replica
				targetURL := aliveReplica + "/read?collection=" + collection + "&resource=" + resource
				proxyRead(w, targetURL)
				return
			}
			// Nếu không có replica sống → fallback đọc tại master
			log.Println("[master] No alive replica, fallback to master read")
		}
	}

	var result interface{}
	if err := db.Read(collection, resource, &result); err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(Response{Message: "Data not found"})
		return
	}

	json.NewEncoder(w).Encode(ReadResponse{Data: result})
}

func deleteHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	collection := r.URL.Query().Get("collection")
	resource   := r.URL.Query().Get("resource")

	if collection == "" || resource == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{Message: "collection and resource are required"})
		return
	}

	if err := db.Delete(collection, resource); err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(Response{Message: "Data not found"})
		return
	}

	json.NewEncoder(w).Encode(Response{Message: "Data deleted successfully"})
}

func replicateHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(Response{Message: "Only POST method is allowed"})
		return
	}

	var req WriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{Message: "Invalid replication request"})
		return
	}

	if err := db.Write(req.Collection, req.Resource, req.Data); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Response{Message: "Failed to replicate data"})
		return
	}

	log.Printf("[replica] Received: collection=%s resource=%s\n", req.Collection, req.Resource)
	json.NewEncoder(w).Encode(Response{Message: "Data replicated successfully"})
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "healthy",
		"role":   nodeRole,
		"port":   serverPort,
	})
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"node_role":     nodeRole,
		"server_port":   serverPort,
		"replica_count": len(replicaURLs),
		"replicas":      replicaURLs,
	})
}

// ============================================================
// Tính năng 1 (nâng cấp): Retry Logic
// Thử gửi lại tối đa maxRetry lần, delay tăng dần (backoff)
// ============================================================
func replicateWithRetry(req WriteRequest, replicaURL string, maxRetry int) error {
	var lastErr error
	for attempt := 1; attempt <= maxRetry; attempt++ {
		lastErr = replicateToReplica(req, replicaURL)
		if lastErr == nil {
			return nil
		}
		log.Printf("[master] Retry %d/%d to %s failed: %v", attempt, maxRetry, replicaURL, lastErr)
		// Backoff: 1s, 2s, 4s
		time.Sleep(time.Duration(attempt) * time.Second)
	}
	return lastErr
}

func replicateToReplica(req WriteRequest, replicaURL string) error {
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Post(
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

// ============================================================
// Tính năng 2 (nâng cấp): Read từ Replica
// Proxy request đọc sang replica còn sống
// ============================================================
func proxyRead(w http.ResponseWriter, targetURL string) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(targetURL)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(Response{Message: "Replica unreachable"})
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Served-By", "replica")
	w.WriteHeader(resp.StatusCode)

	var result interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	json.NewEncoder(w).Encode(result)
}

func getAliveReplica() string {
	replicaStatusMu.RLock()
	defer replicaStatusMu.RUnlock()
	for url, status := range replicaStatusMap {
		if status.Alive {
			return url
		}
	}
	return ""
}

// ============================================================
// Tính năng 3: Pending Queue
// Lưu request chưa replicate được, tự động retry khi replica sống lại
// ============================================================
func enqueuePending(req WriteRequest, replicaURL string) {
	pendingMu.Lock()
	defer pendingMu.Unlock()
	pendingQueue = append(pendingQueue, PendingEntry{
		Req:        req,
		ReplicaURL: replicaURL,
		FailedAt:   time.Now(),
	})
	log.Printf("[pending] Queued: collection=%s resource=%s → %s", req.Collection, req.Resource, replicaURL)
}

func runPendingRetry() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		retryPending()
	}
}

func retryPending() {
	pendingMu.Lock()
	if len(pendingQueue) == 0 {
		pendingMu.Unlock()
		return
	}
	// Lấy toàn bộ queue hiện tại, reset queue
	current := pendingQueue
	pendingQueue = []PendingEntry{}
	pendingMu.Unlock()

	log.Printf("[pending] Retrying %d pending entries...", len(current))

	var failed []PendingEntry
	for _, entry := range current {
		err := replicateToReplica(entry.Req, entry.ReplicaURL)
		if err != nil {
			log.Printf("[pending] Still failed: %s → %s", entry.Req.Resource, entry.ReplicaURL)
			failed = append(failed, entry)
		} else {
			log.Printf("[pending] Recovered: %s → %s", entry.Req.Resource, entry.ReplicaURL)
		}
	}

	// Đưa lại những cái vẫn thất bại
	if len(failed) > 0 {
		pendingMu.Lock()
		pendingQueue = append(failed, pendingQueue...)
		pendingMu.Unlock()
	}
}

func pendingHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	pendingMu.Lock()
	defer pendingMu.Unlock()

	type PendingInfo struct {
		Collection string    `json:"collection"`
		Resource   string    `json:"resource"`
		ReplicaURL string    `json:"replica_url"`
		FailedAt   time.Time `json:"failed_at"`
	}
	var infos []PendingInfo
	for _, e := range pendingQueue {
		infos = append(infos, PendingInfo{
			Collection: e.Req.Collection,
			Resource:   e.Req.Resource,
			ReplicaURL: e.ReplicaURL,
			FailedAt:   e.FailedAt,
		})
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pending_count": len(infos),
		"entries":       infos,
	})
}

// ============================================================
// Tính năng 4: Auto Health Check
// Mỗi 10 giây master ping /health của từng replica
// ============================================================
func runHealthChecker() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		checkAllReplicas()
	}
}

func checkAllReplicas() {
	for _, url := range replicaURLs {
		alive := pingReplica(url)

		replicaStatusMu.Lock()
		status := replicaStatusMap[url]
		wasAlive := status.Alive
		status.Alive = alive
		status.LastCheck = time.Now()
		replicaStatusMu.Unlock()

		if wasAlive && !alive {
			log.Printf("[health] Replica DOWN: %s", url)
		} else if !wasAlive && alive {
			log.Printf("[health] Replica RECOVERED: %s", url)
			// Khi replica sống lại → ngay lập tức retry pending
			go retryPending()
		}
	}
}

func pingReplica(replicaURL string) bool {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(replicaURL + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func replicasStatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	replicaStatusMu.RLock()
	defer replicaStatusMu.RUnlock()

	type ReplicaInfo struct {
		URL       string    `json:"url"`
		Alive     bool      `json:"alive"`
		LastCheck time.Time `json:"last_check"`
	}
	var infos []ReplicaInfo
	for _, status := range replicaStatusMap {
		infos = append(infos, ReplicaInfo{
			URL:       status.URL,
			Alive:     status.Alive,
			LastCheck: status.LastCheck,
		})
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"replicas": infos,
	})
}