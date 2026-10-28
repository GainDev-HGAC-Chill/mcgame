set check_function_bodies = off;

create or replace function "player"."complete_guild_loop_atomic"(
    p_character_id bigint,
    p_progress_id bigint,
    p_quest_id integer,
    p_expected_ft integer,
    p_money bigint,
    p_money_bind bigint,
    p_contribution bigint,
    p_guild_exp bigint,
    p_items jsonb,
    p_pet_ids bigint[],
    p_next_quest_id integer,
    p_next_objectives jsonb,
    p_next_started_at timestamptz
)
returns table(
    guild_id bigint,
    money bigint,
    money_bind bigint,
    gold bigint,
    guild_contrib bigint,
    donate_contrib bigint,
    restore_normal bigint,
    restore_donate bigint,
    member_normal bigint,
    member_total bigint,
    member_weekly bigint,
    guild_experience bigint,
    ft integer,
    active boolean,
    item_states jsonb,
    consumed_pet_ids bigint[],
    next_progress_id bigint,
    next_quest_id integer,
    next_objectives jsonb,
    next_started_at timestamptz,
    diary_task_count integer,
    diary_vitality integer,
    diary_vitality_delta integer
)
language plpgsql
security invoker
set search_path to ''
as $function$
declare
    v_character "player"."characters"%rowtype;
    v_wallet "player"."character_wallet"%rowtype;
    v_member "player"."guild_members"%rowtype;
    v_loop "player"."character_quest_loops"%rowtype;
    v_progress "player"."character_quests"%rowtype;
    v_item record;
    v_item_row "player"."character_items"%rowtype;
    v_item_remaining integer;
    v_item_states jsonb := '[]'::jsonb;
    v_guild_id bigint;
    v_guild_exp bigint;
    v_normal_balance bigint;
    v_next_progress_id bigint;
    v_affected integer;
    v_pet_count integer;
    v_diary "player"."character_daily_act"%rowtype;
    v_diary_count integer;
    v_diary_vitality integer;
    v_diary_delta integer;
