create or replace function "player"."guild_warehouse_authorized"(p_guild_id bigint, p_actor_id bigint)
returns boolean
language plpgsql
security invoker
set search_path to ''
as $function$
declare
    v_rank integer;
    v_leader_id bigint;
    v_allowed boolean;
begin
    select "leader_id" into v_leader_id from "player"."guilds" where "id" = p_guild_id for update;
    if not found then
        return false;
    end if;
    select "rank" into v_rank from "player"."guild_members"
    where "guild_id" = p_guild_id and "character_id" = p_actor_id for update;
    if not found then
        return false;
    end if;
    if v_rank = 1 and v_leader_id = p_actor_id then
        return true;
    end if;
    if v_rank not between 2 and 6 then
        return false;
    end if;
    select "can_slot" into v_allowed from "player"."guild_rank_settings"
    where "guild_id" = p_guild_id and "rank" = v_rank for update;
    return coalesce(v_allowed, false);
end;
$function$;

create or replace function "player"."move_guild_warehouse_atomic"(
    p_guild_id bigint, p_actor_id bigint, p_source_id bigint, p_source_index integer,
    p_target_index integer, p_template_id integer, p_max_stack integer, p_page_count integer
)
returns table(source_id bigint, target_id bigint, source_count integer, target_count integer, mode text)
language plpgsql
security invoker
set search_path to ''
as $function$
declare
    v_source "player"."guild_warehouse"%rowtype;
    v_target "player"."guild_warehouse"%rowtype;
    v_page_count integer;
    v_combined bigint;
begin
    perform 1 from "player"."guilds" where "id" = p_guild_id for update;
    if not found or not "player"."guild_warehouse_authorized"(p_guild_id, p_actor_id) then
        raise exception 'GUILD_WAREHOUSE_UNAUTHORIZED';
    end if;
    v_page_count := coalesce(("player"."load_latest_guild_log_details"(p_guild_id, 9002)->>'pageCount')::integer, 1);
    if v_page_count <> p_page_count or v_page_count not between 1 and 4 or p_source_index < 0 or p_target_index < 0
       or p_source_index >= v_page_count * 30 or p_target_index >= v_page_count * 30
       or p_source_index = p_target_index or p_max_stack <= 0 then
        raise exception 'INVALID_GUILD_WAREHOUSE_MOVE';
    end if;
    perform 1 from "player"."guild_warehouse"
    where "guild_id" = p_guild_id and "slot_index" in (p_source_index, p_target_index)
    order by "id" for update;
    select * into v_source from "player"."guild_warehouse"
    where "guild_id" = p_guild_id and "id" = p_source_id and "slot_index" = p_source_index;
    if not found or v_source."template_id" <> p_template_id or v_source."stack_count" <= 0 or v_source."stack_count" > p_max_stack then
        raise exception 'INVALID_GUILD_WAREHOUSE_MOVE';
    end if;
    select * into v_target from "player"."guild_warehouse"
    where "guild_id" = p_guild_id and "slot_index" = p_target_index;
    if not found then
        update "player"."guild_warehouse" set "slot_index" = p_target_index where "id" = v_source."id";
        return query select v_source."id", 0::bigint, v_source."stack_count", 0, 'move'::text;
        return;
    end if;
    if v_target."template_id" <> v_source."template_id" or v_target."stack_count" <= 0 or v_target."stack_count" >= p_max_stack then
        raise exception 'INVALID_GUILD_WAREHOUSE_MOVE';
    end if;
    v_combined := v_source."stack_count"::bigint + v_target."stack_count";
    if v_combined <= p_max_stack then
        update "player"."guild_warehouse" set "stack_count" = v_combined::integer where "id" = v_target."id";
        delete from "player"."guild_warehouse" where "id" = v_source."id";
        return query select v_source."id", v_target."id", 0, v_combined::integer, 'merge'::text;
        return;
    end if;
    update "player"."guild_warehouse" set "stack_count" = p_max_stack where "id" = v_target."id";
    update "player"."guild_warehouse" set "stack_count" = (v_combined - p_max_stack)::integer where "id" = v_source."id";
    return query select v_source."id", v_target."id", (v_combined - p_max_stack)::integer, p_max_stack, 'partial'::text;
end;
$function$;

create or replace function "player"."drop_guild_warehouse_atomic"(p_guild_id bigint, p_actor_id bigint, p_slot_id bigint)
returns table(template_id integer, stack_count integer)
language plpgsql
security invoker
set search_path to ''
as $function$
begin
    perform 1 from "player"."guilds" where "id" = p_guild_id for update;
    if not found or not "player"."guild_warehouse_authorized"(p_guild_id, p_actor_id) then
        raise exception 'GUILD_WAREHOUSE_UNAUTHORIZED';
    end if;
    return query delete from "player"."guild_warehouse" as w
    where w."guild_id" = p_guild_id and w."id" = p_slot_id
    returning w."template_id", w."stack_count";
end;
$function$;

revoke execute on function "player"."guild_warehouse_authorized"(bigint, bigint) from public, anon, authenticated;
revoke execute on function "player"."move_guild_warehouse_atomic"(bigint, bigint, bigint, integer, integer, integer, integer, integer) from public, anon, authenticated;
revoke execute on function "player"."drop_guild_warehouse_atomic"(bigint, bigint, bigint) from public, anon, authenticated;
grant execute on function "player"."guild_warehouse_authorized"(bigint, bigint) to service_role;
grant execute on function "player"."move_guild_warehouse_atomic"(bigint, bigint, bigint, integer, integer, integer, integer, integer) to service_role;
grant execute on function "player"."drop_guild_warehouse_atomic"(bigint, bigint, bigint) to service_role;
