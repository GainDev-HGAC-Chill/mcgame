create table if not exists "player"."guild_rank_settings" (
    "guild_id" bigint not null references "player"."guilds"("id") on delete cascade,
    "rank" integer not null check ("rank" between 1 and 6),
    "name" varchar(50) not null check (length("name") > 0),
    "can_add" boolean not null,
    "can_quest" boolean not null,
    "can_slot" boolean not null,
    "can_info" boolean not null,
    "can_del" boolean not null,
    "can_duty" boolean not null,
    primary key ("guild_id", "rank")
);

alter table "player"."guild_rank_settings" enable row level security;

create or replace function "player"."list_guild_rank_settings"(p_guild_id bigint)
returns table(
    rank integer,
    name varchar,
    can_add boolean,
    can_quest boolean,
    can_slot boolean,
    can_info boolean,
    can_del boolean,
    can_duty boolean
)
language sql
security invoker
set search_path to ''
as $function$
    select s."rank", s."name", s."can_add", s."can_quest", s."can_slot", s."can_info", s."can_del", s."can_duty"
    from "player"."guild_rank_settings" as s
    where s."guild_id" = p_guild_id
    order by s."rank";
$function$;

create or replace function "player"."replace_guild_rank_settings"(p_guild_id bigint, p_settings jsonb)
returns boolean
language plpgsql
security invoker
set search_path to ''
as $function$
declare
    v_count integer;
begin
    if jsonb_typeof(p_settings) <> 'array' or jsonb_array_length(p_settings) <> 6 then
        return false;
    end if;

    perform 1 from "player"."guilds" where "id" = p_guild_id for update;
    if not found then
        return false;
    end if;

    select count(distinct x."rank")
    into v_count
    from jsonb_to_recordset(p_settings) as x(
        "rank" integer,
        "name" text,
        "canAdd" boolean,
        "canQuest" boolean,
        "canSlot" boolean,
        "canInfo" boolean,
        "canDel" boolean,
        "canDuty" boolean
    )
    where x."rank" between 1 and 6
      and length(x."name") > 0
      and (x."rank" <> 1 or (x."canAdd" and x."canQuest" and x."canSlot" and x."canInfo" and x."canDel" and x."canDuty"));

    if v_count <> 6 then
        return false;
    end if;

    delete from "player"."guild_rank_settings" where "guild_id" = p_guild_id;
    insert into "player"."guild_rank_settings" (
        "guild_id", "rank", "name", "can_add", "can_quest", "can_slot", "can_info", "can_del", "can_duty"
    )
    select p_guild_id, x."rank", x."name", x."canAdd", x."canQuest", x."canSlot", x."canInfo", x."canDel", x."canDuty"
    from jsonb_to_recordset(p_settings) as x(
        "rank" integer,
        "name" text,
        "canAdd" boolean,
        "canQuest" boolean,
        "canSlot" boolean,
        "canInfo" boolean,
        "canDel" boolean,
        "canDuty" boolean
    );
    return true;
end;
$function$;

create or replace function "player"."create_guild_atomic"(
    p_leader_id bigint,
    p_name text,
    p_cost bigint,
    p_rank_settings jsonb
)
returns table(
    guild_id bigint,
    guild_name text,
    money bigint,
    money_bind bigint,
    gold bigint,
    guild_contrib bigint,
    donate_contrib bigint,
    restore_normal bigint,
    restore_donate bigint
)
language plpgsql
security invoker
set search_path to ''
as $function$
declare
    v_character "player"."characters"%rowtype;
    v_wallet "player"."character_wallet"%rowtype;
    v_level integer;
    v_guild_id bigint;
    v_normal bigint;
    v_donate bigint;
