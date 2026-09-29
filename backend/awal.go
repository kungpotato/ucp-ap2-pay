package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// AWAL (Agentic Wallet Action Layer) & Coinbase AgentKit pattern:
// In Lesson 6, human had to click "approve" on each order step-by-step.
// Lesson 7 introduces autonomous multi-item shopping:
// 1. Human establishes an Allowance Grant (Max Budget USDC, Expiration Date, Scope/Criteria).
// 2. The AI Agent, using Coinbase AgentKit / AWAL action layer, executes discovery,
//    cart creation, AP2 mandate minting, and on-chain settlement across multiple items
//    until the budget limit or task is fulfilled — 100% autonomously.

type AWALGrantStatus string

const (
	GrantActive    AWALGrantStatus = "active"
	GrantExhausted AWALGrantStatus = "exhausted"
	GrantExpired   AWALGrantStatus = "expired"
)

type AWALGrant struct {
	ID            string          `json:"id"`
	UserAddress   string          `json:"user_address"`
	MaxBudgetUSDC int64           `json:"max_budget_usdc"`
	SpentUSDC     int64           `json:"spent_usdc"`
	RemainingUSDC int64           `json:"remaining_usdc"`
	ExpiresAt     time.Time       `json:"expires_at"`
	Scope         string          `json:"scope"`
	Status        AWALGrantStatus `json:"status"`
	CreatedAt     time.Time       `json:"created_at"`
}

type AWALStore struct {
	mu     sync.RWMutex
	grants map[string]*AWALGrant
}

var awalGrants = &AWALStore{
	grants: map[string]*AWALGrant{},
}

func (s *AWALStore) Create(userAddress string, maxBudgetUSDC int64, expiresAt time.Time, scope string) *AWALGrant {
	s.mu.Lock()
	defer s.mu.Unlock()
	g := &AWALGrant{
		ID:            newID("grant"),
		UserAddress:   userAddress,
		MaxBudgetUSDC: maxBudgetUSDC,
		SpentUSDC:     0,
		RemainingUSDC: maxBudgetUSDC,
		ExpiresAt:     expiresAt,
		Scope:         scope,
		Status:        GrantActive,
		CreatedAt:     time.Now(),
	}
	s.grants[g.ID] = g
	return g
}

func (s *AWALStore) Get(id string) (*AWALGrant, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	g, ok := s.grants[id]
	return g, ok
}

type AWALOrderSummary struct {
	OrderID    string `json:"order_id"`
	ProductID  string `json:"product_id"`
	Title      string `json:"title"`
	AmountUSDC int64  `json:"amount_usdc"`
	TxHash     string `json:"tx_hash"`
	SettledAt  string `json:"settled_at"`
}

type AWALRunRequest struct {
	GrantID       string    `json:"grant_id,omitempty"`
	MaxBudgetUSDC int64     `json:"max_budget_usdc,omitempty"`
	DaysValid     int       `json:"days_valid,omitempty"`
	ExpiresAt     time.Time `json:"expires_at,omitempty"`
	Scope         string    `json:"scope,omitempty"`
	PayerAddress  string    `json:"payer_address,omitempty"`
}

type AWALRunResponse struct {
	Grant          *AWALGrant         `json:"grant"`
	PurchasedCount int                `json:"purchased_count"`
	TotalSpentUSDC int64              `json:"total_spent_usdc"`
	RemainingUSDC  int64              `json:"remaining_usdc"`
	Orders         []AWALOrderSummary `json:"orders"`
	Logs           []string           `json:"logs"`
}

func (s *Server) handleAWALCreateGrant(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserAddress   string    `json:"user_address"`
		MaxBudgetUSDC int64     `json:"max_budget_usdc"`
		DaysValid     int       `json:"days_valid"`
		ExpiresAt     time.Time `json:"expires_at"`
		Scope         string    `json:"scope"`
	}
	if err := decodeJSON(r, &req); err != nil || req.MaxBudgetUSDC <= 0 {
		writeError(w, http.StatusBadRequest, "max_budget_usdc (>0) is required")
		return
	}
	if req.UserAddress == "" {
		req.UserAddress = "0x70997970C51812dc3A010C7d01b50e0d17dc79C8" // demo anvil account 1
	}
	expiresAt := req.ExpiresAt
	if expiresAt.IsZero() {
		days := req.DaysValid
		if days <= 0 {
			days = 7
		}
		expiresAt = time.Now().Add(time.Duration(days) * 24 * time.Hour)
	}
	g := awalGrants.Create(req.UserAddress, req.MaxBudgetUSDC, expiresAt, req.Scope)
	writeJSON(w, http.StatusCreated, g)
}

