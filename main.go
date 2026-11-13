package main

import (
	"fmt"
	"sync"
)

type Payment struct {
	ID          int64  `json:"id"`
	Amount      int64  `json:"amount"`
	Description string `json:"description"`
}

var (
	mu       sync.Mutex
	payments = make(map[int64]Payment)
)

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
}
