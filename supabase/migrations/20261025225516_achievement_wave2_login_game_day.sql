set check_function_bodies = off;

create index idx_character_achievement_login_events
    on player.character_achievement_events (character_id, occurred_at)
    where event_key like 'login-day:v1:%' or event_key like 'login-day:v2:%';

create or replace function player.record_character_achievement_login(
    p_character_id bigint,
    p_updates jsonb,
    p_occurred_at timestamptz
)
returns jsonb
language plpgsql
set search_path to ''
as $function$
declare
    v_game_day date;
    v_day_start timestamptz;
    v_day_end timestamptz;
begin
    if p_character_id <= 0 or p_occurred_at is null or jsonb_typeof(p_updates) <> 'array' then
        raise exception 'invalid achievement login event';
    end if;

    v_game_day := (p_occurred_at at time zone 'Asia/Ho_Chi_Minh')::date;
    v_day_start := v_game_day::timestamp at time zone 'Asia/Ho_Chi_Minh';
    v_day_end := (v_game_day + 1)::timestamp at time zone 'Asia/Ho_Chi_Minh';

    perform pg_catalog.pg_advisory_xact_lock(
        pg_catalog.hashtextextended('achievement-login:' || p_character_id::text, 0)
    );

    if exists (
        select 1
        from player.character_achievement_events e
        where e.character_id = p_character_id
          and (e.event_key like 'login-day:v1:%' or e.event_key like 'login-day:v2:%')
          and e.occurred_at >= v_day_start
          and e.occurred_at < v_day_end
    ) then
        return jsonb_build_object('requirements', '[]'::jsonb, 'completed', '[]'::jsonb);
    end if;

    return player.record_character_achievement_event(
        p_character_id,
        'login-day:v2:' || v_game_day::text,
        p_updates,
        p_occurred_at
    );
end;
$function$;

revoke execute on function player.record_character_achievement_login(bigint, jsonb, timestamptz) from public, anon, authenticated;

grant execute on function player.record_character_achievement_login(bigint, jsonb, timestamptz) to service_role;
