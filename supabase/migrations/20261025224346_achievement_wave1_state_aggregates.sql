set check_function_bodies = off;

create index idx_pet_arena_battles_winner_created
    on player.pet_arena_battles (winner_id, created_at)
    where winner_id is not null;

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

revoke execute on function player.get_achievement_marriage_state(bigint) from public, anon, authenticated;

grant execute on function player.get_achievement_marriage_state(bigint) to service_role;

revoke execute on function player.get_achievement_pet_arena_state(bigint) from public, anon, authenticated;

grant execute on function player.get_achievement_pet_arena_state(bigint) to service_role;
