package main

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// AP2 (Agent Payment Protocol) models a chain of three signed mandates.
// Real AP2 lets each party (human, agent, merchant) hold its own keypair
// and exchange signed JSON over the wire. This workshop keeps two demo
// keypairs *inside the backend* (documented, not a real trust boundary) so
// the whole chain is inspectable in one process — see docs/lesson-04.md for
// why that's a simplification and how to split it into real actors.
var (
	userPub, userPriv         = mustGenerateKey()
	merchantPub, merchantPriv = mustGenerateKey()
)

func mustGenerateKey() (ed25519.PublicKey, ed25519.PrivateKey) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		panic(err)
	}
	return pub, priv
}

func sign(priv ed25519.PrivateKey, payload any) (string, string) {
	b, _ := json.Marshal(payload)
	sig := ed25519.Sign(priv, b)
	return hex.EncodeToString(b), hex.EncodeToString(sig)
}

func verify(pub ed25519.PublicKey, payloadHex, sigHex string) bool {
	payload, err1 := hex.DecodeString(payloadHex)
	sig, err2 := hex.DecodeString(sigHex)
	if err1 != nil || err2 != nil {
		return false
	}
	return ed25519.Verify(pub, payload, sig)
}

// IntentMandate: the human states a budget and what they're willing to let
// the agent shop for. Signed with the user's key — this is the mandate
// that scopes everything the agent is allowed to do downstream.
type IntentMandate struct {
	ID           string    `json:"id"`
	Query        string    `json:"query"`
	MaxTotalUSDC int64     `json:"max_total_usdc"`
	ExpiresAt    time.Time `json:"expires_at"`
	Payload      string    `json:"payload"`   // hex-encoded signed bytes
	Signature    string    `json:"signature"` // hex-encoded ed25519 signature
	UserPubKey   string    `json:"user_pub_key"`
}

// CartMandate: the merchant proposes a concrete order within the intent's
// budget. Signed with the merchant's key — proves *this* order is what the
// merchant is offering to fulfil for that price, not something an agent or
// a compromised frontend fabricated.
type CartMandate struct {
	ID               string    `json:"id"`
	OrderID          string    `json:"order_id"`
	IntentMandateID  string    `json:"intent_mandate_id"`
	TotalUSDC        int64     `json:"total_usdc"`
	CreatedAt        time.Time `json:"created_at"`
	Payload          string    `json:"payload"`
	Signature        string    `json:"signature"`
	MerchantPubKey   string    `json:"merchant_pub_key"`
}

// PaymentMandate: the human-in-the-loop approval to actually move money —
// the last signature before X402/on-chain settlement is allowed to run.
type PaymentMandate struct {
	ID            string    `json:"id"`
	CartMandateID string    `json:"cart_mandate_id"`
	PayerAddress  string    `json:"payer_address"`
	CreatedAt     time.Time `json:"created_at"`
	Payload       string    `json:"payload"`
	Signature     string    `json:"signature"`
	UserPubKey    string    `json:"user_pub_key"`
}

type mandateStore struct {
	mu       sync.RWMutex
	intents  map[string]*IntentMandate
	carts    map[string]*CartMandate
	payments map[string]*PaymentMandate
}

var mandates = &mandateStore{
	intents:  map[string]*IntentMandate{},
	carts:    map[string]*CartMandate{},
	payments: map[string]*PaymentMandate{},
}

// --- HTTP handlers --------------------------------------------------------

type intentMandateRequest struct {
	Query        string `json:"query"`
	MaxTotalUSDC int64  `json:"max_total_usdc"`
}

func (s *Server) handleIntentMandate(w http.ResponseWriter, r *http.Request) {
	var req intentMandateRequest
	if err := decodeJSON(r, &req); err != nil || req.MaxTotalUSDC <= 0 {
		writeError(w, http.StatusBadRequest, "query and max_total_usdc (>0) are required")
		return
	}
	m := &IntentMandate{
		ID:           newID("intent"),
		Query:        req.Query,
		MaxTotalUSDC: req.MaxTotalUSDC,
		ExpiresAt:    time.Now().Add(15 * time.Minute),
		UserPubKey:   hex.EncodeToString(userPub),
	}
	m.Payload, m.Signature = sign(userPriv, struct {
		ID           string `json:"id"`
		Query        string `json:"query"`
		MaxTotalUSDC int64  `json:"max_total_usdc"`
		ExpiresAt    string `json:"expires_at"`
	}{m.ID, m.Query, m.MaxTotalUSDC, m.ExpiresAt.Format(time.RFC3339)})

	mandates.mu.Lock()
	mandates.intents[m.ID] = m
	mandates.mu.Unlock()

	writeJSON(w, http.StatusCreated, m)
}

