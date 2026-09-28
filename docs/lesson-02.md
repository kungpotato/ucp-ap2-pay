# บทที่ 2 — UCP: หน้าร้าน Next.js

## เป้าหมาย

สร้าง storefront จริงด้วย Next.js (App Router) ที่คุยกับ UCP backend จาก
บทที่ 1 ตรง ๆ ไม่มี backend-for-frontend คั่นกลาง — ครบ 4 หน้า: แคตตาล็อก,
รายละเอียดหนังสือ, ตะกร้า, และหน้าคำสั่งซื้อ (ที่จะขยายในบทที่ 3-6)

## เรียนไปทำไม

บทที่ 1 พิสูจน์ว่า backend มี API ที่พอสำหรับทั้งมนุษย์และ agent แล้ว
บทนี้พิสูจน์ครึ่งแรก — ว่ามนุษย์ใช้ API ชุดเดียวกันได้จริงผ่าน UI ปกติ
ไม่ต้องมี endpoint พิเศษสำหรับ "คนคลิกเว็บ" การแยก concern แบบนี้ (state
ธุรกิจอยู่ backend, presentation อยู่ frontend) ทำให้บทที่ 6 (agent
self-checkout) เพิ่ม "ผู้ใช้" อีกคนเข้ามาได้โดยไม่ต้องแก้ backend เลย

## สิ่งที่สร้าง

- [`frontend/lib/api.ts`](../frontend/lib/api.ts) — fetch wrapper แบบบาง
  ๆ ที่ mirror ตาราง route ใน `backend/main.go` 1:1 ไม่มี code
  generation, ไม่มี SDK — เห็น endpoint ไหนมาจากบทไหนได้ทันที
- [`frontend/app/page.tsx`](../frontend/app/page.tsx) — server component
  ดึงแคตตาล็อกตรง ๆ (ไม่ต้อง client-side fetch สำหรับหน้าที่ไม่มี
  interactivity)
- [`frontend/app/book/[id]/AddToCartButton.tsx`](../frontend/app/book/[id]/AddToCartButton.tsx)
  — เก็บ cart id ไว้ใน `localStorage` เพราะ UCP cart ไม่มีแนวคิด session,
  แค่ id เดียวก็พอ
- [`frontend/app/cart/[id]/page.tsx`](../frontend/app/cart/[id]/page.tsx)
  — "สร้างคำสั่งซื้อ" เรียก `POST /ucp/cart/{id}/order` แล้วพาไปหน้า
  order (บทที่ 3 จะเข้ามาต่อจากตรงนี้)

## เส้นทางหน้าเว็บ

![เส้นทางหน้าร้าน Next.js](diagrams/02-frontend-flow.svg)

## ทางเลือกอื่นที่พิจารณา

- **ทำ API routes ใน Next.js เป็นตัวกลาง (BFF)** — ปัดตก เพราะ UCP
  backend ทำ CORS แบบเปิดไว้แล้วสำหรับ local dev (`withCORS` ใน
  `main.go`) การเพิ่มชั้น BFF จะซ่อนไว้ว่า "agent เรียก endpoint เดียวกับ
  ที่มนุษย์เรียก" ซึ่งเป็นประเด็นสำคัญของบทนี้
- **ใช้ Redux/Zustand จัดการ state** — ปัดตก เพราะ state จริงทั้งหมดอยู่
  backend (cart, order) frontend แค่ fetch/refetch ไม่มี client state ที่
  ซับซ้อนพอจะต้องใช้ state library

## ทดสอบเอง

```bash
cd frontend
npm install
cp .env.local.example .env.local
npm run dev
```

เปิด `http://localhost:3000` (ต้องมี backend รันที่ `:8080` จากบทที่ 1)

## ต่อยอดได้อะไร

หน้า `/order/[id]` ที่เพิ่งสร้างจะกลายเป็นเวทีของบทที่ 3-6 — มนุษย์กด
ปุ่มสร้าง mandate ทีละขั้น ก่อนจะให้ agent รับช่วงจ่ายเงินต่อในบทที่ 6