begin
    perform pg_advisory_xact_lock(hashtextextended('guild-character:' || p_leader_id::text, 0));
    if p_cost <> 1000000 or p_name is null or length(btrim(p_name)) = 0 or length(p_name) > 50 then
        raise exception 'INSUFFICIENT_FUNDS';
    end if;

    select * into v_character
    from "player"."characters"
    where "id" = p_leader_id
    for update;
    if not found then
        raise exception 'NOT_IN_GUILD';
    end if;
    select "level" into v_level from "player"."character_progression" where "character_id" = p_leader_id for update;
    if not found or v_level < 50 then
        raise exception 'INSUFFICIENT_LEVEL';
    end if;
    select * into v_wallet from "player"."character_wallet" where "character_id" = p_leader_id for update;
    if not found then
        raise exception 'INSUFFICIENT_FUNDS';
    end if;
    if v_character."guild_id" is not null or exists (
        select 1 from "player"."guild_members" where "character_id" = p_leader_id
    ) or exists (select 1 from "player"."guild_applications" where "character_id" = p_leader_id) then
        raise exception 'ALREADY_IN_GUILD';
    end if;
    if exists (select 1 from "player"."guilds" where "name" = p_name) then
        raise exception 'GUILD_NAME_TAKEN';
    end if;
    if v_wallet."money_bind" >= p_cost then
        v_wallet."money_bind" := v_wallet."money_bind" - p_cost;
    elsif v_wallet."money" >= p_cost then
        v_wallet."money" := v_wallet."money" - p_cost;
    else
        raise exception 'INSUFFICIENT_FUNDS';
    end if;

    insert into "player"."guilds" (
        "name", "leader_id", "level", "experience", "population", "max_population", "icon", "funds",
        "contribution_total", "announcement", "description", "join_level_req", "join_approval_required", "activity_points"
    ) values (p_name, p_leader_id, 1, 0, 1, 50, 0, 0, 0, '', '', 1, true, 0)
    returning "id" into v_guild_id;

    insert into "player"."guild_members" ("guild_id", "character_id", "rank")
    values (v_guild_id, p_leader_id, 1);

    if not "player"."replace_guild_rank_settings"(v_guild_id, p_rank_settings) then
        raise exception 'INVALID_RANK_SETTINGS';
    end if;

    insert into "player"."character_currencies" ("character_id", "currency_type", "amount", "season")
    values (p_leader_id, 20, v_character."guild_restore_contrib", 0)
    on conflict ("character_id", "currency_type", "season")
    do update set "amount" = "player"."character_currencies"."amount" + excluded."amount"
    returning "amount" into v_normal;

    insert into "player"."character_currencies" ("character_id", "currency_type", "amount", "season")
    values (p_leader_id, 21, v_character."guild_restore_donate", 0)
    on conflict ("character_id", "currency_type", "season")
    do update set "amount" = "player"."character_currencies"."amount" + excluded."amount"
    returning "amount" into v_donate;

    update "player"."guild_members" as m
    set "contribution_normal" = v_normal,
        "contribution_donate" = v_donate,
        "contribution_total" = v_normal + v_donate
    where m."guild_id" = v_guild_id and m."character_id" = p_leader_id;

    update "player"."character_wallet"
    set "money" = v_wallet."money", "money_bind" = v_wallet."money_bind"
    where "character_id" = p_leader_id;

    update "player"."characters"
    set "guild_id" = v_guild_id,
        "guild_restore_contrib" = 0,
        "guild_restore_donate" = 0
    where "id" = p_leader_id;

    return query select v_guild_id, p_name, v_wallet."money", v_wallet."money_bind", v_wallet."gold",
        v_normal, v_donate, 0::bigint, 0::bigint;
end;
$function$;

create or replace function "player"."contribute_guild_money_atomic"(
    p_guild_id bigint,
    p_character_id bigint,
    p_silver bigint,
    p_gold bigint
)
returns table(
    character_name text,
    money bigint,
    money_bind bigint,
    gold bigint,
    guild_contrib bigint,
    donate_contrib bigint,
    restore_normal bigint,
    restore_donate bigint,
    guild_funds bigint,
    member_donate bigint,
    member_total bigint,
    member_weekly bigint
)
language plpgsql
security invoker
set search_path to ''
as $function$
declare
    v_character "player"."characters"%rowtype;
    v_wallet "player"."character_wallet"%rowtype;
    v_member "player"."guild_members"%rowtype;
    v_funds_gain bigint;
    v_donate_gain bigint;
    v_donate_balance bigint;
    v_guild_funds bigint;
