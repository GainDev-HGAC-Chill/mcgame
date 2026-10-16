-- Preview-only STH progression; no rewards or purchases.
CREATE TABLE IF NOT EXISTS player.character_sth_progress (
 character_id bigint PRIMARY KEY,
 refresh_date date NOT NULL,
 position integer NOT NULL DEFAULT 1 CHECK(position BETWEEN 1 AND 103),
 normal_dice integer NOT NULL DEFAULT 0 CHECK(normal_dice >= 0),
 lucky_dice integer NOT NULL DEFAULT 0 CHECK(lucky_dice >= 0),
 bought_dice integer NOT NULL DEFAULT 0 CHECK(bought_dice >= 0),
 visited integer[] NOT NULL DEFAULT '{}'::integer[],
 moving boolean NOT NULL DEFAULT false,
 updated_at timestamptz NOT NULL DEFAULT now()
);
