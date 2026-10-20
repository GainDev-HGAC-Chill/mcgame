BEGIN;

CREATE TABLE IF NOT EXISTS player.character_sth_progress (
    character_id bigint PRIMARY KEY,
    refresh_date date NOT NULL,
    position integer NOT NULL DEFAULT 1 CHECK (position BETWEEN 1 AND 103),
    normal_dice integer NOT NULL DEFAULT 0 CHECK (normal_dice >= 0),
    lucky_dice integer NOT NULL DEFAULT 0 CHECK (lucky_dice >= 0),
    bought_dice integer NOT NULL DEFAULT 0 CHECK (bought_dice >= 0),
    visited integer[] NOT NULL DEFAULT '{}'::integer[],
    moving boolean NOT NULL DEFAULT false,
    blocked_until timestamptz,
    updated_at timestamptz NOT NULL DEFAULT now()
);
ALTER TABLE player.character_sth_progress
    ADD COLUMN IF NOT EXISTS blocked_until timestamptz;

CREATE TABLE IF NOT EXISTS player.sth_reward_pool (
    tier integer NOT NULL CHECK (tier IN (4, 5, 6)),
    table_type integer NOT NULL CHECK (table_type IN (19, 29)),
    item_id integer NOT NULL,
    quantity integer NOT NULL CHECK (quantity BETWEEN 1 AND 9999),
    weight integer NOT NULL CHECK (weight BETWEEN 1 AND 1000000),
    enabled boolean NOT NULL DEFAULT true,
    PRIMARY KEY (tier, table_type, item_id)
);
ALTER TABLE player.sth_reward_pool
    ADD COLUMN IF NOT EXISTS table_type integer;
UPDATE player.sth_reward_pool SET table_type = 29 WHERE table_type IS NULL;
ALTER TABLE player.sth_reward_pool ALTER COLUMN table_type SET NOT NULL;
ALTER TABLE player.sth_reward_pool DROP CONSTRAINT IF EXISTS sth_reward_pool_table_type_check;
ALTER TABLE player.sth_reward_pool ADD CONSTRAINT sth_reward_pool_table_type_check
    CHECK (table_type IN (19, 29));
ALTER TABLE player.sth_reward_pool DROP CONSTRAINT IF EXISTS sth_reward_pool_pkey;
ALTER TABLE player.sth_reward_pool ADD CONSTRAINT sth_reward_pool_pkey
    PRIMARY KEY (tier, table_type, item_id);

INSERT INTO player.sth_reward_pool (tier, table_type, item_id, quantity, weight) VALUES
    (4, 29, 5298, 1, 65),
    (4, 29, 5299, 1, 30),
    (4, 29, 2335, 1, 5),
    (5, 29, 5299, 1, 50),
    (5, 29, 5300, 1, 30),
    (5, 29, 2335, 1, 15),
    (5, 29, 1652, 1, 5),
    (6, 29, 5300, 1, 45),
    (6, 29, 5301, 1, 30),
    (6, 29, 2335, 2, 15),
    (6, 29, 1652, 1, 10)
ON CONFLICT (tier, table_type, item_id) DO NOTHING;

CREATE TABLE IF NOT EXISTS player.sth_event_claim (
    character_id bigint NOT NULL,
    position integer NOT NULL CHECK (position BETWEEN 1 AND 103),
    event_type integer NOT NULL,
    table_type integer,
    item_id integer,
    quantity integer,
    state text NOT NULL DEFAULT 'pending' CHECK (state IN ('pending', 'processing', 'delivered')),
    created_at timestamptz NOT NULL DEFAULT now(),
    delivered_at timestamptz,
    PRIMARY KEY (character_id, position)
);
ALTER TABLE player.sth_event_claim ADD COLUMN IF NOT EXISTS table_type integer;
UPDATE player.sth_event_claim
SET table_type = 29
WHERE item_id IS NOT NULL AND table_type IS NULL;
ALTER TABLE player.sth_event_claim DROP CONSTRAINT IF EXISTS sth_event_claim_table_type_check;
ALTER TABLE player.sth_event_claim ADD CONSTRAINT sth_event_claim_table_type_check CHECK (
    (item_id IS NULL AND table_type IS NULL)
    OR (item_id IS NOT NULL AND table_type IN (19, 29))
);

