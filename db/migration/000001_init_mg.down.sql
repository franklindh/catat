-- 1. Hapus Tabel (Urutan: Anak -> Induk)
-- Menggunakan CASCADE akan otomatis menghapus Trigger yang menempel di tabel ini
DROP TABLE IF EXISTS "budgets" CASCADE;
DROP TABLE IF EXISTS "transactions" CASCADE;
DROP TABLE IF EXISTS "categories" CASCADE;
DROP TABLE IF EXISTS "accounts" CASCADE;
DROP TABLE IF EXISTS "users" CASCADE;

-- 2. Hapus Function (Logic updated_at)
-- Function harus dihapus terpisah karena dia objek global, bukan milik tabel
DROP FUNCTION IF EXISTS update_updated_at_column() CASCADE;