begin
    perform pg_advisory_xact_lock(hashtextextended('guild-character:' || p_character_id::text, 0));
    if p_silver < 0 or p_gold < 0 or (p_silver = 0 and p_gold = 0) or p_gold > 4611686018427387
       or p_silver > 9223372036854775807 - p_gold * 2000 then
        raise exception 'INSUFFICIENT_FUNDS';
    end if;
    v_funds_gain := p_silver + p_gold * 2000;
    v_donate_gain := v_funds_gain / 8;

    perform 1 from "player"."guilds" where "id" = p_guild_id for update;
    if not found then
        raise exception 'NOT_IN_GUILD';
    end if;
    select * into v_character from "player"."characters" where "id" = p_character_id for update;
    if not found or v_character."guild_id" is distinct from p_guild_id then
        raise exception 'NOT_IN_GUILD';
    end if;
    select * into v_wallet from "player"."character_wallet" where "character_id" = p_character_id for update;
    if not found then
        raise exception 'INSUFFICIENT_FUNDS';
    end if;
    select * into v_member
    from "player"."guild_members"
    where "guild_id" = p_guild_id and "character_id" = p_character_id
    for update;
    if not found then
        raise exception 'NOT_IN_GUILD';
    end if;
    if v_wallet."money" < p_silver or v_wallet."gold" < p_gold then
        raise exception 'INSUFFICIENT_FUNDS';
    end if;

    update "player"."character_wallet" as w
    set "money" = w."money" - p_silver,
        "gold" = w."gold" - p_gold
    where w."character_id" = p_character_id;

    insert into "player"."character_currencies" ("character_id", "currency_type", "amount", "season")
    values (p_character_id, 21, v_donate_gain, 0)
    on conflict ("character_id", "currency_type", "season")
    do update set "amount" = "player"."character_currencies"."amount" + excluded."amount"
    returning "amount" into v_donate_balance;

    update "player"."guild_members"
    set "contribution_donate" = "contribution_donate" + v_donate_gain,
        "contribution_total" = "contribution_total" + v_donate_gain,
        "contribution_weekly" = "contribution_weekly" + v_donate_gain
    where "id" = v_member."id"
    returning "contribution_donate", "contribution_total", "contribution_weekly"
    into v_member."contribution_donate", v_member."contribution_total", v_member."contribution_weekly";

    update "player"."guilds"
    set "funds" = "funds" + v_funds_gain,
        "contribution_total" = "contribution_total" + v_donate_gain
    where "id" = p_guild_id
    returning "funds" into v_guild_funds;

    return query select v_character."name"::text, v_wallet."money" - p_silver, v_wallet."money_bind",
        v_wallet."gold" - p_gold,
        coalesce((select "amount" from "player"."character_currencies" where "character_id" = p_character_id and "currency_type" = 20 and "season" = 0), 0),
        v_donate_balance, v_character."guild_restore_contrib"::bigint, v_character."guild_restore_donate"::bigint,
        v_guild_funds, v_member."contribution_donate", v_member."contribution_total", v_member."contribution_weekly";
end;
$function$;

create or replace function "player"."contribute_guild_material_atomic"(
    p_guild_id bigint,
    p_character_id bigint,
    p_item_id bigint,
    p_item_template_id integer,
    p_item_count integer,
    p_warehouse_template_id integer,
    p_warehouse_count integer,
    p_warehouse_max_stack integer,
    p_warehouse_page_count integer,
    p_donate_gain bigint
)
returns table(
    character_name text,
    item_remaining integer,
    warehouse_slot_id bigint,
    warehouse_slot_index integer,
    warehouse_stack_count integer,
    money bigint,
    money_bind bigint,
    gold bigint,
    guild_contrib bigint,
    donate_contrib bigint,
    restore_normal bigint,
    restore_donate bigint,
    member_donate bigint,
    member_total bigint,
    member_weekly bigint
)
language plpgsql
security invoker
set search_path to ''
as $function$
declare
    v_character "player"."characters"%rowtype;
    v_wallet "player"."character_wallet"%rowtype;
    v_member "player"."guild_members"%rowtype;
    v_item "player"."character_items"%rowtype;
    v_item_remaining integer;
    v_donate_balance bigint;
    v_slot_id bigint;
    v_slot_index integer;
    v_slot_count integer;
