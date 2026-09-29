package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAWALGrantAndAutonomousShopping(t *testing.T) {
	s := &Server{
		catalog: NewCatalog(),
		carts:   NewCartStore(),
		orders:  NewOrderStore(),
	}

	// 1. Create a delegation grant: 35 USDC, valid for 7 days, scope: "systems"
	grantReq := map[string]any{
		"user_address":    "0x70997970C51812dc3A010C7d01b50e0d17dc79C8",
		"max_budget_usdc": 35_000_000,
		"days_valid":      7,
		"scope":           "systems",
	}
	body, _ := json.Marshal(grantReq)
	req := httptest.NewRequest(http.MethodPost, "/agent/awal/grant", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	s.handleAWALCreateGrant(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var grant AWALGrant
	if err := json.Unmarshal(rec.Body.Bytes(), &grant); err != nil {
		t.Fatalf("failed to decode grant: %v", err)
	}

	if grant.MaxBudgetUSDC != 35_000_000 || grant.RemainingUSDC != 35_000_000 {
		t.Fatalf("unexpected allowance balance: %+v", grant)
	}
	if grant.Status != GrantActive {
		t.Fatalf("expected grant active, got %s", grant.Status)
	}

	// 2. Run autonomous multi-product shopping with this grant
	runReq := AWALRunRequest{
		GrantID: grant.ID,
		Scope:   "systems",
	}
	runBody, _ := json.Marshal(runReq)
	req2 := httptest.NewRequest(http.MethodPost, "/agent/awal/run", bytes.NewReader(runBody))
	rec2 := httptest.NewRecorder()
	s.handleAWALRun(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("expected status 200 from handleAWALRun, got %d: %s", rec2.Code, rec2.Body.String())
	}

	var runResp AWALRunResponse
	if err := json.Unmarshal(rec2.Body.Bytes(), &runResp); err != nil {
		t.Fatalf("failed to decode run response: %v", err)
	}

	if runResp.PurchasedCount == 0 {
		t.Fatalf("expected at least 1 book purchased, got 0")
	}

	if runResp.TotalSpentUSDC <= 0 || runResp.RemainingUSDC >= 35_000_000 {
		t.Fatalf("allowance was not correctly deducted: total_spent=%d, remaining=%d",
			runResp.TotalSpentUSDC, runResp.RemainingUSDC)
	}

	for _, ord := range runResp.Orders {
		if ord.TxHash == "" {
			t.Errorf("order %s missing on-chain tx_hash", ord.OrderID)
		}
		// verify order in store is marked paid
		stored, ok := s.orders.Get(ord.OrderID)
		if !ok || stored.Status != OrderPaid {
			t.Errorf("order %s not marked paid in store", ord.OrderID)
		}
	}

	t.Logf("Successfully purchased %d books autonomously. Spent: %.2f USDC, Remaining: %.2f USDC",
		runResp.PurchasedCount, float64(runResp.TotalSpentUSDC)/1_000_000, float64(runResp.RemainingUSDC)/1_000_000)
}

func TestAWALExpiredGrantRejection(t *testing.T) {
	s := &Server{
		catalog: NewCatalog(),
		carts:   NewCartStore(),
		orders:  NewOrderStore(),
	}

	expiredGrant := awalGrants.Create("0x70997970C51812dc3A010C7d01b50e0d17dc79C8", 50_000_000, time.Now().Add(-1*time.Hour), "all")

	runReq := AWALRunRequest{GrantID: expiredGrant.ID}
	body, _ := json.Marshal(runReq)
	req := httptest.NewRequest(http.MethodPost, "/agent/awal/run", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	s.handleAWALRun(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for expired grant, got %d", rec.Code)
	}
}
