"use client";

import { useParams } from "next/navigation";
import { useEffect, useState } from "react";
import {
  agentCheckout,
  createCartMandate,
  createIntentMandate,
  createPaymentMandate,
  getOrder,
  type Order,
} from "@/lib/api";

// Anvil's well-known dev account #1 — the demo "buyer wallet" this whole
// workshop uses for lesson 6. Never a real key; see contracts/README.md.
const DEMO_PAYER_ADDRESS = "0x70997970C51812dc3A010C7d01b50e0d17dc79C8";

type StepState = "idle" | "loading" | "done" | "error";

export default function OrderPage() {
  const { id } = useParams<{ id: string }>();
  const [order, setOrder] = useState<Order | null>(null);
  const [intentMandateId, setIntentMandateId] = useState<string | null>(null);
  const [cartMandateId, setCartMandateId] = useState<string | null>(null);
  const [step, setStep] = useState<StepState>("idle");
  const [error, setError] = useState<string | null>(null);

  const refresh = () => getOrder(id).then(setOrder);

  useEffect(() => {
    refresh();
  }, [id]);

  async function run(action: () => Promise<void>) {
    setStep("loading");
    setError(null);
    try {
      await action();
      await refresh();
      setStep("done");
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      setStep("error");
    }
  }

  if (!order) return <p>กำลังโหลดคำสั่งซื้อ…</p>;

  const budgetUsdc = order.total_usdc; // demo: intent budget == exact order total

  return (
    <div>
      <h1 className="mb-2 text-2xl font-semibold">คำสั่งซื้อ {order.id}</h1>
      <p className="mb-6 text-black/60">
        ยอดรวม {(order.total.amount / 100).toLocaleString("th-TH")} บาท ·{" "}
        {(order.total_usdc / 1_000_000).toFixed(2)} USDC · สถานะปัจจุบัน:{" "}
        <span className="font-medium">{order.status}</span>
      </p>

      <ol className="space-y-4">
        <li className="rounded-lg border border-black/10 p-4">
          <div className="font-medium">1. Intent mandate — มนุษย์ตั้งงบและมอบอำนาจ</div>
          <p className="mb-2 text-sm text-black/60">
            เซ็น (ed25519) ว่ายินดีให้ agent ใช้จ่ายได้ไม่เกิน{" "}
            {(budgetUsdc / 1_000_000).toFixed(2)} USDC สำหรับคำสั่งซื้อนี้
          </p>
          <button
            disabled={!!order.intent_mandate_id}
            onClick={() =>
              run(async () => {
                const m = await createIntentMandate("ร้านหนังสือ agentic", budgetUsdc);
                setIntentMandateId(m.id);
              })
            }
            className="rounded-md bg-[var(--accent)] px-3 py-1.5 text-sm text-white disabled:opacity-40"
          >
            {order.intent_mandate_id ? "สร้างแล้ว ✓" : "สร้าง Intent Mandate"}
          </button>
        </li>

        <li className="rounded-lg border border-black/10 p-4">
          <div className="font-medium">2. Cart mandate — merchant ยืนยันคำสั่งซื้ออยู่ในงบ</div>
          <p className="mb-2 text-sm text-black/60">merchant เซ็นรับรองยอดนี้ ถ้าเกินงบจะถูกปฏิเสธ</p>
          <button
            disabled={!intentMandateId || !!order.cart_mandate_id}
            onClick={() =>
              run(async () => {
                if (!intentMandateId) return;
                const m = await createCartMandate(order.id, intentMandateId);
                setCartMandateId(m.id);
              })
            }
            className="rounded-md bg-[var(--accent)] px-3 py-1.5 text-sm text-white disabled:opacity-40"
          >
            {order.cart_mandate_id ? "สร้างแล้ว ✓" : "สร้าง Cart Mandate"}
          </button>
        </li>

        <li className="rounded-lg border border-black/10 p-4">
          <div className="font-medium">3. Payment mandate — มนุษย์กดยืนยันจ่ายเงินจริง</div>
          <p className="mb-2 text-sm text-black/60">
            human-in-the-loop ครั้งสุดท้ายก่อนเงินจะเคลื่อนบนเชน (wallet ตัวอย่าง{" "}
            {DEMO_PAYER_ADDRESS.slice(0, 10)}…)
          </p>
          <button
            disabled={!cartMandateId || !!order.payment_mandate_id}
            onClick={() =>
              run(async () => {
                if (!cartMandateId) return;
                await createPaymentMandate(order.id, cartMandateId, DEMO_PAYER_ADDRESS);
              })
            }
            className="rounded-md bg-[var(--accent)] px-3 py-1.5 text-sm text-white disabled:opacity-40"
          >
            {order.payment_mandate_id ? "อนุมัติแล้ว ✓" : "อนุมัติจ่ายเงิน"}
          </button>
        </li>

        <li className="rounded-lg border border-black/10 p-4">
          <div className="font-medium">4. X402 + settlement บนเชน — agent จ่ายเงินเอง</div>
          <p className="mb-2 text-sm text-black/60">
            agent ขอราคาผ่าน HTTP 402, approve + เรียก Settlement.pay() บน anvil fork ของ mainnet ด้วย
            USDC จริง
          </p>
          <button
            disabled={!order.payment_mandate_id || order.status === "paid"}
            onClick={() => run(() => agentCheckout(order.id).then(() => undefined))}
            className="rounded-md bg-[var(--accent)] px-3 py-1.5 text-sm text-white disabled:opacity-40"
          >
            {order.status === "paid" ? "ชำระเงินแล้ว ✓" : "ให้ agent จ่ายเงิน (on-chain)"}
          </button>
          {order.tx_hash && (
            <p className="mt-2 break-all text-sm text-black/60">tx: {order.tx_hash}</p>
          )}
        </li>
      </ol>

      {step === "loading" && <p className="mt-4 text-sm">กำลังทำรายการ…</p>}
      {error && <p className="mt-4 text-sm text-red-600">{error}</p>}
    </div>
  );
}
