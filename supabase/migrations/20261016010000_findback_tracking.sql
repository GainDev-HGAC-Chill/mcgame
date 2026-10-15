-- Findback: strictly separate the activity day from the later claim day.
-- Does not generate missed rewards automatically; only server-verified progress
-- may be written to this table. A subsequent implementation must integrate
-- daily quest/job/pet activity completion events before enabling redemption.
CREATE TABLE IF NOT EXISTS player.character_findback_days (
    character_id BIGINT NOT NULL,
    activity_day DATE NOT NULL,
    activity_key TEXT NOT NULL CHECK (activity_key IN
      ('xiuxing','xueyuan','chongwu','panjun','feimo','xuanshang','xuanshang_banner')),
    completed_count INTEGER NOT NULL DEFAULT 0 CHECK (completed_count >= 0),
    claimed_count INTEGER NOT NULL DEFAULT 0 CHECK (claimed_count >= 0),
    level_snapshot INTEGER NOT NULL CHECK (level_snapshot BETWEEN 1 AND 160),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (character_id, activity_day, activity_key),
    CHECK (claimed_count <= 10)
);
CREATE INDEX IF NOT EXISTS character_findback_days_day_idx
  ON player.character_findback_days (activity_day, character_id);
CREATE TABLE IF NOT EXISTS player.character_findback_claims (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    character_id BIGINT NOT NULL,
    activity_day DATE NOT NULL,
    activity_key TEXT NOT NULL,
    claim_mode SMALLINT NOT NULL CHECK (claim_mode IN (1,2)),
    count_claimed INTEGER NOT NULL CHECK (count_claimed > 0),
    exp_awarded BIGINT NOT NULL CHECK (exp_awarded >= 0),
    cost_currency TEXT NOT NULL CHECK (cost_currency IN ('money','gold')),
    cost_amount BIGINT NOT NULL CHECK (cost_amount >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS character_findback_claims_character_idx
 ON player.character_findback_claims(character_id,activity_day);
