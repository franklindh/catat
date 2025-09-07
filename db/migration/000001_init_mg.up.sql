CREATE TABLE "users" (
  "id" uuid PRIMARY KEY DEFAULT (gen_random_uuid()),
  "email" text UNIQUE NOT NULL,
  "password_hash" text NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "updated_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE "accounts" (
  "id" uuid PRIMARY KEY DEFAULT (gen_random_uuid()),
  "user_id" uuid NOT NULL,
  "name" text NOT NULL,
  "type" text NOT NULL,
  "balance" numeric(19,4) NOT NULL DEFAULT 0,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "updated_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE "categories" (
  "id" uuid PRIMARY KEY DEFAULT (gen_random_uuid()),
  "user_id" uuid NOT NULL,
  "name" text NOT NULL,
  "type" text NOT NULL,
  "parent_id" uuid,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "updated_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE "transactions" (
  "id" uuid PRIMARY KEY DEFAULT (gen_random_uuid()),
  "user_id" uuid NOT NULL,
  "account_id" uuid NOT NULL,
  "category_id" uuid,
  "amount" numeric(19,4) NOT NULL,
  "description" text NOT NULL,
  "transaction_date" timestamptz NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE "receipts" (
  "id" uuid PRIMARY KEY DEFAULT (gen_random_uuid()),
  "transaction_id" uuid UNIQUE NOT NULL,
  "image_url" text NOT NULL,
  "raw_text" text,
  "created_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE UNIQUE INDEX ON "categories" ("user_id", "name");

CREATE INDEX ON "transactions" ("user_id", "transaction_date");

COMMENT ON COLUMN "accounts"."type" IS 'Contoh: "depository", "credit", "cash"';

COMMENT ON COLUMN "categories"."type" IS '"income" atau "expense"';

COMMENT ON COLUMN "categories"."parent_id" IS 'Untuk sub-kategori, mereferensikan dirinya sendiri';

COMMENT ON COLUMN "transactions"."amount" IS 'Positif untuk pendapatan, negatif untuk pengeluaran';

COMMENT ON COLUMN "receipts"."image_url" IS 'Path ke file gambar di storage (lokal atau cloud)';

ALTER TABLE "accounts" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id");

ALTER TABLE "categories" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id");

ALTER TABLE "categories" ADD FOREIGN KEY ("parent_id") REFERENCES "categories" ("id");

ALTER TABLE "transactions" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id");

ALTER TABLE "transactions" ADD FOREIGN KEY ("account_id") REFERENCES "accounts" ("id");

ALTER TABLE "transactions" ADD FOREIGN KEY ("category_id") REFERENCES "categories" ("id");

ALTER TABLE "receipts" ADD FOREIGN KEY ("transaction_id") REFERENCES "transactions" ("id");
