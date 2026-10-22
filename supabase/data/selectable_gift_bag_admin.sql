create or replace function data.box_item_award_revision(p_item_id bigint)
returns text
language sql
stable
security invoker
set search_path to ''
as $$
    select md5(coalesce(
        jsonb_agg(
            jsonb_build_array(
                a.id,
                a.item_id,
                a.award_id,
                a.type,
                a.count,
                a.rate,
                a.quality,
                a.pre_name_type,
                a.payload
            ) order by a.id
        )::text,
        '[]'
    ))
    from data.data_tbl_item_award as a
    where a.item_id = p_item_id;
$$;

create or replace function data.box_item_admin_save(
    p_item_id bigint,
    p_mode text,
    p_selection_count integer,
    p_expected_revision text,
    p_awards jsonb
)
returns void
language plpgsql
security invoker
set search_path to ''
as $$
declare
    v_current_revision text;
    v_next_id bigint;
    v_award_count integer;
begin
    if p_item_id <= 0 or p_mode not in ('award', 'selectable') then
        raise exception 'invalid box item configuration';
    end if;
    if jsonb_typeof(p_awards) <> 'array' then
        raise exception 'awards must be an array';
    end if;

    v_award_count := jsonb_array_length(p_awards);
    if v_award_count > 200 then
        raise exception 'invalid award count';
    end if;
    if p_mode = 'selectable'
       and (v_award_count < 1 or p_selection_count < 1 or p_selection_count > v_award_count) then
        raise exception 'invalid selection count';
    end if;

    perform 1
    from data.data_tbl_item_template as i
    where i.id = p_item_id
      and (
        p_mode = 'award'
        or (i.type in (508, 550) and i.use_type = 1)
      )
    for update;
    if not found then
        raise exception 'invalid selectable package item';
    end if;

    lock table data.data_tbl_item_award in share row exclusive mode;
    v_current_revision := data.box_item_award_revision(p_item_id);
    if coalesce(p_expected_revision, '') <> v_current_revision then
        raise exception 'stale box item award revision';
    end if;

    if exists (
        select 1
        from jsonb_array_elements(p_awards) as entry
        where (entry->>'award_id') is null
           or (entry->>'type') is null
           or (entry->>'count') is null
           or (entry->>'rate') is null
           or (entry->>'quality') is null
           or (entry->>'pre_name_type') is null
           or jsonb_typeof(coalesce(entry->'payload', '{}'::jsonb)) <> 'object'
           or coalesce((entry->>'id')::bigint, 0) < 0
           or (entry->>'award_id')::bigint < 0
           or (entry->>'count')::integer < 1
           or (entry->>'count')::integer > 100000000
           or (entry->>'rate')::integer < 0
           or (entry->>'quality')::integer not between 0 and 25
           or (entry->>'pre_name_type')::integer not between 0 and 5
           or (p_mode = 'selectable' and (entry->>'type')::integer not in (12, 19, 29))
    ) then
        raise exception 'invalid award row';
    end if;

    if exists (
        select 1
        from jsonb_array_elements(p_awards) as entry
        where coalesce((entry->>'id')::bigint, 0) > 0
          and not exists (
              select 1
              from data.data_tbl_item_award as current_award
              where current_award.id = (entry->>'id')::bigint
                and current_award.item_id = p_item_id
          )
    ) then
        raise exception 'award row does not belong to package';
    end if;

    if exists (
        select 1
        from jsonb_array_elements(p_awards) as entry
        where coalesce((entry->>'id')::bigint, 0) > 0
        group by (entry->>'id')::bigint
        having count(*) > 1
    ) then
        raise exception 'duplicate award row id';
    end if;

    if p_mode = 'selectable' and exists (
        select 1
        from jsonb_array_elements(p_awards) as entry
        where case (entry->>'type')::integer
            when 12 then not exists (
                select 1 from data.data_tbl_creature as c
                where c.id = (entry->>'award_id')::bigint
            )
            when 19 then not exists (
                select 1 from data.data_tbl_equipt_template as e
                where e.id = (entry->>'award_id')::bigint
            )
            when 29 then not exists (
                select 1 from data.data_tbl_item_template as i
                where i.id = (entry->>'award_id')::bigint
            )
            else true
        end
    ) then
        raise exception 'selectable reward template does not exist';
    end if;

    delete from data.data_tbl_item_award
    where item_id = p_item_id;

    select greatest(
        coalesce((select max(id) from data.data_tbl_item_award), 0),
        coalesce((
            select max((entry->>'id')::bigint)
            from jsonb_array_elements(p_awards) as entry
        ), 0)
    ) + 1 into v_next_id;

    insert into data.data_tbl_item_award (
        id,
        item_id,
        award_id,
        type,
        count,
        rate,
        quality,
        pre_name_type,
        payload
    )
    select
        case
            when coalesce((entry.value->>'id')::bigint, 0) > 0 then
                (entry.value->>'id')::bigint
            else v_next_id + sum(
                case
                    when coalesce((entry.value->>'id')::bigint, 0) = 0 then 1
                    else 0
                end
            ) over (order by entry.ordinality) - 1
        end,
        p_item_id,
        (entry.value->>'award_id')::numeric,
        (entry.value->>'type')::numeric,
        (entry.value->>'count')::numeric,
        (entry.value->>'rate')::numeric,
        (entry.value->>'quality')::numeric,
        (entry.value->>'pre_name_type')::numeric,
        case
            when p_mode = 'selectable' then
                (coalesce(entry.value->'payload', '{}'::jsonb) - 'guaranteed' - 'countMin' - 'countMax')
                || jsonb_build_object(
                    'selection',
                    jsonb_build_object(
                        'order', entry.ordinality - 1,
                        'getNum', p_selection_count
                    )
                )
            else coalesce(entry.value->'payload', '{}'::jsonb) - 'selection'
        end
    from jsonb_array_elements(p_awards) with ordinality as entry(value, ordinality);
end;
$$;

revoke all on function data.box_item_award_revision(bigint) from public, anon, authenticated;
revoke all on function data.box_item_admin_save(bigint, text, integer, text, jsonb) from public, anon, authenticated;
grant execute on function data.box_item_award_revision(bigint) to service_role;
grant execute on function data.box_item_admin_save(bigint, text, integer, text, jsonb) to service_role;
