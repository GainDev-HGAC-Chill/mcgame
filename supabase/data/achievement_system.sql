create table if not exists player.character_achievement_requirements (
    character_id bigint not null references player.characters(id) on delete cascade,
    requirement_id bigint not null,
    achievement_id integer not null,
    progress integer not null default 0 check (progress >= 0),
    target integer not null check (target >= 0),
    is_done boolean not null default false,
    updated_at timestamptz not null default current_timestamp,
    primary key (character_id, requirement_id)
);

create index if not exists idx_character_achievement_requirements_achievement
    on player.character_achievement_requirements (character_id, achievement_id);

create index if not exists idx_pet_arena_battles_winner_created
    on player.pet_arena_battles (winner_id, created_at)
    where winner_id is not null;

alter table player.character_achievement_requirements enable row level security;

create table if not exists player.character_achievement_events (
    character_id bigint not null references player.characters(id) on delete cascade,
    event_key text not null check (event_key <> ''),
    occurred_at timestamptz not null,
    created_at timestamptz not null default current_timestamp,
    primary key (character_id, event_key)
);

create index if not exists idx_character_achievement_login_events
    on player.character_achievement_events (character_id, occurred_at)
    where event_key like 'login-day:v1:%' or event_key like 'login-day:v2:%';

alter table player.character_achievement_events enable row level security;

create table if not exists player.character_achievement_claim_reservations (
    character_id bigint not null references player.characters(id) on delete cascade,
    achievement_id integer not null,
    claim_token uuid not null unique,
    reserved_at timestamptz not null,
    primary key (character_id, achievement_id)
);

alter table player.character_achievement_claim_reservations enable row level security;

create table if not exists player.character_achievement_reward_grants (
    character_id bigint not null references player.characters(id) on delete cascade,
    achievement_id integer not null,
    claim_token uuid not null unique,
    reward_kind smallint not null check (reward_kind between 1 and 3),
    template_id integer not null check (template_id > 0),
    quantity integer not null check (quantity > 0),
    status text not null check (status in ('reserved', 'granting', 'granted', 'finalized', 'failed')),
    receipt jsonb,
    last_error text not null default '',
    lease_expires_at timestamptz,
    created_at timestamptz not null default current_timestamp,
    updated_at timestamptz not null default current_timestamp,
    granted_at timestamptz,
    finalized_at timestamptz,
    primary key (character_id, achievement_id)
);

alter table player.character_achievement_reward_grants enable row level security;

revoke all on table player.character_achievement_reward_grants from public, anon, authenticated;
grant select, insert, update on table player.character_achievement_reward_grants to service_role;

create unique index if not exists idx_character_items_achievement_reward_key
    on player.character_items (character_id, ((properties ->> 'achievement_reward_key')))
    where properties ? 'achievement_reward_key';

create or replace function player.list_character_achievement_requirements(p_character_id bigint)
returns table (
    character_id bigint,
    requirement_id bigint,
    achievement_id bigint,
    progress integer,
    target integer,
    is_done boolean,
    updated_at timestamptz
)
language sql
set search_path to ''
as $function$
    select
        r.character_id,
        r.requirement_id,
        r.achievement_id::bigint,
        r.progress,
        r.target,
        r.is_done,
        r.updated_at
    from player.character_achievement_requirements r
    where r.character_id = p_character_id
    order by r.requirement_id;
$function$;

create or replace function player.get_achievement_marriage_state(p_character_id bigint)
returns table (
    normal_weddings integer,
    marriage_count integer,
    current_married_days integer
)
language sql
stable
set search_path to ''
as $function$
    with current_marriage as (
        select m.ring_type, m.married_at
        from player.marriages m
        where m.divorced_at is null
          and (m.partner1_id = p_character_id or m.partner2_id = p_character_id)
        limit 1
    ), history as (
        select
            count(*)::integer as marriage_count,
            count(*) filter (where (a.payload->>'ring_type')::integer = 1)::integer as normal_weddings
        from player.marriage_audit_log a
        where a.event = 'married'
          and (a.partner1_id = p_character_id or a.partner2_id = p_character_id)
    )
    select
        greatest(h.normal_weddings, case when c.ring_type = 1 then 1 else 0 end)::integer,
        greatest(h.marriage_count, case when c.married_at is not null then 1 else 0 end)::integer,
        coalesce(
            (current_timestamp at time zone 'Asia/Ho_Chi_Minh')::date
                - (c.married_at at time zone 'Asia/Ho_Chi_Minh')::date,
            0
        )::integer
    from history h
    left join current_marriage c on true;
