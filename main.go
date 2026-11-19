package main

import (
	"encoding/json"
	"errors"
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
var ErrPaymentNotFound = errors.New("payment not found")

type Payment struct {
	ID          int64  `json:"id"`
	Amount      int64  `json:"amount"`
	Description string `json:"description"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type PaymentStore struct {
	mu       sync.Mutex
	payments map[int64]Payment
	nextID   int64
}

func NewPaymentStore() *PaymentStore {
	return &PaymentStore{payments: make(map[int64]Payment), nextID: 1}
}

func (s *PaymentStore) Create(p Payment) Payment {
	s.mu.Lock()
	defer s.mu.Unlock()
	p.ID = s.nextID
	s.nextID++
	s.payments[p.ID] = p
	return p
}

func (s *PaymentStore) Get(id int64) (Payment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.payments[id]
	if !ok {
		return Payment{}, ErrPaymentNotFound
	}
	return p, nil
}

func (s *PaymentStore) List() []Payment {
	s.mu.Lock()
	defer s.mu.Unlock()
	list := make([]Payment, 0, len(s.payments))
	for _, p := range s.payments {
		list = append(list, p)
	}
	return list
}

func (s *PaymentStore) Update(id int64, p Payment) (Payment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.payments[id]
	if !ok {
		return Payment{}, ErrPaymentNotFound
	}
	p.ID = id
	s.payments[id] = p
	return p, nil
}

func (s *PaymentStore) Delete(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.payments[id]
	if !ok {
		return ErrPaymentNotFound
	}
	delete(s.payments, id)
	return nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorResponse{Error: msg})
}

func parseID(r *http.Request) (int64, error) {
	idStr := r.PathValue("id")
	return strconv.ParseInt(idStr, 10, 64)
}

func validatePayment(p Payment) error {
	if p.Amount <= 0 {
		return errors.New("amount must be positive")
	}

	if strings.TrimSpace(p.Description) == "" {
		return errors.New("description must not be empty")
	}
	return nil
}

func getPayment(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

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

	if err := validatePayment(p); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
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
	id, err := parseID(r)
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
	id, err := parseID(r)
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

	if err := validatePayment(p); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
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
