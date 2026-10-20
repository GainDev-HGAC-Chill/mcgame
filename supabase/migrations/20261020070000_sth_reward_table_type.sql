
BEGIN;

-- Phần thưởng sự kiện Bí Cảnh
ALTER TABLE player.sth_reward_pool
ADD COLUMN IF NOT EXISTS table_type integer;

UPDATE player.sth_reward_pool
SET table_type = 29
WHERE table_type IS NULL;

ALTER TABLE player.sth_reward_pool
ALTER COLUMN table_type SET NOT NULL;

ALTER TABLE player.sth_reward_pool
ADD CONSTRAINT sth_reward_pool_table_type_check
CHECK (table_type IN (19, 29));

ALTER TABLE player.sth_reward_pool
DROP CONSTRAINT sth_reward_pool_pkey;

ALTER TABLE player.sth_reward_pool
ADD CONSTRAINT sth_reward_pool_pkey
PRIMARY KEY (tier, table_type, item_id);


-- Phần thưởng chiến đấu Bí Cảnh
ALTER TABLE player.sth_battle_reward_pool
ADD COLUMN IF NOT EXISTS table_type integer;

UPDATE player.sth_battle_reward_pool
SET table_type = 29
WHERE table_type IS NULL;

ALTER TABLE player.sth_battle_reward_pool
ALTER COLUMN table_type SET NOT NULL;

ALTER TABLE player.sth_battle_reward_pool
ADD CONSTRAINT sth_battle_reward_pool_table_type_check
CHECK (table_type IN (19, 29));

ALTER TABLE player.sth_battle_reward_pool
DROP CONSTRAINT sth_battle_reward_pool_pkey;

ALTER TABLE player.sth_battle_reward_pool
ADD CONSTRAINT sth_battle_reward_pool_pkey
PRIMARY KEY (table_type, item_id);


-- Lưu loại vật phẩm khi sự kiện đã tạo phần thưởng
ALTER TABLE player.sth_event_claim
ADD COLUMN IF NOT EXISTS table_type integer;

UPDATE player.sth_event_claim
SET table_type = 29
WHERE item_id IS NOT NULL
  AND table_type IS NULL;

ALTER TABLE player.sth_event_claim
ADD CONSTRAINT sth_event_claim_table_type_check
CHECK (
    (item_id IS NULL AND table_type IS NULL)
    OR
    (item_id IS NOT NULL AND table_type IN (19, 29))
);


-- Lưu loại vật phẩm khi chiến đấu đã tạo phần thưởng
ALTER TABLE player.sth_battle_runs
ADD COLUMN IF NOT EXISTS table_type integer;

UPDATE player.sth_battle_runs
SET table_type = 29
WHERE item_id IS NOT NULL
  AND table_type IS NULL;

ALTER TABLE player.sth_battle_runs
ADD CONSTRAINT sth_battle_runs_table_type_check
CHECK (
    (item_id IS NULL AND table_type IS NULL)
    OR
    (item_id IS NOT NULL AND table_type IN (19, 29))
);

COMMIT;