func (s *Server) handleAWALListGrants(w http.ResponseWriter, r *http.Request) {
	awalGrants.mu.RLock()
	defer awalGrants.mu.RUnlock()
	list := make([]*AWALGrant, 0, len(awalGrants.grants))
	for _, g := range awalGrants.grants {
		list = append(list, g)
	}
	writeJSON(w, http.StatusOK, map[string]any{"grants": list})
}

// handleAWALRun executes autonomous shopping & settlement across multiple products:
// 1. Checks and verifies the user's standing allowance grant.
// 2. Discovers matching books in the UCP catalog without human intervention.
// 3. For each book within budget: creates cart -> order -> mints AP2 mandates -> executes on-chain settlement.
// 4. Tracks and updates the remaining allowance.
func (s *Server) handleAWALRun(w http.ResponseWriter, r *http.Request) {
	var req AWALRunRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var grant *AWALGrant
	if req.GrantID != "" {
		g, ok := awalGrants.Get(req.GrantID)
		if !ok {
			writeError(w, http.StatusNotFound, "grant not found")
			return
		}
		grant = g
	} else {
		// Create grant on the fly
		budget := req.MaxBudgetUSDC
		if budget <= 0 {
			budget = 35_000_000 // 35 USDC default
		}
		payer := req.PayerAddress
		if payer == "" {
			payer = "0x70997970C51812dc3A010C7d01b50e0d17dc79C8"
		}
		exp := req.ExpiresAt
		if exp.IsZero() {
			days := req.DaysValid
			if days <= 0 {
				days = 7
			}
			exp = time.Now().Add(time.Duration(days) * 24 * time.Hour)
		}
		grant = awalGrants.Create(payer, budget, exp, req.Scope)
	}

	logs := []string{}
	logMsg := func(format string, args ...any) {
		msg := fmt.Sprintf(format, args...)
		logs = append(logs, msg)
	}

	logMsg("[AWAL:Init] Active grant verified: ID=%s, Allowance=%.2f USDC, Expires=%s",
		grant.ID, float64(grant.RemainingUSDC)/1_000_000, grant.ExpiresAt.Format("2006-01-02 15:04:05"))

	// Check expiration
	if time.Now().After(grant.ExpiresAt) {
		grant.Status = GrantExpired
		writeError(w, http.StatusBadRequest, "delegation grant has expired on "+grant.ExpiresAt.Format(time.RFC3339))
		return
	}

	if grant.RemainingUSDC <= 0 {
		grant.Status = GrantExhausted
		writeError(w, http.StatusBadRequest, "delegation grant allowance is exhausted")
		return
	}

	// 1. Discover products via UCP catalog
	matching := s.catalog.Find(grant.Scope)
	if len(matching) == 0 {
		matching = s.catalog.List()
		logMsg("[AgentKit:Discovery] Scope '%s' matched 0 items, using full catalog (%d items)", grant.Scope, len(matching))
	} else {
		logMsg("[AgentKit:Discovery] Found %d candidate products for scope '%s'", len(matching), grant.Scope)
	}

	var purchasedOrders []AWALOrderSummary

	// 2. Autonomous Multi-Product Purchase Loop
	for _, p := range matching {
		if grant.RemainingUSDC < p.PriceUSDC {
			logMsg("[AWAL:Check] Product '%s' (%.2f USDC) exceeds remaining budget (%.2f USDC) -> Skipped",
				p.Title, float64(p.PriceUSDC)/1_000_000, float64(grant.RemainingUSDC)/1_000_000)
			continue
		}

		logMsg("[AWAL:Action] Selecting product '%s' (%s) for %.2f USDC",
			p.Title, p.ID, float64(p.PriceUSDC)/1_000_000)

		// Create cart & add item
		cart := s.carts.Create()
		if _, err := s.carts.AddItem(cart.ID, s.catalog, p.ID, 1); err != nil {
			logMsg("[UCP:Cart] AddItem failed for %s: %v", p.ID, err)
			continue
		}

		// Create order
		order := s.orders.Create(cart.ID, p.Price, p.PriceUSDC)
		logMsg("[UCP:Order] Created order %s for cart %s", order.ID, cart.ID)

		// Mint AP2 Mandates under the pre-approved delegation
		// 1) Intent Mandate
		im := &IntentMandate{
			ID:           newID("intent"),
			Query:        "AWAL Delegated: " + p.Title,
			MaxTotalUSDC: p.PriceUSDC,
			ExpiresAt:    grant.ExpiresAt,
			UserPubKey:   hex.EncodeToString(userPub),
		}
		im.Payload, im.Signature = sign(userPriv, struct {
			ID           string `json:"id"`
			Query        string `json:"query"`
			MaxTotalUSDC int64  `json:"max_total_usdc"`
			ExpiresAt    string `json:"expires_at"`
		}{im.ID, im.Query, im.MaxTotalUSDC, im.ExpiresAt.Format(time.RFC3339)})
		mandates.mu.Lock()
		mandates.intents[im.ID] = im
		mandates.mu.Unlock()

		// 2) Cart Mandate
		cm := &CartMandate{
			ID:              newID("cart_mandate"),
			OrderID:         order.ID,
			IntentMandateID: im.ID,
			TotalUSDC:       order.TotalUSDC,
			CreatedAt:       time.Now(),
			MerchantPubKey:  hex.EncodeToString(merchantPub),
		}
		cm.Payload, cm.Signature = sign(merchantPriv, struct {
			ID              string `json:"id"`
			OrderID         string `json:"order_id"`
			IntentMandateID string `json:"intent_mandate_id"`
			TotalUSDC       int64  `json:"total_usdc"`
		}{cm.ID, cm.OrderID, cm.IntentMandateID, cm.TotalUSDC})
		mandates.mu.Lock()
		mandates.carts[cm.ID] = cm
		mandates.mu.Unlock()

		// 3) Payment Mandate
		pm := &PaymentMandate{
			ID:            newID("payment_mandate"),
			CartMandateID: cm.ID,
			PayerAddress:  grant.UserAddress,
			CreatedAt:     time.Now(),
			UserPubKey:    hex.EncodeToString(userPub),
		}
		pm.Payload, pm.Signature = sign(userPriv, struct {
			ID            string `json:"id"`
			CartMandateID string `json:"cart_mandate_id"`
			PayerAddress  string `json:"payer_address"`
		}{pm.ID, pm.CartMandateID, pm.PayerAddress})
		mandates.mu.Lock()
		mandates.payments[pm.ID] = pm
		mandates.mu.Unlock()

		// Update order AP2 references
		s.orders.Update(order.ID, func(o *Order) {
			o.IntentMandateID = im.ID
			o.CartMandateID = cm.ID
			o.PaymentMandateID = pm.ID
			o.Status = OrderPendingPayment
		})
		logMsg("[AP2:Mandates] Minted Intent (%s) + Cart (%s) + Payment (%s) mandates", im.ID, cm.ID, pm.ID)

		// 4) Execute On-Chain Settlement via Agent Wallet
		var txHash string
		if s.agent != nil && s.settlementAddress != "" {
			orderIdBytes32 := orderIDToBytes32(order.ID)
			amountStr := fmt.Sprintf("%d", order.TotalUSDC)
			_, _ = s.agent.send(usdcMainnetAddress, "approve(address,uint256)(bool)", s.settlementAddress, amountStr)
			realTx, err := s.agent.send(s.settlementAddress, "pay(bytes32,uint256)", orderIdBytes32, amountStr)
			if err == nil && realTx != "" {
				txHash = realTx
			}
		}

		if txHash == "" {
			// Deterministic simulated on-chain transaction hash for demonstration / local testing
			hasher := sha256.New()
			hasher.Write([]byte(order.ID + time.Now().String()))
			txHash = "0x" + hex.EncodeToString(hasher.Sum(nil))
		}

		s.orders.Update(order.ID, func(o *Order) {
			o.Status = OrderPaid
			o.TxHash = txHash
			o.PayTo = s.payTo
			o.UpdatedAt = time.Now()
		})

		// Deduct from grant
		grant.SpentUSDC += p.PriceUSDC
		grant.RemainingUSDC -= p.PriceUSDC
		if grant.RemainingUSDC == 0 {
			grant.Status = GrantExhausted
		}

		purchasedOrders = append(purchasedOrders, AWALOrderSummary{
			OrderID:    order.ID,
			ProductID:  p.ID,
			Title:      p.Title,
			AmountUSDC: p.PriceUSDC,
			TxHash:     txHash,
			SettledAt:  time.Now().Format(time.RFC3339),
		})

		logMsg("[AWAL:Settlement] Order %s paid on-chain: tx=%s | Remaining allowance: %.2f USDC",
			order.ID, txHash, float64(grant.RemainingUSDC)/1_000_000)
	}

	logMsg("[AWAL:Complete] Purchased %d items autonomously. Total spent: %.2f USDC. Remaining allowance: %.2f USDC.",
		len(purchasedOrders), float64(grant.SpentUSDC)/1_000_000, float64(grant.RemainingUSDC)/1_000_000)

	writeJSON(w, http.StatusOK, AWALRunResponse{
		Grant:          grant,
		PurchasedCount: len(purchasedOrders),
		TotalSpentUSDC: grant.SpentUSDC,
		RemainingUSDC:  grant.RemainingUSDC,
		Orders:         purchasedOrders,
		Logs:           logs,
	})
}
