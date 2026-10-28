drop function if exists "player"."finish_guild_building_atomic"(bigint, bigint, bigint, integer, bigint, bigint, bigint, integer, integer, integer);

create or replace function "player"."finish_guild_building_atomic"(
    p_guild_id bigint, p_actor_id bigint, p_building_id bigint, p_target_template_id integer,
    p_expected_revision bigint, p_money_cost bigint, p_experience_cost bigint,
    p_general_cost integer, p_rare_cost integer, p_special_cost integer, p_hall_level integer
)
returns table(id bigint, guild_id bigint, extend_id integer, template_id integer, pending_template_id integer, mode integer, actor_id bigint, revision bigint, created_at timestamp with time zone, updated_at timestamp with time zone)
language plpgsql
security invoker
set search_path to ''
as $function$
declare
    v_guild "player"."guilds"%rowtype;
    v_building "player"."guild_buildings"%rowtype;
    v_material_ids integer[] := array[2026, 2027, 2039];
    v_material_costs integer[] := array[p_general_cost, p_rare_cost, p_special_cost];
    v_index integer;
    v_remaining integer;
    v_slot "player"."guild_warehouse"%rowtype;
    v_take integer;
begin
    if p_money_cost < 0 or p_experience_cost < 0 or p_general_cost < 0 or p_rare_cost < 0 or p_special_cost < 0 or p_hall_level < 0 then
        raise exception 'INVALID_GUILD_BUILDING_COST';
    end if;
    select * into v_guild from "player"."guilds" as g
    where g."id" = p_guild_id and g."leader_id" = p_actor_id for update;
    if not found or not exists (
        select 1 from "player"."guild_members" as m
        where m."guild_id" = p_guild_id and m."character_id" = p_actor_id and m."rank" = 1
    ) then
        raise exception 'GUILD_BUILDING_UNAUTHORIZED';
    end if;
    select * into v_building from "player"."guild_buildings" as b
    where b."guild_id" = p_guild_id and b."id" = p_building_id for update;
    if not found or v_building."pending_template_id" <> p_target_template_id or v_building."revision" <> p_expected_revision then
        raise exception 'INVALID_GUILD_BUILDING_FINISH';
    end if;
    if (p_hall_level > 0 and not exists (
        select 1 from "data"."data_tbl_building" as t
        where t."id" = p_target_template_id and t."code_name" = 'BUILD100001'
          and t."level" = p_hall_level and t."prop1" = 4 and t."prop1_value" = p_hall_level
    )) or (p_hall_level = 0 and exists (
        select 1 from "data"."data_tbl_building" as t
        where t."id" = p_target_template_id and t."code_name" = 'BUILD100001'
    )) or (p_hall_level > 0 and (p_hall_level < v_guild."level" or p_hall_level > v_guild."level" + 1)) then
        raise exception 'INVALID_GUILD_HALL_LEVEL';
    end if;
    if v_guild."funds" < p_money_cost or v_guild."experience" < p_experience_cost then
        raise exception 'INSUFFICIENT_GUILD_BUILDING_RESOURCES';
    end if;
    for v_index in 1..3 loop
        v_remaining := v_material_costs[v_index];
        if coalesce((select sum(w."stack_count") from "player"."guild_warehouse" as w where w."guild_id" = p_guild_id and w."template_id" = v_material_ids[v_index]), 0) < v_remaining then
            raise exception 'INSUFFICIENT_GUILD_BUILDING_MATERIALS';
        end if;
        for v_slot in select * from "player"."guild_warehouse" as w
            where w."guild_id" = p_guild_id and w."template_id" = v_material_ids[v_index]
            order by w."id" for update
        loop
            exit when v_remaining = 0;
            v_take := least(v_remaining, v_slot."stack_count");
            if v_take = v_slot."stack_count" then
                delete from "player"."guild_warehouse" as w where w."id" = v_slot."id";
            else
                update "player"."guild_warehouse" as w set "stack_count" = w."stack_count" - v_take where w."id" = v_slot."id";
            end if;
            v_remaining := v_remaining - v_take;
        end loop;
    end loop;
    update "player"."guilds" as g
    set "funds" = g."funds" - p_money_cost, "experience" = g."experience" - p_experience_cost,
        "level" = case when p_hall_level > 0 then p_hall_level else g."level" end
    where g."id" = p_guild_id;
    update "player"."guild_buildings" as b
    set "template_id" = p_target_template_id, "pending_template_id" = null, "mode" = null, "actor_id" = null,
        "revision" = b."revision" + 1, "updated_at" = now()
    where b."id" = p_building_id returning * into v_building;
    return query select v_building."id", v_building."guild_id", v_building."extend_id", coalesce(v_building."template_id", 0), 0, 0, 0::bigint, v_building."revision", v_building."created_at", v_building."updated_at";
end;
$function$;

create or replace function "player"."disband_guild_atomic"(p_guild_id bigint, p_leader_id bigint, p_expected_member_ids bigint[])
returns jsonb
language plpgsql
security invoker
set search_path to ''
as $function$
declare
    v_guild "player"."guilds"%rowtype;
    v_member_ids bigint[];
    v_character_id bigint;
    v_normal bigint;
    v_donate bigint;
    v_restore_normal bigint;
    v_restore_donate bigint;
    v_characters jsonb := '[]'::jsonb;
    v_rows integer;
