-- A bounty award is eligible only after a trustworthy server event explicitly
-- records the matching historical day, completion count, and bounty color.
-- NULL is intentionally ineligible; no retroactive entitlements are fabricated.
BEGIN;
ALTER TABLE player.character_findback_days
 ADD COLUMN IF NOT EXISTS bounty_color SMALLINT;
ALTER TABLE player.character_findback_days
 DROP CONSTRAINT IF EXISTS findback_bounty_color_range;
ALTER TABLE player.character_findback_days
 ADD CONSTRAINT findback_bounty_color_range CHECK (bounty_color IS NULL OR bounty_color BETWEEN 0 AND 5);
COMMIT;