CREATE TABLE IF NOT EXISTS player.sth_event_settings (
    event_type integer PRIMARY KEY,
    blocked_seconds integer NOT NULL DEFAULT 0 CHECK (blocked_seconds >= 0),
    target_position integer CHECK (target_position BETWEEN 1 AND 103)
);
INSERT INTO player.sth_event_settings (event_type, blocked_seconds, target_position) VALUES
    (9, 0, 57),
    (13, 0, 1),
    (14, 30, NULL),
    (15, 120, NULL),
    (16, 180, NULL),
    (17, 300, NULL)
ON CONFLICT (event_type) DO NOTHING;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM data.data_tbl_npc_creature) THEN
        WITH pairs(nid, cid) AS (
            VALUES (2348, 2115), (2348, 2119), (2349, 2116), (2349, 2120), (2350, 2117), (2350, 2121)
        ), missing AS (
            SELECT p.nid, p.cid
            FROM pairs p
            WHERE NOT EXISTS (
                SELECT 1
                FROM data.data_tbl_npc_creature c
                WHERE c.nid::integer = p.nid AND c.cid::integer = p.cid
            )
        ), numbered AS (
            SELECT (SELECT COALESCE(MAX(id), 0) FROM data.data_tbl_npc_creature)
                + row_number() OVER (ORDER BY nid, cid) AS new_id,
                nid,
                cid
            FROM missing
        )
        INSERT INTO data.data_tbl_npc_creature
            (id, nid, cid, level, q_level, boss_flag, start_rate, end_rate, exp, exp_multi, exp_set, unique_flag)
        SELECT new_id, nid, cid, 0, 3, 0, 100, 100, 0, 1, 0, 0
        FROM numbered;
    END IF;
END
$$;

CREATE TABLE IF NOT EXISTS player.sth_battle_runs (
    character_id bigint NOT NULL,
    activity_day date NOT NULL,
    position integer NOT NULL CHECK (position BETWEEN 1 AND 103),
    npc_id integer NOT NULL,
    battle_id text,
    status text NOT NULL CHECK (status IN ('starting', 'active', 'failed', 'won', 'lost', 'pending', 'processing', 'delivered')),
    table_type integer,
    item_id integer,
    quantity integer,
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (character_id, activity_day, position)
);
ALTER TABLE player.sth_battle_runs ADD COLUMN IF NOT EXISTS table_type integer;
UPDATE player.sth_battle_runs
SET table_type = 29
WHERE item_id IS NOT NULL AND table_type IS NULL;
ALTER TABLE player.sth_battle_runs DROP CONSTRAINT IF EXISTS sth_battle_runs_table_type_check;
ALTER TABLE player.sth_battle_runs ADD CONSTRAINT sth_battle_runs_table_type_check CHECK (
    (item_id IS NULL AND table_type IS NULL)
    OR (item_id IS NOT NULL AND table_type IN (19, 29))
);
CREATE UNIQUE INDEX IF NOT EXISTS sth_battle_runs_battle_id_uidx
    ON player.sth_battle_runs (battle_id)
    WHERE battle_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS player.sth_battle_reward_pool (
    table_type integer NOT NULL CHECK (table_type IN (19, 29)),
    item_id integer NOT NULL,
    quantity integer NOT NULL CHECK (quantity > 0),
    weight integer NOT NULL CHECK (weight > 0),
    enabled boolean NOT NULL DEFAULT true,
    PRIMARY KEY (table_type, item_id)
);
ALTER TABLE player.sth_battle_reward_pool ADD COLUMN IF NOT EXISTS table_type integer;
UPDATE player.sth_battle_reward_pool SET table_type = 29 WHERE table_type IS NULL;
ALTER TABLE player.sth_battle_reward_pool ALTER COLUMN table_type SET NOT NULL;
ALTER TABLE player.sth_battle_reward_pool DROP CONSTRAINT IF EXISTS sth_battle_reward_pool_table_type_check;
ALTER TABLE player.sth_battle_reward_pool ADD CONSTRAINT sth_battle_reward_pool_table_type_check
    CHECK (table_type IN (19, 29));
ALTER TABLE player.sth_battle_reward_pool DROP CONSTRAINT IF EXISTS sth_battle_reward_pool_pkey;
ALTER TABLE player.sth_battle_reward_pool ADD CONSTRAINT sth_battle_reward_pool_pkey
    PRIMARY KEY (table_type, item_id);

INSERT INTO player.sth_battle_reward_pool (table_type, item_id, quantity, weight) VALUES
    (29, 5299, 1, 50),
    (29, 5300, 1, 30),
    (29, 2335, 1, 15),
    (29, 1652, 1, 5)
ON CONFLICT (table_type, item_id) DO NOTHING;

COMMIT;
