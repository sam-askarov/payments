package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

var (
	mu       sync.Mutex
	payments       = make(map[int64]Payment)
	nextID   int64 = 3
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

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorResponse{Error: msg})
}

func getPayment(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	log.Println("id:", id)

	mu.Lock()
	defer mu.Unlock()
	p, ok := payments[id]
	if !ok {
		writeError(w, http.StatusNotFound, "payment not found")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func listPayments(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	list := make([]Payment, 0, len(payments))
	for _, p := range payments {
		list = append(list, p)
	}
	writeJSON(w, http.StatusOK, list)
}

func createPayment(w http.ResponseWriter, r *http.Request) {
	var p Payment
	err := json.NewDecoder(r.Body).Decode(&p)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	if p.Amount <= 0 {
		writeError(w, http.StatusBadRequest, "amount must be positive")
		return
	}

	if strings.TrimSpace(p.Description) == "" {
		writeError(w, http.StatusBadRequest, "description must not be empty")
		return
	}

	mu.Lock()
	defer mu.Unlock()

	p.ID = nextID
	nextID++
	payments[p.ID] = p
	writeJSON(w, http.StatusCreated, p)
}

func deletePayment(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	mu.Lock()
	defer mu.Unlock()
	_, ok := payments[id]
	if !ok {
		writeError(w, http.StatusNotFound, "payment not found")
		return
	}

	delete(payments, id)
	w.WriteHeader(http.StatusNoContent)
}

func updatePayment(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var p Payment
	err = json.NewDecoder(r.Body).Decode(&p)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	if p.Amount <= 0 {
		writeError(w, http.StatusBadRequest, "amount must be positive")
		return
	}

	if strings.TrimSpace(p.Description) == "" {
		writeError(w, http.StatusBadRequest, "description must not be empty")
		return
	}

	mu.Lock()
	defer mu.Unlock()
	_, ok := payments[id]
	if !ok {
		writeError(w, http.StatusNotFound, "payment not found")
		return
	}
	p.ID = id
	payments[id] = p
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
	mux.HandleFunc("GET /payments", listPayments)
	mux.HandleFunc("POST /payments", createPayment)
	mux.HandleFunc("DELETE /payments/{id}", deletePayment)
	mux.HandleFunc("PUT /payments/{id}", updatePayment)
	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
