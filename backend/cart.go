package main

import (
	"errors"
	"sync"
	"time"
)

// Cart mirrors the shape of UCP's cart object, trimmed to line items only.
type LineItem struct {
	ProductID string `json:"product_id"`
	Title     string `json:"title"`
	Quantity  int    `json:"quantity"`
	UnitPrice Money  `json:"unit_price"`
	UnitUSDC  int64  `json:"unit_usdc"`
}

type Cart struct {
	ID        string     `json:"id"`
	Items     []LineItem `json:"items"`
	CreatedAt time.Time  `json:"created_at"`
}

func (c *Cart) Subtotal() Money {
	if len(c.Items) == 0 {
		return Money{Amount: 0, Currency: "thb"}
	}
	var total int64
	cur := c.Items[0].UnitPrice.Currency
	for _, it := range c.Items {
		total += it.UnitPrice.Amount * int64(it.Quantity)
	}
	return Money{Amount: total, Currency: cur}
}

// SubtotalUSDC is the amount the on-chain settlement contract (lesson 6)
// will actually pull from the payer's wallet.
func (c *Cart) SubtotalUSDC() int64 {
	var total int64
	for _, it := range c.Items {
		total += it.UnitUSDC * int64(it.Quantity)
	}
	return total
}

var ErrNotFound = errors.New("not found")
var ErrOutOfStock = errors.New("out of stock")

type CartStore struct {
	mu    sync.RWMutex
	carts map[string]*Cart
}

func NewCartStore() *CartStore { return &CartStore{carts: map[string]*Cart{}} }

func (s *CartStore) Create() *Cart {
	s.mu.Lock()
	defer s.mu.Unlock()
	cart := &Cart{ID: newID("cart"), Items: []LineItem{}, CreatedAt: time.Now()}
	s.carts[cart.ID] = cart
	return cart
}

func (s *CartStore) Get(id string) (*Cart, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.carts[id]
	return c, ok
}

// AddItem is the single mutation UCP calls "cart update" — the agent and
// the human UI funnel through one clamped operation so they share identical
// business rules.
func (s *CartStore) AddItem(cartID string, catalog *Catalog, productID string, qty int) (*Cart, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cart, ok := s.carts[cartID]
	if !ok {
		return nil, ErrNotFound
	}
	product, ok := catalog.Get(productID)
	if !ok {
		return nil, ErrNotFound
	}
	if qty < 1 {
		qty = 1
	}
	if qty > product.Stock {
		return nil, ErrOutOfStock
	}

	for i, it := range cart.Items {
		if it.ProductID == productID {
			cart.Items[i].Quantity = qty
			return cart, nil
		}
	}
	cart.Items = append(cart.Items, LineItem{
		ProductID: product.ID,
		Title:     product.Title,
		Quantity:  qty,
		UnitPrice: product.Price,
		UnitUSDC:  product.PriceUSDC,
	})
	return cart, nil
}
