-- Real STH fights: NPC->creature templates. Level zero is intentionally overridden
-- from the player's actual level at battle start.
WITH pairs(nid,cid) AS (
 VALUES (2348,2115),(2348,2119),(2349,2116),(2349,2120),(2350,2117),(2350,2121)
), missing AS (
 SELECT p.nid,p.cid FROM pairs p
 WHERE NOT EXISTS (SELECT 1 FROM data.data_tbl_npc_creature c WHERE c.nid::integer=p.nid AND c.cid::integer=p.cid)
), numbered AS (
 SELECT (SELECT COALESCE(MAX(id),0) FROM data.data_tbl_npc_creature)
        + row_number() OVER (ORDER BY nid,cid) AS new_id,nid,cid FROM missing
)
INSERT INTO data.data_tbl_npc_creature
 (id,nid,cid,level,q_level,boss_flag,start_rate,end_rate,exp,exp_multi,exp_set,unique_flag)
SELECT new_id,nid,cid,0,3,0,100,100,0,1,0,0 FROM numbered;
CREATE TABLE IF NOT EXISTS player.sth_battle_runs (
 character_id bigint NOT NULL,
 activity_day date NOT NULL,
 position integer NOT NULL CHECK (position BETWEEN 1 AND 103),
 npc_id integer NOT NULL,
 battle_id text,
 status text NOT NULL CHECK (status IN ('starting','active','failed','won','lost','pending','processing','delivered')),
 item_id integer,
 quantity integer,
 updated_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(character_id,activity_day,position)
);
CREATE UNIQUE INDEX IF NOT EXISTS sth_battle_runs_battle_id_uidx
 ON player.sth_battle_runs(battle_id) WHERE battle_id IS NOT NULL;
CREATE TABLE IF NOT EXISTS player.sth_battle_reward_pool (
 item_id integer PRIMARY KEY,
 quantity integer NOT NULL CHECK(quantity>0),
 weight integer NOT NULL CHECK(weight>0),
 enabled boolean NOT NULL DEFAULT true
);
INSERT INTO player.sth_battle_reward_pool(item_id,quantity,weight) VALUES
 (5299,1,50),(5300,1,30),(2335,1,15),(1652,1,5)
ON CONFLICT(item_id) DO NOTHING;
