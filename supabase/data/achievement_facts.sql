create table if not exists player.character_achievement_facts (
    fact_id text primary key check (fact_id <> ''),
    character_id bigint not null references player.characters(id) on delete cascade,
    kind text not null check (kind <> ''),
    payload jsonb not null check (jsonb_typeof(payload) = 'object'),
    occurred_at timestamptz not null,
    processed_at timestamptz
);

create index if not exists idx_character_achievement_facts_pending
    on player.character_achievement_facts (character_id, occurred_at, fact_id)
    where processed_at is null;

alter table player.character_achievement_facts enable row level security;

revoke all on table player.character_achievement_facts from public, anon, authenticated;
grant select, insert, update on table player.character_achievement_facts to service_role;

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

create or replace function player.list_pending_character_achievement_facts(
    p_character_id bigint,
    p_limit integer
)
returns table (
    fact_id text,
    character_id bigint,
    kind text,
    payload jsonb,
    occurred_at timestamptz
)
language sql
stable
set search_path to ''
as $function$
    select f.fact_id, f.character_id, f.kind, f.payload, f.occurred_at
    from player.character_achievement_facts f
    where f.character_id = p_character_id
      and f.processed_at is null
      and p_limit between 1 and 100
    order by f.occurred_at, f.fact_id
    limit p_limit;
$function$;

create or replace function player.mark_character_achievement_fact_processed(
    p_fact_id text,
    p_processed_at timestamptz
)
returns boolean
language sql
set search_path to ''
as $function$
    with changed as (
        update player.character_achievement_facts
        set processed_at = p_processed_at
        where fact_id = p_fact_id
          and processed_at is null
        returning 1
    )
    select exists(select 1 from changed)
        or exists(
            select 1
            from player.character_achievement_facts
            where fact_id = p_fact_id
              and processed_at is not null
        );
$function$;

create or replace function player.update_quest_progress(
    p_id bigint,
    p_status integer,
    p_objectives jsonb,
    p_completed_at timestamptz,
    p_expires_at timestamptz,
    p_completion_count integer,
    p_last_reset timestamptz
)
returns boolean
language plpgsql
set search_path to ''
as $function$
declare
    v_character_id bigint;
    v_quest_id integer;
    v_previous_completion_count integer;
begin
    select q.character_id, q.quest_id, q.completion_count
    into v_character_id, v_quest_id, v_previous_completion_count
    from player.character_quests q
    where q.id = p_id
    for update;

    if not found then
        return false;
    end if;

    update player.character_quests
    set status = p_status,
        objectives = p_objectives,
        completed_at = p_completed_at,
        expires_at = p_expires_at,
        completion_count = p_completion_count,
        last_reset = p_last_reset
    where id = p_id;

    if p_status = 1
       and p_completion_count = v_previous_completion_count + 1
       and p_completed_at is not null then
        insert into player.character_achievement_facts (
            fact_id,
            character_id,
            kind,
            payload,
            occurred_at
        ) values (
            'quest-completed:v1:' || p_id::text || ':' || p_completion_count::text,
            v_character_id,
            'quest-completed',
            jsonb_build_object(
                'progress_id', p_id,
                'quest_id', v_quest_id,
                'completion_count', p_completion_count
            ),
            p_completed_at
        ) on conflict (fact_id) do nothing;
    end if;

    return true;
end;
$function$;

revoke execute on function player.list_pending_character_achievement_facts(bigint, integer) from public, anon, authenticated;
grant execute on function player.list_pending_character_achievement_facts(bigint, integer) to service_role;
revoke execute on function player.mark_character_achievement_fact_processed(text, timestamptz) from public, anon, authenticated;
grant execute on function player.mark_character_achievement_fact_processed(text, timestamptz) to service_role;
revoke execute on function player.append_character_achievement_fact(text, bigint, text, jsonb, timestamptz) from public, anon, authenticated;
grant execute on function player.append_character_achievement_fact(text, bigint, text, jsonb, timestamptz) to service_role;