type cartMandateRequest struct {
	IntentMandateID string `json:"intent_mandate_id"`
}

// handleCartMandate has the merchant check the order fits inside the
// intent's budget and expiry, then sign it — refuse otherwise. This is the
// enforcement point: an agent cannot walk away with a cart the human never
// authorized the budget for.
func (s *Server) handleCartMandate(w http.ResponseWriter, r *http.Request) {
	orderID := r.PathValue("id")
	order, ok := s.orders.Get(orderID)
	if !ok {
		writeError(w, http.StatusNotFound, "order not found")
		return
	}
	var req cartMandateRequest
	if err := decodeJSON(r, &req); err != nil || req.IntentMandateID == "" {
		writeError(w, http.StatusBadRequest, "intent_mandate_id is required")
		return
	}

	mandates.mu.RLock()
	intent, ok := mandates.intents[req.IntentMandateID]
	mandates.mu.RUnlock()
	if !ok {
		writeError(w, http.StatusNotFound, "intent mandate not found")
		return
	}
	if !verify(userPub, intent.Payload, intent.Signature) {
		writeError(w, http.StatusForbidden, "intent mandate signature invalid")
		return
	}
	if time.Now().After(intent.ExpiresAt) {
		writeError(w, http.StatusForbidden, "intent mandate expired")
		return
	}
	if order.TotalUSDC > intent.MaxTotalUSDC {
		writeError(w, http.StatusForbidden, "order exceeds intent mandate budget")
		return
	}

	m := &CartMandate{
		ID:              newID("cartm"),
		OrderID:         orderID,
		IntentMandateID: intent.ID,
		TotalUSDC:       order.TotalUSDC,
		CreatedAt:       time.Now(),
		MerchantPubKey:  hex.EncodeToString(merchantPub),
	}
	m.Payload, m.Signature = sign(merchantPriv, struct {
		ID              string `json:"id"`
		OrderID         string `json:"order_id"`
		IntentMandateID string `json:"intent_mandate_id"`
		TotalUSDC       int64  `json:"total_usdc"`
	}{m.ID, m.OrderID, m.IntentMandateID, m.TotalUSDC})

	mandates.mu.Lock()
	mandates.carts[m.ID] = m
	mandates.mu.Unlock()

	s.orders.Update(orderID, func(o *Order) { o.IntentMandateID = intent.ID; o.CartMandateID = m.ID })
	writeJSON(w, http.StatusCreated, m)
}

type paymentMandateRequest struct {
	CartMandateID string `json:"cart_mandate_id"`
	PayerAddress  string `json:"payer_address"`
}

// handlePaymentMandate is the human-in-the-loop click: "yes, pay this
// cart". Without this signature X402 (lesson 5) refuses to even quote a
// price to the agent.
func (s *Server) handlePaymentMandate(w http.ResponseWriter, r *http.Request) {
	orderID := r.PathValue("id")
	if _, ok := s.orders.Get(orderID); !ok {
		writeError(w, http.StatusNotFound, "order not found")
		return
	}
	var req paymentMandateRequest
	if err := decodeJSON(r, &req); err != nil || req.CartMandateID == "" || req.PayerAddress == "" {
		writeError(w, http.StatusBadRequest, "cart_mandate_id and payer_address are required")
		return
	}

	mandates.mu.RLock()
	cartMandate, ok := mandates.carts[req.CartMandateID]
	mandates.mu.RUnlock()
	if !ok || cartMandate.OrderID != orderID {
		writeError(w, http.StatusNotFound, "cart mandate not found for this order")
		return
	}
	if !verify(merchantPub, cartMandate.Payload, cartMandate.Signature) {
		writeError(w, http.StatusForbidden, "cart mandate signature invalid")
		return
	}

	m := &PaymentMandate{
		ID:            newID("paym"),
		CartMandateID: cartMandate.ID,
		PayerAddress:  req.PayerAddress,
		CreatedAt:     time.Now(),
		UserPubKey:    hex.EncodeToString(userPub),
	}
	m.Payload, m.Signature = sign(userPriv, struct {
		ID            string `json:"id"`
		CartMandateID string `json:"cart_mandate_id"`
		PayerAddress  string `json:"payer_address"`
	}{m.ID, m.CartMandateID, m.PayerAddress})

	mandates.mu.Lock()
	mandates.payments[m.ID] = m
	mandates.mu.Unlock()

	s.orders.Update(orderID, func(o *Order) {
		o.PaymentMandateID = m.ID
		o.Status = OrderPendingPayment
	})
	writeJSON(w, http.StatusCreated, m)
}