begin
    perform pg_advisory_xact_lock(hashtextextended('guild-character:' || p_character_id::text, 0));
    if p_item_count <= 0 or p_warehouse_count <= 0 or p_warehouse_max_stack <= 0
       or p_warehouse_count > p_warehouse_max_stack or p_warehouse_page_count not between 1 and 4
       or p_donate_gain < 0 then
        raise exception 'INVALID_CONTRIBUTION';
    end if;
    perform 1 from "player"."guilds" where "id" = p_guild_id for update;
    if not found then
        raise exception 'NOT_IN_GUILD';
    end if;
    select * into v_character from "player"."characters" where "id" = p_character_id for update;
    if not found or v_character."guild_id" is distinct from p_guild_id then
        raise exception 'NOT_IN_GUILD';
    end if;
    select * into v_wallet from "player"."character_wallet" where "character_id" = p_character_id for update;
    if not found then
        raise exception 'INSUFFICIENT_FUNDS';
    end if;
    select * into v_member
    from "player"."guild_members"
    where "guild_id" = p_guild_id and "character_id" = p_character_id
    for update;
    if not found then
        raise exception 'NOT_IN_GUILD';
    end if;
    select * into v_item
    from "player"."character_items"
    where "id" = p_item_id and "character_id" = p_character_id
    for update;
    if not found or v_item."template_id" <> p_item_template_id or v_item."slot_type" <> 0 or v_item."stack_count" < p_item_count then
        raise exception 'INVALID_CONTRIBUTION';
    end if;

    v_item_remaining := v_item."stack_count" - p_item_count;
    if v_item_remaining = 0 then
        delete from "player"."character_items" where "id" = p_item_id;
    else
        update "player"."character_items" set "stack_count" = v_item_remaining where "id" = p_item_id;
    end if;

    if coalesce(("player"."load_latest_guild_log_details"(p_guild_id, 9002)->>'pageCount')::integer, 1) <> p_warehouse_page_count then
        raise exception 'INVALID_CONTRIBUTION';
    end if;
    perform 1 from "player"."guild_warehouse" as w where w."guild_id" = p_guild_id order by w."id" for update;
    select w."id", w."slot_index", w."stack_count"
    into v_slot_id, v_slot_index, v_slot_count
    from "player"."guild_warehouse" as w
    where w."guild_id" = p_guild_id and w."template_id" = p_warehouse_template_id
      and w."stack_count" > 0
      and w."stack_count"::bigint + p_warehouse_count::bigint <= p_warehouse_max_stack::bigint
    order by w."id"
    limit 1
    for update;
    if found then
        update "player"."guild_warehouse"
        set "stack_count" = "stack_count" + p_warehouse_count
        where "id" = v_slot_id
        returning "stack_count" into v_slot_count;
    else
        select min(candidate.slot_index)
        into v_slot_index
        from (
            select series.slot_index
            from generate_series(0, p_warehouse_page_count * 30 - 1) as series(slot_index)
            where not exists (
                select 1 from "player"."guild_warehouse" as occupied
                where occupied."guild_id" = p_guild_id and occupied."slot_index" = series.slot_index
            )
        ) as candidate;
        if v_slot_index is null then
            raise exception 'GUILD_WAREHOUSE_FULL';
        end if;
        insert into "player"."guild_warehouse" ("guild_id", "slot_index", "template_id", "stack_count", "deposited_by")
        values (p_guild_id, v_slot_index, p_warehouse_template_id, p_warehouse_count, p_character_id)
        returning "id", "stack_count" into v_slot_id, v_slot_count;
    end if;

    insert into "player"."character_currencies" ("character_id", "currency_type", "amount", "season")
    values (p_character_id, 21, p_donate_gain, 0)
    on conflict ("character_id", "currency_type", "season")
    do update set "amount" = "player"."character_currencies"."amount" + excluded."amount"
    returning "amount" into v_donate_balance;

    update "player"."guild_members"
    set "contribution_donate" = "contribution_donate" + p_donate_gain,
        "contribution_total" = "contribution_total" + p_donate_gain,
        "contribution_weekly" = "contribution_weekly" + p_donate_gain
    where "id" = v_member."id"
    returning "contribution_donate", "contribution_total", "contribution_weekly"
    into v_member."contribution_donate", v_member."contribution_total", v_member."contribution_weekly";

    update "player"."guilds"
    set "contribution_total" = "contribution_total" + p_donate_gain
    where "id" = p_guild_id;

    return query select v_character."name"::text, v_item_remaining, v_slot_id, v_slot_index, v_slot_count,
        v_wallet."money", v_wallet."money_bind", v_wallet."gold",
        coalesce((select "amount" from "player"."character_currencies" where "character_id" = p_character_id and "currency_type" = 20 and "season" = 0), 0),
        v_donate_balance, v_character."guild_restore_contrib"::bigint, v_character."guild_restore_donate"::bigint,
        v_member."contribution_donate", v_member."contribution_total", v_member."contribution_weekly";
