package main

import "sync"

// Product is a deliberately small slice of the UCP "Product" object (see
// lesson 1). PriceUSDC is the settlement-layer price used from lesson 6
// onward when payment moves on-chain as USDC (6 decimals) instead of THB.
type Product struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Author      string `json:"author"`
	Description string `json:"description"`
	ImageURL    string `json:"image_url"`
	Price       Money  `json:"price"`
	PriceUSDC   int64  `json:"price_usdc"` // smallest USDC unit (1 USDC = 1_000_000)
	Stock       int    `json:"stock"`
}

type Money struct {
	Amount   int64  `json:"amount"`   // smallest currency unit (satang)
	Currency string `json:"currency"` // ISO 4217, lower-case
}

type Catalog struct {
	mu       sync.RWMutex
	products map[string]*Product
	order    []string
}

func NewCatalog() *Catalog {
	c := &Catalog{products: map[string]*Product{}}
	seed := []*Product{
		{ID: "bk-01", Title: "The Pragmatic Agent", Author: "L. Novak",
			Description: "A field guide to shipping AI agents that actually finish tasks.",
			ImageURL:    "https://picsum.photos/seed/bk-01/480/640",
			Price:       Money{Amount: 45900, Currency: "thb"}, PriceUSDC: 12_900_000, Stock: 12},
		{ID: "bk-02", Title: "Protocols of Trust", Author: "R. Adeyemi",
			Description: "Why open commerce protocols beat bespoke API integrations.",
			ImageURL:    "https://picsum.photos/seed/bk-02/480/640",
			Price:       Money{Amount: 39000, Currency: "thb"}, PriceUSDC: 10_900_000, Stock: 8},
		{ID: "bk-03", Title: "Go Without Frameworks", Author: "S. Iwata",
			Description: "Building production HTTP services with only the standard library.",
			ImageURL:    "https://picsum.photos/seed/bk-03/480/640",
			Price:       Money{Amount: 52000, Currency: "thb"}, PriceUSDC: 14_500_000, Stock: 20},
		{ID: "bk-04", Title: "Settlement Layers", Author: "M. Okonkwo",
			Description: "How money actually moves after the agent clicks \"pay\" — HTTP 402 and on-chain settlement.",
			ImageURL:    "https://picsum.photos/seed/bk-04/480/640",
			Price:       Money{Amount: 61500, Currency: "thb"}, PriceUSDC: 17_200_000, Stock: 5},
		{ID: "bk-05", Title: "Small Models, Sharp Tools", Author: "C. Bergstrom",
			Description: "Getting reliable tool-calling out of fast, cheap LLMs.",
			ImageURL:    "https://picsum.photos/seed/bk-05/480/640",
			Price:       Money{Amount: 35000, Currency: "thb"}, PriceUSDC: 9_800_000, Stock: 15},
		{ID: "bk-06", Title: "Mandates, Not Passwords", Author: "P. Suwannakij",
			Description: "Scoping what an agent is allowed to pay for, cryptographically.",
			ImageURL:    "https://picsum.photos/seed/bk-06/480/640",
			Price:       Money{Amount: 42900, Currency: "thb"}, PriceUSDC: 12_000_000, Stock: 9},
	}
	for _, p := range seed {
		c.products[p.ID] = p
		c.order = append(c.order, p.ID)
	}
	return c
}

func (c *Catalog) List() []*Product {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]*Product, 0, len(c.order))
	for _, id := range c.order {
		out = append(out, c.products[id])
	}
	return out
}

func (c *Catalog) Get(id string) (*Product, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	p, ok := c.products[id]
	return p, ok
}

// Find does a UCP-style free-text search over title/author/description.
func (c *Catalog) Find(query string) []*Product {
	c.mu.RLock()
	defer c.mu.RUnlock()
	q := normalize(query)
	if q == "" {
		out := make([]*Product, 0, len(c.order))
		for _, id := range c.order {
			out = append(out, c.products[id])
		}
		return out
	}
	var out []*Product
	for _, id := range c.order {
		p := c.products[id]
		if contains(normalize(p.Title), q) || contains(normalize(p.Author), q) || contains(normalize(p.Description), q) {
			out = append(out, p)
		}
	}
	return out
}
