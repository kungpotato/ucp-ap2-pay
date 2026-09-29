import type { Metadata } from "next";
import Link from "next/link";
import "./globals.css";

export const metadata: Metadata = {
  title: "ร้านหนังสือ Agentic Commerce",
  description: "UCP -> AP2 -> X402 -> on-chain settlement, ตัวอย่าง workshop",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="th">
      <body>
        <header className="border-b border-black/10 px-6 py-4 flex items-center justify-between">
          <Link href="/" className="text-xl font-semibold no-underline">
            📚 ร้านหนังสือ Agentic
          </Link>
          <nav className="flex items-center gap-4 text-sm font-medium">
            <Link href="/" className="hover:text-purple-700">
              หน้าร้าน
            </Link>
            <Link
              href="/autonomous"
              className="rounded-full bg-purple-100 px-3 py-1 text-purple-800 hover:bg-purple-200 transition"
            >
              🤖 Autonomous Agent (AgentKit/AWAL)
            </Link>
          </nav>
        </header>
        <main className="mx-auto max-w-4xl px-6 py-8">{children}</main>
      </body>
    </html>
  );
}
