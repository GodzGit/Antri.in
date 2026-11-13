-- 1. Tabel Roles & Permissions (Isna)
CREATE TABLE roles (
    id BIGSERIAL PRIMARY KEY,
    nama VARCHAR(50) UNIQUE NOT NULL
);

CREATE TABLE permissions (
    id BIGSERIAL PRIMARY KEY,
    nama VARCHAR(100) UNIQUE NOT NULL
);

CREATE TABLE role_permissions (
    role_id BIGINT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id BIGINT NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

-- 2. Tabel Users & Refresh Tokens (Isna)
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    nama VARCHAR(100) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role_id BIGINT NOT NULL REFERENCES roles(id) ON DELETE RESTRICT,
    is_active BOOLEAN DEFAULT TRUE NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT NOW() NOT NULL
);

CREATE TABLE refresh_tokens (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) UNIQUE NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL
);

-- 3. Tabel Layanan & Loket (Shidqi)
CREATE TABLE layanan (
    id BIGSERIAL PRIMARY KEY,
    kode VARCHAR(10) UNIQUE NOT NULL,
    nama VARCHAR(100) NOT NULL,
    deskripsi TEXT,
    jam_buka TIME,
    jam_tutup TIME,
    kuota_harian INT,
    rata_durasi_menit INT DEFAULT 5 NOT NULL,
    is_active BOOLEAN DEFAULT TRUE NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT NOW() NOT NULL
);

CREATE TABLE loket (
    id BIGSERIAL PRIMARY KEY,
    layanan_id BIGINT NOT NULL REFERENCES layanan(id) ON DELETE CASCADE,
    nama VARCHAR(50) NOT NULL,
    petugas_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    status VARCHAR(20) DEFAULT 'tutup' NOT NULL CHECK (status IN ('buka', 'tutup')),
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT NOW() NOT NULL
);

-- 4. Tabel Antrean & Tiket (Shidqi)
CREATE TABLE antrean_harian (
    layanan_id BIGINT NOT NULL REFERENCES layanan(id) ON DELETE CASCADE,
    tanggal DATE NOT NULL,
    nomor_terakhir INT DEFAULT 0 NOT NULL,
    PRIMARY KEY (layanan_id, tanggal)
);

CREATE TABLE tiket (
    id BIGSERIAL PRIMARY KEY,
    layanan_id BIGINT NOT NULL REFERENCES layanan(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    loket_id BIGINT REFERENCES loket(id) ON DELETE SET NULL,
    tanggal DATE NOT NULL,
    nomor INT NOT NULL,
    kode VARCHAR(20) NOT NULL,
    status VARCHAR(20) DEFAULT 'menunggu' NOT NULL CHECK (status IN ('menunggu', 'dipanggil', 'selesai', 'dilewati', 'batal')),
    diambil_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    dipanggil_at TIMESTAMPTZ,
    selesai_at TIMESTAMPTZ,
    version INT DEFAULT 1 NOT NULL,
    UNIQUE (layanan_id, tanggal, nomor)
);

-- Partial Unique Index: 1 user hanya boleh punya 1 tiket aktif per layanan per hari
CREATE UNIQUE INDEX idx_tiket_aktif_per_user_layanan 
ON tiket (user_id, layanan_id, tanggal) 
WHERE status IN ('menunggu', 'dipanggil');

-- 5. Tabel Outbox (Irfan)
CREATE TABLE outbox (
    id BIGSERIAL PRIMARY KEY,
    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    status VARCHAR(20) DEFAULT 'pending' NOT NULL CHECK (status IN ('pending', 'processed', 'failed')),
    attempts INT DEFAULT 0 NOT NULL,
    last_error TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    processed_at TIMESTAMPTZ
);