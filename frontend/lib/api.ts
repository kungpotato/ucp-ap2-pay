// Thin fetch wrapper around the Go backend from backend/main.go. No SDK,
// no code generation — the routes here are a 1:1 mirror of the mux table
// in backend/main.go so it's obvious which lesson introduced each call.
const BASE = process.env.NEXT_PUBLIC_BACKEND_URL ?? "http://localhost:8080";

export type Money = { amount: number; currency: string };

export type Product = {
  id: string;
  title: string;
  author: string;
  description: string;
  image_url: string;
  price: Money;
  price_usdc: number;
  stock: number;
};

export type LineItem = {
  product_id: string;
  title: string;
  quantity: number;
  unit_price: Money;
  unit_usdc: number;
};

export type Cart = { id: string; items: LineItem[]; created_at: string };

export type Order = {
  id: string;
  cart_id: string;
  status:
    | "pending_mandate"
    | "pending_payment"
    | "paid"
    | "failed";
  total: Money;
  total_usdc: number;
  created_at: string;
  updated_at: string;
  intent_mandate_id?: string;
  cart_mandate_id?: string;
  payment_mandate_id?: string;
  pay_to?: string;
  tx_hash?: string;
};

async function api<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    ...init,
    headers: { "Content-Type": "application/json", ...init?.headers },
    cache: "no-store",
  });
  if (!res.ok) {
    const body = await res.text();
    throw new Error(`${init?.method ?? "GET"} ${path} -> ${res.status}: ${body}`);
  }
  return res.json() as Promise<T>;
}

// --- lesson 1-2: catalog, cart, order handoff ------------------------------

export const listCatalog = (q = "") =>
  api<{ products: Product[] }>(`/ucp/catalog${q ? `?q=${encodeURIComponent(q)}` : ""}`);

export const getProduct = (id: string) => api<Product>(`/ucp/catalog/${id}`);

export const createCart = () => api<Cart>("/ucp/cart", { method: "POST" });

export const getCart = (id: string) => api<Cart>(`/ucp/cart/${id}`);

export const addItem = (cartId: string, productId: string, quantity: number) =>
  api<Cart>(`/ucp/cart/${cartId}/items`, {
    method: "POST",
    body: JSON.stringify({ product_id: productId, quantity }),
  });

export const createOrder = (cartId: string) =>
  api<Order>(`/ucp/cart/${cartId}/order`, { method: "POST" });

export const getOrder = (id: string) => api<Order>(`/orders/${id}`);

// --- lesson 4: AP2 mandate chain -------------------------------------------

export const createIntentMandate = (query: string, maxTotalUsdc: number) =>
  api<{ id: string }>("/ap2/intent-mandate", {
    method: "POST",
    body: JSON.stringify({ query, max_total_usdc: maxTotalUsdc }),
  });

export const createCartMandate = (orderId: string, intentMandateId: string) =>
  api<{ id: string }>(`/ap2/orders/${orderId}/cart-mandate`, {
    method: "POST",
    body: JSON.stringify({ intent_mandate_id: intentMandateId }),
  });

export const createPaymentMandate = (orderId: string, cartMandateId: string, payerAddress: string) =>
  api<{ id: string }>(`/ap2/orders/${orderId}/payment-mandate`, {
    method: "POST",
    body: JSON.stringify({ cart_mandate_id: cartMandateId, payer_address: payerAddress }),
  });

// --- lesson 6: agent pays on-chain ------------------------------------------

export const agentCheckout = (orderId: string) =>
  api<Order>(`/agent/orders/${orderId}/checkout`, { method: "POST" });
