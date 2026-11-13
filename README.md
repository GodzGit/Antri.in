# ANTRI.IN — Sistem Antrean Digital

ANTRI.IN adalah backend REST API untuk sistem antrean digital. Pengunjung dapat mengambil nomor antrean dari ponsel, melihat estimasi waktu tunggu, dan menerima notifikasi saat gilirannya hampir tiba. Petugas dapat memanggil antrean berikutnya, sedangkan admin dapat mengelola layanan dan melihat statistik.

## Tech stack

- **Bahasa dan framework:** Go, Fiber v2
- **Database utama:** PostgreSQL (pgx v5)
- **Database riwayat dan notifikasi:** MongoDB (driver v2)
- **Message broker dan cache:** Redis, Asynq worker
- **Autentikasi:** JWT
- **Dokumentasi API:** Swagger
- **Deployment:** Docker dan Docker Compose

## Arsitektur

Proyek ini menggunakan pola Clean Architecture dengan pembagian utama berikut:

- `app/model/` — struct data, request, dan response
- `app/repository/` — akses database; mendukung `DBTX` untuk transaksi
- `app/service/` — logika bisnis dan handler HTTP (Fiber)
- `route/` — pendaftaran endpoint
- `middleware/` — autentikasi, permission, dan logging
- `helper/` — response standar, error handling, validator, dan pagination
- `database/` — koneksi PostgreSQL, MongoDB, dan Redis
- `migrations/` — migrasi skema database
- `worker/` — Asynq worker dan pemrosesan outbox

## Menjalankan dengan Docker

### Prasyarat

- Docker Desktop atau Docker Engine dengan Docker Compose
- Git

### Langkah

1. Clone repository dan masuk ke direktori project:

   ```bash
   git clone https://github.com/GodzGit/Antri.in.git
   cd Antri.in
   ```

2. Bangun dan jalankan service:

   ```bash
   docker compose up -d --build
   ```

3. Terapkan migrasi skema PostgreSQL:

   ```bash
   docker exec -i antriin-postgres psql -U postgres -d antriin < migrations/000001_init_schema.up.sql
   ```

4. API tersedia di `http://localhost:3000`.

Untuk melihat log atau menghentikan service:

```bash
docker compose logs -f api
docker compose down
```

## Menjalankan secara lokal

### Prasyarat

- Go 1.22 atau lebih baru
- PostgreSQL, MongoDB, dan Redis

### Langkah

1. Buat database PostgreSQL bernama `antriin`.
2. Salin konfigurasi contoh, lalu isi kredensial lokal:

   ```bash
   cp .env.example .env
   ```

   Pastikan nilai `DB_USER`, `DB_PASSWORD`, dan host/port service sesuai dengan lingkunganmu.
3. Terapkan migrasi:

   ```bash
   psql -U postgres -d antriin -f migrations/000001_init_schema.up.sql
   ```

4. Unduh dependency dan jalankan API:

   ```bash
   go mod download
   go run .
   ```

API berjalan di `http://localhost:3000` (atau port dari `APP_PORT`).

## Dokumentasi API

Saat server berjalan, buka Swagger di [http://localhost:3000/swagger/index.html](http://localhost:3000/swagger/index.html).

## Tim — Kelompok 13

- **Bagus** — fondasi, konfigurasi, database, helper, migrasi, Docker, dan Git
- **Isna** — autentikasi, manajemen pengguna, JWT, dan middleware
- **Shidqi** — layanan, loket, tiket, transaksi antrean, dan unit test
- **Irfan** — outbox pattern, Asynq worker, notifikasi, statistik, dan Swagger
