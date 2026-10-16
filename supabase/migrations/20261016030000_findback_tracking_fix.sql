-- Correct stage2's per-round counting: a FindBack activity is a COMPLETED
-- loop cycle, not an individual quest child. The last child deactivates the
-- loop without advancing ft, so detect the terminal transition as well.
-- The historic stage2 data cannot reliably be converted into cycle counts.
-- DO NOT claim rewards based on stage2 values; archive them as invalid.
-- No historical entitlement is fabricated by this migration.
BEGIN;
CREATE TABLE IF NOT EXISTS player.character_findback_stage2_round_archive (
    character_id BIGINT NOT NULL,
    activity_day DATE NOT NULL,
    activity_key TEXT NOT NULL,
    old_completed_count INTEGER NOT NULL,
    archived_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY(character_id,activity_day,activity_key)
);
INSERT INTO player.character_findback_stage2_round_archive(character_id,activity_day,activity_key,old_completed_count)
SELECT character_id,activity_day,activity_key,completed_count
FROM player.character_findback_days
WHERE activity_key IN ('xiuxing','chongwu','panjun','feimo')
ON CONFLICT DO NOTHING;
-- Clearing ambiguous counters avoids wrongful retroactive awards.
-- Preserve claimed rows, and never produce completed_count below claimed_count.
UPDATE player.character_findback_days
SET completed_count=0, updated_at=now()
WHERE activity_key IN ('xiuxing','chongwu','panjun','feimo') AND claimed_count=0;
CREATE OR REPLACE FUNCTION player.track_findback_loop_round() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    activity TEXT;
    cycle_size INT;
    playerlevel INT;
    finished BOOLEAN;
    vietnam_day DATE;
BEGIN
    IF NEW.character_id IS DISTINCT FROM OLD.character_id
       OR NEW.loop_id IS DISTINCT FROM OLD.loop_id THEN
        RETURN NEW;
    END IF;
    SELECT x.activity, x.cycle_size INTO activity,cycle_size
    FROM (VALUES
        (6,'xiuxing',20),
        (5,'chongwu',20),
        (4,'panjun',10),
        (16,'feimo',10)
    ) AS x(loop_id,activity,cycle_size) WHERE x.loop_id=NEW.loop_id;
    IF activity IS NULL THEN RETURN NEW; END IF;
    -- The last quest finishes a cycle; LoopRepository.Deactivate flips active
    -- to false without modifying ft. A premature cancel also deactivates,
    -- but then ft is smaller than cycle_size - 1 and must NOT be counted.
    finished := OLD.active AND NOT NEW.active AND OLD.ft=cycle_size-1;
    IF NOT finished THEN RETURN NEW; END IF;
    vietnam_day := (now() AT TIME ZONE 'Asia/Ho_Chi_Minh')::date;
    SELECT level INTO playerlevel
    FROM player.character_progression WHERE character_id=NEW.character_id;
    IF playerlevel IS NULL THEN RETURN NEW; END IF;
    INSERT INTO player.character_findback_days
        (character_id,activity_day,activity_key,completed_count,claimed_count,level_snapshot)
    VALUES(NEW.character_id,vietnam_day,activity,1,0,LEAST(160,GREATEST(1,playerlevel)))
    ON CONFLICT (character_id,activity_day,activity_key)
    DO UPDATE SET
        completed_count=LEAST(10,player.character_findback_days.completed_count+1),
        updated_at=now();
    RETURN NEW;
END $$;
DROP TRIGGER IF EXISTS findback_loop_round_progress ON player.character_quest_loops;
CREATE TRIGGER findback_loop_round_progress
AFTER UPDATE OF active, ft ON player.character_quest_loops
FOR EACH ROW EXECUTE FUNCTION player.track_findback_loop_round();
COMMIT;
