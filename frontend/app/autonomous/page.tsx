"use client";

import { useState } from "react";
import Link from "next/link";
import { runAWALAutonomous, type AWALRunResult } from "@/lib/api";

export default function AutonomousPage() {
  const [budgetUsdc, setBudgetUsdc] = useState<number>(35);
  const [daysValid, setDaysValid] = useState<number>(7);
  const [scope, setScope] = useState<string>("Protocols & Systems");
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);
  const [result, setResult] = useState<AWALRunResult | null>(null);

  async function handleRun() {
    setLoading(true);
    setError(null);
    try {
      const res = await runAWALAutonomous({
        max_budget_usdc: Math.round(budgetUsdc * 1_000_000),
        days_valid: daysValid,
        scope,
      });
      setResult(res);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }

  return (
    <div>
      <div className="mb-6">
        <div className="inline-block rounded bg-purple-100 px-2 py-1 text-xs font-semibold text-purple-800 mb-2">
          บทที่ 7 — Autonomous Agentic Commerce
        </div>
        <h1 className="text-2xl font-bold">Coinbase AgentKit / AWAL Autonomous Shopping</h1>
        <p className="text-sm text-black/60 mt-1">
          ผู้ใช้กำหนดกรอบวงเงิน (Allowance) และวันสิ้นสุด (Expiry) เพียงครั้งเดียว
          จากนั้น AI Agent จะค้นหาสินค้า สร้างตะกร้า และสั่งจ่ายเงิน On-chain หลายรายการเองอัตโนมัติ
          โดยที่ผู้ใช้ไม่ต้องกดอนุมัติทีละ Order
        </p>
      </div>

      {/* Delegation & Allowance Setup Card */}
      <div className="rounded-xl border border-black/10 bg-white p-6 shadow-sm mb-6">
        <h2 className="text-lg font-semibold mb-4 flex items-center gap-2">
          <span>🛡️</span> 1. กำหนดสิทธิ์และวงเงินให้ Agent (Standing Delegation)
        </h2>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-4">
          <div>
            <label className="block text-xs font-medium text-black/70 mb-1">
              วงเงินสูงสุด (Max Allowance USDC)
            </label>
            <div className="relative">
              <input
                type="number"
                min="1"
                max="1000"
                value={budgetUsdc}
                onChange={(e) => setBudgetUsdc(parseFloat(e.target.value) || 0)}
                className="w-full rounded-md border border-black/20 px-3 py-2 text-sm focus:border-purple-600 focus:outline-none"
              />
              <span className="absolute right-3 top-2 text-xs text-black/40">USDC</span>
            </div>
          </div>

          <div>
            <label className="block text-xs font-medium text-black/70 mb-1">
              อายุสิทธิ์ (Valid For)
            </label>
            <div className="relative">
              <input
                type="number"
                min="1"
                max="365"
                value={daysValid}
                onChange={(e) => setDaysValid(parseInt(e.target.value, 10) || 1)}
                className="w-full rounded-md border border-black/20 px-3 py-2 text-sm focus:border-purple-600 focus:outline-none"
              />
              <span className="absolute right-3 top-2 text-xs text-black/40">วัน (Days)</span>
            </div>
          </div>

          <div>
            <label className="block text-xs font-medium text-black/70 mb-1">
              ขอบเขต / หมวดหมู่สินค้า (Shopping Scope)
            </label>
            <input
              type="text"
              value={scope}
              onChange={(e) => setScope(e.target.value)}
              placeholder="e.g. Protocols, AI, Systems"
              className="w-full rounded-md border border-black/20 px-3 py-2 text-sm focus:border-purple-600 focus:outline-none"
            />
          </div>
        </div>

        <button
          onClick={handleRun}
          disabled={loading || budgetUsdc <= 0}
          className="w-full md:w-auto rounded-lg bg-purple-700 px-6 py-2.5 text-sm font-medium text-white transition hover:bg-purple-800 disabled:opacity-50"
        >
          {loading ? "กำลังค้นหาและชำระเงิน On-chain อัตโนมัติ..." : "🚀 มอบสิทธิ์และเริ่มการซื้ออัตโนมัติ (AWAL Run)"}
        </button>

        {error && <p className="mt-3 text-sm text-red-600">{error}</p>}
      </div>

      {/* Results Section */}
      {result && (
        <div className="space-y-6">
          {/* Grant Summary Card */}
          <div className="rounded-xl border border-purple-200 bg-purple-50/50 p-5">
            <div className="flex flex-wrap items-center justify-between gap-2 border-b border-purple-200 pb-3 mb-3">
              <div>
                <span className="text-xs font-bold uppercase tracking-wider text-purple-600">AWAL Grant ID:</span>
                <span className="ml-2 font-mono text-xs">{result.grant.id}</span>
              </div>
              <span className="rounded-full bg-green-100 px-2.5 py-0.5 text-xs font-medium text-green-800">
                สถานะ: {result.grant.status}
              </span>
            </div>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
              <div>
                <div className="text-xs text-black/60">วงเงินที่ได้รับมอบหมาย</div>
                <div className="font-semibold">{(result.grant.max_budget_usdc / 1_000_000).toFixed(2)} USDC</div>
              </div>
              <div>
                <div className="text-xs text-black/60">ใช้จ่ายไปแล้วจริง</div>
                <div className="font-semibold text-purple-900">{(result.total_spent_usdc / 1_000_000).toFixed(2)} USDC</div>
              </div>
              <div>
                <div className="text-xs text-black/60">วงเงินคงเหลือ</div>
                <div className="font-semibold text-green-700">{(result.remaining_usdc / 1_000_000).toFixed(2)} USDC</div>
              </div>
              <div>
                <div className="text-xs text-black/60">วันหมดอายุของสิทธิ์</div>
                <div className="font-mono text-xs text-black/80">
                  {new Date(result.grant.expires_at).toLocaleDateString("th-TH")}
                </div>
              </div>
            </div>
          </div>

          {/* Autonomous Purchases List */}
          <div className="rounded-xl border border-black/10 bg-white p-6 shadow-sm">
            <h2 className="text-lg font-semibold mb-3 flex items-center justify-between">
              <span>🛒 คำสั่งซื้อที่ AI Agent ชำระเงินเรียบร้อย ({result.purchased_count} รายการ)</span>
              <span className="text-xs font-normal text-black/50">Settled บน Anvil Mainnet Fork</span>
            </h2>

            <div className="divide-y divide-black/10">
              {result.orders.map((ord, idx) => (
                <div key={ord.order_id} className="py-4 flex flex-col md:flex-row md:items-center justify-between gap-3">
                  <div>
                    <div className="font-medium text-base">
                      {idx + 1}. {ord.title}
                    </div>
                    <div className="text-xs text-black/50 font-mono mt-0.5">
                      Order: {ord.order_id} · Product: {ord.product_id}
                    </div>
                    <div className="text-xs text-purple-800 font-mono mt-1 break-all">
                      Tx: {ord.tx_hash}
                    </div>
                  </div>
                  <div className="flex items-center gap-4">
                    <span className="font-semibold text-sm">
                      {(ord.amount_usdc / 1_000_000).toFixed(2)} USDC
                    </span>
                    <Link
                      href={`/order/${ord.order_id}`}
                      className="rounded border border-black/20 px-2.5 py-1 text-xs hover:bg-black/5"
                    >
                      ดู Order
                    </Link>
                  </div>
                </div>
              ))}
            </div>
          </div>

          {/* Execution Timeline Logs */}
          <div className="rounded-xl border border-black/10 bg-gray-900 p-5 text-gray-200 shadow-inner">
            <h3 className="font-mono text-xs uppercase tracking-wider text-gray-400 mb-3 flex items-center gap-2">
              <span>⚡</span> Coinbase AgentKit / AWAL Autonomous Action Log
            </h3>
            <div className="font-mono text-xs space-y-1.5 overflow-x-auto max-h-60 overflow-y-auto">
              {result.logs.map((log, i) => (
                <div key={i} className="text-emerald-400">
                  <span className="text-gray-500 mr-2">{">"}</span>
                  {log}
                </div>
              ))}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
