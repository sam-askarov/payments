package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
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

type PaymentHandler struct {
	store *PaymentStore
}

func NewPaymentHandler(store *PaymentStore) *PaymentHandler {
	return &PaymentHandler{store: store}
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

func (h *PaymentHandler) getPayment(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	p, err := h.store.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, ErrPaymentNotFound):
			writeError(w, http.StatusNotFound, "payment not found")
		default:
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *PaymentHandler) listPayments(w http.ResponseWriter, r *http.Request) {
	list := h.store.List()
	writeJSON(w, http.StatusOK, list)
}

func (h *PaymentHandler) createPayment(w http.ResponseWriter, r *http.Request) {
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

	p = h.store.Create(p)
	writeJSON(w, http.StatusCreated, p)
}

func (h *PaymentHandler) deletePayment(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	err = h.store.Delete(id)
	if err != nil {
		switch {
		case errors.Is(err, ErrPaymentNotFound):
			writeError(w, http.StatusNotFound, "payment not found")
		default:
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *PaymentHandler) updatePayment(w http.ResponseWriter, r *http.Request) {
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

	p, err = h.store.Update(id, p)
	if err != nil {
		switch {
		case errors.Is(err, ErrPaymentNotFound):
			writeError(w, http.StatusNotFound, "payment not found")
		default:
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func main() {
	store := NewPaymentStore()
	store.Create(Payment{Amount: 1000, Description: "transfer"})
	store.Create(Payment{Amount: 5000, Description: "transfer"})

	h := NewPaymentHandler(store)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /payments/{id}", h.getPayment)
	mux.HandleFunc("GET /payments", h.listPayments)
	mux.HandleFunc("POST /payments", h.createPayment)
	mux.HandleFunc("DELETE /payments/{id}", h.deletePayment)
	mux.HandleFunc("PUT /payments/{id}", h.updatePayment)
	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