begin
    if p_guild_id <= 0 or p_leader_id <= 0 or cardinality(p_expected_member_ids) <= 0 then
        raise exception 'INVALID_GUILD_DISBAND';
    end if;
    select * into v_guild from "player"."guilds" as g where g."id" = p_guild_id for update;
    if not found then raise exception 'GUILD_NOT_FOUND'; end if;
    if v_guild."leader_id" <> p_leader_id then raise exception 'GUILD_UNAUTHORIZED'; end if;

    select array_agg(m."character_id" order by m."character_id") into v_member_ids
    from "player"."guild_members" as m where m."guild_id" = p_guild_id;
    if v_member_ids is distinct from (
        select array_agg(expected_id order by expected_id) from unnest(p_expected_member_ids) as expected(expected_id)
    ) or cardinality(v_member_ids) <> v_guild."population" then
        raise exception 'MEMBERSHIP_DRIFT';
    end if;

    foreach v_character_id in array v_member_ids loop
        perform pg_advisory_xact_lock(hashtextextended('guild-character:' || v_character_id::text, 0));
    end loop;
    perform 1 from "player"."guild_members" as m where m."guild_id" = p_guild_id order by m."character_id" for update;
    perform 1 from "player"."characters" as c where c."id" = any(v_member_ids) and c."guild_id" = p_guild_id order by c."id" for update;
    if (select count(*) from "player"."characters" as c where c."id" = any(v_member_ids) and c."guild_id" = p_guild_id) <> cardinality(v_member_ids) then
        raise exception 'MEMBERSHIP_DRIFT';
    end if;
    perform 1 from "player"."character_wallet" as w where w."character_id" = any(v_member_ids) order by w."character_id" for update;
    if (select count(*) from "player"."character_wallet" as w where w."character_id" = any(v_member_ids)) <> cardinality(v_member_ids) then
        raise exception 'MEMBERSHIP_DRIFT';
    end if;

    foreach v_character_id in array v_member_ids loop
        insert into "player"."character_currencies" ("character_id", "currency_type", "amount", "season")
        values (v_character_id, 20, 0, 0), (v_character_id, 21, 0, 0)
        on conflict ("character_id", "currency_type", "season") do nothing;
    end loop;
    perform 1 from "player"."character_currencies" as cc
    where cc."character_id" = any(v_member_ids) and cc."currency_type" in (20, 21) and cc."season" = 0
    order by cc."character_id", cc."currency_type" for update;
    if (select count(*) from "player"."character_currencies" as cc where cc."character_id" = any(v_member_ids) and cc."currency_type" in (20, 21) and cc."season" = 0) <> cardinality(v_member_ids) * 2 then
        raise exception 'MEMBERSHIP_DRIFT';
    end if;

    foreach v_character_id in array v_member_ids loop
        select cc."amount" into v_normal from "player"."character_currencies" as cc
        where cc."character_id" = v_character_id and cc."currency_type" = 20 and cc."season" = 0;
        select cc."amount" into v_donate from "player"."character_currencies" as cc
        where cc."character_id" = v_character_id and cc."currency_type" = 21 and cc."season" = 0;
        v_restore_normal := v_normal * 9 / 10;
        v_restore_donate := v_donate * 9 / 10;
        if v_restore_normal > 2147483647 or v_restore_donate > 2147483647 then
            raise exception 'GUILD_RESTORE_OVERFLOW';
        end if;
        update "player"."character_currencies" set "amount" = 0
        where "character_id" = v_character_id and "currency_type" in (20, 21) and "season" = 0;
        get diagnostics v_rows = row_count;
        if v_rows <> 2 then raise exception 'MEMBERSHIP_DRIFT'; end if;
        update "player"."characters"
        set "guild_id" = null, "guild_restore_contrib" = v_restore_normal, "guild_restore_donate" = v_restore_donate
        where "id" = v_character_id and "guild_id" = p_guild_id;
        get diagnostics v_rows = row_count;
        if v_rows <> 1 then raise exception 'MEMBERSHIP_DRIFT'; end if;
        v_characters := v_characters || jsonb_build_array("player"."guild_character_state_json"(v_character_id));
    end loop;

    delete from "player"."guilds" where "id" = p_guild_id and "leader_id" = p_leader_id;
    get diagnostics v_rows = row_count;
    if v_rows <> 1 then raise exception 'MEMBERSHIP_DRIFT'; end if;
    return jsonb_build_object('guild_id', p_guild_id, 'characters', v_characters);
end;
$function$;

revoke execute on function "player"."finish_guild_building_atomic"(bigint, bigint, bigint, integer, bigint, bigint, bigint, integer, integer, integer, integer) from public, anon, authenticated;
revoke execute on function "player"."disband_guild_atomic"(bigint, bigint, bigint[]) from public, anon, authenticated;
grant execute on function "player"."finish_guild_building_atomic"(bigint, bigint, bigint, integer, bigint, bigint, bigint, integer, integer, integer, integer) to service_role;
grant execute on function "player"."disband_guild_atomic"(bigint, bigint, bigint[]) to service_role;