begin
    perform pg_advisory_xact_lock(hashtextextended('guild-character:' || p_character_id::text, 0));
    if p_expected_ft not between 0 and 14 or p_money < 0 or p_money_bind < 0
       or p_contribution < 0 or p_guild_exp < 0 or p_items is null or jsonb_typeof(p_items) <> 'array'
       or p_pet_ids is null then
        raise exception 'INVALID_GUILD_LOOP_COMMAND';
    end if;
    if (p_expected_ft < 14 and (p_next_quest_id is null or p_next_objectives is null or p_next_started_at is null))
       or (p_expected_ft = 14 and (p_next_quest_id is not null or p_next_objectives is not null or p_next_started_at is not null)) then
        raise exception 'INVALID_GUILD_LOOP_COMMAND';
    end if;

    select c."guild_id" into v_guild_id
    from "player"."characters" as c
    where c."id" = p_character_id;
    if v_guild_id is null then
        raise exception 'NOT_IN_GUILD';
    end if;
    perform 1 from "player"."guilds" as g where g."id" = v_guild_id for update;
    if not found then
        raise exception 'NOT_IN_GUILD';
    end if;
    select * into v_character from "player"."characters" as c where c."id" = p_character_id for update;
    if not found or v_character."guild_id" is distinct from v_guild_id then
        raise exception 'NOT_IN_GUILD';
    end if;
    select * into v_wallet from "player"."character_wallet" as w where w."character_id" = p_character_id for update;
    if not found then
        raise exception 'INVALID_GUILD_LOOP_COMMAND';
    end if;
    select * into v_member
    from "player"."guild_members" as m
    where m."guild_id" = v_guild_id and m."character_id" = p_character_id
    for update;
    if not found then
        raise exception 'NOT_IN_GUILD';
    end if;
    select * into v_loop
    from "player"."character_quest_loops" as l
    where l."character_id" = p_character_id and l."loop_id" = 2
    for update;
    if not found or not v_loop."active" or v_loop."active_quest_id" <> p_quest_id or v_loop."ft" <> p_expected_ft then
        raise exception 'GUILD_LOOP_STATE_CHANGED';
    end if;
    select * into v_progress
    from "player"."character_quests" as q
    where q."id" = p_progress_id and q."character_id" = p_character_id and q."quest_id" = p_quest_id
    for update;
    if not found or v_progress."status" <> 0 or exists (
        select 1
        from jsonb_array_elements(v_progress."objectives") as objective(value)
        where coalesce((objective.value->>'required')::integer, 0) <= 0
           or coalesce((objective.value->>'current')::integer, 0) < coalesce((objective.value->>'required')::integer, 0)
    ) then
        raise exception 'GUILD_LOOP_OBJECTIVE_INCOMPLETE';
    end if;
    if coalesce((select sum((objective.value->>'required')::integer) from jsonb_array_elements(v_progress."objectives") as objective(value) where (objective.value->>'type')::integer = 2), 0)
       <> coalesce((select sum(parsed."count") from jsonb_to_recordset(p_items) as parsed("itemId" bigint, "templateId" integer, "count" integer)), 0)
       or exists (
           select 1
           from jsonb_array_elements(v_progress."objectives") as objective(value)
           where (objective.value->>'type')::integer = 2
             and coalesce((select sum(parsed."count") from jsonb_to_recordset(p_items) as parsed("itemId" bigint, "templateId" integer, "count" integer) where parsed."templateId" = (objective.value->>'target')::integer), 0)
                 < (objective.value->>'required')::integer
       ) then
        raise exception 'INVALID_GUILD_LOOP_COMMAND';
    end if;

    for v_item in
        select parsed."itemId" as item_id, parsed."templateId" as template_id, sum(parsed."count")::integer as consume_count
        from jsonb_to_recordset(p_items) as parsed("itemId" bigint, "templateId" integer, "count" integer)
        group by parsed."itemId", parsed."templateId"
        order by parsed."itemId"
    loop
        if v_item.consume_count <= 0 then
            raise exception 'INVALID_GUILD_LOOP_COMMAND';
        end if;
        select * into v_item_row
        from "player"."character_items" as i
        where i."id" = v_item.item_id and i."character_id" = p_character_id
        for update;
        if not found or v_item_row."template_id" <> v_item.template_id
           or v_item_row."slot_type" not in (0, 4) or v_item_row."stack_count" < v_item.consume_count then
            raise exception 'GUILD_LOOP_ITEM_CHANGED';
        end if;
        if not exists (
            select 1
            from jsonb_array_elements(v_progress."objectives") as objective(value)
            where (objective.value->>'type')::integer = 2
              and (objective.value->>'target')::integer = v_item.template_id
              and (coalesce((objective.value->>'quality')::integer, 0) < 0
                   or v_item_row."color_code" = least(5, greatest(0, (coalesce((objective.value->>'quality')::integer, 0) + 4) / 5)))
        ) then
            raise exception 'GUILD_LOOP_ITEM_CHANGED';
        end if;
        v_item_remaining := v_item_row."stack_count" - v_item.consume_count;
        if v_item_remaining = 0 then
            delete from "player"."character_items" where "id" = v_item.item_id;
        else
            update "player"."character_items" set "stack_count" = v_item_remaining where "id" = v_item.item_id;
        end if;
        v_item_states := v_item_states || jsonb_build_array(jsonb_build_object('itemId', v_item.item_id, 'remaining', v_item_remaining));
    end loop;

    select count(distinct pet_id)::integer into v_pet_count from unnest(p_pet_ids) as pet_id;
    if v_pet_count <> cardinality(p_pet_ids)
       or v_pet_count <> coalesce((select sum((objective.value->>'required')::integer) from jsonb_array_elements(v_progress."objectives") as objective(value) where (objective.value->>'type')::integer = 8), 0) then
        raise exception 'INVALID_GUILD_LOOP_COMMAND';
    end if;
    if v_pet_count > 0 then
        perform 1
        from "player"."character_pets" as p
        where p."id" = any(p_pet_ids) and p."character_id" = p_character_id
        order by p."id"
        for update;
        if (select count(*) from "player"."character_pets" as p where p."id" = any(p_pet_ids) and p."character_id" = p_character_id) <> v_pet_count then
            raise exception 'GUILD_LOOP_PET_CHANGED';
        end if;
        if exists (
            select 1
            from jsonb_array_elements(v_progress."objectives") as objective(value)
            where (objective.value->>'type')::integer = 8
              and (select count(*)
                   from "player"."character_pets" as p
                   where p."id" = any(p_pet_ids) and p."character_id" = p_character_id
                     and p."template_id" = (objective.value->>'target')::integer
                     and (case when p."grow_rate" <= 1.1 then 0 when p."grow_rate" <= 1.3 then 1 when p."grow_rate" <= 1.6 then 2 when p."grow_rate" <= 2.0 then 3 else 4 end)
                         >= (case when coalesce((objective.value->>'quality')::numeric, 0) / 10.0 <= 1.1 then 0 when coalesce((objective.value->>'quality')::numeric, 0) / 10.0 <= 1.3 then 1 when coalesce((objective.value->>'quality')::numeric, 0) / 10.0 <= 1.6 then 2 when coalesce((objective.value->>'quality')::numeric, 0) / 10.0 <= 2.0 then 3 else 4 end))
                  < (objective.value->>'required')::integer
        ) then
            raise exception 'GUILD_LOOP_PET_CHANGED';
        end if;
        delete from "player"."character_pets" as p where p."id" = any(p_pet_ids) and p."character_id" = p_character_id;
        get diagnostics v_affected = row_count;
        if v_affected <> v_pet_count then
            raise exception 'GUILD_LOOP_PET_CHANGED';
        end if;
    end if;

    if v_wallet."money" > 9223372036854775807 - p_money
       or v_wallet."money_bind" > 9223372036854775807 - p_money_bind then
        raise exception 'GUILD_LOOP_REWARD_OVERFLOW';
    end if;
    update "player"."character_wallet" as w
    set "money" = w."money" + p_money,
        "money_bind" = w."money_bind" + p_money_bind
    where w."character_id" = p_character_id
    returning w."money", w."money_bind", w."gold" into v_wallet."money", v_wallet."money_bind", v_wallet."gold";

    insert into "player"."character_currencies" ("character_id", "currency_type", "amount", "season")
    values (p_character_id, 20, p_contribution, 0)
    on conflict ("character_id", "currency_type", "season")
    do update set "amount" = "player"."character_currencies"."amount" + excluded."amount"
    returning "amount" into v_normal_balance;

    update "player"."guild_members" as m
    set "contribution_normal" = m."contribution_normal" + p_contribution,
        "contribution_total" = m."contribution_total" + p_contribution,
        "contribution_weekly" = m."contribution_weekly" + p_contribution
    where m."id" = v_member."id"
    returning m."contribution_normal", m."contribution_total", m."contribution_weekly"
    into v_member."contribution_normal", v_member."contribution_total", v_member."contribution_weekly";

    update "player"."guilds" as g
    set "experience" = g."experience" + p_guild_exp
    where g."id" = v_guild_id
    returning g."experience" into v_guild_exp;

    delete from "player"."character_quests" where "id" = p_progress_id;
    get diagnostics v_affected = row_count;
    if v_affected <> 1 then
        raise exception 'GUILD_LOOP_STATE_CHANGED';
    end if;

    if p_next_quest_id is not null then
        insert into "player"."character_quests" ("character_id", "quest_id", "status", "objectives", "started_at", "completion_count")
        values (p_character_id, p_next_quest_id, 0, p_next_objectives, p_next_started_at, 0)
        returning "id" into v_next_progress_id;
        update "player"."character_quest_loops"
        set "ft" = p_expected_ft + 1, "active_quest_id" = p_next_quest_id, "updated_at" = now()
        where "id" = v_loop."id";
    else
        update "player"."character_quest_loops"
        set "ft" = p_expected_ft + 1, "active" = false, "active_quest_id" = null, "updated_at" = now()
        where "id" = v_loop."id";
    end if;

    select * into v_diary
    from "player"."character_daily_act" as d
    where d."character_id" = p_character_id and d."day" = timezone('Asia/Ho_Chi_Minh', now())::date
    for update;
    if not found then
        v_diary_count := 1;
        v_diary_delta := 2;
        v_diary_vitality := 2;
        insert into "player"."character_daily_act" ("character_id", "day", "task_counts", "vitality_points", "updated_at")
        values (p_character_id, timezone('Asia/Ho_Chi_Minh', now())::date, jsonb_build_object('5', 1), 2, now());
    else
        v_diary_count := least(coalesce((v_diary."task_counts"->>'5')::integer, 0) + 1, 15);
        v_diary_delta := case when coalesce((v_diary."task_counts"->>'5')::integer, 0) < 15 then 2 else 0 end;
        v_diary_vitality := v_diary."vitality_points" + v_diary_delta;
        update "player"."character_daily_act"
        set "task_counts" = v_diary."task_counts" || jsonb_build_object('5', v_diary_count),
            "vitality_points" = v_diary_vitality,
            "updated_at" = now()
        where "character_id" = p_character_id and "day" = timezone('Asia/Ho_Chi_Minh', now())::date;
    end if;

    return query select v_guild_id, v_wallet."money", v_wallet."money_bind", v_wallet."gold"::bigint,
        v_normal_balance, coalesce((select c."amount" from "player"."character_currencies" as c where c."character_id" = p_character_id and c."currency_type" = 21 and c."season" = 0), 0),
        v_character."guild_restore_contrib"::bigint, v_character."guild_restore_donate"::bigint,
        v_member."contribution_normal", v_member."contribution_total", v_member."contribution_weekly", v_guild_exp,
        p_expected_ft + 1, p_next_quest_id is not null, v_item_states, p_pet_ids,
        v_next_progress_id, p_next_quest_id, p_next_objectives, p_next_started_at,
        v_diary_count, v_diary_vitality, v_diary_delta;
end;
$function$;

revoke all on function "player"."complete_guild_loop_atomic"(bigint, bigint, integer, integer, bigint, bigint, bigint, bigint, jsonb, bigint[], integer, jsonb, timestamptz) from public;
revoke all on function "player"."complete_guild_loop_atomic"(bigint, bigint, integer, integer, bigint, bigint, bigint, bigint, jsonb, bigint[], integer, jsonb, timestamptz) from anon;
revoke all on function "player"."complete_guild_loop_atomic"(bigint, bigint, integer, integer, bigint, bigint, bigint, bigint, jsonb, bigint[], integer, jsonb, timestamptz) from authenticated;
grant execute on function "player"."complete_guild_loop_atomic"(bigint, bigint, integer, integer, bigint, bigint, bigint, bigint, jsonb, bigint[], integer, jsonb, timestamptz) to service_role;