end;
$function$;

create or replace function "player"."develop_guild_skill_atomic"(
    p_guild_id bigint, p_actor_id bigint, p_skill_id integer, p_previous_id integer,
    p_level integer, p_exp_cost bigint, p_money_cost bigint, p_family_ids integer[]
)
returns boolean
language plpgsql
security invoker
set search_path to ''
as $function$
declare
    v_guild "player"."guilds"%rowtype;
    v_count integer;
    v_current integer;
begin
    if p_level < 1 or p_level > 6 or p_exp_cost < 0 or p_money_cost < 0 or p_family_ids is null
       or cardinality(p_family_ids) <> 6
       or (select count(distinct family_id) from unnest(p_family_ids) as family(family_id)) <> 6
       or not (p_skill_id = any(p_family_ids)) then
        return false;
    end if;
    select * into v_guild from "player"."guilds" where "id" = p_guild_id for update;
    if not found or v_guild."leader_id" <> p_actor_id then
        return false;
    end if;
    if not exists (
        select 1 from "player"."guild_members"
        where "guild_id" = p_guild_id and "character_id" = p_actor_id and "rank" = 1
    ) then
        return false;
    end if;
    select count(*), coalesce(min("skill_id"), 0) into v_count, v_current
    from "player"."guild_skills"
    where "guild_id" = p_guild_id and "skill_id" = any(p_family_ids);
    if (p_previous_id = 0 and (v_count <> 0 or p_level <> 1))
       or (p_previous_id <> 0 and (v_count <> 1 or v_current <> p_previous_id or p_level <= 1)) then
        return false;
    end if;
    if v_guild."experience" < p_exp_cost or v_guild."funds" < p_money_cost then
        return false;
    end if;
    update "player"."guilds"
    set "experience" = "experience" - p_exp_cost, "funds" = "funds" - p_money_cost
    where "id" = p_guild_id;
    if p_previous_id <> 0 then
        delete from "player"."guild_skills" where "guild_id" = p_guild_id and "skill_id" = p_previous_id;
    end if;
    insert into "player"."guild_skills" ("guild_id", "skill_id", "level") values (p_guild_id, p_skill_id, p_level);
    return true;
end;
$function$;

create or replace function "player"."set_guild_name"(p_guild_id bigint, p_actor_id bigint, p_name text)
returns boolean
language sql
security invoker
set search_path to ''
as $function$
    with changed as (
        update "player"."guilds"
        set "name" = p_name
        where "id" = p_guild_id and "leader_id" = p_actor_id and length(btrim(p_name)) > 0
        returning 1
    )
    select exists(select 1 from changed);
$function$;

create or replace function "player"."guild_permission_authorized"(p_guild_id bigint, p_actor_id bigint, p_permission text)
returns boolean
language plpgsql
security invoker
set search_path to ''
as $function$
declare
    v_leader_id bigint;
    v_rank integer;
    v_allowed boolean;
begin
    select g."leader_id" into v_leader_id from "player"."guilds" as g where g."id" = p_guild_id for update;
    if not found then return false; end if;
    select m."rank" into v_rank from "player"."guild_members" as m where m."guild_id" = p_guild_id and m."character_id" = p_actor_id for update;
    if not found then return false; end if;
    if v_rank = 1 and v_leader_id = p_actor_id then return true; end if;
    if v_rank not between 2 and 6 or p_permission not in ('add', 'del', 'info', 'duty') then return false; end if;
    if p_permission = 'add' then
        select r."can_add" into v_allowed from "player"."guild_rank_settings" as r where r."guild_id" = p_guild_id and r."rank" = v_rank for update;
    elsif p_permission = 'del' then
        select r."can_del" into v_allowed from "player"."guild_rank_settings" as r where r."guild_id" = p_guild_id and r."rank" = v_rank for update;
    elsif p_permission = 'info' then
        select r."can_info" into v_allowed from "player"."guild_rank_settings" as r where r."guild_id" = p_guild_id and r."rank" = v_rank for update;
    else
        select r."can_duty" into v_allowed from "player"."guild_rank_settings" as r where r."guild_id" = p_guild_id and r."rank" = v_rank for update;
    end if;
    return coalesce(v_allowed, false);
