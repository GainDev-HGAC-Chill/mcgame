-- Bootstrap verified missing days for supported loop activities.
-- Tracking was deployed on 2026-10-16 (Asia/Ho_Chi_Minh). Historical
 -- dates before that are NOT reconstructed, since there is no event ledger.
BEGIN;
CREATE OR REPLACE FUNCTION player.seed_findback_missing_day(p_character_id BIGINT)
RETURNS VOID LANGUAGE plpgsql SECURITY DEFINER
SET search_path = player, pg_temp
AS $$
DECLARE
    yesterday DATE := (now() AT TIME ZONE 'Asia/Ho_Chi_Minh')::date - 1;
    player_level INT;
    created_day DATE;
BEGIN
    -- Do not infer entitlements before this system started recording activity.
    IF yesterday < DATE '2026-10-16' THEN RETURN; END IF;
    SELECT p.level, (c.created_at AT TIME ZONE 'Asia/Ho_Chi_Minh')::date
      INTO player_level, created_day
      FROM player.characters c
      JOIN player.character_progression p ON p.character_id = c.id
     WHERE c.id = p_character_id;
    IF player_level IS NULL OR created_day IS NULL OR created_day > yesterday THEN RETURN; END IF;

    -- Only the four activity keys instrumented by the Stage3 loop trigger.
    -- A missed day is zero completed; recorded successful cycles already have
    -- a row and ON CONFLICT ensures they are never overwritten.
    INSERT INTO player.character_findback_days
      (character_id,activity_day,activity_key,completed_count,claimed_count,level_snapshot)
    SELECT p_character_id, yesterday, v.key, 0, 0, LEAST(160,GREATEST(1,player_level))
      FROM (VALUES ('xiuxing',70),('chongwu',50),('panjun',30),('feimo',50)) AS v(key,min_level)
     WHERE player_level >= v.min_level
    ON CONFLICT (character_id,activity_day,activity_key) DO NOTHING;
END;
$$;
COMMIT;
