"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { addItem, createCart } from "@/lib/api";

// UCP has no notion of "session" — the cart id is the only handle the
// buyer (human or agent) needs, so we just keep it in localStorage. A real
// storefront would tie this to an authenticated session instead.
const CART_KEY = "ucp-ap2-pay:cart-id";

export default function AddToCartButton({ productId }: { productId: string }) {
  const router = useRouter();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleClick() {
    setLoading(true);
    setError(null);
    try {
      let cartId = localStorage.getItem(CART_KEY);
      if (!cartId) {
        const cart = await createCart();
        cartId = cart.id;
        localStorage.setItem(CART_KEY, cartId);
      }
      await addItem(cartId, productId, 1);
      router.push(`/cart/${cartId}`);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }

  return (
    <div>
      <button
        onClick={handleClick}
        disabled={loading}
        className="rounded-md bg-[var(--accent)] px-4 py-2 text-white disabled:opacity-50"
      >
        {loading ? "กำลังหยิบใส่ตะกร้า…" : "หยิบใส่ตะกร้า"}
      </button>
      {error && <p className="mt-2 text-sm text-red-600">{error}</p>}
    </div>
  );
}
