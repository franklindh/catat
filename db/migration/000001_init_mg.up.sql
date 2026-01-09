DROP TABLE IF EXISTS "budgets" CASCADE;
DROP TABLE IF EXISTS "transactions" CASCADE;
DROP TABLE IF EXISTS "categories" CASCADE;
DROP TABLE IF EXISTS "accounts" CASCADE;
DROP TABLE IF EXISTS "users" CASCADE;


CREATE TABLE "users" (
  "id" bigserial PRIMARY KEY,
  "email" varchar(255) UNIQUE NOT NULL,
  "name" varchar(255),
  "avatar_url" text,
  
  "google_auth_id" varchar(255), 
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "updated_at" timestamptz
);

CREATE TABLE "accounts" (
  "id" bigserial PRIMARY KEY,
  "user_id" bigint NOT NULL,
  "name" varchar(100) NOT NULL, 
  "current_balance" numeric(19,4) NOT NULL DEFAULT 0, 
  "is_main_account" boolean DEFAULT false,
  "deleted_at" timestamptz, 
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
  "account_id" bigint NOT NULL, 
  "category_id" bigint, 
  
  
  "amount" numeric(19,4) NOT NULL CHECK (amount > 0),
  
  "description" text,
  "transaction_date" timestamptz NOT NULL,
  
  
  "type" varchar(20) NOT NULL CHECK (type IN ('INCOME', 'EXPENSE', 'TRANSFER')),
  
  
  "related_transfer_account_id" bigint, 
  
  "deleted_at" timestamptz,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "updated_at" timestamptz
);



ALTER TABLE "accounts" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON DELETE CASCADE;

ALTER TABLE "categories" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON DELETE CASCADE;

ALTER TABLE "transactions" ADD FOREIGN KEY ("account_id") REFERENCES "accounts" ("id") ON DELETE CASCADE;
ALTER TABLE "transactions" ADD FOREIGN KEY ("category_id") REFERENCES "categories" ("id") ON DELETE SET NULL;
ALTER TABLE "transactions" ADD FOREIGN KEY ("related_transfer_account_id") REFERENCES "accounts" ("id");


CREATE INDEX "idx_accounts_user" ON "accounts" ("user_id");
CREATE INDEX "idx_transactions_account" ON "transactions" ("account_id");
CREATE INDEX "idx_transactions_date" ON "transactions" ("transaction_date");
CREATE INDEX "idx_categories_user" ON "categories" ("user_id");


CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';


CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_accounts_updated_at BEFORE UPDATE ON accounts FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_categories_updated_at BEFORE UPDATE ON categories FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_transactions_updated_at BEFORE UPDATE ON transactions FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();