end;
$function$;

create or replace function "player"."replace_guild_rank_settings_authorized"(p_guild_id bigint, p_actor_id bigint, p_settings jsonb)
returns boolean
language plpgsql
security invoker
set search_path to ''
as $function$
begin
    perform 1 from "player"."guilds" as g where g."id" = p_guild_id and g."leader_id" = p_actor_id for update;
    if not found then return false; end if;
    return "player"."replace_guild_rank_settings"(p_guild_id, p_settings);
end;
$function$;

create or replace function "player"."set_guild_notice"(p_guild_id bigint, p_actor_id bigint, p_announcement text, p_description text)
returns boolean
language plpgsql
security invoker
set search_path to ''
as $function$
begin
    perform 1 from "player"."guilds" as g where g."id" = p_guild_id for update;
    if not found or not "player"."guild_permission_authorized"(p_guild_id, p_actor_id, 'info') then return false; end if;
    update "player"."guilds" set "announcement" = p_announcement, "description" = p_description where "id" = p_guild_id;
    return true;
end;
$function$;

revoke all on table "player"."guild_rank_settings" from public, anon, authenticated;
grant select, insert, update, delete on table "player"."guild_rank_settings" to service_role;
revoke execute on function "player"."list_guild_rank_settings"(bigint) from public, anon, authenticated;
revoke execute on function "player"."replace_guild_rank_settings"(bigint, jsonb) from public, anon, authenticated;
revoke execute on function "player"."create_guild_atomic"(bigint, text, bigint, jsonb) from public, anon, authenticated;
revoke execute on function "player"."contribute_guild_money_atomic"(bigint, bigint, bigint, bigint) from public, anon, authenticated;
revoke execute on function "player"."contribute_guild_material_atomic"(bigint, bigint, bigint, integer, integer, integer, integer, integer, integer, bigint) from public, anon, authenticated;
revoke execute on function "player"."develop_guild_skill_atomic"(bigint, bigint, integer, integer, integer, bigint, bigint, integer[]) from public, anon, authenticated;
revoke execute on function "player"."set_guild_name"(bigint, bigint, text) from public, anon, authenticated;
revoke execute on function "player"."guild_permission_authorized"(bigint, bigint, text) from public, anon, authenticated;
revoke execute on function "player"."replace_guild_rank_settings_authorized"(bigint, bigint, jsonb) from public, anon, authenticated;
revoke execute on function "player"."set_guild_notice"(bigint, bigint, text, text) from public, anon, authenticated;
grant execute on function "player"."list_guild_rank_settings"(bigint) to service_role;
grant execute on function "player"."replace_guild_rank_settings"(bigint, jsonb) to service_role;
grant execute on function "player"."create_guild_atomic"(bigint, text, bigint, jsonb) to service_role;
grant execute on function "player"."contribute_guild_money_atomic"(bigint, bigint, bigint, bigint) to service_role;
grant execute on function "player"."contribute_guild_material_atomic"(bigint, bigint, bigint, integer, integer, integer, integer, integer, integer, bigint) to service_role;
grant execute on function "player"."develop_guild_skill_atomic"(bigint, bigint, integer, integer, integer, bigint, bigint, integer[]) to service_role;
grant execute on function "player"."set_guild_name"(bigint, bigint, text) to service_role;
grant execute on function "player"."guild_permission_authorized"(bigint, bigint, text) to service_role;
grant execute on function "player"."replace_guild_rank_settings_authorized"(bigint, bigint, jsonb) to service_role;
grant execute on function "player"."set_guild_notice"(bigint, bigint, text, text) to service_role;