$function$;

create or replace function player.get_achievement_pet_arena_state(p_character_id bigint)
returns table (
    total_wins integer,
    fifteen_win_day_streak integer,
    twenty_win_day_streak integer
)
language sql
stable
set search_path to ''
as $function$
    with daily_wins as (
        select
            (b.created_at at time zone 'Asia/Ho_Chi_Minh')::date as win_day,
            count(*)::integer as wins
        from player.pet_arena_battles b
        where b.winner_id = p_character_id
        group by (b.created_at at time zone 'Asia/Ho_Chi_Minh')::date
    ), fifteen_groups as (
        select win_day, win_day - row_number() over (order by win_day)::integer as island
        from daily_wins
        where wins >= 15
    ), twenty_groups as (
        select win_day, win_day - row_number() over (order by win_day)::integer as island
        from daily_wins
        where wins >= 20
    ), fifteen_streak as (
        select coalesce(max(day_count), 0)::integer as value
        from (
            select count(*)::integer as day_count
            from fifteen_groups
            group by island
        ) grouped
    ), twenty_streak as (
        select coalesce(max(day_count), 0)::integer as value
        from (
            select count(*)::integer as day_count
            from twenty_groups
            group by island
        ) grouped
    )
    select
        coalesce((select sum(wins) from daily_wins), 0)::integer,
        (select value from fifteen_streak),
        (select value from twenty_streak);
$function$;

revoke execute on function player.get_achievement_marriage_state(bigint) from public, anon, authenticated;
grant execute on function player.get_achievement_marriage_state(bigint) to service_role;
revoke execute on function player.get_achievement_pet_arena_state(bigint) from public, anon, authenticated;
grant execute on function player.get_achievement_pet_arena_state(bigint) to service_role;

create or replace function player.record_character_achievement_event(
    p_character_id bigint,
    p_event_key text,
    p_updates jsonb,
    p_occurred_at timestamptz
)
returns jsonb
language plpgsql
set search_path to ''
as $function$
declare
    v_inserted integer;
    v_requirements jsonb := '[]'::jsonb;
    v_completed jsonb := '[]'::jsonb;
