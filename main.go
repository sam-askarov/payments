package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
)

var (
	mu       sync.Mutex
	payments = make(map[int64]Payment)
)

type Payment struct {
	ID          int64  `json:"id"`
	Amount      int64  `json:"amount"`
	Description string `json:"description"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func getPayment(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid id"})
		return
	}
	log.Println("id:", id)

	mu.Lock()
	defer mu.Unlock()
	p, ok := payments[id]
	if !ok {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "payment not found"})
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func main() {
	p1 := Payment{ID: 1, Amount: 1000, Description: "transfer"}
	p2 := Payment{ID: 2, Amount: 5000, Description: "transfer"}

	mu.Lock()
	payments[p1.ID] = p1
	mu.Unlock()

	mu.Lock()
	payments[p2.ID] = p2
	mu.Unlock()

	fmt.Println(payments)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /payments/{id}", getPayment)
	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
