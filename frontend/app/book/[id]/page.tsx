import Image from "next/image";
import { notFound } from "next/navigation";
import { getProduct } from "@/lib/api";
import AddToCartButton from "./AddToCartButton";

export default async function BookPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const product = await getProduct(id).catch(() => null);
  if (!product) notFound();

  return (
    <div className="grid grid-cols-1 gap-8 sm:grid-cols-[240px_1fr]">
      <Image
        src={product.image_url}
        alt={product.title}
        width={240}
        height={320}
        className="w-full rounded-lg object-cover"
      />
      <div>
        <h1 className="text-2xl font-semibold">{product.title}</h1>
        <p className="text-black/60">{product.author}</p>
        <p className="mt-4">{product.description}</p>
        <p className="mt-4 text-lg">
          {(product.price.amount / 100).toLocaleString("th-TH")} บาท ·{" "}
          {(product.price_usdc / 1_000_000).toFixed(2)} USDC · เหลือ {product.stock} เล่ม
        </p>
        <div className="mt-6">
          <AddToCartButton productId={product.id} />
        </div>
      </div>
    </div>
  );
}