begin
    if p_character_id <= 0 or p_event_key is null or p_event_key = '' or jsonb_typeof(p_updates) <> 'array' then
        raise exception 'invalid achievement event';
    end if;

    insert into player.character_achievement_events (character_id, event_key, occurred_at)
    values (p_character_id, p_event_key, p_occurred_at)
    on conflict do nothing;
    get diagnostics v_inserted = row_count;

    if v_inserted = 0 then
        return jsonb_build_object('requirements', '[]'::jsonb, 'completed', '[]'::jsonb);
    end if;

    with requested as (
        select
            u.requirement_id,
            u.achievement_id,
            greatest(coalesce(u.delta, 0), 0) as delta,
            u.absolute,
            greatest(coalesce(u.target, 0), 0) as requested_target,
            u.done
        from jsonb_to_recordset(p_updates) as u(
            requirement_id bigint,
            achievement_id bigint,
            delta integer,
            absolute integer,
            target integer,
            done boolean
        )
    ), canonical as (
        select
            q.requirement_id,
            q.achievement_id::integer,
            q.delta,
            greatest(coalesce(q.absolute, 0), 0) as absolute,
            q.absolute is not null as has_absolute,
            coalesce(q.done, false) as force_done,
            greatest(r.num::integer, q.requested_target, 0) as target
        from requested q
        join data.data_tbl_achievement_require r
          on r.id = q.requirement_id
         and r.aid::bigint = q.achievement_id
    ), changed as (
        insert into player.character_achievement_requirements (
            character_id,
            requirement_id,
            achievement_id,
            progress,
            target,
            is_done,
            updated_at
        )
        select
            p_character_id,
            c.requirement_id,
            c.achievement_id,
            case
                when c.target > 0 then least(case when c.has_absolute then c.absolute else c.delta end, c.target)
                else case when c.has_absolute then c.absolute else c.delta end
            end,
            c.target,
            c.force_done or (c.target > 0 and (case when c.has_absolute then c.absolute else c.delta end) >= c.target),
            p_occurred_at
        from canonical c
        on conflict (character_id, requirement_id) do update set
            progress = case
                when excluded.target > 0 then least(
                    case
                        when (select x.has_absolute from canonical x where x.requirement_id = excluded.requirement_id)
                            then greatest(player.character_achievement_requirements.progress, excluded.progress)
                        else player.character_achievement_requirements.progress + excluded.progress
                    end,
                    excluded.target
                )
                else case
                    when (select x.has_absolute from canonical x where x.requirement_id = excluded.requirement_id)
                        then greatest(player.character_achievement_requirements.progress, excluded.progress)
                    else player.character_achievement_requirements.progress + excluded.progress
                end
            end,
            target = excluded.target,
            is_done = player.character_achievement_requirements.is_done
                or excluded.is_done
                or (
                    excluded.target > 0
                    and case
                        when (select x.has_absolute from canonical x where x.requirement_id = excluded.requirement_id)
                            then greatest(player.character_achievement_requirements.progress, excluded.progress)
                        else player.character_achievement_requirements.progress + excluded.progress
                    end >= excluded.target
                ),
            updated_at = excluded.updated_at
        returning *
    )
    select coalesce(jsonb_agg(jsonb_build_object(
        'character_id', c.character_id,
        'requirement_id', c.requirement_id,
        'achievement_id', c.achievement_id,
        'done', c.is_done,
        'progress', c.progress,
        'target', c.target,
        'updated_at', c.updated_at
    ) order by c.requirement_id), '[]'::jsonb)
    into v_requirements
    from changed c;

    with touched as (
        select distinct (value->>'achievement_id')::integer as achievement_id
        from jsonb_array_elements(v_requirements)
    ), newly_completed as (
        insert into player.character_achievements (
            character_id,
            achievement_id,
            progress,
            target,
            is_completed,
            is_claimed,
            completed_at,
            claimed_at
        )
        select
            p_character_id,
            t.achievement_id,
            count(*) filter (where rp.is_done)::integer,
            count(*)::integer,
            true,
            false,
            p_occurred_at,
            null
        from touched t
        join data.data_tbl_achievement_require d on d.aid::integer = t.achievement_id
        left join player.character_achievement_requirements rp
          on rp.character_id = p_character_id
         and rp.requirement_id = d.id
        group by t.achievement_id
        having bool_and(coalesce(rp.is_done, false))
        on conflict (character_id, achievement_id) do update set
            progress = excluded.progress,
            target = excluded.target,
            is_completed = true,
            completed_at = coalesce(player.character_achievements.completed_at, excluded.completed_at)
        where player.character_achievements.is_completed = false
        returning achievement_id, completed_at
    )
    select coalesce(jsonb_agg(jsonb_build_object(
        'achievement_id', achievement_id,
        'done_at', floor(extract(epoch from completed_at) * 1000)::bigint
    ) order by achievement_id), '[]'::jsonb)
    into v_completed
    from newly_completed;

    return jsonb_build_object('requirements', v_requirements, 'completed', v_completed);
exception
    when others then
        delete from player.character_achievement_events
        where character_id = p_character_id and event_key = p_event_key;
        raise;
end;
$function$;

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

create or replace function player.reserve_character_achievement_claim(
    p_character_id bigint,
    p_achievement_id bigint,
    p_claim_token uuid,
    p_reserved_at timestamptz
)
returns boolean
language plpgsql
set search_path to ''
as $function$
declare
    v_inserted integer;
begin
    if not exists (
        select 1
        from player.character_achievements a
        where a.character_id = p_character_id
          and a.achievement_id = p_achievement_id
          and a.is_completed = true
          and a.is_claimed = false
    ) then
        return false;
    end if;

    insert into player.character_achievement_claim_reservations (
        character_id,
        achievement_id,
        claim_token,
        reserved_at
    ) values (
        p_character_id,
        p_achievement_id,
        p_claim_token,
        p_reserved_at
    ) on conflict do nothing;
    get diagnostics v_inserted = row_count;
    return v_inserted = 1;
