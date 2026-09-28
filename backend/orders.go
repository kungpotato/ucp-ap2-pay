package main

import (
	"sync"
	"time"
)

type OrderStatus string

const (
	OrderPendingMandate OrderStatus = "pending_mandate" // waiting for AP2 payment mandate (lesson 4)
	OrderPendingPayment OrderStatus = "pending_payment" // 402 issued, waiting for on-chain payment (lesson 5)
	OrderPaid           OrderStatus = "paid"            // on-chain tx confirmed (lesson 6)
	OrderFailed         OrderStatus = "failed"
)

// Order is UCP's "post-purchase handoff" object: what the buyer (human or
// agent) polls after checkout to know whether settlement finished. From
// lesson 4 onward it also carries the AP2 mandate chain that authorized it,
// and from lesson 6 onward the on-chain receipt that proves it was paid.
type Order struct {
	ID        string      `json:"id"`
	CartID    string      `json:"cart_id"`
	Status    OrderStatus `json:"status"`
	Total     Money       `json:"total"`
	TotalUSDC int64       `json:"total_usdc"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`

	// AP2 mandate chain (lesson 4), all optional until that lesson.
	IntentMandateID  string `json:"intent_mandate_id,omitempty"`
	CartMandateID    string `json:"cart_mandate_id,omitempty"`
	PaymentMandateID string `json:"payment_mandate_id,omitempty"`

	// X402 + on-chain settlement (lessons 5-6).
	PayTo  string `json:"pay_to,omitempty"`  // merchant wallet address
	TxHash string `json:"tx_hash,omitempty"` // settlement tx on the anvil fork
}

type OrderStore struct {
	mu     sync.RWMutex
	orders map[string]*Order
}

func NewOrderStore() *OrderStore {
	return &OrderStore{orders: map[string]*Order{}}
}

func (s *OrderStore) Create(cartID string, total Money, totalUSDC int64) *Order {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	o := &Order{
		ID:        newID("order"),
		CartID:    cartID,
		Status:    OrderPendingMandate,
		Total:     total,
		TotalUSDC: totalUSDC,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.orders[o.ID] = o
	return o
}

func (s *OrderStore) Get(id string) (*Order, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	o, ok := s.orders[id]
	return o, ok
}

func (s *OrderStore) Update(id string, mutate func(*Order)) (*Order, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.orders[id]
	if !ok {
		return nil, false
	}
	mutate(o)
	o.UpdatedAt = time.Now()
	return o, true
}
