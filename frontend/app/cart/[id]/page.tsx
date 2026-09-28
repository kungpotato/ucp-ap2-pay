"use client";

import { useRouter, useParams } from "next/navigation";
import { useEffect, useState } from "react";
import { createOrder, getCart, type Cart } from "@/lib/api";

export default function CartPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const [cart, setCart] = useState<Cart | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [creating, setCreating] = useState(false);

  useEffect(() => {
    getCart(id).then(setCart).catch((e) => setError(String(e)));
  }, [id]);

  async function handleCheckout() {
    setCreating(true);
    setError(null);
    try {
      const order = await createOrder(id);
      router.push(`/order/${order.id}`);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setCreating(false);
    }
  }

  if (error) return <p className="text-red-600">{error}</p>;
  if (!cart) return <p>กำลังโหลดตะกร้า…</p>;

  const subtotalThb = cart.items.reduce((s, it) => s + it.unit_price.amount * it.quantity, 0);
  const subtotalUsdc = cart.items.reduce((s, it) => s + it.unit_usdc * it.quantity, 0);

  return (
    <div>
      <h1 className="mb-6 text-2xl font-semibold">ตะกร้าของคุณ</h1>
      {cart.items.length === 0 ? (
        <p>ตะกร้าว่าง — กลับไปเลือกหนังสือก่อนนะ</p>
      ) : (
        <>
          <ul className="divide-y divide-black/10">
            {cart.items.map((it) => (
              <li key={it.product_id} className="flex justify-between py-3">
                <span>
                  {it.title} × {it.quantity}
                </span>
                <span>{((it.unit_price.amount * it.quantity) / 100).toLocaleString("th-TH")} บาท</span>
              </li>
            ))}
          </ul>
          <div className="mt-4 flex justify-between font-semibold">
            <span>รวม</span>
            <span>
              {(subtotalThb / 100).toLocaleString("th-TH")} บาท · {(subtotalUsdc / 1_000_000).toFixed(2)}{" "}
              USDC
            </span>
          </div>
          <button
            onClick={handleCheckout}
            disabled={creating}
            className="mt-6 rounded-md bg-[var(--accent)] px-4 py-2 text-white disabled:opacity-50"
          >
            {creating ? "กำลังสร้างคำสั่งซื้อ…" : "สร้างคำสั่งซื้อ (ไป AP2 mandate)"}
          </button>
        </>
      )}
    </div>
  );
}