end;
$function$;

create or replace function player.finalize_character_achievement_claim(
    p_character_id bigint,
    p_achievement_id bigint,
    p_claim_token uuid,
    p_claimed_at timestamptz
)
returns setof player.character_achievements
language plpgsql
set search_path to ''
as $function$
begin
    if not exists (
        select 1
        from player.character_achievement_claim_reservations r
        where r.character_id = p_character_id
          and r.achievement_id = p_achievement_id
          and r.claim_token = p_claim_token
        for update
    ) then
        return;
    end if;

    return query
    update player.character_achievements a
    set is_claimed = true,
        claimed_at = p_claimed_at
    where a.character_id = p_character_id
      and a.achievement_id = p_achievement_id
      and a.is_completed = true
      and a.is_claimed = false
    returning a.*;

    if found then
        delete from player.character_achievement_claim_reservations r
        where r.character_id = p_character_id
          and r.achievement_id = p_achievement_id
          and r.claim_token = p_claim_token;
    end if;
end;
$function$;

create or replace function player.release_character_achievement_claim(
    p_character_id bigint,
    p_achievement_id bigint,
    p_claim_token uuid
)
returns boolean
language sql
set search_path to ''
as $function$
    with deleted as (
        delete from player.character_achievement_claim_reservations
        where character_id = p_character_id
          and achievement_id = p_achievement_id
          and claim_token = p_claim_token
        returning 1
    )
    select exists(select 1 from deleted);
$function$;

create or replace function player.prepare_character_achievement_reward_grant(
    p_character_id bigint,
    p_achievement_id bigint,
    p_reward_kind integer,
    p_template_id integer,
    p_quantity integer,
    p_claim_token uuid,
    p_prepared_at timestamptz
)
returns setof player.character_achievement_reward_grants
language plpgsql
set search_path to ''
as $function$
declare
    v_grant player.character_achievement_reward_grants%rowtype;
begin
    perform pg_advisory_xact_lock(p_character_id);

    select * into v_grant
    from player.character_achievement_reward_grants g
    where g.character_id = p_character_id
      and g.achievement_id = p_achievement_id
    for update;

    if not found then
        if not exists (
            select 1
            from player.character_achievements a
            where a.character_id = p_character_id
              and a.achievement_id = p_achievement_id
              and a.is_completed = true
              and a.is_claimed = false
        ) then
            return;
        end if;

        insert into player.character_achievement_reward_grants (
            character_id, achievement_id, claim_token, reward_kind,
            template_id, quantity, status, created_at, updated_at
        ) values (
            p_character_id, p_achievement_id, p_claim_token, p_reward_kind,
            p_template_id, p_quantity, 'reserved', p_prepared_at, p_prepared_at
        ) returning * into v_grant;
    end if;

    if v_grant.reward_kind <> p_reward_kind
       or v_grant.template_id <> p_template_id
       or v_grant.quantity <> p_quantity then
        raise exception 'achievement reward definition drift for character %, achievement %', p_character_id, p_achievement_id;
    end if;

    if v_grant.status in ('granted', 'finalized') then
        return next v_grant;
        return;
    end if;

    if v_grant.status = 'granting'
       and v_grant.lease_expires_at is not null
       and v_grant.lease_expires_at > p_prepared_at then
        return next v_grant;
        return;
    end if;

    update player.character_achievement_reward_grants g
    set claim_token = p_claim_token,
        status = 'granting',
        last_error = '',
        lease_expires_at = p_prepared_at + interval '5 minutes',
        updated_at = p_prepared_at
    where g.character_id = p_character_id
      and g.achievement_id = p_achievement_id
    returning * into v_grant;

    return next v_grant;
end;
$function$;

