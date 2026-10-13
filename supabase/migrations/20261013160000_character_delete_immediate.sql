-- MCGame: immediate character deletion (after valid secondary password on server).
-- Requires the preexisting player.delete_character(bigint) RETURNS boolean.
-- PostgreSQL transaction is atomic: any failure rolls back the deletion.
-- Never discard pending requests from the old seven-day experiment silently.
DO $preflight$
DECLARE
    v_pending bigint;
BEGIN
    IF to_regprocedure('player.delete_character(bigint)') IS NULL THEN
        RAISE EXCEPTION 'Missing player.delete_character(bigint)';
    END IF;
    IF pg_get_function_result(to_regprocedure('player.delete_character(bigint)')) != 'boolean' THEN
        RAISE EXCEPTION 'player.delete_character(bigint) must return boolean';
    END IF;
    IF to_regclass('player.character_deletion_requests') IS NOT NULL THEN
        EXECUTE 'select count(*) from player.character_deletion_requests' INTO v_pending;
        IF v_pending > 0 THEN
            RAISE EXCEPTION 'There are % pending character deletion requests. Restore or process them explicitly before migrating.', v_pending;
        END IF;
    END IF;
END
$preflight$;

CREATE OR REPLACE FUNCTION player.delete_character_immediately(
    p_account_id uuid, p_character_id bigint
) RETURNS boolean
LANGUAGE plpgsql
SECURITY INVOKER
SET search_path TO ''
AS $fn$
DECLARE
    v_id bigint;
    v_deleted boolean;
BEGIN
    IF p_account_id IS NULL OR p_character_id IS NULL OR p_character_id <= 0 THEN
        RETURN false;
    END IF;
    SELECT c.id INTO v_id
    FROM player.characters AS c
    WHERE c.id = p_character_id AND c.account_id = p_account_id
    FOR UPDATE;
    IF NOT FOUND THEN
        RETURN false;
    END IF;
    SELECT player.delete_character(v_id) INTO v_deleted;
    IF NOT coalesce(v_deleted, false) THEN
        RAISE EXCEPTION 'player.delete_character(%) returned false', v_id;
    END IF;
    RETURN true;
END
$fn$;

REVOKE ALL ON FUNCTION player.delete_character_immediately(uuid, bigint) FROM PUBLIC;
REVOKE ALL ON FUNCTION player.delete_character_immediately(uuid, bigint) FROM anon, authenticated;
GRANT EXECUTE ON FUNCTION player.delete_character_immediately(uuid, bigint) TO service_role;

-- Expand-only deploy: leave legacy API/table intact until the new Go server is running.
-- Then execute tools/character_deletion_legacy_cleanup.sql after verification.
