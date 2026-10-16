-- STH event configuration. Safe to rerun. Defaults are newly designed, not original drop rates.
CREATE TABLE IF NOT EXISTS player.sth_reward_pool (
  tier integer NOT NULL CHECK (tier IN (4,5,6)),
  item_id integer NOT NULL,
  quantity integer NOT NULL CHECK (quantity BETWEEN 1 AND 9999),
  weight integer NOT NULL CHECK (weight BETWEEN 1 AND 1000000),
  enabled boolean NOT NULL DEFAULT true,
  PRIMARY KEY (tier,item_id)
);
INSERT INTO player.sth_reward_pool (tier,item_id,quantity,weight) VALUES
(4,5298,1,65),(4,5299,1,30),(4,2335,1,5),
(5,5299,1,50),(5,5300,1,30),(5,2335,1,15),(5,1652,1,5),
(6,5300,1,45),(6,5301,1,30),(6,2335,2,15),(6,1652,1,10)
ON CONFLICT (tier,item_id) DO NOTHING;
CREATE TABLE IF NOT EXISTS player.sth_event_claim (
 character_id bigint NOT NULL,
 position integer NOT NULL CHECK(position BETWEEN 1 AND 103),
 event_type integer NOT NULL,
 item_id integer,
 quantity integer,
 state text NOT NULL DEFAULT 'pending' CHECK (state IN ('pending','processing','delivered')),
 created_at timestamptz NOT NULL DEFAULT now(),
 delivered_at timestamptz,
 PRIMARY KEY(character_id,position)
);
ALTER TABLE player.character_sth_progress ADD COLUMN IF NOT EXISTS blocked_until timestamptz;
CREATE TABLE IF NOT EXISTS player.sth_event_settings (
 event_type integer PRIMARY KEY,
 blocked_seconds integer NOT NULL DEFAULT 0 CHECK (blocked_seconds >= 0),
 target_position integer CHECK (target_position BETWEEN 1 AND 103)
);
INSERT INTO player.sth_event_settings(event_type,blocked_seconds,target_position) VALUES
(9,0,57),(13,0,1),(14,30,NULL),(15,120,NULL),(16,180,NULL),(17,300,NULL)
ON CONFLICT (event_type) DO NOTHING;
-- Configuration view (edit underlying tables):
-- SELECT * FROM player.sth_reward_pool ORDER BY tier,weight DESC;
-- UPDATE player.sth_reward_pool SET weight=20 WHERE tier=4 AND item_id=2335;