create or replace function player.mark_character_achievement_reward_granted(
    p_character_id bigint,
    p_achievement_id bigint,
    p_claim_token uuid,
    p_receipt jsonb,
    p_granted_at timestamptz
)
returns setof player.character_achievement_reward_grants
language sql
set search_path to ''
as $function$
    update player.character_achievement_reward_grants g
    set status = 'granted',
        receipt = p_receipt,
        lease_expires_at = null,
        updated_at = p_granted_at,
        granted_at = coalesce(g.granted_at, p_granted_at)
    where g.character_id = p_character_id
      and g.achievement_id = p_achievement_id
      and g.claim_token = p_claim_token
      and g.status = 'granting'
    returning g.*;
$function$;

create or replace function player.mark_character_achievement_reward_failed(
    p_character_id bigint,
    p_achievement_id bigint,
    p_claim_token uuid,
    p_reason text,
    p_failed_at timestamptz
)
returns boolean
language sql
set search_path to ''
as $function$
    with updated as (
        update player.character_achievement_reward_grants g
        set status = 'failed',
            last_error = left(coalesce(p_reason, 'reward grant failed'), 1000),
            lease_expires_at = null,
            updated_at = p_failed_at
        where g.character_id = p_character_id
          and g.achievement_id = p_achievement_id
          and g.claim_token = p_claim_token
          and g.status = 'granting'
        returning 1
    )
    select exists(select 1 from updated);
$function$;

create or replace function player.finalize_character_achievement_reward_grant(
    p_character_id bigint,
    p_achievement_id bigint,
    p_finalized_at timestamptz
)
returns setof player.character_achievements
language plpgsql
set search_path to ''
as $function$
declare
    v_status text;
begin
    perform pg_advisory_xact_lock(p_character_id);

    select g.status into v_status
    from player.character_achievement_reward_grants g
    where g.character_id = p_character_id
      and g.achievement_id = p_achievement_id
    for update;

    if v_status = 'finalized' then
        return query
        select a.*
        from player.character_achievements a
        where a.character_id = p_character_id
          and a.achievement_id = p_achievement_id
          and a.is_claimed = true;
        return;
    end if;

    if v_status <> 'granted' then
        return;
    end if;

    return query
    update player.character_achievements a
    set is_claimed = true,
        claimed_at = p_finalized_at
    where a.character_id = p_character_id
      and a.achievement_id = p_achievement_id
      and a.is_completed = true
      and a.is_claimed = false
    returning a.*;

    if found then
        update player.character_achievement_reward_grants g
        set status = 'finalized',
            finalized_at = p_finalized_at,
            updated_at = p_finalized_at
        where g.character_id = p_character_id
          and g.achievement_id = p_achievement_id;
    end if;
end;
$function$;

create or replace function player.list_granted_character_achievement_rewards(
    p_character_id bigint,
    p_limit integer default 100
)
returns setof player.character_achievement_reward_grants
language sql
stable
set search_path to ''
as $function$
    select g.*
    from player.character_achievement_reward_grants g
    where g.character_id = p_character_id
      and g.status = 'granted'
    order by g.granted_at, g.achievement_id
    limit greatest(1, least(coalesce(p_limit, 100), 1000));
$function$;

revoke execute on function player.prepare_character_achievement_reward_grant(bigint, bigint, integer, integer, integer, uuid, timestamptz) from public, anon, authenticated;
grant execute on function player.prepare_character_achievement_reward_grant(bigint, bigint, integer, integer, integer, uuid, timestamptz) to service_role;
revoke execute on function player.mark_character_achievement_reward_granted(bigint, bigint, uuid, jsonb, timestamptz) from public, anon, authenticated;
grant execute on function player.mark_character_achievement_reward_granted(bigint, bigint, uuid, jsonb, timestamptz) to service_role;
revoke execute on function player.mark_character_achievement_reward_failed(bigint, bigint, uuid, text, timestamptz) from public, anon, authenticated;
grant execute on function player.mark_character_achievement_reward_failed(bigint, bigint, uuid, text, timestamptz) to service_role;
revoke execute on function player.finalize_character_achievement_reward_grant(bigint, bigint, timestamptz) from public, anon, authenticated;
grant execute on function player.finalize_character_achievement_reward_grant(bigint, bigint, timestamptz) to service_role;
revoke execute on function player.list_granted_character_achievement_rewards(bigint, integer) from public, anon, authenticated;
grant execute on function player.list_granted_character_achievement_rewards(bigint, integer) to service_role;
