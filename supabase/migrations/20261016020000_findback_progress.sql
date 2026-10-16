-- Tracks only verified loop-round progress. No synthetic missed days.
-- Note: the final round of a loop is not reflected by an ft increment and
-- requires a future direct completion hook. Only partial cycles are counted.
CREATE OR REPLACE FUNCTION player.track_findback_loop_round() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE activity TEXT; playerlevel INT; n INT;
BEGIN
  IF NOT NEW.active OR NEW.ft <= OLD.ft OR NEW.character_id <> OLD.character_id OR NEW.loop_id <> OLD.loop_id THEN
    RETURN NEW;
  END IF;
  activity := CASE NEW.loop_id
    WHEN 6 THEN 'xiuxing' WHEN 5 THEN 'chongwu'
    WHEN 4 THEN 'panjun' WHEN 16 THEN 'feimo' ELSE NULL END;
  IF activity IS NULL THEN RETURN NEW; END IF;
  n := LEAST(10, NEW.ft-OLD.ft);
  SELECT level INTO playerlevel FROM player.character_progression WHERE character_id=NEW.character_id;
  IF playerlevel IS NULL THEN RETURN NEW; END IF;
  INSERT INTO player.character_findback_days(character_id,activity_day,activity_key,completed_count,claimed_count,level_snapshot)
  VALUES(NEW.character_id, (now() AT TIME ZONE 'Asia/Ho_Chi_Minh')::date, activity,n,0,LEAST(160,GREATEST(1,playerlevel)))
  ON CONFLICT(character_id,activity_day,activity_key)
  DO UPDATE SET completed_count=LEAST(10,player.character_findback_days.completed_count+EXCLUDED.completed_count),updated_at=now();
  RETURN NEW;
END $$;
DROP TRIGGER IF EXISTS findback_loop_round_progress ON player.character_quest_loops;
CREATE TRIGGER findback_loop_round_progress AFTER UPDATE OF ft ON player.character_quest_loops
FOR EACH ROW EXECUTE FUNCTION player.track_findback_loop_round();
