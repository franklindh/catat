CREATE TABLE "users" (
  "id" bigserial PRIMARY KEY,
  "email" varchar(255) UNIQUE NOT NULL,
  "name" varchar(255),
  "avatar_url" text,
  "google_auth_id" varchar(255), 
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "updated_at" timestamptz
);

CREATE TABLE "categories" (
  "id" bigserial PRIMARY KEY,
  "user_id" bigint NOT NULL,
  "name" varchar(255) NOT NULL,
  "type" varchar(20) NOT NULL CHECK (type IN ('INCOME', 'EXPENSE')),
  "icon_url" text,
  "deleted_at" timestamptz,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "updated_at" timestamptz,
  
  UNIQUE ("user_id", "name", "type") 
);

CREATE TABLE "transactions" (
  "id" bigserial PRIMARY KEY,
  "user_id" bigint NOT NULL REFERENCES "users" ("id") ON DELETE CASCADE,
  "category_id" bigint REFERENCES "categories" ("id") ON DELETE SET NULL, 
  "amount" numeric(19,4) NOT NULL CHECK (amount > 0) ,
  "description" text,
  "transaction_date" timestamptz NOT NULL,
  "type" varchar(20) NOT NULL CHECK (type IN ('INCOME', 'EXPENSE')),
  "deleted_at" timestamptz,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "updated_at" timestamptz
);

ALTER TABLE "users" 
ADD COLUMN "role" varchar(20) NOT NULL DEFAULT 'USER' 
CHECK (role IN ('USER', 'ADMIN'));

CREATE INDEX "idx_users_role" ON "users" ("role");
CREATE INDEX "idx_transactions_user" ON "transactions" ("user_id");
CREATE INDEX "idx_transactions_date" ON "transactions" ("transaction_date");
CREATE INDEX "idx_categories_user" ON "categories" ("user_id", LOWER("name"));

CREATE UNIQUE INDEX "idx_unique_category_name_type" ON "categories" ("user_id", LOWER("name"), "type");

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';


CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_categories_updated_at BEFORE UPDATE ON categories FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_transactions_updated_at BEFORE UPDATE ON transactions FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
