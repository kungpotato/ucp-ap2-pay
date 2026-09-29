package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

// Server wires together every layer this workshop builds, lesson by lesson:
//
//	lesson 1-2: catalog, carts            (this file)
//	lesson 4:   AP2 mandates               (mandate.go)
//	lesson 5:   X402 payment-required flow (x402.go)
//	lesson 6:   on-chain settlement        (chain.go, settlement.go)
type Server struct {
	catalog           *Catalog
	carts             *CartStore
	orders            *OrderStore
	chain             *ChainClient
	payTo             string       // merchant wallet address on the anvil fork
	settlementAddress string       // deployed Settlement.sol contract address
	agent             *AgentWallet // the shopping agent's own on-chain wallet (lesson 6)
}

func main() {
	loadDotEnv("../.env", ".env")

	port := getenv("PORT", "8080")
	anvilRPC := getenv("ANVIL_RPC_URL", "http://127.0.0.1:8545")
	s := &Server{
		catalog:           NewCatalog(),
		carts:             NewCartStore(),
		orders:            NewOrderStore(),
		chain:             NewChainClient(anvilRPC),
		payTo:             getenv("MERCHANT_WALLET_ADDRESS", ""),
		settlementAddress: getenv("SETTLEMENT_CONTRACT_ADDRESS", ""),
	}
	if key := os.Getenv("AGENT_PRIVATE_KEY"); key != "" {
		s.agent = NewAgentWallet(key, anvilRPC)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealth)

	// --- UCP-inspired commerce surface (lesson 1-2) --------------------
	mux.HandleFunc("GET /ucp/catalog", s.handleListCatalog)
	mux.HandleFunc("GET /ucp/catalog/{id}", s.handleGetProduct)
	mux.HandleFunc("POST /ucp/cart", s.handleCreateCart)
	mux.HandleFunc("GET /ucp/cart/{id}", s.handleGetCart)
	mux.HandleFunc("POST /ucp/cart/{id}/items", s.handleAddItem)
	mux.HandleFunc("POST /ucp/cart/{id}/order", s.handleCreateOrder)
	mux.HandleFunc("GET /orders/{id}", s.handleGetOrder)

	// --- AP2 mandate chain (lesson 4) -----------------------------------
	mux.HandleFunc("POST /ap2/intent-mandate", s.handleIntentMandate)
	mux.HandleFunc("POST /ap2/orders/{id}/cart-mandate", s.handleCartMandate)
	mux.HandleFunc("POST /ap2/orders/{id}/payment-mandate", s.handlePaymentMandate)

	// --- X402 + on-chain settlement (lesson 5-6) ------------------------
	mux.HandleFunc("POST /x402/orders/{id}/pay", s.handleX402Pay)
	mux.HandleFunc("POST /agent/orders/{id}/checkout", s.handleAgentCheckout)

	// --- Lesson 7: Coinbase AgentKit / AWAL Autonomous Shopping ---------
	mux.HandleFunc("POST /agent/awal/grant", s.handleAWALCreateGrant)
	mux.HandleFunc("GET /agent/awal/grants", s.handleAWALListGrants)
	mux.HandleFunc("POST /agent/awal/run", s.handleAWALRun)

	log.Printf("ucp-ap2-pay backend listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, withCORS(withLogging(mux))))
}

// --- middleware ---------------------------------------------------------

func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

// withCORS is permissive on purpose: this is a local workshop backend the
// Next.js dev server (a different origin/port) calls directly from the
// browser. Lock this down to a real allow-list before deploying anywhere.
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Payment")
		w.Header().Set("Access-Control-Expose-Headers", "X-Payment-Required")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// --- JSON helpers ---------------------------------------------------------

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func decodeJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return err
	}
	if len(strings.TrimSpace(string(body))) == 0 {
		return nil
	}
	return json.Unmarshal(body, v)
}

// --- lesson 1-2 handlers: catalog, cart, order handoff --------------------

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleListCatalog(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	writeJSON(w, http.StatusOK, map[string]any{"products": s.catalog.Find(q)})
}

func (s *Server) handleGetProduct(w http.ResponseWriter, r *http.Request) {
	p, ok := s.catalog.Get(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, "product not found")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) handleCreateCart(w http.ResponseWriter, r *http.Request) {
	cart := s.carts.Create()
	writeJSON(w, http.StatusCreated, cart)
}

func (s *Server) handleGetCart(w http.ResponseWriter, r *http.Request) {
	cart, ok := s.carts.Get(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, "cart not found")
		return
	}
	writeJSON(w, http.StatusOK, cart)
}

type addItemRequest struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

func (s *Server) handleAddItem(w http.ResponseWriter, r *http.Request) {
	var req addItemRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	cart, err := s.carts.AddItem(r.PathValue("id"), s.catalog, req.ProductID, req.Quantity)
	switch err {
	case nil:
		writeJSON(w, http.StatusOK, cart)
	case ErrNotFound:
		writeError(w, http.StatusNotFound, "cart or product not found")
	case ErrOutOfStock:
		writeError(w, http.StatusConflict, "requested quantity exceeds stock")
	default:
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}

// handleCreateOrder is the UCP -> AP2 handoff point: it freezes the cart
// into an Order the buyer (human or agent) can now attach a mandate chain
// to (lesson 4) before any payment is even discussed.
func (s *Server) handleCreateOrder(w http.ResponseWriter, r *http.Request) {
	cart, ok := s.carts.Get(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, "cart not found")
		return
	}
	if len(cart.Items) == 0 {
		writeError(w, http.StatusUnprocessableEntity, "cart is empty")
		return
	}
	order := s.orders.Create(cart.ID, cart.Subtotal(), cart.SubtotalUSDC())
	writeJSON(w, http.StatusCreated, order)
}

func (s *Server) handleGetOrder(w http.ResponseWriter, r *http.Request) {
	order, ok := s.orders.Get(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, "order not found")
		return
	}
	writeJSON(w, http.StatusOK, order)
}
