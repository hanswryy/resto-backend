# Resto Backend — Order Management API

Backend API untuk aplikasi pemesanan menu (order management) sederhana. Dibangun
sebagai proyek portofolio dengan fokus pada **status code HTTP yang benar**,
**autentikasi JWT**, dan **integrasi push notification (FCM)**.

Aplikasi mobile (React Native + Expo) ada di repo terpisah: **resto-mobile**.

## Tech Stack

| Komponen | Teknologi |
|---|---|
| Bahasa | Go |
| Web framework | [Gin](https://github.com/gin-gonic/gin) |
| Database | PostgreSQL |
| DB driver | [pgx v5](https://github.com/jackc/pgx) — **raw SQL, tanpa ORM** |
| Auth | JWT (HS256, satu secret) |
| Password hashing | bcrypt |
| Push notification | [Firebase Admin SDK Go](https://firebase.google.com/docs/admin/setup) (FCM) |

## Daftar Endpoint

| Method | Endpoint | Deskripsi | Auth | Status Sukses | Status Error |
|---|---|---|:---:|:---:|---|
| POST | `/auth/login` | Login, terbitkan JWT | — | `200` | `400` payload rusak · `401` kredensial salah |
| GET | `/menu` | Daftar menu | — | `200` | — |
| GET | `/menu/:id` | Detail item menu | — | `200` | `400` id bukan angka · `404` item tidak ada |
| POST | `/orders` | Buat pesanan baru | ✅ | `201` | `400` payload rusak · `401` tanpa/invalid token · `422` validasi gagal (item kosong / qty ≤ 0 / menu tidak ada) |
| GET | `/orders/:id` | Cek status pesanan | ✅ | `200` | `401` tanpa/invalid token · `404` order tidak ada / bukan milik user |
| PATCH | `/orders/:id/status` | Update status pesanan | ✅ | `200` | `400` status tidak dikenal · `401` tanpa/invalid token · `404` order tidak ada |
| GET | `/health` | Health check (app + DB) | — | `200` | `500` DB tidak sehat |

**Endpoint terproteksi** membutuhkan header:
```
Authorization: Bearer <JWT>
```

### Catatan desain auth
- `POST /orders` diproteksi agar setiap order punya pemilik (`user_id` diambil dari
  token) — dibutuhkan untuk mengarahkan push notification FCM.
- `GET /orders/:id` diproteksi **dan** memeriksa kepemilikan: user hanya bisa melihat
  order miliknya sendiri. Order milik orang lain dijawab `404` (bukan `403`) agar tidak
  membocorkan keberadaan order.
- Auth sengaja disederhanakan: **semua user terautentikasi** dapat mengubah status
  order. Pemisahan role staf/customer (RBAC) berada di luar scope proyek ini.

### Kredensial contoh (seed data)
| Email | Password |
|---|---|
| `customer@resto.test` | `password123` |
| `staff@resto.test` | `password123` |

### Contoh request

```bash
# Login → dapat token
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"customer@resto.test","password":"password123"}'

# Buat order (butuh token)
curl -X POST http://localhost:8080/orders \
  -H "Authorization: Bearer <JWT>" \
  -H "Content-Type: application/json" \
  -d '{"items":[{"menu_item_id":1,"quantity":2},{"menu_item_id":3,"quantity":1}]}'

# Update status → "ready" (memicu push notification ke pemilik order)
curl -X PATCH http://localhost:8080/orders/1/status \
  -H "Authorization: Bearer <JWT>" \
  -H "Content-Type: application/json" \
  -d '{"status":"ready"}'
```

## Skema Database

Empat tabel dengan pola header–detail untuk pesanan (`orders` + `order_items`).

```
users ──1:N── orders ──1:N── order_items ──N:1── menu_items
```

### `users`
| Kolom | Tipe | Keterangan |
|---|---|---|
| `id` | BIGSERIAL PK | |
| `email` | TEXT UNIQUE NOT NULL | |
| `password_hash` | TEXT NOT NULL | hash bcrypt |
| `device_token` | TEXT (nullable) | token FCM device, di-update saat login |
| `created_at` | TIMESTAMPTZ | default `now()` |

### `menu_items`
| Kolom | Tipe | Keterangan |
|---|---|---|
| `id` | BIGSERIAL PK | |
| `name` | TEXT NOT NULL | |
| `description` | TEXT NOT NULL | default `''` |
| `price` | INTEGER NOT NULL | rupiah, disimpan sebagai integer (mis. `25000`) |
| `is_available` | BOOLEAN NOT NULL | default `true` |
| `created_at` | TIMESTAMPTZ | default `now()` |

### `orders`
| Kolom | Tipe | Keterangan |
|---|---|---|
| `id` | BIGSERIAL PK | |
| `user_id` | BIGINT FK → `users(id)` | pemilik order |
| `status` | TEXT NOT NULL | `pending` \| `preparing` \| `ready` \| `completed` \| `cancelled` (CHECK constraint) |
| `total` | INTEGER NOT NULL | dihitung backend dari harga DB |
| `created_at` | TIMESTAMPTZ | default `now()` |
| `updated_at` | TIMESTAMPTZ | default `now()` |

### `order_items`
| Kolom | Tipe | Keterangan |
|---|---|---|
| `id` | BIGSERIAL PK | |
| `order_id` | BIGINT FK → `orders(id)` ON DELETE CASCADE | |
| `menu_item_id` | BIGINT FK → `menu_items(id)` | |
| `quantity` | INTEGER NOT NULL | CHECK `> 0` |
| `price_at_order` | INTEGER NOT NULL | snapshot harga saat pesan |

> **Kenapa `price_at_order`?** Harga menu bisa berubah; pesanan lama tetap mencatat
> harga saat transaksi terjadi.

Skema lengkap + seed data ada di [`db/schema.sql`](db/schema.sql).

## Environment Variables

| Variable | Wajib | Deskripsi |
|---|:---:|---|
| `DATABASE_URL` | ✅ | Connection string PostgreSQL, mis. `postgres://user@localhost:5432/resto?sslmode=disable` |
| `JWT_SECRET` | ✅ | Secret untuk sign/verify JWT |
| `FIREBASE_SERVICE_ACCOUNT_JSON` | — | Path ke file service account Firebase. Jika kosong, FCM dinonaktifkan (app tetap jalan) |

Buat file `.env` di root (lihat `.env.example`):
```
DATABASE_URL=postgres://USERNAME@localhost:5432/resto?sslmode=disable
JWT_SECRET=ganti-dengan-secret-anda
FIREBASE_SERVICE_ACCOUNT_JSON=./firebase-service-account.json
```

## Cara Menjalankan

### Prasyarat
- Go 1.27+
- PostgreSQL 14+

### 1. Setup database
```bash
# buat database
createdb resto

# jalankan skema + seed data
psql resto -f db/schema.sql
```

### 2. Konfigurasi environment
```bash
cp .env.example .env
# edit .env sesuai kredensial PostgreSQL Anda
```

### 3. Jalankan server
```bash
go mod download
go run .
```
Server berjalan di `http://localhost:8080`. Cek dengan:
```bash
curl http://localhost:8080/health
# {"status":"healthy"}
```

## Push Notification (FCM)

Satu alur notifikasi: saat `PATCH /orders/:id/status` mengubah status menjadi
**`ready`** ("siap diambil"), backend mengirim push notification ke `device_token`
milik pemilik order lewat Firebase Admin SDK.

Setup:
1. Buat Firebase project di [console.firebase.google.com](https://console.firebase.google.com).
2. **Project Settings → Service accounts → Generate new private key**.
3. Simpan file JSON sebagai `firebase-service-account.json` (sudah ada di `.gitignore`).
4. Set `FIREBASE_SERVICE_ACCOUNT_JSON` di `.env` ke path file tersebut.

Device token disimpan per user dan di-update melalui field opsional `device_token`
pada `POST /auth/login`.

## Screenshot Aplikasi

Screenshot antarmuka mobile (React Native + Expo) tersedia di repo **resto-mobile**.

<!-- TODO: tambahkan 2-3 screenshot frontend di sini setelah aplikasi mobile selesai -->
<!-- Contoh:
![Daftar Menu](docs/screenshot-menu.png)
![Keranjang](docs/screenshot-cart.png)
![Status Pesanan](docs/screenshot-order-status.png)
-->

## Struktur Project

```
resto-backend/
├── main.go                 # entry point: load env, connect DB, setup router
├── db/
│   └── schema.sql          # CREATE TABLE + seed data
├── internal/
│   ├── db/                 # koneksi pgx pool
│   ├── models/             # struct data
│   ├── auth/               # sign & verify JWT
│   ├── middleware/         # middleware autentikasi
│   ├── fcm/                # Firebase Cloud Messaging
│   └── handlers/           # HTTP handlers (auth, menu, orders)
└── .github/workflows/      # GitHub Actions CI
```
