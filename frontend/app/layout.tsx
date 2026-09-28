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
        <header className="border-b border-black/10 px-6 py-4">
          <Link href="/" className="text-xl font-semibold no-underline">
            📚 ร้านหนังสือ Agentic
          </Link>
        </header>
        <main className="mx-auto max-w-4xl px-6 py-8">{children}</main>
      </body>
    </html>
  );
}
