set check_function_bodies = off;

create or replace function player.append_character_achievement_fact(
    p_fact_id text,
    p_character_id bigint,
    p_kind text,
    p_payload jsonb,
    p_occurred_at timestamptz
)
returns boolean
language sql
set search_path to ''
as $function$
    with inserted as (
        insert into player.character_achievement_facts (
            fact_id,
            character_id,
            kind,
            payload,
            occurred_at
        ) values (
            p_fact_id,
            p_character_id,
            p_kind,
            p_payload,
            p_occurred_at
        )
        on conflict (fact_id) do nothing
        returning 1
    )
    select exists(select 1 from inserted)
        or exists(
            select 1
            from player.character_achievement_facts
            where fact_id = p_fact_id
              and character_id = p_character_id
              and kind = p_kind
              and payload = p_payload
              and occurred_at = p_occurred_at
        );
$function$;

revoke execute on function player.append_character_achievement_fact(text, bigint, text, jsonb, timestamptz) from public, anon, authenticated;
grant execute on function player.append_character_achievement_fact(text, bigint, text, jsonb, timestamptz) to service_role;
