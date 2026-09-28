import Image from "next/image";
import Link from "next/link";
import { listCatalog } from "@/lib/api";

export default async function HomePage() {
  const { products } = await listCatalog();

  return (
    <div>
      <h1 className="mb-6 text-2xl font-semibold">แคตตาล็อกหนังสือ (UCP)</h1>
      <div className="grid grid-cols-2 gap-6 sm:grid-cols-3">
        {products.map((p) => (
          <Link
            key={p.id}
            href={`/book/${p.id}`}
            className="block rounded-lg border border-black/10 p-3 no-underline transition hover:border-black/30"
          >
            <Image
              src={p.image_url}
              alt={p.title}
              width={240}
              height={320}
              className="mb-2 h-40 w-full rounded object-cover"
            />
            <div className="font-medium text-[var(--ink)]">{p.title}</div>
            <div className="text-sm text-black/60">{p.author}</div>
            <div className="mt-1 text-sm">
              {(p.price.amount / 100).toLocaleString("th-TH")} บาท ·{" "}
              {(p.price_usdc / 1_000_000).toFixed(2)} USDC
            </div>
          </Link>
        ))}
      </div>
    </div>
  );
}
